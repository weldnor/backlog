## Context

See proposal.md — Why. The constraints that actually shape the approach are
properties of the existing code:

- **Reading is deliberately tolerant.** `task.Parse` fails only when the
  frontmatter is absent or unparseable; every other deviation becomes an
  `Issue`, so that a hand-edited file stays usable and `backlog validate` — not
  a crash in an unrelated command — is what reports it. A new field must join
  that model rather than introduce a new way for reads to fail.
- **Frontmatter is emitted by hand, not by a YAML encoder** (`serialize.go`),
  because these files are read in diffs. Field order is fixed by
  `task.TopLevelKeys`, and unknown keys are re-emitted in the position the
  author wrote them.
- **`scope.apply` is shared by `list` and `search`.** It sorts by identifier,
  and `search.Search` then re-ranks by match location. Anything that changes the
  order coming out of `scope.apply` changes search results too.
- **Repair is driven by parse issues.** `validate.repair` skips any task where
  `HasBlockingIssue()` is true (a non-repairable error), and otherwise rewrites
  files flagged by `HasRepairableIssue()`. A new repairable warning needs no new
  machinery, and a new non-repairable error is automatically excluded from
  repair.
- **Skill files are embedded and checksummed.** `skills.Render` stamps the
  binary version and a checksum; `validate` warns only when the installed
  version is numerically older than the running one.

## Goals / Non-Goals

**Goals:**

- One representation of priority that every read path resolves identically, so
  no call site has to remember what an absent value means.
- An existing backlog keeps working untouched, and reaches the new convention
  through one `backlog validate --fix` run.
- Priority changes the order of `list` without changing the order of `search`.

**Non-Goals:**

- No `--priority` filter on `search`. Search answers "has this already been
  recorded", which is a question about content; narrowing it by severity would
  let a duplicate hide behind a filter. This is a deliberate asymmetry with
  `list`, and the `task-search` spec's existing wording — "the same status and
  tag filters as listing" — is left as it is precisely because it stays true.
- No change to the on-disk `metadata.schema` version. The field is additive and
  every reader of schema 1 keeps working.
- No ordering of priority against status. A `low` task in `doing` is not
  reordered ahead of a `high` task in `todo`; grouping by status stays the outer
  structure of the human listing.

## Decisions

### Priority is a top-level field, not a metadata key

`metadata` is tool-owned with a **closed** key set, where an unrecognised key is
an error. `id`, `title`, `status` and `tags` are author-owned and hand-editable.
The README states the test directly: *would a person edit this field on
purpose?* For priority the answer is unambiguously yes — a human triaging the
backlog re-judges severity by hand, and an agent sets it at capture. It
therefore joins `TopLevelKeys`.

*Alternative considered:* `metadata.priority`. Rejected — it would make the
field subject to a closed key set and, worse, put a value people are meant to
edit into the block the README describes as tool-owned.

### Emitted after `status`, before `tags`

`TopLevelKeys` becomes `id, title, status, priority, tags, metadata`. Placing it
directly after `status` groups the two single-word judgements about the task,
and keeps `tags` — the only variable-height field before `metadata` — adjacent
to the block below it. Adding the key to `TopLevelKeys` is also what stops
`notePreservedKeys` from reporting `priority` as an unrecognised field.

### `Task.Priority` is never empty after a successful parse

`Parse` resolves the default: a file with no `priority` key yields
`Priority = "medium"` plus a **repairable warning**, exactly as
`metadata.schema is missing` behaves today. Every consumer — filtering,
ordering, `show`, JSON — reads a real value and needs no fallback of its own.

*Alternative considered:* keep `Priority == ""` meaning "unset" and resolve the
default at each use site. Rejected — it scatters one rule across the filter, the
comparator, two output paths and the JSON view, and each of those is a place to
forget it.

