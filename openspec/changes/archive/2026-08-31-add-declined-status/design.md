## Context

See proposal.md — Why. The constraints that shape the approach are properties of
the existing code:

- **`scope.selected()` is shared by `list` and `search`** (`internal/cli/scope.go`).
  It returns `todo + doing`, adds `done` under `--all`, and takes an explicit
  `--status` at face value. Both commands get their status set from this one
  function, so any asymmetry between them has to be introduced deliberately —
  the precedent is `registerPriority`, which binds `--priority` for `list`
  alone.
- **`Store.DirFor(status)` is a two-way map.** `done` resolves to `archive/`,
  everything else to `tasks/`. Status determines directory, and `validate`'s
  cross-file checks plus `--fix` enforce that in both directions.
- **Reading is deliberately tolerant.** `task.Parse` fails only on absent or
  unparseable frontmatter; every other deviation becomes an `Issue` so that
  `backlog validate` reports it rather than an unrelated command crashing.
- **Repair is driven by parse issues.** `validate.repair` skips a task with
  `HasBlockingIssue()` and rewrites one with `HasRepairableIssue()`. A new
  non-repairable error needs no new machinery to be excluded from repair.
- **Frontmatter is emitted by hand** (`serialize.go`), with field order fixed by
  `task.TopLevelKeys` and unknown keys re-emitted where the author put them.
- **`TaskView` already carries `status`** (`output.go:26`), so an agent reading
  `backlog search --json` can already distinguish a declined hit from a live one
  without any change to the JSON shape beyond the new `reason` string.

## Goals / Non-Goals

**Goals:**

- A declined finding stays findable by the mechanism that exists to prevent
  duplicates, without the caller having to know to ask for it.
- The reason for a decline is stored next to the decline, not in a transcript.
- `rm` and `declined` end up with disjoint meanings, so a reviewer choosing
  between them is choosing between two different situations rather than two
  spellings of the same one.

**Non-Goals:**

- No second decline status. "Real but not worth it" and "no longer true" are
  both declines; the difference lives in the `reason` prose, not in the schema.
  A second value would double the vocabulary to sharpen a distinction that only
  a human reader acts on.
- No `reason` on any status other than `declined`. It is not a general note
  field; `title` and the description body already carry the finding's content.
- `reason` is not searchable content. Search asks whether a *finding* has been
  recorded, and the reason describes the disposition, not the finding.
- No change to the on-disk `metadata.schema` version. Both additions are
  additive at the top level.
- No automatic reopening. A declined task returns to `todo` only because someone
  ran `backlog set`.

## Decisions

### `declined` is a status value, not a type or a resolution field

The axis already exists. `status` records where a task stands, and `done` is
already a terminal position on it; `declined` is the second terminal position —
the finding was closed without being acted on. Nothing about a declined task is
a different *kind* of finding, which is what a type would model, and a type
would not change when the reviewer decides.

*Alternative considered:* a separate `resolution` field (`completed`,
`wontfix`, `obsolete`) alongside a `closed` status, as Bugzilla does. Rejected —
it makes two fields that must agree, so every read path grows a question about
what a `todo` task with `resolution: wontfix` means, and `validate` grows a
matrix of invalid combinations to police. One field with four values has no
invalid combinations at all.

*Alternative considered:* a `declined` tag. Rejected — tags are free-form,
unvalidated searchable content, and `list` could not exclude declined tasks by
default without teaching it that one particular tag is structural.

### `declined` lives in `archive/`, and the layout keeps two directories

`DirFor` gains one case. The layout invariant is not "done here, everything else
there" but **not terminal / terminal**, and `declined` is terminal, so the shape
of the store is unchanged and the existing cross-file placement check and its
repair extend by wording alone.

*Alternative considered:* a third directory `.backlog/declined/`. Rejected —
`--all` would have to grow a notion of *which* archives it includes, `Tasks()`
would walk three roots, and the store's documented "exactly two task
directories" would become three for no gain a reader of `status` cannot already
see.

### `reason` is a top-level author-owned field

The README states the test: *would a person edit this field on purpose?* A
reviewer rereading a decline six months later and sharpening why it was declined
is exactly that. It joins `TopLevelKeys` after `priority`, which also stops
`notePreservedKeys` from reporting it as an unrecognised field.

*Alternative considered:* `metadata.decline_reason`. Rejected on the same test —
`metadata` is tool-owned with a closed key set, and this is prose a human writes.

