## 1. The task model

- [x] 1.1 Add `StatusDeclined` to `internal/task/task.go`, append it to `Statuses` so `ValidStatus` accepts it, and add an `IsTerminal(status string) bool` helper returning true for `done` and `declined`. Verify with unit tests in `internal/task/task_test.go` covering each permitted status, the two terminal ones, and an unknown value.
- [x] 1.2 Add the `Reason` field to the `Task` struct and add `"reason"` to `TopLevelKeys`, positioned between `"priority"` and `"tags"`. Verify `go build ./...` succeeds and the `internal/task` tests still compile.
- [x] 1.3 Map `declined` to the archive directory in `Store.DirFor` in `internal/store/store.go`, expressed through `task.IsTerminal` rather than by naming the two statuses separately. Verify with a store test that a declined task is written to and discovered in `.backlog/archive/`.

## 2. Reading and writing the field

- [x] 2.1 Add `readReason` to `internal/task/parse.go` and call it from `Parse` after `readPriority`. A non-scalar value records a non-repairable error; a scalar is stored verbatim. Verify with table tests in `internal/task/parse_test.go` for a string, a list and a mapping.
- [x] 2.2 Add the pairing check to `Parse`: status `declined` with an absent, empty or whitespace-only `reason` records a non-repairable error, and a non-empty `reason` on any other status records a non-repairable error. Verify with parse tests covering all four combinations of terminal status and reason presence.
- [x] 2.3 Confirm `notePreservedKeys` no longer reports `reason` as an unrecognised top-level field now that it is in `TopLevelKeys`. Verify with a parse test asserting a declined file declaring a reason produces no unrecognised-field warning.
- [x] 2.4 Emit `reason` from `Task.Bytes` in `internal/task/serialize.go` by adding a case to the `emit` switch, written only when non-empty so a live task gains no blank line. Verify with serialize tests that a declined task round-trips with its reason, that a `todo` task emits no `reason` key, and that a multi-word reason containing a colon is quoted correctly.
- [x] 2.5 Verify that an unrelated write preserves the reason: a test that parses a declined task, changes only its priority, and asserts the rendered bytes still declare the original `reason`.

## 3. Validation and repair

- [x] 3.1 Confirm the two pairing findings surface through `internal/validate`, which already forwards parse issues, and that both are classified as errors. Verify with tests in `internal/validate/validate_test.go` that a declined task with no reason and a `todo` task carrying a reason each report an error and cause a non-zero exit.
- [x] 3.2 Extend the cross-file directory check in `internal/validate/validate.go` to use `task.IsTerminal`, so a declined task in `tasks/` and a `todo` task in `archive/` are both reported. Verify with tests covering a declined task in the task directory and an active task in the archive.
- [x] 3.3 Confirm `validate --fix` moves a misplaced declined task to the archive through the existing directory repair, and leaves both pairing errors alone because `HasBlockingIssue()` excludes them. Verify with tests that `--fix` relocates the declined task, does not invent a reason, and does not delete a misplaced one. Extend the doc comment on `repair` to name the new cases.

## 4. Command-line surface

- [x] 4.1 Add `--reason` to `backlog set` in `internal/cli/lifecycle.go` and implement the rules: `declined` without a reason is a usage error; a reason with any status other than `declined` is a usage error; a reason alone is permitted only when the task is already declined; and setting a declined task to another status clears `Reason`. Update the "nothing to do" usage error to mention the reason. Verify with CLI tests covering each of those five paths plus the successful decline.
- [x] 4.2 Verify the file moves that follow from a status change: CLI tests that declining moves the file into `archive/`, that reopening a declined task moves it back into `tasks/` with no `reason` in the written file, and that `done` → `declined` updates the status without moving the file.
- [x] 4.3 Add an `alwaysDeclined` flag to `internal/cli/scope.go` set by `search` and not by `list`, and extend `selected()` so that when no explicit `--status` was given it adds `declined` for search while `done` stays behind `--all`. Leave the explicit-`--status` branch taking its values at face value. Verify with scope tests covering the search default, the list default, `--all` on each, and an explicit `--status todo` excluding declined tasks from a search.
- [x] 4.4 Add `declined` to the status group order in `writeTaskLines` in `internal/cli/output.go`, after `done`. Verify with a list test that `--all` prints the four groups in the order `todo`, `doing`, `done`, `declined`.
- [x] 4.5 Show the reason in `writeTaskDetail` in `internal/cli/output.go` for a declined task and omit the line otherwise. Verify with tests that `backlog show` prints the reason for a declined task and prints no reason line for a `todo` one.
- [x] 4.6 Add `Reason` to `TaskView` in `internal/cli/output.go` as a top-level JSON string positioned after `priority`, always present and empty for a task that is not declined. Verify with a test asserting the JSON of `add`, `list`, `show`, `set` and `search` all carry the field and that it is `""` rather than absent on a live task.
- [x] 4.7 Verify the search behaviour end to end in `internal/cli/search_test.go`: a declined task matching the query is returned without `--all`, its result reports status `declined`, a `done` task matching the query is still excluded without `--all`, and `--status declined` returns only declined tasks.

## 5. Agent guidance

- [x] 5.1 Rewrite the **Drop** disposition in `internal/skills/files/backlog-triage.md` as the four distinct outcomes: promote, fix now and set `done`, keep, decline with `backlog set <id> declined --reason "..."`. State that an already-fixed finding is `done`, that `rm` is for an entry that should never have been recorded, and that every decline carries its reason on the task rather than only in the summary. Verify against the `agent-integration` delta spec scenarios.
- [x] 5.2 Add declined-match guidance to `internal/skills/files/backlog-capture.md`: a search hit whose status is `declined` means the finding was already weighed and rejected, so do not record a second task, and report the existing task and its reason to the user rather than dropping the observation silently. State that reopening is the user's call. Verify against the `agent-integration` delta spec scenarios.
- [x] 5.3 Bump `Version` in `internal/cli/cli.go` so the staleness check prompts existing projects to re-run `backlog init`. Verify with a test in `internal/cli/skills_test.go` that a project carrying the previous version's skills is reported as stale.

## 6. Documentation

- [x] 6.1 Update the README's status vocabulary: the four statuses, the two terminal ones, and the `.backlog/` layout comment that currently reads `archive/ # status done`. Verify by reading the format section for consistency with the `task-store` delta spec.
- [x] 6.2 Update the README's frontmatter section: add `reason` to the list of author-owned top-level fields, state the rule that it is present exactly when the status is `declined`, and explain that reopening removes it because git keeps the history.
- [x] 6.3 Update the README's `set` section with the decline examples and the reason rules, and the `rm` section with the boundary between deleting a mis-capture and declining a finding.
- [x] 6.4 Update the README's `search` section to state that declined tasks are always in scope while `done` stays behind `--all`, and why the two differ. Update the `list` section for the fourth status and the group order, and the `validate`/`--fix` lists for the two new errors that are reported and left alone.

## 7. End-to-end verification

- [x] 7.1 Add an end-to-end case to `e2e_test.go` covering the full loop: add a task, decline it with a reason, confirm the file moved to `archive/` and carries the reason, confirm a plain `backlog search` finds it and reports it as declined, confirm `backlog list` does not show it, reopen it and confirm the reason is gone.
- [x] 7.2 Run `go build ./...`, `go test ./...` and `backlog validate` against a scratch backlog containing one task in each of the four statuses, and confirm a clean build, passing tests and a zero exit from validate.
