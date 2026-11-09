---
description: Show MeMo-AI status, active memory and orchestration configuration
---

Report the current MeMo-AI state. Do not change any files.

1. Read `~/.config/opencode/memo-ai/state.json` if it exists. If it does not, say that
   MeMo-AI is not configured and stop.
2. If Engram tools are available, call `mem_current_project` and `mem_context`.
3. Summarize in this order:
   - primary agent and model;
   - enabled subagents (or "subagents disabled");
   - commit mode: supervised by default, autonomous only when declared;
   - Engram status: configured or not, and how many relevant memories were recovered;
   - the last durable memory worth knowing about, if any.
4. Finish with one line: `Next: /memo-recall <topic>` or `Next: /memo-handoff` when the
   session is getting long.

Keep the report under 200 words. Answer in the user's language.