### `reason` is present exactly when the status is `declined`

Both directions are enforced, and both are **non-repairable errors**: a declined
task with no reason needs prose written, and a reason on a live task needs prose
deleted. Neither has a single correct answer a tool can supply, so `--fix`
leaves both alone — which `HasBlockingIssue()` already arranges.

Leaving `declined` therefore clears the reason. This applies the project's
existing position rather than inventing a new one: the store has no `updated`
field because git already records modification time, and by the same argument
git already records what the reason said before the task was reopened. Carrying
a stale reason forward on a live task would be a field that describes a state
the task is no longer in.

*Alternatives considered:* keeping `reason` on every status as a general triage
note — rejected, it turns a precise field into a vague one and removes the
pairing check that makes a reasonless decline impossible. Making `declined`
irreversible so the question cannot arise — rejected, `set` moves every other
status in both directions and a single one-way status is a special case the CLI
would have to explain; the history git keeps is enough.

### Search always includes `declined`; `done` stays behind `--all`

This is the requirement the change turns on, and the asymmetry is deliberate.
`scope.go` already documents the principle for the neighbouring case — search
has no `--priority` filter because letting a severity filter narrow it would let
a duplicate hide behind the filter. Letting the archive scope hide a declined
task lets a duplicate hide behind the scope in exactly the same way.

`done` is genuinely different and keeps its behaviour: a fixed problem
reappearing is a regression, which is new information and *should* be recorded
again. A declined problem reappearing is the same decision arriving a second
time.

Mechanically, `scope` gains a flag that `search` sets and `list` does not,
mirroring the `register` / `registerPriority` split. The forced inclusion
applies only when no explicit `--status` was given: `selected()` already takes an
explicit `--status` at face value — including when it names `done` — and
`--status todo` on a search meaning "todo, and also declined" would be
surprising in a way the current rule is not.

### Declined tasks form the last group in a human listing

Group order becomes `todo, doing, done, declined`. The canonical order stays
descending priority then ascending identifier, as one sequence in JSON;
`writeTaskLines` buckets by status in input order, so this is a change to the
bucket order alone. A declined task keeps its priority — a `high` finding
someone consciously chose not to fix is worth being able to see as such — and
priority never implies a disposition.

### The binary version is bumped

Both skill files change. `skills.Install` refreshes an unmodified file whenever
the rendered content differs, but `validate`'s staleness warning — the thing
that tells a project to re-run `init` — fires only when the installed version is
numerically older than the running one. Without a bump, projects keep the old
guidance with nothing prompting them to refresh.

## Risks / Trade-offs

- **An old decline blocks a capture that a changed codebase now justifies.** →
  The capture skill directs the agent to *report* the declined match to the user
  rather than silently drop the finding, so the decision surfaces to someone who
  can reopen the task. Silently re-recording and silently discarding are both
  worse.
- **`archive/` grows without bound**, since declines accumulate and nothing
  prunes them. → Accepted; that is the feature. `list` is unaffected by default,
  and `rm` remains available for an entry that should never have been recorded.
- **A scripted `search --status todo` still misses declines.** → Documented
  behaviour, not a bug: an explicit status filter is taken at face value
  everywhere else too. The default path — which is what the capture skill uses —
  is the one that matters.
- **Rollback is not clean for tasks already declined.** Unlike the priority
  change, an older binary does not degrade gracefully here: it reports
  `status: declined` as an invalid status error and `reason` as an unrecognised
  top-level field. The files stay readable and nothing is destroyed, but
  `validate` will fail on them until the binary is upgraded again or the tasks
  are moved back to `done`. This is inherent in adding a value to a closed
  enumeration and is the price of `declined` being a status rather than a tag.
- **Two errors that `--fix` will not fix** appear in a tool whose selling point
  is that repair handles the unambiguous cases. → Correct and intended; the
  README already documents that judgement calls are reported and left alone, and
  these are judgement calls about prose.

## Migration Plan

No data migration, and no step is mandatory:

1. A backlog written before this change is unaffected. No existing file can
   carry `declined` or `reason`, so nothing is newly reported and
   `validate --fix` has nothing new to do.
2. `backlog init` refreshes the two skill files, after which triage starts
   using the four outcomes.
3. Existing tasks that were dropped before this change are gone and cannot be
   recovered; the change is forward-looking only.

Rollback is reverting the binary, with the caveat recorded under Risks: any task
declined in the meantime reads as an invalid status until the binary is restored
or the task is set back to `done`.
