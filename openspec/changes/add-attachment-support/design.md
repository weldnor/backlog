## Context

See proposal.md — Why. Today two independent mechanisms attach "something
extra" to a task:

- `metadata.source.files` — a list of source-code paths, supplied only at
  `add` time via `--file`, never editable afterward, recorded alongside the
  git `branch`/`commit` that were captured automatically.
- `metadata.refs` — a list of free-form strings, appendable at `add` and
  `set` time via `--ref`, stored and reported verbatim.

Both are pure references: nothing is ever copied. `.backlog/` currently holds
only `tasks/` and `hooks/`; there is no facility for the tool to own a file.

The user has explicitly authorized breaking the on-disk format: this design
merges the two mechanisms and adds physical storage, without preserving the
old shape.

## Goals / Non-Goals

**Goals:**
- One `metadata.attachments` list replacing `source.files` and `refs`, with
  two kinds: `file` (a path, project-root-relative) and `link` (a free-form
  string, never interpreted — the old `refs` behavior exactly).
- A `file` attachment's path MAY point at an ordinary tracked file (today's
  `--file` behavior: cheap, no copy) or at a copy the backlog physically owns
  under `.backlog/attachments/<id>/`.
- A CLI path to physically attach a local file to an existing task, and to
  detach (and, when backlog-owned, delete) one.
- `backlog validate` treats a backlog-owned attachment the way it already
  treats task files: a missing one is an error, an unreferenced one on disk
  is a warning.
- Parity between the CLI and the browse UI/JSON API, matching every other
  capability in this codebase.

**Non-Goals:**
- No automatic migration of existing `metadata.source.files` /
  `metadata.refs` task files. See Migration Plan.
- No attachment metadata beyond type and value — no titles, captions,
  content-type sniffing, thumbnails, or size limits. `refs` already
  established that this tool records strings and paths verbatim; the same
  restraint applies to stored files.
- No full-replace editing of the attachment list (no `edit --attach`). See
  Decisions.
- No upload at `add` time. An identifier does not exist until `Store.Create`
  returns, and `.backlog/attachments/<id>/` is named by it; uploading is only
  possible once a task exists. `add --attach` stays reference-only, matching
  `--file`'s existing zero-I/O behavior.
- No change to how `branch`/`commit` provenance is captured or displayed.

## Decisions

### One `Attachment{Type, Value}`, not separate file/link slices

```go
type AttachmentType string

const (
    AttachmentFile AttachmentType = "file"
    AttachmentLink AttachmentType = "link"
)

type Attachment struct {
    Type  AttachmentType
    Value string
}
```

`Value` is a project-root-relative, forward-slash path for `file`, and a
free-form string for `link`, stored and reported verbatim exactly as `refs`
is today. Frontmatter renders the two kinds with different keys —
`path` for `file`, `url` for `link` — because the file format is
hand-edited and a bare `value:` key would leave a reader guessing which kind
of thing it is; `AttachmentKeys = ["type", "path", "url"]` is the closed key
set per list entry, checked the same way `SourceKeys` is today. Go and JSON
keep the single `Value` field — a UI or script consuming the JSON never has
to branch on which key name to read.

```yaml
metadata:
  attachments:
    - type: file
      path: internal/auth/session.go
    - type: file
      path: .backlog/attachments/007/screenshot.png
    - type: link
      url: "issue:1423"
```

