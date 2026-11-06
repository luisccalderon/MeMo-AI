# MeMo-AI

**Memory-first orchestration for [OpenCode](https://opencode.ai).**

MeMo-AI installs and configures a small, opinionated stack: a primary orchestrator agent, optional
subagents, a persistent memory server ([Engram](https://github.com/Gentleman-Programming/engram)),
slash commands, skills and a TUI plugin that renders the MeMo-AI wordmark.

It is a single Go binary with no runtime dependencies. It edits only the OpenCode files it owns,
snapshots your configuration before changing it, and can remove everything it installed.

> [Leer en español](README.es.md)

---

## Why

Coding agents forget. They re-ask decisions, re-read the same files and lose the thread after
compaction. MeMo-AI makes memory and evidence part of the workflow instead of an afterthought:

- **Remember before asking** — recover context from Engram (`mem_context`, `mem_search`).
- **Orchestrate deliberately** — work inline by default; delegate only for a named reason.
- **Prove before claiming done** — every completion includes a command and its result.
- **Persist what matters** — durable decisions, fixes and handoffs are saved to Engram.

## Requirements

- **OpenCode v2** in `PATH` (or `OPENCODE_BIN`).
- **Go 1.23+** only if you build from source.
- **Engram** is optional. MeMo-AI can detect it or download the latest verified release for you.
- A local model runtime (Ollama, LM Studio, vLLM) is optional. Any `provider/model` OpenCode can use works.

## Install

### From source

```sh
git clone https://github.com/luisccalderon/MeMo-AI.git
cd MeMo-AI
make build
./bin/memo-ai install
```

### With the install script

```sh
./scripts/install.sh          # builds this checkout and installs to ~/.local/bin
```

### With Go

```sh
go install github.com/luisccalderon/MeMo-AI/cmd/memo-ai@latest
memo-ai install
```

## Quick start

```sh
memo-ai                 # interactive wizard (Spanish by default)
memo-ai --lang en       # interactive wizard in English
memo-ai install --yes   # accept recommended values
memo-ai doctor          # verify the installation
memo-ai agents          # list agents, models and current configuration
```

Non-interactive examples:

```sh
# Use a local Ollama model and enable the full subagent stack
memo-ai install --yes --model ollama/qwen3:8b

# Keep it inline: no subagents, no TUI plugin
memo-ai install --yes --no-subagents --no-plugin

# Use the built-in build agent as primary, with memory only
memo-ai install --yes --agent build --model anthropic/claude-sonnet-4-5 --no-subagents
```

Then open OpenCode and run `/memo`.

## What gets installed

Everything lives under the global OpenCode config directory
(`${XDG_CONFIG_HOME:-~/.config}/opencode`):

| Path | Purpose |
| --- | --- |
| `agents/memo-*.md` | Orchestrator plus explorer, writer, reviewer and curator subagents |
| `skills/memo-*/SKILL.md` | Memory, delegation, handoff and local-model procedures |
| `commands/memo*.md` | `/memo`, `/memo-recall`, `/memo-remember`, `/memo-handoff`, `/memo-review`, `/memo-agents` |
| `plugins/memo-ai/` | TUI wordmark, `MEMO` badge, `MeMo-AI · About` command and `memo_status` tool |
| `memo-ai/state.json` | Manifest of installed files and their SHA-256 hashes |
| `memo-ai/backups/` | Configuration and file snapshots created before writes |

`opencode.json` (or `opencode.jsonc`) is merged, not replaced:

- `default_agent` and `model` point at your choices.
- `agents.<primary>.permissions` gets an ordered subagent policy.
- `mcp.servers.engram` is added when Engram is enabled.
- `experimental.subagent_depth` is set to `1` when subagents are enabled.

Existing keys, comments outside of JSONC limits, and unrelated configuration are preserved. JSONC
comments are lost when the file is rewritten, and a backup is always kept.

## Agents

| Agent | Mode | Role |
| --- | --- | --- |
| `memo-orchestrator` | primary | Memory-first default: recover, route, verify, persist. |
| `memo-explorer` | subagent | Read-only code map with `path:line` evidence. |
| `memo-writer` | subagent | One bounded implementation unit with a disjoint edit surface. |
| `memo-reviewer` | subagent | Independent read-only review, ordered by severity. |
| `memo-curator` | subagent | Consolidates durable findings in Engram. |

Subagents are optional and capped at depth 1. When disabled, the primary agent receives
`deny subagent *` and does all work inline.

## Commands and skills

| Command | What it does |
| --- | --- |
| `/memo` | Show the active memory and orchestration configuration. |
| `/memo-recall <topic>` | Search project memory and answer with evidence. |
| `/memo-remember` | Save a durable decision, fix or discovery. |
| `/memo-handoff` | Write a resumable session summary before pausing or compacting. |
| `/memo-review` | Run a read-only independent review. |
| `/memo-agents` | Explain the installed agents and routing. |

Skills are auto-invoked by OpenCode when their description matches the task:
`memo-memory`, `memo-delegation`, `memo-handoff`, `memo-local-models`.

## TUI plugin

The bundled plugin installs under `plugins/memo-ai/` and is discovered automatically. It provides:

- The MeMo-AI wordmark in `home.footer` (Vesper palette).
- A `MEMO` status badge.
- A `MeMo-AI · About` palette command and `/memo-about` slash command.
- A server-side `memo_status` tool for agents.

Disable it with `--no-plugin`. Reconfigure with `memo-ai configure` at any time.

## Memory with Engram

MeMo-AI configures Engram as a local MCP server:

```json
{
  "mcp": {
    "servers": {
      "engram": {
        "type": "local",
        "command": ["/usr/bin/engram", "mcp", "--tools=agent"]
      }
    }
  }
}
```

If Engram is missing:

```sh
memo-ai engram status
memo-ai engram install   # downloads the latest release and verifies its checksum
```

See [docs/engram.md](docs/engram.md) for the memory protocol and troubleshooting.

## Local models

MeMo-AI detects Ollama (`:11434`), LM Studio (`:1234`) and vLLM (`:8000`) and offers their models
during the wizard. Any `provider/model` OpenCode supports can be entered manually.

```sh
memo-ai install --yes --model ollama/qwen3:8b
memo-ai install --yes --model lmstudio/qwen2.5-coder-7b
```

See [docs/local-models.md](docs/local-models.md) for capability and context-window guidance.

## Safety

- Only files recorded in `memo-ai/state.json` are removed.
- Files modified by you are skipped unless `--force` is used.
- Configuration is restored from the pre-install snapshot on uninstall.
- `--dry-run` shows every planned write without touching disk.
- No data leaves your machine except the optional Engram release download.

```sh
memo-ai uninstall          # restore config + remove installed files
memo-ai uninstall --force  # also remove files you modified
```

## Commands reference

```text
memo-ai [command] [flags]

Commands
  install       Install or reconfigure the stack (interactive)
  configure     Reconfigure agents, model and subagents
  doctor        Read-only health check (--json supported)
  agents        List agents, local models and current configuration
  engram        Engram status or install
  uninstall     Remove only what MeMo-AI installed
  version       Print the version

Global flags
  --lang es|en
  --yes
  --dry-run
  --agent <id>
  --model <provider/model>
  --subagent-model <provider/model>
  --no-subagents
  --no-engram
  --no-plugin
  --install-engram
  --force
  --restore
  --json
```

## Development

```sh
make test     # go test ./...
make vet      # go vet ./...
make fmt      # gofmt -w
make build    # bin/memo-ai
make install  # install to PREFIX/bin
```

Project layout:

```text
cmd/memo-ai/            entrypoint
internal/cli/           flag parsing, wizard, commands
internal/install/       asset writes, config merge, subagent policy
internal/uninstall/     safe removal and config restore
internal/config/        JSONC load/save and nested helpers
internal/detect/        OpenCode, Engram and local model detection
internal/doctor/        health checks
internal/engram/        release download, checksum and install
internal/assets/files/  embedded agents, skills, commands and plugin
packages/opencode-memo/ plugin source package
```

See [docs/architecture.md](docs/architecture.md) for the install lifecycle and data model.

## License

[MIT](LICENSE)
