## Why

A backlog that only records findings cannot answer the one question triage
actually asks: *which of these matters most?* Today every task is equal, so
reviewing an accumulated backlog means re-reading and re-judging each entry from
scratch — discarding the judgement the agent already made at the moment of
capture, when the surrounding context was still in hand. Recording severity when
it is cheapest to know it is what makes the later review fast.

This is deliberately an attribute of the *finding*, not scheduling machinery.
The README currently states that the tool has "no board, no priorities, no
milestones, no assignment"; that sentence bundles one useful thing in with three
that stay out of scope. Priority answers "how bad is this", which the capturing
agent knows. It does not answer "what do we do next", which remains the job of
the planning system the triage skill promotes into. No board, no milestones and
no assignment are still out of scope, and the README is amended to say exactly
that rather than to forbid priority along with them.

## What Changes

- A new author-owned top-level frontmatter field `priority` on every task, with
  exactly three values in descending order of severity: `high`, `medium`, `low`.
  It sits beside `status` and `tags` — not inside the tool-owned `metadata`
  block — because it records deliberate human or agent judgement and is meant to
  be edited by hand.
- `backlog add` gains `--priority`. When it is omitted the task is written with
  `priority: medium`, so the field is always present on a newly created task.
- `backlog set` gains `--priority`, so a task's severity can be revised without
  touching its status. `set` continues to require at least one of status,
  reference or — now — priority.
- `backlog list` gains a repeatable `--priority` filter that combines with the
  existing status and tag filters, and orders tasks within each status group by
  descending priority, then by ascending identifier.
- `backlog show` displays the priority, and the JSON of every command that
  reports tasks carries a new top-level `priority` string.
- `backlog validate` reports a priority outside the permitted set as an error,
  and a task file with no `priority` field at all as a repairable warning.
- `backlog validate --fix` adds `priority: medium` to a task file that has none.
  This is the migration path: an existing backlog stays readable untouched, and
  a single `--fix` run brings it up to the new convention.
- Search behaviour is unchanged, and the spec is amended to say so explicitly:
  `priority` joins `status` on the list of structural fields the query is never
  matched against.
- The capture skill gains guidance on choosing a priority at capture time, and
  the triage skill gains guidance on revising it during review — without which
  the field would exist but never be set to anything but its default.
- Not breaking. A task file written before this change parses exactly as it did,
  is treated as `medium`, and is reported only as a warning.

## Capabilities

### New Capabilities

None. Priority is a new field and a new set of options on existing commands, not
a new area of behaviour.

### Modified Capabilities

- `task-store`: the author-owned frontmatter requirement gains `priority` as a
  fourth top-level field, with its permitted values, its default, and the rule
  that a file lacking it is read as `medium` rather than rejected.
- `task-management`: `add` and `set` gain a priority option, `list` gains a
  priority filter and a priority-aware order, `show` and the JSON shape both
  expose the field.
- `backlog-validation`: the per-file checks gain a priority value check, and the
  automatic repair rule gains the insertion of the default priority as a further
  unambiguous correction.
- `task-search`: the searchable-content requirement is amended so that
  `priority`, like `status`, is named as a field the query never matches.
- `agent-integration`: the capture skill must say how to choose a priority, and
  the triage skill must treat priority as something it reads and revises.

## Impact

- `internal/task`: `Task` gains a `Priority` field; `parse.go` gains a reader
  and its issues, `serialize.go` gains an emitter positioned after `status`, and
  `TopLevelKeys` gains the key so that `priority` stops being reported as an
  unrecognised field preserved verbatim.
- `internal/cli`: `lifecycle.go` (`add`, `set`, `show`), `scope.go` (the
  `--priority` filter), `output.go` (`TaskView`, the listing order and the task
  line).
- `internal/validate`: the value check and the repair that inserts the default.
- `internal/skills/files/`: both skill markdown files, which are embedded in the
  binary and therefore change its output; every project re-running
  `backlog init` picks the new text up.
- `README.md`: the "no priorities" sentence, the frontmatter example, the field
  table for `add`, the `list` and `set` sections, and the `--fix` list.
- No new dependencies. The on-disk format version stays at 1: the field is
  additive and optional to read, so nothing that reads schema 1 today breaks.
