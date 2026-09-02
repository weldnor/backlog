---
name: backlog-capture
description: Record a problem you noticed but are not fixing right now, so it is not lost. Use when, in the middle of unrelated work, you find a defect, a race, a flaky test, a missing check, a leaky abstraction or another concrete piece of technical debt that is outside the scope of what you were asked to do. Do not use when reviewing the backlog, planning work, or fixing the problem now.
---

# Recording a finding

You are mid-task. You found something that is real, that matters, and that is
not what you were asked to do. Record it in one command and get back to work.

Capturing must stay cheap. Two commands, no deliberation beyond the threshold
below, then return to what you were doing. Never let a capture turn into an
investigation.

## The threshold

Record a finding only when **all three** are true:

1. **It is outside the scope of the current task.** If it is inside the scope,
   it is part of the work — do it, do not file it.
2. **You are not fixing it now.** If you are about to fix it, fix it. A task
   created and closed in the same session is noise.
3. **It concerns the repository, not this session.** It has to be reproducible
   by someone who reads the code tomorrow. A confusing tool result, a stale
   build, a mistake you already corrected, or anything about the conversation
   itself is not a finding.

If any of the three is false, record nothing and move on.

## Do not record

These are the categories that turn a backlog into a dump nobody reads. None of
them belongs here, however tempting:

- **Stylistic preferences.** Naming you would have chosen differently, a
  formatting habit, a file you would have split. Not a finding.
- **Speculative refactoring.** "This could be generalised", "this might not
  scale", "we could extract an interface here" — with no defect and no concrete
  cost being paid today. Not a finding.
- **Anything already covered by work in progress.** If the current change, an
  open plan, or an existing task already addresses it, adding another entry
  makes the backlog worse, not better.
- **Anything you have not actually seen.** Do not file a hypothesis you have
  not verified in the code.
- **Session-local trouble.** A flaky network call, a missing local dependency, a
  command you typed wrong.

When in doubt, do not record. A backlog of thirty real findings is useful; a
backlog of two hundred observations is the same as no backlog at all.

## Always search first

The CLI does no duplicate detection — deciding whether two descriptions mean the
same thing is your job, and you are far better at it than any string match.

```
backlog search "<a few distinctive words>" --json
```

Search the words that would appear in someone else's description of the same
problem, not the words you happen to have in mind. Try a second query with
different terms if the first returns nothing.

Then:

- **A task describes the same problem** → do not create a second one. If you
  learned something the existing task does not say, add what you know as a
  reference or leave it alone. Then return to your work.
- **A match is in status `declined`** → do not create a second one either. Someone
  has already weighed this finding and decided against acting on it, and the
  task carries the `reason` they gave. Search covers every status, so the plain
  search above already surfaces this decision before you file the same one
  again.
- **A task is related but distinct** → create yours, and say in the description
  how it relates.
- **Nothing relevant** → create the task.

A declined match is not something to swallow. Tell the user which task covers
the finding and what its reason says, in the same one line you would have used
to report a new task — then return to your work. If the reason no longer holds
because the code has changed underneath it, say that too, and leave the decision
to reopen the task to the user. Reopening it yourself, or filing a duplicate to
get around the decline, both throw away the judgement the decline recorded.

## Closing a task the same search turns up

The search above sometimes surfaces a task that your current change has
already resolved, or one that is an exact duplicate of what you were about to
file. Resolve it on the spot rather than leaving it for a later triage that may
not come soon:

- **Your current change fixed it** — close it, with the commit that did:
  ```
  backlog set <id> done --ref "commit:<sha>"
  ```
- **It is an exact duplicate of the finding you were about to record** — delete
  it rather than filing a second copy:
  ```
  backlog rm <id>
  ```

Do this only for the unambiguous cases above. Anything short of "my change
plainly fixed it" or "this is the same finding, word for word" is a judgement
call for `backlog-triage`, not something to decide in passing — leave it as
it is and move on.

## Recording it

```
backlog add "<title>" \
  --description "<what is wrong, and why it matters>" \
  --priority <high|medium|low> \
  --tag <tag> \
  --file <path> --file <path>
```

- **Title** — the problem, specifically, in one line. `Session cache is not
  safe for concurrent readers`, not `Fix cache`. Someone reading only titles in
  three months has to be able to tell what this is.
- **Description** — what is wrong, how you noticed it, and what the consequence
  is. Include the reasoning you already have in context; reconstructing it later
  costs far more than writing it now.
- **`--file` is required whenever the finding is tied to a location in the
  code**, and it almost always is. You have the paths in front of you right now.
  Whoever picks this up does not, and finding them again is the expensive part.
  Pass one `--file` per path.
- **Priority** records how bad the finding is if nobody ever fixes it — see
  below. Omit it and the task is recorded as `medium`.
- **Tags** are optional and free-form. Use them only if this project already
  uses a tag consistently — check with `backlog list --json`.

Branch and commit are recorded automatically, so a later reader can tell how
old the observation is.

## Choosing the priority

Priority says **how bad this is if nobody ever fixes it**. It is not a schedule,
a deadline, or a claim about what will be worked on next — that decision belongs
to whoever triages the backlog and to the project's planning system, not to you
in the middle of unrelated work.

Judge it by the consequence of leaving the finding in place:

- **`high`** — leaving it unfixed causes real damage. Data is lost or corrupted,
  a correctness or security guarantee does not hold, a user-facing path is
  broken, or the failure is silent and will be discovered long after the fact.
- **`medium`** — leaving it unfixed keeps costing something without breaking
  anything outright. A race that has not bitten yet, a test that fails
  intermittently, a missing check that today's callers happen to satisfy, an
  abstraction that makes every change in the area more expensive.
- **`low`** — leaving it unfixed is survivable indefinitely. Real, worth writing
  down, but nothing degrades while it sits there.

Choose from what you have already read. If you cannot tell what the consequence
would be without investigating — opening call sites you have not looked at,
tracing a path you have not followed, running something to find out — then
**leave the priority at the default**. Capture stays cheap, and a `medium` that
honestly means "I could not tell from here" is worth more to the next reviewer
than a `high` you guessed at. Triage re-judges priority with the code in front
of it.

## Afterwards

Say one line to the user — what you filed and its identifier — and continue the
task you were on. Do not start working on the finding, do not expand it, and do
not open a discussion about it.
