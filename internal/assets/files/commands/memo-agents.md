---
description: Explain the installed agents and how work will be routed
---

Explain the MeMo-AI agent stack and the current routing.

1. Call the `memo_status` tool when it is available. Otherwise read
   `~/.omp/agent/memo-ai/state.json` for the configured model and enabled subagents.
2. Read the installed agents under `~/.omp/agent/agents/` (only `memo-*.md`).
3. Present:
   - the main session and what the orchestrator prompt makes it do;
   - each enabled subagent, its role, its tools and that it cannot spawn other agents;
   - the delegation triggers that would route work to each subagent through the `task` tool;
   - the commit mode (supervised or autonomous) and how it is declared;
   - what is disabled and how to enable it (`memo-ai configure`).
4. End with one example request that would stay inline and one that would delegate.

Keep it under 300 words. Answer in the user's language.
