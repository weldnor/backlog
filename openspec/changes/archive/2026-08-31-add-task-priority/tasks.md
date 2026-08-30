## 1. The task model

- [x] 1.1 Add the priority constants and helpers to `internal/task/task.go`: `PriorityHigh`, `PriorityMedium`, `PriorityLow`, a `Priorities` slice in descending severity order, `DefaultPriority` (`medium`), `ValidPriority(string) bool`, and a `PriorityRank(string) int` that ranks unknown values after `low`. Verify with a unit test in `internal/task/task_test.go` (new file) covering each permitted value, an unknown value, and the empty string.
- [x] 1.2 Add the `Priority` field to the `Task` struct and add `"priority"` to `TopLevelKeys`, positioned between `"status"` and `"tags"`. Verify `go build ./...` succeeds and that `internal/task` tests still compile.
- [x] 1.3 Add `SortByPriorityThenID` to `internal/task/task.go`, ordering by descending priority rank then ascending identifier, using a stable sort. Verify with a unit test covering mixed priorities, ties broken by identifier, and a task carrying an invalid priority sorting last.

## 2. Reading and writing the field

- [x] 2.1 Add `readPriority` to `internal/task/parse.go` and call it from `Parse` after `readStatus`. An absent key sets `Priority` to `medium` and records a repairable warning; a non-scalar value records a non-repairable error; a scalar outside the permitted set is kept verbatim and records a non-repairable error naming the permitted values. Verify with table tests in `internal/task/parse_test.go` for all four cases.
- [x] 2.2 Confirm that `notePreservedKeys` no longer reports `priority` as an unrecognised top-level field, now that it is in `TopLevelKeys`. Verify with a parse test asserting a file declaring `priority: high` produces no warning about an unrecognised field.
- [x] 2.3 Emit `priority` from `Task.Bytes` in `internal/task/serialize.go` by adding a case to the `emit` switch, so it is written unconditionally on every task. Verify with a serialize test that a task parsed from a file with no priority round-trips to a file declaring `priority: medium`, and that a file with `priority: high` round-trips unchanged.
- [x] 2.4 Set `Priority` to `DefaultPriority` in `task.New` and give it a `priority` parameter so `add` can pass an explicit value. Update every existing caller. Verify `go build ./...` and `go test ./internal/task/...` pass.
- [x] 2.5 Verify that an unrelated write preserves priority: a test that parses a task with `priority: high`, changes only its status, and asserts the rendered bytes still declare `priority: high`.

## 3. Validation and repair

- [x] 3.1 Confirm the per-file priority findings surface through `internal/validate`, which already forwards parse issues. Verify with tests in `internal/validate/validate_test.go` that a backlog containing a task with an unknown priority reports an error, and one containing a task with no priority reports a repairable warning and still exits zero without `--strict`.
- [x] 3.2 Confirm `validate --fix` inserts the default: because the missing-priority warning is repairable, `repair` already rewrites the file through its `HasRepairableIssue()` path. Verify with a test that `--fix` rewrites a priority-less file to declare `priority: medium` and reports the action.
- [x] 3.3 Verify `--fix` leaves an out-of-range priority alone: a test asserting that a task declaring `priority: urgent` is not modified, keeps its value, and is still reported as an error. Extend the doc comment on `repair` to name the new repair.

## 4. Command-line surface

- [x] 4.1 Add `--priority` to `backlog add` in `internal/cli/lifecycle.go`, defaulting to `medium`, rejecting an unknown value with a usage error listing the permitted values, and passing the value through to `task.New`. Verify with CLI tests covering the default, an explicit value, and a rejected value creating no task.
- [x] 4.2 Add `--priority` to `backlog set`, allow any combination of status, priority and reference, and update the "nothing to do" usage error to mention priority. Verify with CLI tests that setting only the priority leaves the status and the file's directory unchanged, that status and priority can be set together, and that an invalid priority leaves the task unchanged.
- [x] 4.3 Add the `priorities` field and its filter to `internal/cli/scope.go`, with a separate registration method so `--priority` is bound by `list` only and not by `search`. Verify with tests covering one value, several values combining as a disjunction, combination with a tag filter, and an unknown value producing a usage error.
- [x] 4.4 Call `task.SortByPriorityThenID` in `runList` after `sc.apply`, and register the priority flag there. Verify with tests that the JSON output is a single sequence in descending priority then ascending identifier order, and that human output preserves that order within each status group.
- [x] 4.5 Show the priority in `writeTaskDetail` and in `taskLine` in `internal/cli/output.go`. Verify with tests that `backlog show` prints the priority and that a listing line displays it alongside the tags.
- [x] 4.6 Add `Priority` to `TaskView` in `internal/cli/output.go` as a top-level JSON string positioned after `status`. Verify with a test asserting the JSON of `add`, `list`, `show`, `set` and `search` all carry the field.
- [x] 4.7 Verify `search` is unaffected: a test asserting that `backlog search` rejects `--priority` as an unknown flag and that its result order is unchanged for a backlog with mixed priorities.

## 5. Agent guidance

- [x] 5.1 Add priority guidance to `internal/skills/files/backlog-capture.md`: how to choose a value, the three values described by the consequence of leaving the finding unfixed rather than by scheduling, and the instruction to leave the default when the consequence cannot be judged without unrequested investigation. Verify against the `agent-integration` delta spec scenarios.
- [x] 5.2 Add priority guidance to `internal/skills/files/backlog-triage.md`: read priority as a provisional judgement, revise it on a task that is kept, use it to order the review, and do not let it decide a disposition on its own. Verify against the `agent-integration` delta spec scenarios.
- [x] 5.3 Bump `Version` in `internal/cli/cli.go` so the staleness check prompts existing projects to re-run `backlog init`. Verify with a test in `internal/cli/skills_test.go` that a project carrying the previous version's skills is reported as stale.

## 6. Documentation

- [x] 6.1 Rewrite the README's "no board, no priorities, no milestones, no assignment" sentence so it excludes only board, milestones and assignment, and states that priority records the severity of the finding rather than a scheduling decision. Verify by reading the surrounding paragraph for consistency with the proposal's Why.
- [x] 6.2 Update the README's task file format section: add `priority` to the frontmatter example, to the list of author-owned top-level fields, and state the default and that a file without the field is read as `medium`.
- [x] 6.3 Update the README's command reference: the `--priority` row in the `add` option table, the `--priority` filter and the new ordering under `list`, the priority option under `set`, and the added repair under `validate --fix`. Note in the `search` section that priority is not searchable and not a search filter, with the reason.

## 7. End-to-end verification

- [x] 7.1 Extend `e2e_test.go` with a full-lifecycle run: `init`, `add` with and without `--priority`, `list --priority`, `set` changing only priority, `show`, and `--json` at each step, asserting the priority is carried throughout.
- [x] 7.2 Add an e2e migration test: build a backlog whose task files declare no priority, assert `validate` reports repairable warnings and exits zero, run `validate --fix`, and assert every file then declares `priority: medium` and `validate --strict` exits zero.
- [x] 7.3 Run `go build ./...` and `go test ./...` and confirm the whole suite passes, then run `openspec validate --changes add-task-priority --strict` and confirm the change still validates.
