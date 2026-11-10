---
name: memo-handoff
description: Preserve a resumable record before compaction, a long pause or a session handoff.
---

# MeMo Handoff

Use this when a session is about to be compacted, paused, or handed to a fresh session.

## Before compaction

1. Call `mem_session_summary` with the structure below.
2. Keep the summary factual: no guesses, no unfilled placeholders.
3. If the work is incomplete, name the exact next action and the file to start from.

```markdown
**Goal**
The user-visible outcome.

**Instructions / constraints**
Durable rules the next session must respect.

**Discoveries**
Non-obvious facts, root causes, gotchas, with path:line where relevant.

**Completed**
What is done and verified, with the command used.

**Next steps**
Ordered, concrete, one action per line.

**Relevant files**
path — why it matters.
```

## After compaction

1. Recover with `mem_context`.
2. If a decision is still unclear, `mem_search` before asking the user.
3. Re-state the plan in two lines and continue.

## Feature-sized work

If the work cannot be resumed from the diff alone, keep one feature document in the repository
with the user's request verbatim, the acceptance criteria, the current state and the next
step. Link to it from the session summary instead of paraphrasing it.
