---
name: memo-delegation
description: Work inline with bounded evidence, because Pi has no subagents; includes the self-review gate for high-risk changes.
---

# MeMo Inline Execution

Pi has no built-in subagents, so MeMo works **inline**. Do not call, name or simulate a
subagent, and never say that you delegated work.

## Evidence budget

Decide and verify with one bounded batch: at most 3 calls and roughly 10k tokens, using targeted
search and line ranges. When understanding needs more, keep reading in small batches and note
what you learned after each one instead of widening a single call.

- Independent read-only calls may run in parallel.
- Writes, builds and anything the next step depends on run one at a time.
- Summarize shell output to counts, `--stat` or tails. Read full logs only on failure.

## Self-review gate

| Tier | When | Verification |
| --- | --- | --- |
| Passive | Structural readback only. | Read the diff. |
| Medium | The change has its own runnable checks. | Run them and report `<command> → <result>`. |
| High | The Risk line names a high-risk item. | Medium, plus one read-only pass over the final diff as a reviewer would. |

For the high tier, run runtime probes in a fresh scratch directory outside the workspace. Say
plainly that the extra pass was a self-review, not an independent one.

- A blocker is a change-caused defect or unmet acceptance item that reproduces with realistic
  input and did not reproduce at the baseline. Pre-existing defects and out-of-domain values are
  advisories.
- Silently ignoring an explicit option with success, or changing existing output nobody asked to
  change, is always a blocker.
- Correction bounds: one batch fixes every blocker, then one recheck limited to those blockers.
  Remaining blockers become one **Needs your decision** result.

## Context backstop

When the context nears its limit, persist a handoff (`mem_session_summary`, or `/memo-handoff`)
before it is compacted, then continue from the recovered context.

## Reserved for later

Subagent routing (explorer, writer, reviewer, curator) is not available on Pi yet. Until it
returns, every task runs through this skill.
