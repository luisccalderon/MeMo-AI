# Memory with Engram

Engram is the persistent memory layer for MeMo-AI. It stores durable project
knowledge — decisions, fixes, constraints, handoffs — and exposes it to
oh-my-pi (`omp`) as an MCP server. The goal is simple: the agent should recover what
the project already knows before asking you again.

- Upstream: <https://github.com/Gentleman-Programming/engram>
- Installed by MeMo-AI as a **stdio** MCP server.
- Optional: MeMo-AI works without Engram, but loses cross-session memory.

## Configuration

`memo-ai install` merges this into `~/.omp/agent/mcp.json` when Engram is
enabled. Other servers in that file are left untouched:

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

`--tools=agent` exposes the agent-oriented tool set instead of the full
surface. The command path is the one detected at install time. If Engram is
moved later, re-run `memo-ai configure` or edit the path.

Disable memory with `--no-engram`. MeMo-AI then removes `mcpServers.engram`
only if a previous run added it and it still points at the same binary; a
server you configured yourself is left alone.

## Checking and installing Engram

```sh
memo-ai engram status
memo-ai engram install
memo-ai install --install-engram
```

`engram install` downloads the latest GitHub release for the current OS and
architecture, requires `checksums.txt`, verifies SHA-256, and refuses to
install an unverified binary. Downloads go to a temporary directory; the
binary is moved into place only after verification.

MeMo-AI never installs Engram silently as part of a plain `install`; the
wizard asks first.

## Memory protocol used by the agents

The bundled `memo-memory` skill and the `/memo-*` commands implement the same
loop:

1. **Recall before asking.** At the start of a task, recover context:
   `mem_context` for the project session, then `mem_search` for the topic,
   file or error at hand.
2. **Cite what you recovered.** Decisions are answered with their source,
   not re-derived from scratch.
3. **Work with evidence.** Read the code, run the command, keep `path:line`
   references.
4. **Persist what is durable.** Save only decisions, fixes, constraints and
   handoffs — not transient chatter.
5. **Hand off before compaction.** Write a resumable summary with the current
   goal, state, next step and open questions.

### When to save

Save when the information will still matter in a future session:

- An architectural decision and its reason.
- A bug fix with the failing symptom and the validated solution.
- A project constraint (style, tooling, deployment target).
- A handoff before a long pause or context compaction.

Do not save: raw command output, secrets, credentials, or facts that are
obvious from the repository.

### Useful tools

| Tool | Use |
| --- | --- |
| `mem_context` | Recover the project's working context at task start. |
| `mem_search` | Find a decision, error or file by topic. |
| `mem_save` | Persist a durable note with a title and body. |
| `mem_timeline` | Inspect the order of past events. |
| `mem_get` | Read a specific memory by ID. |

Tool names follow the Engram release; run `engram mcp --help` for the exact
list on your version.

## Troubleshooting

**omp does not list Engram.** omp reads MCP servers at startup. Restart it
after install, then run `/mcp` inside a session. `memo-ai doctor` shows the
command recorded in `mcp.json`.

**Tools have a prefix.** Depending on the omp version, Engram tools may appear
as `mem_*` or with a server prefix. The prompts refer to the short names; the
agent uses whichever the runtime exposes.

**`engram` not found.** Run `memo-ai doctor`; it reports the detected path and
version. Install with `memo-ai engram install` or point `engram_path` at your
binary by re-running `memo-ai configure`.

**Tools fail with a schema error.** MeMo-AI pins `--tools=agent`; if you
changed it, restore it and restart omp.

**Nothing is remembered across sessions.** Memory is scoped to the project
directory. Confirm you are in the same project, then run `/memo-recall
<topic>` to test retrieval before assuming the write failed.

**Checksum mismatch on install.** A proxy or mirror altered the download.
Retry on a clean network; MeMo-AI will not install the archive.
