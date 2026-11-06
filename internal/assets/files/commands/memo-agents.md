---
description: Explain the installed agents and how work will be routed
---

Explain the MeMo-AI agent stack and the current routing.

1. Read `~/.config/opencode/memo-ai/state.json` for the primary agent, model and enabled
   subagents.
2. Read the installed agents under `~/.config/opencode/agents/` (only `memo-*.md`).
3. Present:
   - the primary agent and when to use it;
   - each enabled subagent, its role and its permission boundary;
   - the delegation reasons that would trigger each subagent;
   - what is disabled and how to enable it (`memo-ai configure`).
4. End with one example request that would stay inline and one that would delegate.

Keep it under 300 words. Answer in the user's language.
