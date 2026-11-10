# MeMo-AI

**Your coding agent, bringing your changes, tasks, and engineering workflow together.**

MeMo-AI turns [Pi](https://pi.dev) (`pi`) into a memory-first engineering agent. It installs an
orchestrator prompt, a persistent memory server
([Engram](https://github.com/Gentleman-Programming/engram)), slash commands, skills and a small
extension.

It is a single Go binary with no runtime dependencies. It edits only what it owns, snapshots
shared files before changing them, and can remove everything it installed.

> [Leer en español](README.es.md)

---

## Quick path

```sh
git clone https://github.com/luisccalderon/MeMo-AI.git
cd MeMo-AI
make build
./bin/memo-ai install      # interactive wizard
./bin/memo-ai doctor       # verify
```

Then open `pi` and run `/memo`.

## Why

Coding agents forget. They re-ask decisions, re-read the same files and lose the thread after
compaction. MeMo-AI keeps three things in one loop:

| | What MeMo-AI does |
| --- | --- |
| **Changes** | Knows what the working tree holds; every code change closes with evidence and a `Risk:` line. `/memo-changes` summarizes it. |
| **Tasks** | Small work stays in the conversation; large work gets one feature document. `/memo-tasks` reports where it stands. |
| **Workflow** | One fixed loop (authorize, remember, explore, classify, implement, verify, persist), run inline in a single session. |

Memory makes it stick: decisions, fixes and handoffs are saved to Engram and recovered before
the agent asks you again.

## Requirements

- **Pi** (`pi`) in `PATH`, in its managed install directory (`~/.pi/agent/bin`), or `PI_BIN`
  pointing at it. Install it from [pi.dev](https://pi.dev).
- **Go 1.23+** only if you build from source.
- **Engram** is optional. MeMo-AI can detect it or download the latest verified release.
- A local model runtime (Ollama, LM Studio, vLLM, llama.cpp) is optional. Any `provider/model`
  Pi can use works.

## Install

| Method | Command |
| --- | --- |
| From source | `make build && ./bin/memo-ai install` |
| Install script | `./scripts/install.sh` (builds this checkout into `~/.local/bin`) |
| Go | `go install github.com/luisccalderon/MeMo-AI/cmd/memo-ai@latest && memo-ai install` |

Non-interactive examples:

```sh
# Local Ollama model, full stack
memo-ai install --yes --model ollama/qwen3:8b

# No extension
memo-ai install --yes --no-extension

# Memory, skills and commands only; leave Pi's system prompt alone
memo-ai install --yes --no-orchestrator

# See every planned write without touching disk
memo-ai install --yes --dry-run
```

## What gets installed

Everything lives under the Pi agent directory: `~/.pi/agent` by default. `PI_CODING_AGENT_DIR`
is honored the same way Pi honors it.

| Path | Purpose | Ownership |
| --- | --- | --- |
| `APPEND_SYSTEM.md` | The MeMo orchestrator, appended to Pi's system prompt | Only the block between `memo-ai:begin` / `memo-ai:end` markers |
| `skills/memo-*/SKILL.md` | Memory, inline execution, handoff and local-model procedures | Whole file |
| `prompts/memo*.md` | The `/memo*` slash commands (Pi prompt templates) | Whole file |
| `extensions/memo-ai/index.ts` | Status badge, `memo_status` tool, `/memo-about`, compaction reminder | Whole file |
| `mcp.json` | Engram as a stdio MCP server | Only `mcpServers.engram` |
| `settings.json` | Default model, only when you pass `--model` or pick one in the wizard | Only `defaultProvider` and `defaultModel` |
| `memo-ai/state.json` | Manifest of installed files and their SHA-256 hashes | Whole file |
| `memo-ai/backups/` | Snapshots taken before a shared file is changed | Whole directory |

Your own text in `APPEND_SYSTEM.md`, your other MCP servers and every other Pi setting are
preserved. A `settings.json` that cannot be parsed is left untouched and reported as a warning.

## Subagents

Pi has no built-in subagents, so MeMo-AI does not install any. The orchestrator works inline in
one session: it budgets its evidence, runs read-only calls in parallel, and for high-risk
changes finishes with a read-only self-review of the final diff. Subagent support is planned for
a later release; the agent definitions stay in the repository for that work.

`--no-subagents` and `--subagent-model` are still accepted so existing scripts keep working, but
they have no effect.

## How the orchestrator works

The orchestrator runs a fixed loop on every request:

1. **Authorize** — read-only unless the request authorizes a change.
2. **Remember** — recover project context from Engram before asking.
3. **Explore** — read the existing code first.
4. **Classify** — small tasks stay inline; large tasks get a feature document.
5. **Implement** — one verifiable unit at a time, test-first where applicable.
6. **Verify** — report `command → result`.
7. **Persist** — save durable knowledge and close substantial sessions.

### Commit policy

- **Supervised (default):** no commit, push, publish, merge or release unless you ask.
- **Autonomous:** when you explicitly declare an automatic, headless or unsupervised run, the
  orchestrator may create work-unit commits on a feature branch with Conventional Commit
  messages. Push, pull requests, merge, release and destructive git operations still require an
  explicit request.

## Commands and skills

| Command | What it does |
| --- | --- |
| `/memo` | Show the active memory and orchestration configuration. |
| `/memo-changes [ref]` | Summarize the working-tree changes with their evidence and risk. |
| `/memo-tasks [feature]` | Show tracked tasks, their status and the next step. |
| `/memo-recall <topic>` | Search project memory and answer with evidence. |
| `/memo-remember <fact>` | Save a durable decision, fix or discovery. |
| `/memo-handoff` | Write a resumable session summary before pausing or compacting. |
| `/memo-review [focus]` | Run a read-only review of the current changes. |
| `/memo-about` | Show the configuration without spending a model turn (extension). |

Skills are listed in Pi's system prompt and loaded when their description matches the task:
`memo-memory`, `memo-delegation` (inline execution), `memo-handoff`, `memo-local-models`.

## Extension

`extensions/memo-ai/index.ts` is discovered automatically by Pi. It provides:

- The MeMo mark printed once at the start of a session.
- A `MEMO` status entry (`MEMO · memory` when Engram is configured).
- A working spinner while the agent runs, colored with the active theme accent.
- A read-only `memo_status` tool, so the agent reads its configuration instead of guessing.
- The `/memo-about` command.
- A reminder to run `/memo-handoff` when the context is about to be compacted.

Extensions run with your full permissions. Review `index.ts` if you want to know exactly what it
does. Disable it with `--no-extension`. Reconfigure with `memo-ai configure` at any time. Run
`/reload` in Pi after changing installed files.

## Memory with Engram

MeMo-AI adds Engram to `~/.pi/agent/mcp.json`:

```json
{
  "mcpServers": {
    "engram": {
      "type": "stdio",
      "command": "/usr/bin/engram",
      "args": ["mcp", "--tools=agent"],
      "exposure": "direct"
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
during the wizard. Pi only lists a model whose provider is declared, so Ollama, LM Studio and
vLLM need an entry in `~/.pi/agent/models.json`. llama.cpp is built in (`/llama`).

```sh
memo-ai install --yes --model ollama/qwen3:8b
memo-ai install --yes --model lm-studio/qwen2.5-coder-7b
```

See [docs/local-models.md](docs/local-models.md) for the `models.json` entry and capability
guidance.

## Free models

Two community packages add free model providers to Pi's catalog:

| Package | What it adds |
| --- | --- |
| `npm:pi-free` | Free model filtering and dynamic model fetching |
| `npm:opencode-pi` | Bridges OpenCode's free models into Pi without an OpenCode login |

The installer offers both in interactive mode and runs `pi install` for each one, so Pi owns the
`packages` entry in `settings.json` and can update them later with `pi update`.

```sh
memo-ai install --free-models      # install them without asking
memo-ai install --no-free-models   # never install or ask about them
```

Non-interactive runs (`--yes`) skip them unless you pass `--free-models`, so a headless install
never downloads packages you did not ask for. The provider list is not embedded in the binary and
carries no warranty; the free quota is whatever the upstream provider allows. `memo-ai doctor`
reports whether they are present.

## Safety

- Only files recorded in `memo-ai/state.json` are removed.
- Files modified by you are skipped unless `--force` is used.
- `APPEND_SYSTEM.md`, `mcp.json` and `settings.json` are edited in place: only the MeMo block,
  the Engram entry and the two default-model keys are touched, and a snapshot is taken first.
- `--dry-run` shows every planned write without touching disk.
- The orchestrator commits only in a declared autonomous run; push, pull requests, merge and
  release always require an explicit request.
- No data leaves your machine except the optional Engram release download.

```sh
memo-ai uninstall            # remove installed files, the prompt block and the Engram entry
memo-ai uninstall --force    # also remove files you modified
```

Uninstall leaves the Pi default model as it is and tells you how to reset it.

## Commands reference

```text
memo-ai [command] [flags]

Commands
  install       Install or reconfigure the stack (interactive)
  configure     Reconfigure the orchestrator, model and extension
  doctor        Read-only health check (--json supported)
  agents        Show the MeMo-AI configuration and detected local models
  engram        Engram status or install
  uninstall     Remove only what MeMo-AI installed
  version       Print the version

Global flags
  --lang es|en
  --yes
  --dry-run
  --model <provider/model>
  --no-orchestrator
  --no-engram
  --no-extension
  --free-models
  --no-free-models
  --install-engram
  --force
  --restore
  --json
  --no-subagents                      (accepted, no effect)
  --subagent-model <provider/model>   (accepted, no effect)
```

## Coming from the oh-my-pi version

The previous version targeted oh-my-pi (`omp`). That installation lives in `~/.omp/agent` and is
not touched by this version. Remove it with the old binary (`memo-ai uninstall`) before
upgrading, then run `memo-ai install` again.

| Before (oh-my-pi) | Now (Pi) |
| --- | --- |
| `~/.omp/agent` | `~/.pi/agent` |
| `OMP_BIN`, `OMP_PROFILE`, `PI_CONFIG_DIR` | `PI_BIN`, `PI_CODING_AGENT_DIR` |
| `commands/memo*.md` | `prompts/memo*.md` |
| `omp config set modelRoles.default` | `defaultProvider` / `defaultModel` in `settings.json` |
| `agents/memo-*.md` task agents | Not installed: Pi has no subagents yet |
| `/memo-agents` | Removed |
| `@oh-my-pi/pi-coding-agent`, `pi.zod` | `@earendil-works/pi-coding-agent`, `typebox` |

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
internal/install/       asset writes, prompt block, mcp.json merge, default model
internal/uninstall/     safe removal and cleanup
internal/promptblock/   managed block inside APPEND_SYSTEM.md
internal/pi/            Pi agent directory resolution
internal/config/        JSON/JSONC load/save and nested helpers
internal/detect/        Pi, Engram and local model detection
internal/doctor/        health checks
internal/engram/        release download, checksum and install
internal/assets/files/  embedded prompt, skills, prompt templates, extension and agents (not installed)
```

See [docs/architecture.md](docs/architecture.md) for the install lifecycle and data model.

## License

[MIT](LICENSE)
