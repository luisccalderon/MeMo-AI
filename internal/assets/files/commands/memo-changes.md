---
description: Summarize the working-tree changes with their evidence and risk
argument-hint: "[base ref]"
---

Summarize the changes in this repository. Do not modify any files and do not commit.

Compare against: $ARGUMENTS (when empty, use the working tree and the staged diff; when a ref
is given, also include the commits since that ref).

1. Run `git status --short` and `git diff --stat` (plus `git diff --stat <ref>...HEAD` and
   `git log --oneline <ref>..HEAD` when a ref is given). Read only the hunks you need.
2. Group the changes into units of behavior: one line each saying what changed and why, with
   the files it touches. Call out anything that looks unrelated to the rest.
3. For each unit, state the evidence that exists: the test or command that covers it, or
   `unverified` when nothing does.
4. Close with one `Risk:` line per unit (`Risk: item N (reason)` or `Risk: none`) using the
   MeMo high-risk list, and a suggested Conventional Commit message per unit.

Keep it under 300 words. Answer in the user's language; commit messages stay in English.
