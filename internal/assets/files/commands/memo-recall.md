---
description: Search project memory and answer with evidence
argument-hint: "<topic>"
---

Recall what the project already knows about: $ARGUMENTS

1. Call `mem_current_project` to confirm the project.
2. Call `mem_search` with focused domain terms. If the results are thin, try one alternative
   query with synonyms.
3. For each promising result, use `mem_get_observation` or `mem_timeline` before relying on it.
4. Answer with:
   - the recovered fact;
   - the topic key and observation ID;
   - the files or systems it applies to;
   - whether it is still valid or possibly stale.
5. If nothing relevant exists, say `No prior memory found` and then inspect the repository for
   evidence. Do not invent a memory.

Keep the answer under 250 words. Answer in the user's language.
