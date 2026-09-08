## Why

The `.backlog/archive/` directory is a second, redundant representation of a fact
already carried by each task's `status` field: `DirFor` is literally
"terminal status → archive, else tasks". It buys one thing — a tidier `tasks/`
listing for the hand-editing workflow — and pays for it with a whole class of
"file in the wrong directory" drift, dual-source-of-truth checks in `validate`,
move-on-transition churn, and per-directory iteration in every read and write
path. Status is the single source of truth; the directory should not exist.

## What Changes

- **BREAKING** A backlog is a single task directory, `.backlog/tasks/`, holding
  tasks in every status. `.backlog/archive/` is removed. No automatic migration
  is performed — an existing backlog's `archive/*.md` files must be moved into
  `tasks/` by hand (the maintainer will do this).
- **BREAKING** `backlog list` no longer has an `--all` flag and no longer has a
  `--status` flag. Bare `backlog list` now shows tasks in **every** status.
  Four status subcommands narrow to one status: `backlog list todo`,
  `backlog list doing`, `backlog list done`, `backlog list declined`. The
  `--tag` and `--priority` filters are unchanged and apply to both forms.
- **BREAKING** `backlog search` no longer has an `--all` flag. Search always
  covers tasks in every status.
- `backlog set`, `backlog show`, `backlog rm`, `backlog init` and
  `backlog validate` stop moving files between directories, stop looking in two
  places, and stop reporting a task as misplaced because of the directory it
  sits in. `validate --fix` no longer relocates files by status.
- The installed agent skills and `README.md` drop every mention of the archive
  directory and the `--all` flag.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `task-store`: the directory layout collapses from two task directories to one;
  identifier allocation and durable writes stop referring to "both directories".
- `task-management`: `init` creates one directory; `list` gains status
  subcommands and loses `--all` and `--status`, with the default scope widening
  to every status; `set`/`show`/`rm` stop describing file moves between an
  active and an archive directory.
- `task-search`: search scope is unconditionally every status; the "all tasks"
  request and its flag are removed.
- `backlog-validation`: structural checks expect one task directory; the
  cross-file "task in the wrong directory for its status" checks are removed;
  automatic repair no longer moves files by status.
- `browse-ui`: the filtering description no longer refers to
  `backlog list --all`, since that flag is gone and the default list scope is
  now every status.

## Impact

- Code: `internal/store` (`store.go`, `write.go` — remove `ArchiveDir`,
  `ArchivePath`, `DirFor`, `Entry.Archived`, two-directory loops),
  `internal/cli` (`scope.go`, `lifecycle.go`, `search.go`, `initcmd.go`,
  `validatecmd.go`, list subcommand wiring), `internal/validate`
  (`validate.go`, `repair.go`), `internal/task` (`task.go` doc comments;
  `IsTerminal` stays, only its "lives in the archive directory" rationale goes).
- Specs: `task-store`, `task-management`, `task-search`, `backlog-validation`,
  `browse-ui`.
- Docs and agent guidance: `README.md`,
  `internal/skills/files/backlog-capture.md`,
  `internal/skills/files/backlog-triage.md` (skill version stamp bumps, so
  `backlog init` refreshes them and `validate` flags older copies).
- Tests: `internal/store`, `internal/cli`, `internal/validate` suites and
  `e2e_test.go` — anywhere that asserts on `archive/` paths or `--all`.
- Existing backlogs: a one-time manual `git mv .backlog/archive/* .backlog/tasks/`
  by each maintainer. The new binary does not read `archive/` at all.