A value outside the permitted set is kept verbatim on the task and recorded as a
**non-repairable error**, mirroring how an unknown `status` is handled. Repair
must not silently replace a value someone deliberately typed; that is a
judgement, and `HasBlockingIssue()` already causes the file to be skipped.

### Every write emits a priority, including on an unrelated change

A hand-written file with no priority gains `priority: medium` the first time any
command rewrites it — a `set` that only changed the status will show both lines
in the diff.

This is a deliberate departure from the neighbouring rule in
`metadataPresent()`, which avoids adding a `metadata` block to a file that has
none. The two cases differ: `metadata` is an optional multi-line tool-owned
block whose absence is a legitimate shape for a hand-written file, whereas
priority is a single required author-owned scalar with one defined default. The
invariant that every file the CLI writes carries an explicit priority is worth
more than avoiding one line of one-time diff noise, and the alternative — only
`--fix` ever adds it — leaves files sitting in a warning state indefinitely.

### Ordering is applied in `list`, not in `scope.apply`

`scope.apply` keeps sorting by identifier. `runList` then applies a
priority-aware sort to the result before either output branch. `search` calls
`scope.apply` and is therefore untouched, which is what keeps its documented
ranking — title matches first, ascending identifier within a rank — exactly as
it is.

The canonical list order becomes descending priority, then ascending identifier.
Grouping by status stays a presentation concern inside `writeTaskLines`, which
appends into per-status buckets in input order and so preserves it; the JSON
array is that same single sequence.

A task whose priority is invalid sorts after `low`. It is already reported as an
error, and the listing needs a total order regardless.

### The filter lives on `scope` but is registered only by `list`

`scope` gains a `priorities` field and a filter step that is a no-op when empty,
so all task selection stays in one function. Registration is split: the existing
`register` continues to bind `--all`, `--status` and `--tag` for both commands,
and a separate call binds `--priority` for `list` alone. Validation of the flag
value reuses the shape of the existing unknown-status usage error.

### The binary version is bumped

`skills.Install` refreshes an unmodified file whenever the rendered content
differs, so re-running `backlog init` picks up the new guidance regardless of
version. But `validate`'s staleness warning — the thing that tells a project to
re-run `init` — fires only when the installed version is numerically older than
the running one. Since both skill files change, `cli.Version` must be bumped, or
projects will keep the old guidance with nothing prompting them to refresh.

## Risks / Trade-offs

- **Defaulting to `medium` produces a backlog where almost everything is
  `medium`, and the field stops discriminating.** → The capture skill is
  required to describe the values by the consequence of not fixing the finding,
  and to reserve the default for the case where the agent genuinely cannot tell.
  The field earns its place only if the skills are written well; that is why
  `agent-integration` is in scope rather than left for later.
- **One-time diff noise** when an unrelated write adds `priority: medium` to a
  pre-existing file. → Accepted, and made predictable by documenting
  `validate --fix` as the way to do it deliberately, in one commit, up front.
- **A stale `.claude/skills/` copy that was edited locally is skipped by
  `init`**, so a project that customised its skills will not receive the new
  priority guidance. → Pre-existing behaviour, surfaced by the staleness
  warning; `init --force` remains the escape hatch.
- **`list` and `search` now accept different filters.** → Deliberate, argued
  under Non-Goals, and the reason is worth a sentence in the README so it does
  not read as an oversight.

## Migration Plan

No data migration is required, and no step is mandatory:

1. A backlog written before this change is read exactly as before, with every
   task treated as `medium`.
2. `backlog validate` reports each such file as a repairable warning. Because
   these are warnings, `validate` still exits zero and any commit hook gating on
   it keeps passing.
3. `backlog validate --fix` writes `priority: medium` into every one of them in
   a single run — the recommended first commit after upgrading.
4. `backlog init` refreshes the two skill files so agents start setting the
   field.

Rollback is reverting the binary: an older build reads `priority` as an
unrecognised top-level field, preserves it verbatim on write, and reports it
only as a warning. No task file becomes unreadable.
