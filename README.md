# MeMo-AI

**Your coding agent, bringing your changes, tasks, and engineering workflow together.**

MeMo-AI turns [oh-my-pi](https://github.com/can1357/oh-my-pi) (`omp`) into a memory-first
engineering agent. It installs an orchestrator prompt, optional subagents, a persistent memory
server ([Engram](https://github.com/Gentleman-Programming/engram)), slash commands, skills and a
small extension.

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

Then open `omp` and run `/memo`.

## Why

Coding agents forget. They re-ask decisions, re-read the same files and lose the thread after
compaction. MeMo-AI keeps three things in one loop:

| | What MeMo-AI does |
| --- | --- |
| **Changes** | Knows what the working tree holds; every code change closes with evidence and a `Risk:` line. `/memo-changes` summarizes it. |
| **Tasks** | Small work stays in the conversation; large work gets one feature document. `/memo-tasks` reports where it stands. |
| **Workflow** | One fixed loop (authorize, remember, explore, classify, implement, verify, persist), with delegation only for a named trigger. |

Memory makes it stick: decisions, fixes and handoffs are saved to Engram and recovered before
the agent asks you again.

## Requirements

- **oh-my-pi** (`omp`) in `PATH`, or `OMP_BIN` pointing at it.
- **Go 1.23+** only if you build from source.
- **Engram** is optional. MeMo-AI can detect it or download the latest verified release.
- A local model runtime (Ollama, LM Studio, vLLM) is optional. Any `provider/model` omp can use
  works.

## Install

| Method | Command |
| --- | --- |
| From source | `make build && ./bin/memo-ai install` |
| Install script | `./scripts/install.sh` (builds this checkout into `~/.local/bin`) |
| Go | `go install github.com/luisccalderon/MeMo-AI/cmd/memo-ai@latest && memo-ai install` |

Non-interactive examples:

```sh
# Local Ollama model and the full subagent stack
memo-ai install --yes --model ollama/qwen3:8b

# Keep it inline: no subagents, no extension
memo-ai install --yes --no-subagents --no-extension

# Memory, skills and commands only; leave the omp system prompt alone
memo-ai install --yes --no-orchestrator --no-subagents

# See every planned write without touching disk
memo-ai install --yes --dry-run
```

## What gets installed

Everything lives under the omp agent directory: `~/.omp/agent` by default. `PI_CODING_AGENT_DIR`,
`PI_CONFIG_DIR` and `OMP_PROFILE` are honored the same way omp honors them.

| Path | Purpose | Ownership |
| --- | --- | --- |
| `APPEND_SYSTEM.md` | The MeMo orchestrator, appended to omp's system prompt | Only the block between `memo-ai:begin` / `memo-ai:end` markers |
| `agents/memo-*.md` | Explorer, writer, reviewer and curator subagents | Whole file |
| `skills/memo-*/SKILL.md` | Memory, delegation, handoff and local-model procedures | Whole file |
| `commands/memo*.md` | The `/memo*` slash commands | Whole file |
| `extensions/memo-ai/index.ts` | Status badge, `memo_status` tool, `/memo-about`, compaction reminder | Whole file |
| `mcp.json` | Engram as a stdio MCP server | Only `mcpServers.engram` |
| `memo-ai/state.json` | Manifest of installed files and their SHA-256 hashes | Whole file |
| `memo-ai/backups/` | Snapshots taken before a shared file is changed | Whole directory |

Your own text in `APPEND_SYSTEM.md` and your other MCP servers are preserved.

The model is the one thing MeMo-AI does not write by hand. `--model` runs
`omp config set modelRoles.default <provider/model>`, so omp validates the value against its own
schema. If `omp` is not found, MeMo-AI prints that command for you instead.

## Agents

omp has one main session and task agents it can spawn. MeMo-AI follows that shape:

| Piece | Kind | Role |
| --- | --- | --- |
| MeMo orchestrator | system prompt block | Memory-first default: recover, route, verify, persist. |
| `memo-explorer` | task agent | Read-only code map with `path:line` evidence. |
| `memo-writer` | task agent | One bounded implementation unit with a disjoint edit surface. |
| `memo-reviewer` | task agent | Independent read-only review, ordered by severity. |
| `memo-curator` | task agent | Consolidates durable findings in Engram. |

Subagents are optional. A subagent is enabled by installing its file and disabled by removing
it; MeMo agents declare `spawns: ""`, so they cannot spawn further agents. With `--no-subagents`
the orchestrator does all work inline. `--subagent-model` pins a model in the agents'
frontmatter.

## How the orchestrator works

The orchestrator runs a fixed loop on every request:

1. **Authorize** — read-only unless the request authorizes a change.
2. **Remember** — recover project context from Engram before asking.
3. **Explore** — read the existing code first.
4. **Classify** — small tasks stay inline; large tasks get a feature document.
5. **Implement** — one verifiable unit at a time, test-first where applicable.
6. **Verify** — report `command → result`.
7. **Persist** — save durable knowledge and close substantial sessions.

Delegation is opt-in by trigger: evidence budget, mapping, parallel writers, context backstop,
per-action work, high-risk verification and memory curation. Every handoff declares its
objective, allowed edit surfaces, constraints, invariants, acceptance check and return format.

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
| `/memo-review [focus]` | Run a read-only independent review. |
| `/memo-agents` | Explain the installed agents and routing. |
| `/memo-about` | Show the configuration without spending a model turn (extension). |

Skills are listed in omp's system prompt and loaded when their description matches the task:
`memo-memory`, `memo-delegation`, `memo-handoff`, `memo-local-models`.

## Extension

`extensions/memo-ai/index.ts` is discovered automatically by omp. It provides:

- A `MEMO` status entry (`MEMO · memory` when Engram is configured).
- A read-only `memo_status` tool, so the agent reads its configuration instead of guessing.
- The `/memo-about` command.
- A reminder to run `/memo-handoff` when the context is about to be compacted.

Disable it with `--no-extension`. Reconfigure with `memo-ai configure` at any time.

## Memory with Engram

MeMo-AI adds Engram to `~/.omp/agent/mcp.json`:

```json
{
  "mcpServers": {
    "engram": {
      "type": "stdio",
      "command": "/usr/bin/engram",
      "args": ["mcp", "--tools=agent"]
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
during the wizard. omp discovers Ollama and LM Studio on its own; vLLM needs a provider entry in
`models.yml`.

```sh
memo-ai install --yes --model ollama/qwen3:8b
memo-ai install --yes --model lm-studio/qwen2.5-coder-7b
```

See [docs/local-models.md](docs/local-models.md) for capability and context-window guidance.

## Safety

- Only files recorded in `memo-ai/state.json` are removed.
- Files modified by you are skipped unless `--force` is used.
- `APPEND_SYSTEM.md` and `mcp.json` are edited in place: only the MeMo block and the Engram
  entry are touched, and a snapshot is taken first.
- `--dry-run` shows every planned write without touching disk.
- The orchestrator commits only in a declared autonomous run; push, pull requests, merge and
  release always require an explicit request.
- No data leaves your machine except the optional Engram release download.

```sh
memo-ai uninstall            # remove installed files, the prompt block and the Engram entry
memo-ai uninstall --force    # also remove files you modified
```

Uninstall leaves the omp default model as it is and tells you how to reset it.

## Commands reference

```text
memo-ai [command] [flags]

Commands
  install       Install or reconfigure the stack (interactive)
  configure     Reconfigure the orchestrator, model and subagents
  doctor        Read-only health check (--json supported)
  agents        List agents, local models and current configuration
  engram        Engram status or install
  uninstall     Remove only what MeMo-AI installed
  version       Print the version

Global flags
  --lang es|en
  --yes
  --dry-run
  --model <provider/model>
  --subagent-model <provider/model>
  --no-orchestrator
  --no-subagents
  --no-engram
  --no-extension
  --install-engram
  --force
  --restore
  --json
```

## Coming from the OpenCode version

Earlier versions targeted OpenCode. That installation lives in `~/.config/opencode` and is not
touched by this version. Remove it with the old binary (`memo-ai uninstall`) before upgrading,
then run `memo-ai install` again.

| Before (OpenCode) | Now (oh-my-pi) |
| --- | --- |
| `memo-orchestrator` primary agent, `--agent` | Prompt block in `APPEND_SYSTEM.md`, `--no-orchestrator` |
| `opencode.json` merge | `mcp.json` merge + `omp config set` |
| Subagent permission rules | Agent files with `spawns: ""` |
| TUI plugin, `--no-plugin` | Extension, `--no-extension` |
| `lmstudio/<model>` | `lm-studio/<model>` |

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
internal/install/       asset writes, prompt block, mcp.json merge, model role
internal/uninstall/     safe removal and cleanup
internal/promptblock/   managed block inside APPEND_SYSTEM.md
internal/omp/           omp agent directory resolution
internal/config/        JSON/JSONC load/save and nested helpers
internal/detect/        omp, Engram and local model detection
internal/doctor/        health checks
internal/engram/        release download, checksum and install
internal/assets/files/  embedded prompt, agents, skills, commands and extension
```

See [docs/architecture.md](docs/architecture.md) for the install lifecycle and data model.

## License

[MIT](LICENSE)
