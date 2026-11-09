# Architecture

MeMo-AI is a single Go binary that writes a small set of files into the global
OpenCode configuration directory and merges a few keys into `opencode.json`.
This document describes the install lifecycle, the data it owns, and the rules
that keep re-runs and uninstalls safe.

## Component map

```text
cmd/memo-ai/            process entrypoint
internal/cli/           flag parsing, commands, interactive wizard
internal/assets/        go:embed of agents, skills, commands and plugin
internal/install/       asset writes, config merge, subagent policy
internal/uninstall/     safe removal, config restore or cleanup
internal/backup/        timestamped config snapshots
internal/config/        JSONC parsing, nested get/set/delete, atomic save
internal/state/         install manifest and SHA-256 tracking
internal/detect/        OpenCode, Engram and local model discovery
internal/doctor/        read-only health checks
internal/engram/        verified release download and install
internal/opencode/      config path resolution
internal/term/          colored terminal output
internal/i18n/          English and Spanish strings
internal/banner/        ANSI wordmark
internal/version/       build metadata
packages/opencode-memo/ plugin source (copied into embedded assets)
```

The runtime has no third-party Go dependencies. Everything embedded in the
binary ships from `internal/assets/files`.

## Paths

All state lives under the OpenCode global config directory:

| Platform | Directory |
| --- | --- |
| Linux / macOS | `${XDG_CONFIG_HOME:-$HOME/.config}/opencode` |
| Windows | `%APPDATA%\opencode` |

If `opencode.jsonc` exists it is edited; otherwise `opencode.json` is created.
Config file resolution happens in `internal/opencode/paths.go`.

```text
<config>/agents/memo-*.md
<config>/skills/memo-*/SKILL.md
<config>/commands/memo*.md
<config>/plugins/memo-ai/
<config>/memo-ai/state.json
<config>/memo-ai/backups/
```

## Install lifecycle

`memo-ai install` (or the bare `memo-ai`) runs the wizard and then calls
`install.Run`:

1. **Detect.** Resolve OpenCode, Engram and local model runtimes. Detection is
   best-effort with a 1.5 s HTTP timeout per runtime.
2. **Plan.** Compute the embedded file list plus the config keys to merge.
   `--dry-run` stops here and prints the plan.
3. **Backup.** Copy the current config to
   `memo-ai/backups/<timestamp>/opencode.json`. The backup path is recorded in
   state so uninstall can restore it.
4. **Write assets.** Each embedded file is written to its mapped destination.
   Agents and commands are flat under their directories; skills keep their
   folder (`skills/memo-memory/SKILL.md`).
5. **Merge config.** Only the keys below are touched. Unrelated keys,
   comments outside JSONC limits and unrelated `$schema` data are preserved.
6. **Record state.** Every written file is stored with its SHA-256 hash, so a
   later re-run or uninstall can tell whether the user edited it.

Re-running install is idempotent: files are rewritten with the same content
unless the user changed them, timestamps in `state.json` update, and no
duplicate backup directories are created for the same run.

### Config keys written

| Key | Value |
| --- | --- |
| `default_agent` | selected primary agent |
| `model` | selected `provider/model` (when provided) |
| `agents.<primary>.mode` | `primary` |
| `agents.<primary>.permissions` | ordered subagent policy |
| `agents.<subagent>.*` | subagent definitions, only for enabled subagents |
| `mcp.servers.engram` | local MCP server (when Engram is enabled) |
| `experimental.subagent_depth` | `1` when at least one subagent is enabled |

### Subagent policy

Permissions are an ordered array. The first matching rule wins, so MeMo-AI
writes `deny *` first and then one `allow` rule per enabled subagent:

```json
"permissions": [
  { "action": "subagent", "effect": "deny", "resource": "*" },
  { "action": "subagent", "effect": "allow", "resource": "memo-explorer" }
]
```

The same policy is applied to the selected primary agent and to
`memo-orchestrator` when it is installed as a subagent of something else.
Choosing `--no-subagents` writes only the `deny` rule and does not set
`experimental.subagent_depth`. A previous "no subagents" decision is preserved
when `configure --yes` runs without an explicit subagent change.

## State schema

`memo-ai/state.json` is the source of truth for uninstall:

```json
{
  "version": 1,
  "language": "en",
  "installed_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "primary_agent": "memo-orchestrator",
  "primary_model": "ollama/qwen3:8b",
  "subagents": ["memo-explorer", "memo-writer"],
  "subagent_model": "",
  "engram_path": "/usr/bin/engram",
  "engram_mcp": true,
  "plugin": true,
  "config": {
    "path": "/home/me/.config/opencode/opencode.json",
    "backup": "/home/me/.config/opencode/memo-ai/backups/20260101-000000.000/opencode.json",
    "existed": true
  },
  "files": [
    {
      "path": "/home/me/.config/opencode/agents/memo-orchestrator.md",
      "sha256": "…",
      "kind": "agent"
    }
  ]
}
```

`subagents: null` records an explicit "no subagents" decision. Writes are
atomic (`tmp` file + rename) so an interrupted run cannot leave partial JSON.

## Uninstall

`memo-ai uninstall` defaults to `--restore` semantics:

1. Load state. If there is no state, warn and exit cleanly.
2. Restore the pre-install config snapshot when it exists, after snapshotting
   the current config into `backups/uninstall-<timestamp>/`.
3. If no snapshot exists, remove only managed keys (`default_agent`, `model`,
   `agents`, `mcp`, `experimental`) and managed permission rules.
4. Delete each recorded file only when its current SHA-256 matches the record.
   Modified files are skipped and reported unless `--force` is used.
5. Remove empty managed directories and `state.json`.

Files outside the recorded manifest are never deleted.

## Orchestrator behavior

The orchestrator is a prompt asset, not runtime code. `agents/memo-orchestrator.md` encodes the
ODD loop (authorize, remember, explore, classify, track, implement, verify, persist), the Task
Size rules, the named delegation triggers, the handoff contract, the delegated verification gate
and the commit policy. The subagent prompts under `agents/memo-*.md` and
`skills/memo-delegation/SKILL.md` must stay consistent with it.

The commit policy is behavioral. Supervised runs never commit, push, publish, merge or release
without an explicit request. A declared autonomous run may create work-unit commits on a feature
branch, but never push, open a pull request, merge, release or rewrite history. MeMo-AI does not
install a runtime switch for this: the user declares the mode in the session.

## Extending the stack

To change what ships:

1. Edit files under `internal/assets/files/`.
2. For the plugin, edit `packages/opencode-memo/` and copy the changed files
   into `internal/assets/files/plugins/memo-ai/`.
3. Add or update strings in `internal/i18n/i18n.go`.
4. Run `make check` and the install/uninstall tests.

The install tests use `t.TempDir()` for the OpenCode root, so they never touch
the real user configuration.
