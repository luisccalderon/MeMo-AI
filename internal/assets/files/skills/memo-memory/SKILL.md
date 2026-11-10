---
name: memo-memory
description: Recover, search, save and hand off durable project knowledge with Engram so the agent stops re-asking the user.
---

# MeMo Memory Protocol

Engram is the project's durable memory. Treat it as a curated notebook, not a transcript sink.

Tools may appear as `mem_*` or, under Code Mode, as `tools.engram.mem_*`. Use whichever
namespace the runtime exposes. If no Engram tool is available, say so once and continue with
repository evidence; do not pretend you remembered something.

## 1. Orient at the start

Before asking the user to re-explain context:

1. `mem_current_project` — confirm which project Engram resolved.
2. `mem_context` — recover recent session history.
3. If the task names a decision, bug or convention, `mem_search` with two or three focused
   terms.

Search results are previews. Use `mem_timeline` for surrounding context and
`mem_get_observation` before relying on a full record.

## 2. Search before repeating

Search before you:

- re-open an already-decided architectural question;
- re-debug a bug that may already have a root cause;
- introduce a convention that may already exist;
- ask the user for a constraint they already stated.

Query with domain words, not full sentences. Example: `auth refresh token rotation`, not
`how did we decide to handle refreshing tokens again`.

## 3. Save deliberately

Save when the knowledge is durable and reusable:

- a completed fix and its root cause;
- an architectural or product decision with its rationale;
- a non-obvious discovery about the codebase or environment;
- a configuration change future sessions must know;
- a durable user constraint or preference.

Do **not** save:

- raw tool output, logs or stack traces;
- every conversational turn;
- secrets, tokens, credentials or personal data;
- temporary states that will be invalid in an hour.

Use a stable `topic_key` for evolving topics (`architecture/auth-model`, `bug/upload-retry`).
Reuse it with `mem_update` instead of creating a competing memory.

### Save template

```markdown
**What**: one sentence describing the durable fact.
**Why**: the reason, constraint or failure it prevents.
**Where**: files, paths or systems it applies to.
**Learned**: the reusable rule for next time.
```

## 4. Hand off before losing context

Before a long pause, a handoff, or automatic compaction:

1. Call `mem_session_summary` with:
   - goal;
   - instructions and constraints;
   - discoveries;
   - work completed;
   - next steps;
   - relevant files.
2. Persist the summary **before** compaction.
3. After compaction, call `mem_context` before continuing.

## 5. Stay honest

- Never invent a memory, an observation ID or a topic key.
- If search returns nothing, say `No prior memory found` and proceed from evidence.
- Tell the user in one line what you saved: `Saved to Engram: bug/upload-retry`.
- If the same fact already exists, update it; do not duplicate it.
