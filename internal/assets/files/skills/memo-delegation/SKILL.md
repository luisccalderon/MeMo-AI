---
name: memo-delegation
description: Route work inline or to explorer, writer, reviewer or curator only when a named trigger truly holds, with a handoff contract and a verification gate.
---

# MeMo Delegation

A subagent costs a fresh session, uncached reads and a handoff. **Inline is the default.**
Delegate only when you can name the trigger and its condition truly holds. When in doubt, work
inline.

## Triggers

| Trigger | Holds when | Agent |
| --- | --- | --- |
| **Evidence budget** | Deciding or verifying needs one parallel batch: at most 3 calls and roughly 10k tokens, using bounded search and line ranges. | Inline |
| **Mapping** | Understanding needs more evidence than the inline budget, or more than about 5 sequential lookups. | `memo-explorer` |
| **Write rule** | 2+ independent units of behavior with disjoint edit surfaces, each several minutes of work (several files or roughly 100+ changed lines), launched together; or the context backstop. | `memo-writer` |
| **Context backstop** | The parent context is around 150k tokens and the next bounded unit should go to a fresh worker. | `memo-writer` or `memo-explorer` |
| **Per-action** | A long test suite, build or install can run in a fresh worker without changing the implementation route. | Fresh worker |
| **Verification** | The change's Risk line names a high-risk item. | `memo-reviewer` |
| **Memory curation** | Several durable findings to consolidate without spending main context. | `memo-curator` |
| **Ask** | A real product or design decision is open. | One focused question; stop and wait |

File count, task size, changed lines or model price never decide the route by themselves. A
delivery that depends on another unit is **not** parallel. Units of about 100 changed lines or
less always run inline.

## Handoff contract

Every delegation must include:

```text
Objective: one bounded outcome.
Allowed edit surfaces: exact repository-relative paths or narrow globs, one per line.
Constraints: patterns to follow, APIs not to change, tests to keep green.
Invariants: existing helpers to reuse for shared state and the invariants they keep
  (ordering, uniqueness, validation, normalization).
Acceptance: the exact command or check that proves success.
Return: the evidence format required.
```

### Allowed edit surfaces

- Exact repository-relative paths or narrow globs, one per line.
- Never `.`, a bare repository root or an absolute path.
- List the directories where new files are authorized and any pre-existing untracked targets.
- Nothing beyond the delegated task.
- If the surface cannot be derived, do not launch the writer. Present the enumerated candidate
  paths as an approve/decline choice and wait.

## After a delegation

1. Read the report and verify the evidence yourself.
2. Inspect the diff, not only the summary.
3. Run the acceptance check when the subagent could not.
4. Save durable findings to Engram.
5. Never say "the subagent verified it" as your only evidence.

If the report lacks evidence, ask once for the missing evidence; do not accept a summary as
proof.

## Verification gate

| Tier | When | Verification |
| --- | --- | --- |
| Passive | Structural readback only. | Parent reads the diff. |
| Medium | Writer self-verification stands. | Writer runs the acceptance commands and reports `<command>: <result>`. |
| High | The change names a high-risk item. | Writer self-verification plus one independent `memo-reviewer`. |

- The parent re-runs one reported command as a spot check.
- A blocker is a change-caused defect or unmet acceptance item that reproduces with realistic
  input and did not reproduce at the baseline. Pre-existing defects and out-of-domain values are
  advisories.
- Silently ignoring an explicit option with success, or changing existing output nobody asked to
  change, is always a blocker.
- Runtime probes run in a fresh scratch directory outside the workspace (`mktemp -d`), executed
  by the parent. The reviewer stays read-only.
- Correction bounds: one batch fixes every reported blocker, then one recheck limited to those
  blockers. A second batch only when the same blocker still fails. A new finding never earns
  one. Remaining blockers become one **Needs your decision** result.

## Launch discipline

- Launch each distinct `(phase, task)` pair at most once per turn. Duplicate launches cause
  conflicting writes and wasted tokens.
- Pass the skills the worker needs under `## Skills to load before work`, using installed skill
  identifiers or workspace-relative `SKILL.md` paths.
- Background execution is only for independent, read-only exploration, audit or review, at most
  two concurrent tasks. Never poll or sleep. Use foreground for decisions, writers, dependent
  verification and anything the next step depends on.
- Every delegated worker closes with a `## Key Learnings` block of 1–5 numbered, standalone
  factual sentences, or omits it when there is nothing reusable.

## Depth

Subagent depth is one. Subagents cannot launch more subagents. If a worker needs exploration, it
should read directly or return the missing information to the parent.

## When subagents are disabled

If the user disabled subagents, the orchestrator does everything inline. Do not attempt to call
a subagent and do not claim a delegation happened.
