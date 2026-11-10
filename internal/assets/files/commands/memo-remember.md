---
description: Save a durable decision, fix or discovery to Engram
argument-hint: "<topic>"
---

Save the following knowledge to Engram:

$ARGUMENTS

Use the MeMo memory protocol:

1. Derive a stable `topic_key` (for example `architecture/auth-model`). If unclear, call
   `mem_suggest_topic_key`.
2. Search with `mem_search` for an existing record on the same topic.
3. If an equivalent record exists, update it with `mem_update`; otherwise save it with
   `mem_save`.
4. Use this content shape:

```markdown
**What**: one sentence.
**Why**: the reason, constraint or failure it prevents.
**Where**: files, paths or systems.
**Learned**: the reusable rule.
```

Do not save secrets, raw tool output or temporary state. Confirm in one line what you saved
and under which topic key. If Engram is unavailable, say so and do not fabricate a record.
