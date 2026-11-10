# Architecture

MeMo-AI is a single Go binary that installs a coding-agent stack into the
oh-my-pi (`omp`) agent directory. It writes its own files, manages one block
inside `APPEND_SYSTEM.md`, merges one entry into `mcp.json`, and sets the
default model through the `omp` CLI. This document describes the install
lifecycle, the data it owns, and the rules that keep re-runs and uninstalls
safe.

## Component map

```text
cmd/memo-ai/            process entrypoint
internal/cli/           flag parsing, commands, interactive wizard
internal/assets/        go:embed of the prompt, agents, skills, commands and extension
internal/install/       asset writes, prompt block, mcp.json merge, model role
internal/uninstall/     safe removal and cleanup
internal/promptblock/   managed block inside a user-owned prompt file
internal/backup/        timestamped snapshots
internal/config/        JSON/JSONC parsing, nested get/set/delete, atomic save
internal/state/         install manifest and SHA-256 tracking
internal/detect/        omp, Engram and local model discovery
internal/doctor/        read-only health checks
internal/engram/        verified release download and install
internal/omp/           agent directory resolution
internal/term/          colored terminal output
internal/i18n/          English and Spanish strings
internal/banner/        ANSI wordmark
internal/version/       build metadata
```

The runtime has no third-party Go dependencies. Everything embedded in the
binary ships from `internal/assets/files`.

## Paths

All state lives under the omp agent directory, resolved in
`internal/omp/paths.go` the same way omp resolves it:

| Condition | Directory |
| --- | --- |
| `OMP_PROFILE` (or legacy `PI_PROFILE`) set | `~/<root>/profiles/<name>/agent` |
| `PI_CODING_AGENT_DIR` set, no profile | that directory |
| default | `~/<root>/agent` |

`<root>` is `.omp` unless `PI_CONFIG_DIR` overrides it.

```text
<agent>/APPEND_SYSTEM.md            managed block only
<agent>/mcp.json                    mcpServers.engram only
<agent>/agents/memo-*.md
<agent>/skills/memo-*/SKILL.md
<agent>/commands/memo*.md
<agent>/extensions/memo-ai/index.ts
<agent>/memo-ai/state.json
<agent>/memo-ai/backups/
```

`<agent>/config.yml` is omp's settings file. MeMo-AI never writes it: omp
quarantines an invalid config file and refuses to start, so the one setting
MeMo-AI needs goes through `omp config set`, which validates it.

## How OpenCode concepts map to omp

| Concept | omp mechanism | Why |
| --- | --- | --- |
| Primary orchestrator agent | Block in `APPEND_SYSTEM.md` | omp has one main session; its prompt is extended, not replaced, so omp's own tool guidance stays intact. |
| Subagents | `agents/*.md` task agents | Spawned by the `task` tool by name. |
| Subagent allow list | Presence of the agent file | The main session has no per-agent allow list; an agent that is not installed cannot be spawned. |
| Depth limit | `spawns: ""` in each MeMo agent | A MeMo worker cannot spawn anything. |
| Subagent model | `model:` frontmatter, injected at install | Keeps the choice next to the agent. |
| Primary model | `omp config set modelRoles.default` | Schema-validated by omp. |
| Engram | `mcpServers.engram` in `mcp.json` | omp's native MCP file. |
| TUI plugin | `extensions/memo-ai/index.ts` | `ExtensionAPI`: status, tool, command, events. |

## Install lifecycle

`memo-ai install` (or the bare `memo-ai`) runs the wizard and then calls
`install.Run`:

1. **Detect.** Resolve `omp`, Engram and local model runtimes. Detection is
   best-effort with a 1.5 s HTTP timeout per runtime.
2. **Plan.** Map each embedded asset to a destination. Agent files are planned
   only for enabled subagents; the extension only when enabled. `--dry-run`
   stops here and prints the plan.
3. **Write assets.** Each planned file is written atomically. An existing file
   with different content is snapshotted first. When a subagent model was
   chosen, `model:` is injected into the agent frontmatter.
4. **Remove stale files.** Files recorded by a previous run that are no longer
   planned (a disabled subagent, the extension, an asset dropped by a newer
   version) are deleted if their hash still matches. Edited files are kept.
5. **Prompt block.** The orchestrator text is inserted between
   `<!-- memo-ai:begin ... -->` and `<!-- memo-ai:end -->` in
   `APPEND_SYSTEM.md`, replacing a previous block in place. With
   `--no-orchestrator` an existing block is stripped.
6. **MCP.** `mcpServers.engram` is merged into `mcp.json`. The file is not
   created when Engram is disabled. A server MeMo-AI enabled earlier is removed
   when Engram is turned off, but only if it still points at the same binary.
7. **Model.** `omp config set modelRoles.default <model>`. Failure, or a
   missing `omp`, is a warning that prints the manual command; state records
   `model_applied: false`.
8. **Record state.** Every written file is stored with its SHA-256 hash.

Re-running install is idempotent: unchanged files, the prompt block and
`mcp.json` stay byte-identical, and no backup directory is created.

### Shared files

`APPEND_SYSTEM.md` and `mcp.json` belong to the user. The first time MeMo-AI
changes one that already exists, it snapshots it to
`memo-ai/backups/<timestamp>/` and records the path in state. Text outside the
markers and every other MCP server are preserved byte-for-byte or key-for-key.

## State schema

`memo-ai/state.json` is the source of truth for uninstall:

```json
{
  "version": 2,
  "language": "en",
  "installed_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "orchestrator": true,
  "primary_model": "ollama/qwen3:8b",
  "model_applied": true,
  "subagents": ["memo-explorer", "memo-writer"],
  "subagent_model": "",
  "engram_path": "/usr/bin/engram",
  "engram_mcp": true,
  "extension": true,
  "config": {
    "path": "/home/me/.omp/agent/mcp.json",
    "backup": "/home/me/.omp/agent/memo-ai/backups/20260101-000000.000/mcp.json",
    "existed": true
  },
  "prompt": {
    "path": "/home/me/.omp/agent/APPEND_SYSTEM.md",
    "existed": false
  },
  "files": [
    {
      "path": "/home/me/.omp/agent/agents/memo-explorer.md",
      "sha256": "…",
      "kind": "agent"
    }
  ]
}
```

`subagents: null` records an explicit "no subagents" decision. Writes are
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
5. Leave the omp default model as it is and say how to reset it.
6. Remove empty managed directories and `state.json`.

Files outside the recorded manifest are never deleted.

## Orchestrator behavior

The orchestrator is a prompt asset, not runtime code.
`prompts/memo-orchestrator.md` encodes the ODD loop (authorize, remember,
explore, classify, track, implement, verify, persist), the Task Size rules,
the named delegation triggers, the handoff contract, the delegated
verification gate and the commit policy. The subagent prompts under
`agents/memo-*.md` and `skills/memo-delegation/SKILL.md` must stay consistent
with it.

The commit policy is behavioral. Supervised runs never commit, push, publish,
merge or release without an explicit request. A declared autonomous run may
create work-unit commits on a feature branch, but never push, open a pull
request, merge, release or rewrite history. MeMo-AI does not install a runtime
switch for this: the user declares the mode in the session.

## Extending the stack

1. Edit files under `internal/assets/files/`.
2. A new command or skill is picked up by `assets.Walk`; a new subagent also
   needs an entry in `install.MemoSubagents`.
3. Add or update strings in `internal/i18n/i18n.go`.
4. Run `make check`.

The install tests build their paths with `omp.At(t.TempDir())`, so they never
touch the real user configuration.
