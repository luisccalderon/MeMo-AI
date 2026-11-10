# Architecture

MeMo-AI is a single Go binary that installs a coding-agent stack into the Pi
(`pi`, from pi.dev) agent directory. It writes its own files, manages one block
inside `APPEND_SYSTEM.md`, merges one entry into `mcp.json`, and sets the
default model keys in `settings.json`. This document describes the install
lifecycle, the data it owns, and the rules that keep re-runs and uninstalls
safe.

## Component map

```text
cmd/memo-ai/            process entrypoint
internal/cli/           flag parsing, commands, interactive wizard
internal/assets/        go:embed of the prompt, skills, prompt templates, extension and agents
internal/install/       asset writes, prompt block, mcp.json merge, default model
internal/uninstall/     safe removal and cleanup
internal/promptblock/   managed block inside a user-owned prompt file
internal/backup/        timestamped snapshots
internal/config/        JSON/JSONC parsing, nested get/set/delete, atomic save
internal/state/         install manifest and SHA-256 tracking
internal/detect/        Pi, Engram and local model discovery
internal/doctor/        read-only health checks
internal/engram/        verified release download and install
internal/pi/            agent directory resolution
internal/term/          colored terminal output
internal/i18n/          English and Spanish strings
internal/banner/        ANSI wordmark
internal/version/       build metadata
```

The runtime has no third-party Go dependencies. Everything embedded in the
binary ships from `internal/assets/files`.

## Paths

All state lives under the Pi agent directory, resolved in
`internal/pi/paths.go` the same way Pi resolves it:

| Condition | Directory |
| --- | --- |
| `PI_CODING_AGENT_DIR` set | that directory |
| default | `~/.pi/agent` |

```text
<agent>/APPEND_SYSTEM.md            managed block only
<agent>/mcp.json                    mcpServers.engram only
<agent>/settings.json               defaultProvider and defaultModel only
<agent>/skills/memo-*/SKILL.md
<agent>/prompts/memo*.md
<agent>/extensions/memo-ai/index.ts
<agent>/memo-ai/state.json
<agent>/memo-ai/backups/
```

`settings.json` is Pi's own settings file. MeMo-AI touches only
`defaultProvider` and `defaultModel`, and only when a model was chosen. A
file that cannot be parsed is never rewritten: the install reports a warning
with the `/model` command to run instead.

### Finding the Pi binary

Detection (`internal/detect.FindPi`) tries, in order: `PI_BIN`, `pi` on
`PATH`, the entrypoint recorded in `<agent dir>/install/managed-install.json`,
the launchers in `<agent dir>/bin`, and the usual Unix install prefixes.

The order matters because the pi.dev installer uses a **managed install**: it
packs each release under `<agent dir>/install/releases/<version>` and writes
`pi`, `pi.cmd` and `pi.ps1` launchers to `<agent dir>/bin`, without adding
that directory to `PATH`. Resolving only `PATH` therefore reports Pi as
missing on a machine where it is installed and working.

## How earlier targets map to Pi

| Concept | Pi mechanism | Why |
| --- | --- | --- |
| Primary orchestrator agent | Block in `APPEND_SYSTEM.md` | Pi has one main session; its prompt is extended, not replaced, so Pi's own tool guidance stays intact. |
| Slash commands | Prompt templates in `prompts/` | Pi exposes each `prompts/*.md` file as `/<name>`. The embedded assets still live under `commands/` and are mapped on install. |
| Subagents | Not supported | Pi has no built-in subagents. The agent assets stay embedded but are never installed. |
| Primary model | `defaultProvider` and `defaultModel` in `settings.json` | Pi reads them at startup. The reference is split at the first `/`, so `vllm/Qwen/Qwen3-Coder` becomes provider `vllm`, model `Qwen/Qwen3-Coder`. |
| Engram | `mcpServers.engram` in `mcp.json` | Pi's native MCP file. |
| Extension | `extensions/memo-ai/index.ts` | `ExtensionAPI` from `@earendil-works/pi-coding-agent`: status, tool, command, events. |

## Install lifecycle

`memo-ai install` (or the bare `memo-ai`) runs the wizard and then calls
`install.Run`:

1. **Detect.** Resolve `pi`, Engram and local model runtimes. Detection is
   best-effort with a 1.5 s HTTP timeout per runtime.
2. **Plan.** Map each embedded asset to a destination. The extension is planned
   only when enabled. Agent assets are never planned. `--dry-run` stops here
   and prints the plan.
3. **Write assets.** Each planned file is written atomically. An existing file
   with different content is snapshotted first.
