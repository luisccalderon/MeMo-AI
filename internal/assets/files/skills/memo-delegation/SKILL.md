---
name: MeMo Delegation
description: Route work inline or to explorer, writer, reviewer or curator subagents only when a named reason applies.
metadata:
  opencode/autoinvoke: true
---

# MeMo Delegation

A subagent costs a fresh session, uncached reads and a handoff. **Inline is the default.**
Delegate only when you can name the reason out loud.

## Decision table

| Reason | Threshold | Agent |
| --- | --- | --- |
| Exploration | More than ~3 inline calls / 10k tokens, or more than ~5 sequential lookups | `memo-explorer` |
| Parallel writers | 2+ independent units with disjoint edit surfaces, each several minutes | `memo-writer` |
| Context backstop | A long session must hand its next bounded unit to fresh context | `memo-writer` |
| Independent verifier | High-risk change (data, security, contracts, concurrency, delivery) or no regression test | `memo-reviewer` |
| Memory curation | Several durable findings to consolidate without spending main context | `memo-curator` |

File count, task size or model price never decide the route by themselves. A delivery that
depends on another unit is **not** parallel.

## Handoff contract

Every delegation must include:

```
Objective: one bounded outcome.
Surface: exact files/directories allowed.
Constraints: patterns to follow, APIs not to change, tests to keep green.
Acceptance: the command or check that proves success.
Return: the evidence format required.
```

If the subagent's report lacks evidence, ask once for the missing evidence; do not accept a
summary as proof.

## After a delegation

1. Read the report and verify the evidence yourself.
2. Inspect the diff, not only the summary.
3. Run the acceptance check when the subagent could not.
4. Save durable findings to Engram.
5. Never say "the subagent verified it" as your only evidence.

## Depth

Subagent depth is one. Subagents cannot launch more subagents. If a worker needs exploration,
it should read directly or return the missing information to the parent.

## When subagents are disabled

If the user disabled subagents, the orchestrator does everything inline. Do not attempt to
call a subagent and do not claim a delegation happened.
