## Context

See `proposal.md` — Why. The archive directory is a pure function of `status`
(`store.DirFor` = `IsTerminal(status) ? archive : tasks`), yet the code carries
it as a first-class concept: `Entry.Archived`, two-directory loops in
`Entries`, `UsedIDs`, `StrayFiles`, `idTakenByAnotherFile` and
`StrayFiles`, file moves in `Save`/`Create`, drift checks in `validate` and a
relocation step in `validate --fix`.

The list and search commands share `internal/cli/scope.go`. `list` has
`--all`, `--status` (repeatable), `--tag` and `--priority`; `search` has
`--all`, `--status`, `--tag` plus `searchScope()` (declined always in scope).
Dispatch in `internal/cli/cli.go` is a flat table of top-level subcommands;
there is currently no nested-subcommand pattern.

The user will migrate existing backlogs by hand
(`git mv .backlog/archive/* .backlog/tasks/`); no migration code is in scope.

## Goals / Non-Goals

**Goals:**
- One task directory, `.backlog/tasks/`, holding every task.
- `store` has no notion of an archive: no `ArchiveDir`, `ArchivePath`,
  `DirFor`, `Entry.Archived`; every directory walk visits one directory.
- `backlog list` shows every status by default; `todo`/`doing`/`done`/`declined`
  subcommands narrow to one status; no `--all`, no `--status`.
- `backlog search` covers every status; no `--all`, no status selector.
- `IsTerminal` stays (it still classifies status for grouping and for the
  decline-reason rules); only its "lives in the archive directory" doc rationale
  goes.

**Non-Goals:**
- Migration tooling or backward-compatible reading of a legacy `archive/`.
- Renaming the `doing` status (raised and declined during planning).
- Any change to the task file format, frontmatter, or the four status values.
- Changing what `browse` shows by default.

## Decisions

### List status selection: subcommands, not a flag

`backlog list <status>` replaces `backlog list --status <status>`. The four
names are a closed set that maps one-to-one to `task.Statuses`, so a subcommand
reads better than a repeatable flag and there is no meaningful "two statuses but
not the third" case that the flag supported and the subcommand does not — the
bare command already covers "everything".

Implementation: `runList` inspects `args[0]`; if it is one of the four status
names, it is consumed and becomes a single-status filter, otherwise the args
are parsed as flags only. An unknown first token that is not a flag is a usage
error listing the four names. `--tag`/`--priority`/`--json` continue to be
parsed from the remaining args in both forms.

Alternative considered: keep `--status` as well. Rejected — two ways to say the
same thing, and the proposal's intent is to shrink the surface.

### `scope` loses `all`, `statuses`, `alwaysDeclined`

`scope.selected()` collapses to: the single status from a list subcommand, or
all four. `search` passes no status constraint at all. `registerPriority` and
the tag filter are unchanged. `searchScope()` and the declined-always-in-scope
reasoning are deleted, because "every status is in scope" subsumes them.

### `store` collapses to one directory

- Delete `ArchiveDir`, `ArchivePath()`, `DirFor()`, `Entry.Archived`.
- `Init` creates only `tasks/`.
- `Entries`, `UsedIDs`, `StrayFiles`, `idTakenByAnotherFile` walk `TasksPath()`
  only. The "archived tasks after active ones of the same identifier" tiebreak
  in `Entries` goes away with `Archived`; ordering falls back to name.
- `Save` and `Create` always target `TasksPath()`; `Save` still renames the
  file in place when the slug changed, but never moves between directories.
  The `t.Path != target` cleanup branch in `Save` still handles a slug rename.
- The concurrent-id-claim logic in `write.go` keeps working unchanged against
  the single directory (it already special-cased "this directory" vs "the
  other one"; the other one is simply gone).

### `validate` drops the placement checks

- Structural check: expect `tasks/` only; drop the `archive/` stat.
- Cross-file: delete the terminal-outside-archive and non-terminal-inside-archive
  findings (`validate.go` ~L179-189).
- `repair.go`: delete the `wantDir := st.DirFor(...)` relocation step.
- `task.IsTerminal` doc comment: drop the archive sentence; keep the function.

### Docs and skills

`README.md` loses the `archive/` layout diagram, the `--all` rows and every
"moved to the archive" phrase. `internal/skills/files/backlog-capture.md` and
`backlog-triage.md` lose their `--all` / "archive" references. Skills carry a
version stamp, so the stamp bumps, `backlog init` refreshes them, and
`validate` flags older installed copies — existing behaviour, no code change
needed beyond the file edits.

## Risks / Trade-offs

- **A user upgrades without moving `archive/`** → their done/declined tasks
  vanish from every command and their ids look free, so `add` could reallocate
  one. Mitigation: the proposal and the `task-store` REMOVED migration note
  both spell out the one-line `git mv`; `validate` will also report the
  now-stray `archive/` directory as an unexpected entry under `.backlog/`.
  Accepted because the user explicitly asked to own migration.
- **`git` history of a moved task** → moving `archive/NNN-*.md` to `tasks/`
  is a rename `git` follows; no content churn. Low.
- **Larger `tasks/` directory over time** → the tidiness the archive bought is
  lost. This is the deliberate trade the proposal makes; `backlog list <status>`
  and `browse` are the ways to focus.
- **Scripts parsing `backlog list --all`** → break loudly with an unknown-flag
  error, which is the intended signal.

## Migration Plan

1. Land the code + spec + doc changes together (single breaking release).
2. Release notes: "Run `git mv .backlog/archive/* .backlog/tasks/ &&
   rmdir .backlog/archive` once after upgrading. Replace `backlog list --all`
   with `backlog list` and `backlog list --status X` with `backlog list X`.
   Drop `--all` from `backlog search`."
3. Rollback: revert the release; a hand-migrated backlog still reads correctly
   on the old binary because every task file is valid in `tasks/` — the old
   binary would just also recreate an empty `archive/` and `validate --fix`
   would move terminal tasks back into it.