A `file` entry whose `path` is not under `.backlog/attachments/<id>/` is a
plain reference (today's `--file`, unchanged in spirit). One whose `path` is
under that directory is backlog-owned: the convention itself carries the
distinction, so no separate `stored: true` field is needed. Alternative
considered — a `stored` boolean alongside `path` — rejected: it can go out of
sync with where the path actually points, whereas the path prefix cannot.

### `metadata.source` keeps `branch` and `commit`, loses `files`

`Source.Files` is deleted from `internal/task.Source`; `Branch`/`Commit` are
untouched, still captured automatically from git and never user-supplied.
`task.New`'s signature drops its `files []string` parameter — its `attach
[]Attachment` parameter replaces both `files` and `refs`.

### Reference vs. upload is two different verbs, not one flag with a bit

`backlog attach add <id> file:<path>` / `link:<value>` records a reference —
no file I/O beyond the write to the task file, exactly like today's
`--ref`/`--file`. `backlog attach add <id> --upload <local-path>` copies
bytes: it reads `<local-path>`, writes it into
`.backlog/attachments/<id>/<name>` (collision-safe: `name.png`, `name-2.png`,
...), and records a `file` attachment pointing at the stored copy. The two
are mutually exclusive on one invocation. Kept separate rather than one flag
that sometimes copies and sometimes doesn't, because whether an operation
touches the filesystem beyond the task file is worth being explicit about at
the call site — the same reasoning that keeps `link add` (an in-repo pointer)
and this command (which can duplicate bytes) visibly different operations.

### `kind:value` syntax, matching `--link type:id`

`--attach` (on `add`) and the positional form of `attach add`/`attach rm` both
take `<kind>:<value>`, split on the first `:` only — `link:issue:1423` parses
to `{type: link, value: "issue:1423"}`. This was chosen over sniffing the
value's shape (e.g. "looks like a URL") specifically because the existing
`--ref` examples in this codebase's own README (`--ref "issue:1423"`) are not
URL-shaped at all; a heuristic would silently misclassify exactly the
examples already in use. It also matches `--link`'s existing `type:id`
convention, so the idiom is already familiar.

### No `edit --attach`

`edit --tag` and `edit --link` are full-replace: the caller restates the
entire list. That is safe for tags and task-links because they are short and
typed by hand. An attachment list can contain a backlog-owned stored path
like `.backlog/attachments/007/screenshot.png` that nobody types from memory;
a full-replace flag would make "forgot to restate one entry" silently drop a
stored file's only reference (the file itself would remain on disk,
undetected until the next `validate`, as an unreferenced stray). Attachments
are therefore mutated only incrementally, through `attach add`/`attach rm` —
there is no bulk form to get wrong.

### `attach add`/`attach rm` fire the edit hooks, not the set hooks

Today's `set --ref` fires `PreSet`/`PostSet` — the workflow-state hooks — because it rides along inside `set`. Once attachment mutation is its own command, it is grouped with `link add`/`link rm`, which fire `PreEdit`/`PostEdit` on the stated reasoning that "a link change is content, not workflow state" (`linkcmd.go`). An attachment is the same kind of thing — evidence and pointers, not a status — so `attach add`/`attach rm` and the two new browse endpoints all fire `PreEdit`/`PostEdit`, matching `link`, not `set`. A project with a hook keyed on `set`'s events for the old `--ref` behavior will need to move it to the edit hook; this is called out for the same reason the format break is: compatibility is not preserved here.

### `attach rm` deletes the backlog-owned file it points at

Matching `type` and `value` exactly (as `link rm` matches `type` and `id`
exactly), `attach rm` removes the entry from the task and, when its `Value`
path is under `.backlog/attachments/<id>/`, deletes that file too — a
detached stored file has no other purpose and no other referrer (attachments
are not shared across tasks). A plain reference (a path elsewhere in the
repo, or a link) is removed from the list only; nothing else exists to
delete.

### `backlog rm` deletes the task's attachment directory

`Store.Remove` now also removes `.backlog/attachments/<id>/` if it exists,
after removing the task file, best-effort (an error removing the directory
does not fail the command — the task is already gone, and `validate` will
flag the leftover directory as orphaned on the next run). Without this, every
`rm` of a task with stored attachments would leave dead weight behind
permanently, since nothing else ever revisits an identifier once it is
freed for reuse.

### Validation additions

- Per-entry: `type` must be `file` or `link`; `file` requires non-empty
  `path` and rejects `url`; `link` requires non-empty `url` and rejects
  `path`; an unknown key under an entry is an error with a "did you mean"
  suggestion, mirroring `SourceKeys`.
- A `file` attachment whose `path` is under `.backlog/attachments/<id>/` but
  does not exist on disk: error, not repairable — the file is gone, and
  fabricating one is not something `--fix` can do.
- A file present under `.backlog/attachments/<id>/` that no attachment in
  task `<id>` references: warning, not repairable, mirroring `StrayFiles`'s
  treatment of the task directory.
- A `.backlog/attachments/<N>/` directory where task `N` no longer exists
  (its file was removed by hand rather than through `rm`, or `rm` failed to
  clean it up): warning, not repairable, for the same reason.
- A `file` attachment whose `path` is *not* under the attachments directory
  is a plain reference and is never checked for existence — matching
  `--file`'s current behavior exactly; requiring existence would make
  `backlog add --attach file:<path>` order-dependent on when the code is
  written relative to when the finding is filed, which is not how it works
  today.

## Risks / Trade-offs

- **Breaking the file format** → Existing task files with
  `metadata.source.files` or `metadata.refs` are read as carrying an
  unrecognised metadata key the moment this ships (an error the parser
  reports on that task, not a crash — the existing tolerant-read path). This
  is deliberate per the request; see Migration Plan.
- **Two verbs to learn instead of one** (`attach add ... file:path` vs.
  `attach add ... --upload path`) → Accepted: the alternative (one flag,
  inferred behavior) hides whether a call duplicates a file's bytes, which
  is exactly the distinction most worth surfacing.
- **`rm`'s directory cleanup is best-effort** → A permissions error or an
  open file handle on the attachments directory could leave it behind after
  `rm` reports success. Accepted: the task itself is correctly gone (the
  primary contract of `rm`), and `validate` catches the orphaned directory
  afterward the same way it catches any other stray.
- **No dedup across tasks** → Two tasks uploading the same screenshot store
  two copies. Accepted: content-addressing or cross-task sharing adds real
  complexity (reference counting on delete) for a case that is expected to
  be rare in a per-repository findings backlog.

## Migration Plan

Breaking, by explicit request. There is no automatic converter. A project
upgrading past this change will see `backlog validate` report every task
file that still carries `metadata.source.files` or `metadata.refs` as
invalid (unrecognised metadata key). The fix is a hand edit — or a small
one-off script, since the transform is mechanical: each `source.files` entry
becomes `{type: file, path: ...}`, each `refs` entry becomes
`{type: link, url: ...}` — but writing and shipping that script is left out
of this change's scope. `SchemaVersion` is not bumped for this: schema
carries the *frontmatter version*, not a migration marker, and this change
does not add a reader path for the old shape for the new schema value to
distinguish.

Rollback is reverting the change (including the rebuilt embedded frontend
bundle); a backlog only ever written by the reverted binary is unaffected,
since nothing here touches files outside a project that has already run the
new binary.
