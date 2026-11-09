---
description: Run a read-only independent review of the current changes
agent: memo-reviewer
subagent: true
---

Review the current changes against this focus: $ARGUMENTS

If no focus is given, review the working diff and the staged diff.

Report findings in severity order with `path:line`, a concrete failure scenario and a
suggested fix. Label each finding `BLOCKER` or `ADVISORY`. Include a verdict: PASS, PASS WITH
NOTES or CHANGES REQUESTED. If you cannot read the diff, say what is missing instead of guessing.
