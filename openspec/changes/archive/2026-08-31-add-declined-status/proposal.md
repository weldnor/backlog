## Why

Triage today has four dispositions, and one of them destroys its own reasoning.
The triage skill tells the reviewer to drop a task with `backlog rm` and to "say
which ones you dropped and why; do not drop silently" — but the only place that
"why" lands is the chat transcript, which nobody reads again. The file is gone,
so `backlog search` cannot find it, so the next capturing agent that walks into
the same code records the same finding again, and it is triaged a second time
from scratch. The decision not to do something is real work, and the backlog
currently has nowhere to keep it.

The gap is narrow and specific: a finding that is *correctly recorded* and
*deliberately declined*. It is not the same as a mis-capture — a duplicate, a
test entry, something that should never have been written down — and the two
have been sharing `rm` because there was no alternative. Separating them gives
`rm` a meaning it does not have today ("this should not be in the backlog") and
gives the decline a durable home ("this is in the backlog, and we decided
against it").

This stays inside the tool's remit. `declined` is the outcome of a judgement
about a finding, in the same way `priority` is a judgement about a finding. It
says nothing about scheduling, and it introduces no board, no milestones and no
assignment.

## What Changes

- A fourth `status` value, `declined`, meaning the finding was recorded
  correctly and a reviewer decided not to act on it. Like `done` it is terminal
  and lives in `.backlog/archive/`, so the directory layout keeps its shape: two
  directories, `tasks/` for live work and `archive/` for finished decisions.
- A new author-owned top-level frontmatter field `reason`, holding the prose
  explanation of the decline. It sits beside `title` and `priority` because a
  reviewer is expected to reread and sharpen it by hand. It is required when the
  status is `declined` and absent otherwise; leaving `declined` clears it, and
  git keeps the history the same way it does for every other edit.
- `backlog set` accepts `declined` and gains `--reason`. Setting a task to
  `declined` without a reason is a usage error, because a decline nobody can
  audit is the state this change exists to eliminate. `--reason` alone revises
  the text of an already declined task; supplying it for any other status is a
  usage error.
- **`backlog search` always searches declined tasks, with or without `--all`.**
  This is the requirement the change turns on: search exists to answer "has this
  already been recorded", and a decline is the strongest possible form of having
  been recorded. `done` deliberately keeps its current behaviour and stays
  behind `--all`, because a fixed problem reappearing is a regression and
  genuinely is new information, while a declined problem reappearing is the same
  decision arriving a second time.
- `backlog list` treats `declined` as terminal: it is excluded by default,
  included by `--all`, and selectable with `--status declined`. Within the
  listing it forms its own group, after `done`.
- `backlog show` displays the reason, and the JSON of every command that reports
  tasks carries a new top-level `reason` string, empty for a task that is not
  declined. `status` is already part of the JSON shape, so an agent reading
  search results can already tell a declined hit from a live one.
- `backlog validate` gains two per-file checks — a `declined` task with no
  `reason` is an error, and a `reason` on a task that is not `declined` is an
  error — and extends the existing directory rule so that a `declined` task
  outside `archive/` is reported and moved by `--fix`. Neither of the two new
  errors is repairable: one needs prose written and the other needs prose
  deleted, and both are judgements.
- The capture skill learns to stop on a declined match: a search hit whose
  status is `declined` means the finding has already been weighed and rejected,
  so it is reported to the user rather than recorded again.
- The triage skill's overloaded **Drop** disposition is split into the three
  distinct outcomes it has always contained: already fixed becomes `done`, a
  mis-capture or duplicate stays `rm`, and below-the-bar or no-longer-true
  becomes `declined` with a reason.
- Not breaking. No existing task file can carry the new status or the new field,
  every old file keeps parsing and validating exactly as it does today, and the
  on-disk format version stays at 1.

## Capabilities

### New Capabilities

None. This is a new value for an existing field, a new field beside it, and new
options on existing commands.

### Modified Capabilities

- `task-store`: `status` gains a fourth permitted value; the directory-layout
  requirement places `declined` in `archive/`; the author-owned frontmatter
  requirement gains `reason` together with the rule that it is present exactly
  when the status is `declined`.
- `task-management`: `set` accepts the new status and gains `--reason` with its
  usage rules, `list` gains the new status in its filter and its grouping,
  `show` and the JSON shape expose `reason`, and the boundary between `rm` and
  `declined` is stated.
- `task-search`: a new requirement that declined tasks are always in scope,
  regardless of `--all`, with the reasoning that separates them from `done`.
- `backlog-validation`: two new per-file checks for the reason/status pairing,
  and the cross-file directory check extended to the new status.
- `agent-integration`: the capture skill must not re-record a declined finding;
  the triage skill must use the four outcomes rather than the overloaded drop.

## Impact

- `internal/task`: `Statuses` and `ValidStatus` gain `declined`; `Task` gains a
  `Reason` field; `parse.go` reads it and reports its issues, `serialize.go`
  emits it after `priority`, and `TopLevelKeys` gains the key so it stops being
  treated as an unrecognised field preserved verbatim.
- `internal/store`: `DirFor` maps `declined` to `archive/`.
- `internal/cli`: `lifecycle.go` (`set`, `show`), `scope.go` (the default status
  set, and the search-specific rule that always admits `declined`),
  `output.go` (`TaskView`, the group order, the task line).
- `internal/validate`: the two pairing checks; the existing directory repair
  covers the new status once `DirFor` knows it.
- `internal/skills/files/`: both skill markdown files, which are embedded in the
  binary, so every project re-running `backlog init` picks the new text up.
- `README.md`: the status list, the frontmatter example and its field split, the
  `set` and `rm` sections, the `search` section's scope paragraph, the `list`
  section, and the validate/`--fix` lists.
- No new dependencies.
