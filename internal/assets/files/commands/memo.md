---
description: Show MeMo-AI status, active memory and orchestration configuration
---

Report the current MeMo-AI state. Do not change any files.

1. Call the `memo_status` tool when it is available. Otherwise read
   `memo-ai/state.json` inside the Pi agent directory (`~/.pi/agent` by default). If neither
   exists, say that MeMo-AI is not configured
   (`memo-ai install`) and stop.
2. If Engram tools are available, call `mem_current_project` and `mem_context`.
3. Summarize in this order:
   - orchestrator prompt: installed or not, and the model MeMo-AI configured;
   - execution model: inline (Pi has no subagents yet);
   - commit mode: supervised by default, autonomous only when declared;
   - Engram status: configured or not, and how many relevant memories were recovered;
   - the last durable memory worth knowing about, if any.
4. Finish with one line: `Next: /memo-recall <topic>`, `Next: /memo-changes` when the working
   tree is dirty, or `Next: /memo-handoff` when the session is getting long.

Keep the report under 200 words. Answer in the user's language.
