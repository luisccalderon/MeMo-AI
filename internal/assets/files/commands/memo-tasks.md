---
description: Show tracked tasks, their status and the next step
argument-hint: "[feature]"
---

Report where the tracked work stands. Do not modify any files.

Feature filter: $ARGUMENTS (when empty, cover every feature document you find).

1. Find the feature documents in the repository (for example `odd/tasks/*.md`, `docs/tasks/`,
   or the path named in `AGENTS.md`). If Engram tools are available, call `mem_context` and
   one focused `mem_search` to recover work that has no document yet.
2. For each feature, list its tasks as `done`, `in progress` or `pending`, with the evidence
   recorded for the done ones (commit or command).
3. Cross-check against `git status --short`: flag a task marked done whose change is still
   uncommitted or missing, and changes in the tree that belong to no task.
4. End with `Next:` and the single most useful next step.

If there is no tracked work, say so in one line and summarize the open changes instead. Keep
it under 250 words. Answer in the user's language.
