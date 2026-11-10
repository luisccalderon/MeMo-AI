---
description: Run a read-only independent review of the current changes
argument-hint: "[focus]"
---

Review the current changes against this focus: $ARGUMENTS

If no focus is given, review the working diff and the staged diff.

1. Collect the evidence yourself first: `git status --short`, `git diff --stat`, and the diff
   of the files in scope. Run any runtime probe the review needs; the reviewer cannot.
2. Review inline, strictly read-only, as an independent reviewer would: judge the diff against
   the focus, not against what you intended to write.
3. Report findings in severity order with `path:line`, a concrete failure scenario and a
   suggested fix. Label each finding `BLOCKER` or `ADVISORY`.
4. Close with a verdict: PASS, PASS WITH NOTES or CHANGES REQUESTED.

Do not modify files. If the diff cannot be read, say what is missing instead of guessing.
