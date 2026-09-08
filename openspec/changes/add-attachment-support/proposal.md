## Why

A task can currently carry two disjoint kinds of "something extra": `--file`, a
source-code path the finding concerns, and `--ref`, a free-form string
pointing at external work. Both are references only — nothing can be
physically attached to a task. A finding that comes with a screenshot, a log
excerpt, or a crash dump has nowhere to go; the agent or reviewer has to
paste it into the description as prose, or leave it out.

`--file` and `--ref` also solve the same problem in slightly different
clothes: each is a pointer from a task to something else. Keeping them as two
separate flags, two separate frontmatter keys, and two separate JSON fields
buys no real distinction — a file path and a URL are both "evidence for this
finding" — while doubling the surface every consumer (CLI, store, browse API,
frontend) has to carry.

This change merges them into one `attachments` concept with two kinds,
`file` and `link`, and adds the missing capability: a `file` attachment can
now be a real file, physically copied into the backlog under
`.backlog/attachments/<id>/`, not just a reference to something that already
exists elsewhere. Compatibility with the old `metadata.source.files` /
`metadata.refs` shape is explicitly not preserved — see design.md.

## What Changes

- **BREAKING**: `metadata.source.files` and `metadata.refs` are removed from
  the task file format and replaced by `metadata.attachments`, a list of
  `{type: file, path: ...}` or `{type: link, url: ...}` entries. `metadata.source`
  keeps `branch` and `commit` only.
- `backlog add --file` and `backlog add --ref` are removed; `backlog add`
  gains a repeatable `--attach file:<path>` / `--attach link:<value>` flag for
  the reference-only case (no file is copied — the same zero-I/O behavior
  `--file` and `--ref` had).
- `backlog set --ref` is removed. Attaching or removing an attachment after a
  task exists moves to a new `backlog attach add|rm <id> ...` command, mirroring
  `backlog link add|rm`. `attach add` also accepts `--upload <local-path>` to
  physically copy a file into `.backlog/attachments/<id>/` and record it as a
  `file` attachment pointing at the stored copy. `attach rm` deletes the
  stored file too when the attachment it removes is one the backlog owns.
- `backlog rm` also removes a task's `.backlog/attachments/<id>/` directory,
  if any, so a deleted task leaves no orphaned files behind.
- `backlog validate` gains checks for the new format: unknown keys, a `file`
  attachment missing exactly one of `path`/`url` for its `type`, a
  backlog-owned file that no longer exists on disk, and a file physically
  present under `.backlog/attachments/<id>/` that no attachment references.
- The browse UI's JSON API replaces `files`/`refs` on create with
  `attachments`, and gains `POST /api/tasks/{id}/attachments` (JSON body for a
  reference, multipart upload for a stored file) and
  `DELETE /api/tasks/{id}/attachments` (by `type`+`value`) so the web UI can
  manage attachments the same way the CLI does.
- The task detail dialog shows one "Attachments" list (file references,
  stored files, and links together) instead of separate source-files and refs
  sections, with inline controls to add a link, upload a file, and remove an
  entry — applied immediately, the same way the dialog's Delete action is.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `task-store`: `metadata.source.files` and `metadata.refs` are replaced by
  `metadata.attachments`; a new requirement covers the
  `.backlog/attachments/<id>/` storage directory and how stored files are
  validated.
- `task-management`: `add` drops `--file`/`--ref` for `--attach`; `set` drops
  `--ref`; a new `attach` command (`add`/`rm` subcommands) covers
  reference-only and stored-file attachments, including `rm`'s cleanup of a
  stored file; `rm` additionally removes a task's attachment directory.
- `browse-ui`: task detail, creation and the JSON API are updated for
  `attachments` in place of `files`/`refs`, and gain endpoints and UI for
  adding and removing attachments (including file upload) from an existing
  task.

## Impact

- Backend: `internal/task` (`Attachment` type, frontmatter read/write,
  closed-key validation), `internal/store` (attachment directory helpers,
  `rm` cleanup), `internal/cli` (`add`, new `attachcmd.go`, `rm`), `internal/
  validate` (new checks), `internal/taskview`, `internal/browse/handlers.go`
  (create/patch bodies, two new routes).
- Frontend: `frontend/src/api.ts` (types, new client calls),
  `frontend/src/components/EditForm.tsx`, `MetadataAside.tsx` (or a new
  `AttachmentsPanel`), `frontend/src/App.tsx` (attach/detach handlers), embedded
  bundle rebuild.
- Docs: `README.md`'s `--file`/`--ref` documentation, and
  `internal/skills/files/backlog-capture.md`, which currently instructs the
  agent to use `--file`.
- No migration tooling: an existing task file with `metadata.source.files` or
  `metadata.refs` is read as having an unrecognised metadata key once this
  ships (reported by `backlog validate`, not a crash) and needs hand-editing
  to the new shape. See design.md's Migration Plan.
