---
name: backlog-triage
description: Review the findings that have accumulated in the project backlog and decide what happens to each one. Use when asked to triage, review, go through or clean up the backlog, to see what has piled up, or to turn recorded findings into planned work. Do not use when recording a single new finding in the middle of other work.
---

# Triaging the backlog

The backlog is a capture inbox, not a plan. Findings were written quickly, by
someone who was busy with something else. Triage is the slow, careful pass that
turns them into decisions.

Unlike capture, this is allowed to take time. Read the code. Check whether the
finding still holds. Give the user a real recommendation.

## Read what is there

```
backlog list --json
```

For anything whose title is not self-explanatory:

```
backlog show <id> --json
```

Each task carries `metadata.source` — the files it concerns, and the branch and
commit it was observed on. Use it:

- **Open the files.** A finding is worth acting on only if it is still true.
- **Check the age.** If the commit is well behind the current branch, the code
  may have moved underneath the finding. A refactor since then may have fixed
  it, changed it, or made it irrelevant.

Group what you find: several tasks often turn out to be one underlying problem,
and are better promoted together than one at a time.

## Work through them in priority order

Each task carries a `priority` — `high`, `medium` or `low` — recorded at capture
as a judgement of how bad the finding is if nobody fixes it. Review the `high`
tasks first, then `medium`, then `low`, so that the most consequential findings
get your attention while it is freshest:

```
backlog list --priority high --json
```

A long backlog may not survive a full pass in one sitting. Ordering the review
this way means the part you do get through is the part that mattered most.

## Priority is a judgement to revise, not a verdict

The priority you are reading was chosen quickly, by an agent that had only what
it happened to have open, and often means no more than "I could not tell from
here". You are the first reader with the code in front of you, so you are the
first one able to judge severity properly.

- **Re-judge every task you keep.** If re-reading the code shows the finding is
  worse than it looked — it can lose data, or a guarantee genuinely does not
  hold — raise it. If it turns out to be harmless in practice, lower it. Record
  the corrected value:

  ```
  backlog set <id> --priority <high|medium|low>
  ```

  Priority can be set on its own; doing so does not change the task's status.

- **Never let priority alone decide a disposition.** A `high` task is not
  automatically promoted and a `low` one is not automatically declined. Every
  task is still checked against the current code first: a `high` finding that a
  refactor has already fixed leaves triage the same way any other fixed finding
  does, and a `low` one that is small and real may be worth fixing on the spot. Priority
  tells you what to look at first and how loudly to argue for it afterwards —
  the disposition follows from what the code actually says.

## Decide, one task at a time

Every task leaves triage with exactly one of four dispositions. They are four
different situations, not four degrees of the same one:

- **Promote** — it is real, it still holds, and it is worth planned work. Turn
  it into a work item in whatever system this project uses (below).
- **Fix now** — it is real and small enough that fixing it costs less than
  tracking it. Do it, then set the task to `done` with a reference to the
  commit or change.
- **Keep** — real, but not worth doing yet. Leave it in `todo`. Sharpen the
  title or description if the finding is clearer to you now than it was to
  whoever wrote it.
- **Decline** — real but not worth doing, or no longer true. Record the
  decision rather than deleting it:

  ```
  backlog set <id> declined --reason "<why this is not being done>"
  ```

  The reason is required, and it lives on the task rather than only in your
  summary to the user. That is the whole point: a declined task stays in the
  backlog, `backlog search` still finds it, and the next agent that walks into
  the same code learns that this was already weighed and rejected instead of
  recording it a second time. Write the reason for that reader — "the call site
  is single-threaded, so the race cannot happen" or "the loader was rewritten in
  a1b2c3d and this no longer applies", not "wontfix".

Two situations look like declining and are not:

- **Already fixed is `done`, not declined.** If the problem a task describes has
  since been fixed, the finding was acted on, whether or not anyone knew the
  task existed. Set it to `done` with a reference to the change that fixed it.
- **`backlog rm` is for an entry that should never have been recorded** — a
  duplicate of another task, a mis-capture, something filed by accident. There
  is no decision worth preserving, so there is nothing to keep. It is not the
  way to record that you decided against a finding; that is what declining is
  for.

Deciding is the point of triage. Do not hand back a summary that leaves every
task exactly where it was.

## Promoting into the project's planning system

The backlog CLI knows nothing about planning systems, deliberately. Work out at
run time what this project actually uses, rather than assuming:

- Look for the marks of a planning workflow in the repository — a directory of
  change or proposal documents, an issue tracker configured for the repo, a
  planning CLI on the path, instructions in `CLAUDE.md` or `README.md` naming
  one.
- If the project has a convention, follow it exactly as its own documentation
  describes. If several are present, ask the user which one to use rather than
  guessing.
- If you find none, say so and stop before inventing one. Promotion then means
  whatever the user asks for — often nothing more than leaving the task in
  `todo` with a better description.

Once a task has been promoted, record the link back on it:

```
backlog set <id> doing --ref "<identifier the planning system gave you>"
```

The reference is a free-form string, stored verbatim and never resolved. Write
whatever identifies the work item unambiguously in this project — a change
name, an issue URL, a ticket key. Prefix it with the system it belongs to
(`issue:`, `pr:`, or the name of the planning tool) so a later reader can tell
what kind of thing it is.

Set the task to `done` — which moves it into the archive — only when the
underlying problem is actually fixed, not when it has been written down
somewhere else.

## Finishing

Report back as a short list: what was promoted and where, what was fixed, what
was kept, what was declined and why, what was deleted as a mis-capture, and
which priorities you changed. The declines are on the tasks themselves, so the
summary is a convenience rather than the record. If the backlog is accumulating
faster than it is being triaged, say so — the fix for that is a stricter capture
threshold, not more triage.
