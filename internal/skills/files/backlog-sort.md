---
name: backlog-sort
description: Check open backlog tasks against the current branch and close the ones already fixed. Use when asked to sort, actualize, sync or refresh the backlog against the code, or to clear out what no longer needs a human decision before a fuller triage. Do not use for judging priority, declining a finding, or promoting into a planning system — that is backlog-triage.
---

# Sorting the backlog against the branch

This is a narrow, mechanical pass, not a review. Its only question is: **has the
code already moved past this finding?** It never judges whether a finding still
matters, never declines, never reprioritises, never promotes. That is
`backlog-triage`'s job, and it stays free to make those calls on a backlog this
pass has already cleared of the obvious cases.

Because it only closes what the code demonstrably already fixed, it is safe to
run often and without a human standing over it.

## What to look at

```
backlog list new --json
backlog list todo --json
backlog list doing --json
```

`done` and `declined` are already terminal — leave them alone. For each
remaining task, read `metadata.source`: the files it concerns, and the branch
and commit it was observed on.

```
backlog show <id> --json
```

## Checking one task

1. **Has anything touched the referenced files since the task's commit?**
   ```
   git log <task's commit>..HEAD -- <file> ...
   ```
   No commits there → no evidence either way. Leave the task exactly as it is
   and move to the next one. Do not open the file just to double-check; the
   absence of change is itself the answer this pass is for.

2. **If something did touch those files, read the current version.** Does the
   specific problem the task describes still exist?
   - **It is gone** — the change fixed it, whether or not the change was made
     with the task in mind. Close it:
     ```
     backlog set <id> done --ref "commit:<sha that fixed it>"
     ```
   - **It still holds, or the file changed in ways unrelated to the finding** —
     leave the task as it is. Do not lower it, raise it, or add commentary;
     that is triage's call, not this pass's.
   - **You cannot tell without investigating further than a diff read** — leave
     it. This pass trades thoroughness for being safe to run unattended; when
     in doubt, it does nothing.

## What this pass never does

- Never declines a task, however clearly obsolete it looks — declining requires
  a `--reason` a human or a full triage pass stands behind, not a mechanical
  sweep.
- Never changes priority.
- Never promotes into a planning system.
- Never creates a task — that is `backlog-capture`.
- Never touches `done` or `declined` tasks.

## Finishing

Report a short summary: how many tasks were checked, which identifiers were
closed and with which commit, and how many were left untouched because the
code hadn't moved or the evidence was inconclusive. If a left-alone task looks
worth a real look, say so — but that look is `backlog-triage`'s, not this
pass's to take.
