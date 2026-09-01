## Why

Every command today is a single-shot terminal invocation, which is right for an
agent but effortful for a person: reviewing a backlog means running `list`,
running `show` for anything worth a closer look, and typing a `set` invocation
by hand for every field to change. There is no way to skim a backlog, open a
task, edit it and move to the next one without re-typing an identifier each
time. `browse` is a local web UI for exactly that human loop — looking at what
has accumulated, creating a task by hand, and editing one — while every
machine-facing path (agents, scripts, `--json`) keeps using the existing
commands untouched.

This is still not a task-execution system. `browse` shows the same four
statuses the CLI already has and lets a person change them; it adds no board,
no milestones, no assignment, and no notion of "next" beyond what `status`
already means. It is a second interface onto the same store, not new
scheduling machinery.

## What Changes

- A new `backlog browse` command that starts a local HTTP server bound to
  `127.0.0.1` by default, serves an embedded single-page web UI, and opens it
  in the default browser (`--no-open` to skip). The port is OS-assigned unless
  `--port` is given; `--host` overrides the bind address for the rare case a
  person needs it reachable from outside the loopback interface, with a
  printed warning that doing so exposes an unauthenticated write API.
- The UI lists tasks grouped by status the way `backlog list --all` does,
  with the same status, tag and priority filters and a client-side text
  filter over what is already loaded. It is a grouped list, not a board: there
  is no drag-and-drop, and moving a task between groups is done through the
  same edit panel as every other field.
- The UI can create a task: title (required), description, tags, priority,
  source files and references — the same fields `backlog add` accepts.
  Everything created through the UI is recorded with `author: human`, since
  that is who is on the other end of a browser.
- The UI can edit every field of an existing task in one panel: title,
  description, tags, priority, status and, when declined, the reason — going
  beyond what `backlog set` exposes today (title, description and tags are
  currently not editable from the CLI at all). Validation mirrors `add` and
  `set` exactly: an empty title is rejected, an invalid status or priority is
  rejected, a reason is required exactly when the resulting status is
  `declined` and rejected otherwise.
- Deleting a task through the UI is out of scope; `rm` remains the only way to
  permanently remove one. This keeps `browse`'s first version to what was
  asked for — creating and editing — without adding a destructive action to a
  surface that has no confirmation-by-typing the way a terminal command does.
- The visual design follows Claude's own product: a warm paper background, a
  serif display face for headings paired with a plain system sans-serif for
  everything else (no network font loading — the binary has to keep working
  offline), the signature terracotta accent for primary actions and priority,
  generous rounded corners and soft borders instead of hard drop shadows, and
  an automatic dark palette that follows `prefers-color-scheme`.
- Not breaking. No existing command, flag, output shape or file format
  changes. `browse` is additive: a project that never runs it is unaffected.

## Capabilities

### New Capabilities

- `browse-ui`: a local web UI, served by a new `backlog browse` command, for
  listing, creating and editing tasks by hand. Covers starting and binding the
  server, the task list and its filters, task creation, task editing and its
  validation parity with `add`/`set`, and the visual design system the UI is
  built on.

### Modified Capabilities

None. `browse` adds a new command and a new HTTP-facing surface without
changing the behaviour, output or file format of any existing command.

## Impact

- `internal/cli`: a new `browsecmd.go` registering `browse` in `commands()`
  and parsing its flags (`--port`, `--host`, `--no-open`, `--json`).
- A new `internal/browse` package: the HTTP server and its handlers (backed by
  `internal/store` and `internal/task`, reusing `TaskView`/`view`/`views` from
  `internal/cli/output.go` — or an equivalent moved somewhere both packages can
  reach — so the JSON shape the UI consumes is identical to the CLI's), the
  create/edit validation, cross-platform browser launching, and graceful
  shutdown on interrupt.
- A new `internal/browse/web` (or similarly named) directory of static
  assets — HTML, CSS and vanilla JS, no build step and no new module
  dependency — embedded into the binary with `go:embed` so `go install`
  continues to produce one self-contained binary.
- `README.md`: a new `browse` section in the command reference.
- No new Go module dependencies: the server uses `net/http` from the standard
  library, and the frontend has no package of its own to depend on.
- No change to the on-disk task format, to `metadata.schema`, or to any
  existing command's behaviour.
