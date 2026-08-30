## 1. Project skeleton

- [x] 1.1 Initialise the Go module and repository layout (`main.go`, `internal/` packages) and verify `go build ./...` succeeds and produces a `backlog` binary
- [x] 1.2 Implement subcommand dispatch over the standard `flag` package with a top-level help listing all eight commands, and verify `backlog` with no arguments and `backlog --help` both print the command list and exit zero
- [x] 1.3 Add a build-time version string exposed via `backlog --version`, and verify it prints a non-empty version and exits zero
- [x] 1.4 Establish the error-reporting convention (diagnostics to stderr, data to stdout, non-zero exit on failure) as a shared helper, and verify an unknown subcommand writes to stderr and exits non-zero with stdout empty

## 2. Task store

- [x] 2.1 Implement backlog root discovery by walking up from the working directory to the nearest `.backlog/`, and verify tests cover discovery from a nested directory, absence anywhere in the tree, and a nested backlog shadowing an outer one
- [x] 2.2 Define the task model and the frontmatter schema with the author-owned fields and the closed tool-owned `metadata` block, and verify round-trip tests parse a task file and re-serialise it byte-identically
- [x] 2.3 Implement tolerant parsing (unknown top-level fields preserved, missing `id` recovered from the file name, invalid status surfaced as a reported error rather than a panic), and verify tests cover each tolerance case
- [x] 2.4 Implement slug derivation from a title with Unicode letters preserved, and verify tests cover ASCII titles, Cyrillic titles, punctuation collapsing and empty-after-reduction titles
- [x] 2.5 Implement identifier allocation as the lowest unused positive integer using exclusive file creation with retry, and verify a concurrency test running parallel creations produces distinct identifiers with no lost writes
- [x] 2.6 Implement atomic task writes (write to a temporary file, then rename) and status-driven placement between `tasks/` and `archive/`, and verify tests cover a task moving to the archive on `done` and back out on `todo`
- [x] 2.7 Implement git provenance capture for branch and commit, degrading silently when the project is not a git repository, and verify tests cover both cases

## 3. Task lifecycle commands

- [x] 3.1 Implement `init` creating `.backlog/tasks/` and `.backlog/archive/` idempotently, and verify re-running it over a backlog containing tasks leaves every task untouched
- [x] 3.2 Implement `add` with title, description, tags and file references, never prompting for input, and verify tests cover minimal capture, full capture, an empty title failing without creating a task, and a run with no terminal on stdin completing
- [x] 3.3 Implement `list` with the default active-only scope, an all-tasks option, and status and tag filters in deterministic order, and verify tests cover each scope, combined filters, and the empty result exiting zero
- [x] 3.4 Implement `show` resolving a task by identifier across both directories, and verify tests cover an active task, an archived task, and an unknown identifier exiting non-zero
- [x] 3.5 Implement `set` for status transitions plus attaching a free-form reference, and verify tests cover each transition, archive movement on `done`, a no-op transition, and an invalid status exiting non-zero and leaving the task unchanged
- [x] 3.6 Implement `rm` deleting from either directory, and verify tests cover removing an existing task and an unknown identifier exiting non-zero with nothing deleted
- [x] 3.7 Implement the dual-audience output layer — human-readable listings grouped by status and a `--json` mode carrying identifier, title, status, tags, description and metadata — and verify tests assert the JSON output parses and that failures in JSON mode keep stdout free of diagnostics

## 4. Search

- [x] 4.1 Implement case-insensitive substring matching over title, description and tags only, and verify tests confirm that status values and `metadata` contents do not match
- [x] 4.2 Add regular-expression matching behind an explicit option, and verify tests cover a successful pattern and an invalid pattern exiting non-zero with a syntax error
- [x] 4.3 Apply the same scope and filters as `list` to search, and verify tests cover an archived-only match being excluded by default and included when all tasks are requested
- [x] 4.4 Implement ranking with title matches ahead of description and tag matches, ascending identifier within each rank, and verify tests assert the order and that repeated runs agree
- [x] 4.5 Implement search output with matched text in context for humans and per-match field identification in JSON, and verify tests cover both forms and that no matches exits zero with an empty result set

## 5. Validation

- [x] 5.1 Implement the finding model with error and warning severities, a strict mode promoting warnings, and exit codes, and verify tests cover clean, warnings-only, warnings-only-under-strict, and error cases
- [x] 5.2 Implement structural checks for the expected directories and stray files, and verify tests cover a missing archive directory and a non-task file among the tasks
- [x] 5.3 Implement per-file checks for frontmatter parsing, required and well-formed fields, permitted status values, tag shape, RFC 3339 timestamps, the closed `metadata` key set with a suggestion for near-miss keys, and identifier agreement with the file name, and verify tests cover each check including a misspelled `metadata` key and an unrecognised top-level field yielding only a warning
- [x] 5.4 Implement cross-file checks for duplicate identifiers, slug drift, and tasks sitting in the wrong directory for their status, and verify tests cover each case
- [x] 5.5 Implement reference checking that requires non-empty strings and resolves nothing, and verify tests cover an arbitrary reference accepted and an empty reference reported as an error
- [x] 5.6 Implement the opt-in repair mode for slug renames, directory moves, a missing format version, timestamp normalisation and tag de-duplication, and verify tests confirm each repair and that duplicate identifiers and unparseable frontmatter are left untouched
- [x] 5.7 Implement validation output grouped by file with severities and repairability, plus a JSON form, and verify tests assert the JSON output parses and carries file, severity, message and repairability per finding

## 6. Agent integration

- [x] 6.1 Author the `backlog-capture` skill content — trigger description, the three-part recording threshold, the explicit do-not-record categories, the mandatory search before add, and the requirement to supply file references — and verify the file is embedded in the binary and written on init
- [x] 6.2 Author the `backlog-triage` skill content — trigger description, reviewing accumulated tasks, run-time detection of the project's planning system, promotion, and recording the result as a free-form reference — and verify it contains no hard-coded dependency on a specific planning system
- [x] 6.3 Wire skill installation into `init`, writing both skills into the project's Claude Code skill directory with the binary version stamped in each, and verify a fresh `init` produces both files carrying the current version
- [x] 6.4 Implement modification detection so `init` skips locally edited skills and only overwrites them under an explicit force option with a warning, and verify tests cover unmodified refresh, modified skip, and forced overwrite
- [x] 6.5 Add the skill staleness check to `validate`, warning when an installed skill carries an older version than the running binary, and verify tests cover a current skill producing no warning and an older skill producing one that names the refresh command

## 7. Integration and delivery

- [x] 7.1 Add an end-to-end test covering the full capture-to-archive path — init, add with provenance, search finding it, set to doing, set to done with a reference, show from the archive — and verify it passes against the built binary
- [x] 7.2 Add an end-to-end test asserting a freshly created backlog validates clean and that a deliberately corrupted one reports the expected findings and exit code, and verify both directions pass
- [x] 7.3 Write the README covering installation, the eight commands, the task file format and the agent workflow, and verify every command example in it runs as written
