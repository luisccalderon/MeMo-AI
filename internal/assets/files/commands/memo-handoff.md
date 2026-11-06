---
description: Save a resumable session summary before pausing or compacting
---

Create a handoff for this session.

1. Call `mem_session_summary` with:
   - goal;
   - instructions and constraints;
   - discoveries (with path:line where relevant);
   - work completed and the command that verified it;
   - next steps, ordered and concrete;
   - relevant files.
2. If the work cannot be resumed from the diff alone, also write or update one feature
   document in the repository and link to it.
3. Return the summary in chat so the user can copy it.

Do not include secrets, raw logs or speculative claims. If Engram is unavailable, write the
summary in chat and say it was not persisted.
