## 1. Store: collapse to one directory

- [x] 1.1 In `internal/store/store.go` remove `ArchiveDir`, `ArchivePath()`, `DirFor()` and the `Archived` field on `Entry`; make `Entries`, `UsedIDs`, `StrayFiles` and `Init` operate on `TasksPath()` only, and drop the `Archived` tiebreak in `Entries`' sort. Verify `go build ./...` passes.
- [x] 1.2 In `internal/store/write.go` make `Create` and `Save` always target `TasksPath()` and reduce `idTakenByAnotherFile` to the single directory, keeping the in-place slug-rename cleanup in `Save`. Verify the concurrent-create test still passes (`go test ./internal/store/`).
- [x] 1.3 Update `internal/store/store_test.go` to drop every `archive/` path assertion and add a test that setting a task to `done` then `declined` leaves the file in `tasks/`. Verify `go test ./internal/store/` passes.

## 2. Task package

- [x] 2.1 In `internal/task/task.go` drop the "lives in the archive directory" wording from the `IsTerminal` and `StatusDeclined` doc comments; keep `IsTerminal` and its behaviour. Verify `go test ./internal/task/` passes.

## 3. CLI: list subcommands and scope

- [x] 3.1 Rewrite `internal/cli/scope.go`: remove `all`, `statuses`, `alwaysDeclined`, `register`'s `--all` and `--status` flags, `searchScope()`; `selected()` returns either one explicitly set status or all four. Keep `registerPriority` and the tag filter. Verify `go build ./...` passes.
- [x] 3.2 In `internal/cli/lifecycle.go` `runList`, consume a leading `todo|doing|done|declined` arg as a single-status filter, reject any other non-flag leading arg with a usage error listing the four names, and parse `--tag`/`--priority`/`--json` from the rest. Verify `backlog list done` and bare `backlog list` behave per the `Listing tasks by status` spec.
- [x] 3.3 In `internal/cli/search.go` remove `--all` and the status filter; search always covers every status; keep `--tag`, `--regex`, `--json`. Verify `go build ./...` passes.
- [x] 3.4 Update `internal/cli/cli.go` help text / subcommand summary for `list` to mention the status subcommands. Verify `backlog list -h` shows them.
- [x] 3.5 Update `internal/cli/scope_test.go`, `search_test.go`, `declined_test.go`, `priority_test.go` and the list tests to the new surface (no `--all`/`--status`, subcommands instead, search covers every status). Verify `go test ./internal/cli/` passes.

## 4. Validation

- [x] 4.1 In `internal/validate/validate.go` make the structural check expect only `tasks/`, and delete the terminal-outside-archive and non-terminal-inside-archive cross-file findings. Verify `go test ./internal/validate/` compiles.
- [x] 4.2 In `internal/validate/repair.go` delete the `DirFor`-based relocation step from `--fix`. Verify remaining repairs (slug, schema, priority, timestamps, tags) still apply.
- [x] 4.3 Update `internal/validate/validatecmd_test.go` and `validate` unit tests: drop "missing archive directory" and misplacement cases, add a case asserting a `done` task in `tasks/` is not flagged. Verify `go test ./internal/validate/` passes.

## 5. Docs and skills

- [x] 5.1 Update `README.md`: replace the two-directory layout block with one `tasks/` directory, remove the `--all` rows from the `list` and `search` tables, document the `backlog list <status>` subcommands, and remove every "moved to the archive" phrase. Verify no remaining `archive/` or `--all` mention with `grep -n "archive\|--all" README.md`.
- [x] 5.2 Update `internal/skills/files/backlog-capture.md` and `internal/skills/files/backlog-triage.md` to drop `--all` and archive references and bump the skill version stamp. Verify `grep -rn "archive\|--all" internal/skills/files/` is clean.

## 6. End-to-end and final check

- [x] 6.1 Update `e2e_test.go` for the single directory, the `list` subcommands and `search` with no `--all`. Verify `go test ./...` passes.
- [x] 6.2 Run `gofmt -l .`, `go vet ./...` and `go test ./...`; verify all clean.
- [x] 6.3 Run `openspec validate remove-archive-directory --strict` and verify it passes.