4. **Remove stale files.** Files recorded by a previous run that are no longer
   planned (the extension when disabled, an asset dropped by a newer version)
   are deleted if their hash still matches. Edited files are kept.
5. **Prompt block.** The orchestrator text is inserted between
   `<!-- memo-ai:begin ... -->` and `<!-- memo-ai:end -->` in
   `APPEND_SYSTEM.md`, replacing a previous block in place. With
   `--no-orchestrator` an existing block is stripped.
6. **MCP.** `mcpServers.engram` is merged into `mcp.json`. The file is not
   created when Engram is disabled. A server MeMo-AI enabled earlier is removed
   when Engram is turned off, but only if it still points at the same binary.
7. **Model.** `defaultProvider` and `defaultModel` are merged into
   `settings.json` after a snapshot of an existing file. An unparseable file or
   an invalid reference is a warning that prints the manual command; state
   records `model_applied: false`. A file that already holds the values is not
   rewritten.
8. **Record state.** Every written file is stored with its SHA-256 hash.

Re-running install is idempotent: unchanged files, the prompt block,
`mcp.json` and `settings.json` stay byte-identical, and no backup directory is
created.

### Shared files

`APPEND_SYSTEM.md`, `mcp.json` and `settings.json` belong to the user. The first
time MeMo-AI changes one that already exists, it snapshots it to
`memo-ai/backups/<timestamp>/`. For `APPEND_SYSTEM.md` and `mcp.json` the path
is also recorded in state. Text outside the markers and every other MCP server
are preserved byte-for-byte or key-for-key. Saving `settings.json` re-encodes
it as two-space JSON, so comments in it are not kept; the snapshot holds the
original.

## State schema

`memo-ai/state.json` is the source of truth for uninstall:

```json
{
  "version": 3,
  "language": "en",
  "installed_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "orchestrator": true,
  "primary_model": "ollama/qwen3:8b",
  "model_applied": true,
  "engram_path": "/usr/bin/engram",
  "engram_mcp": true,
  "extension": true,
  "config": {
    "path": "/home/me/.pi/agent/mcp.json",
    "backup": "/home/me/.pi/agent/memo-ai/backups/20260101-000000.000/mcp.json",
    "existed": true
  },
  "prompt": {
    "path": "/home/me/.pi/agent/APPEND_SYSTEM.md",
    "existed": false
  },
  "files": [
    {
      "path": "/home/me/.pi/agent/skills/memo-memory/SKILL.md",
      "sha256": "…",
      "kind": "skill"
    }
  ]
}
```

Version 3 dropped the `subagents` and `subagent_model` fields. Writes are
atomic (`tmp` file + rename) so an interrupted run cannot leave partial JSON.
The extension reads this file to answer `memo_status`.

## Uninstall

1. Load state. If there is no state, warn and exit cleanly.
2. Delete each recorded file only when its current SHA-256 matches the record.
   Modified files are skipped and reported unless `--force` is used.
3. Strip the managed block from `APPEND_SYSTEM.md`. The file is deleted only
   when nothing else is left in it.
4. `mcp.json`: with `--restore` and a pre-install snapshot, restore the
   snapshot (after snapshotting the current file). Otherwise remove
   `mcpServers.engram` only if it still points at the binary MeMo-AI
   configured, and delete the file if MeMo-AI created it and it is now empty.
5. Leave `settings.json` as it is and say how to reset the default model.
6. Remove empty managed directories and `state.json`.

Files outside the recorded manifest are never deleted.

## Orchestrator behavior

The orchestrator is a prompt asset, not runtime code.
`prompts/memo-orchestrator.md` encodes the ODD loop (authorize, remember,
explore, classify, track, implement, verify, persist), the Task Size rules,
the inline execution model and the commit policy. The skills under
`skills/memo-*/` must stay consistent with it.

Pi has no subagents, so the orchestrator never delegates. For a change whose
Risk line names a high-risk item it finishes with a read-only self-review of
the final diff and says so plainly. The agent definitions under
`internal/assets/files/agents/` are kept for the day subagent support returns;
`install.destination` skips them.

The commit policy is behavioral. Supervised runs never commit, push, publish,
merge or release without an explicit request. A declared autonomous run may
create work-unit commits on a feature branch, but never push, open a pull
request, merge, release or rewrite history. MeMo-AI does not install a runtime
switch for this: the user declares the mode in the session.

## Extending the stack

1. Edit files under `internal/assets/files/`.
2. A new prompt template (`commands/`) or skill is picked up by `assets.Walk`.
3. Add or update strings in `internal/i18n/i18n.go`.
4. Run `make check`.

The install tests build their paths with `pi.At(t.TempDir())`, so they never
touch the real user configuration.
