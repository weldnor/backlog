## Context

See proposal.md — Why. The constraints that actually shape the approach are
properties of the existing code and of what the binary promises today:

- **One self-contained binary, no runtime dependencies.** `go install
  github.com/weldnor/backlog@latest` has to keep producing a single binary
  that needs nothing else on disk or on the network at run time. A web UI
  therefore has to be embedded (`go:embed`), and its assets cannot load fonts,
  scripts or styles from a CDN — the binary is expected to work offline, on an
  airgapped machine, the same as every other command.
- **The store's write paths are already safe for concurrent callers.**
  `Store.Create` claims an identifier with an exclusive file create and
  re-checks after claiming (internal/store/write.go), and `Store.Save` writes
  through a temp-file-plus-rename (`WriteFileAtomic`). Both were built to
  survive two separate CLI processes racing; concurrent goroutines inside one
  `browse` process racing the same way are no more dangerous, so the server
  needs no additional locking to stay correct.
- **`internal/cli` already has the JSON shape the UI needs.** `TaskView` and
  `view`/`views` in `internal/cli/output.go` are the exact contract agents
  already consume from `--json`. The UI should see the same shape, not a
  second one that can quietly drift from it.
- **Only `status`, `priority` and `reason` can be changed today, and only by
  `set`.** `title`, `description` and `tags` have no CLI editing path at all —
  `add` sets them once at creation. `browse`'s edit panel is therefore the
  first place these fields can be changed after creation, so its validation
  has to be written from the same rules `add` and `set` already enforce
  (non-empty title, valid status/priority, reason iff declined), not
  delegated to them, since neither command currently accepts a title or
  description to change.
- **The project's existing design vocabulary is plain text.** There is no
  prior UI, no CSS, no frontend tooling and no design file to extend —
  "Claude's own product" is the reference to build the visual language from,
  translated into a static, offline, dependency-free page.

## Goals / Non-Goals

**Goals:**

- A person can start `backlog browse`, see the current backlog grouped by
  status, create a task, and edit any field of an existing one, all without
  touching a terminal after the initial command.
- The UI and the CLI never disagree about what a task looks like: the same
  `TaskView` shape, the same validation rules, the same file on disk.
- The server is safe to run against a real project by default: bound to
  loopback, with the exposure of binding wider being an explicit, warned-about
  choice rather than a default.
- The binary stays exactly what it is today otherwise — one file, no new
  dependencies, every existing command byte-for-byte unchanged.

**Non-Goals:**

- No task deletion from the UI. `rm` stays the only way, as argued in
  proposal.md.
- No board, no drag-and-drop, no multi-task/bulk actions. The list groups by
  status for orientation; moving a task between groups happens through the
  edit panel like any other field change, not by dragging a card.
- No authentication, no HTTPS, no CSRF protection. The trust boundary is the
  same one the CLI already has — whoever can run `backlog` already has full
  read/write access to `.backlog/` on disk — and is discussed under Decisions.
- No regex search and no server-side text search endpoint. The UI's text
  filter runs client-side over the already-fetched list, matching the same
  substring-only spirit as `search` but not calling into `internal/search` or
  exposing its ranking; a person who needs `search`'s regex mode or its
  declined-always-in-scope behaviour still reaches for the CLI.
- No manual light/dark toggle. The palette follows `prefers-color-scheme`
  automatically; a toggle is a reasonable future addition, not part of this
  change.
- No live updates (polling or websockets) when the backlog changes on disk
  from elsewhere (an agent running `backlog add` concurrently, a hand edit).
  The list reflects what was loaded when the page opened or last refreshed
  itself after an action taken in the UI.

## Decisions

### `browse` is a new package, not more code in `internal/cli`

The HTTP server, its handlers, the create/edit validation and the embedded
assets live in a new `internal/browse` package. `internal/cli/browsecmd.go`
stays a thin adapter — parse flags, open the store, call
`browse.Serve(store, options)` — the same shape `initcmd.go` has around
`internal/skills`.

*Alternative considered:* put the handlers directly in `internal/cli`.
Rejected — the package already carries eight commands' worth of flag parsing
and output formatting; an HTTP server and a directory of embedded static
assets are a different kind of code with a different kind of test, and keeping
them apart keeps `go vet`/review scope narrow when only one side changes.

### The API reuses `TaskView`, not a second JSON shape

`internal/browse` calls into the existing `view`/`views` functions (exported,
or the minimal exported wrapper needed to reach them from another package)
rather than defining its own task JSON struct. The UI's fetch layer and
`backlog --json` output are then guaranteed to agree, and a future field added
to `TaskView` reaches the UI automatically.

*Alternative considered:* a UI-specific view struct, free to add UI-only
fields. Rejected — nothing the UI needs falls outside what `TaskView` already
carries, and a second shape is a second place for the two to drift apart.

### Loopback by default; `--host` is an explicit, warned override

The server binds `127.0.0.1` unless `--host` says otherwise. There is no
authentication: the API can create, edit, and read every task with no
credential, which is fine as long as only local processes on the same machine
can reach it — exactly the access a person already has to `.backlog/*.md` on
that machine. Passing `--host 0.0.0.0` (or any non-loopback address) makes
that write API reachable from the network, so `browse` prints a warning to
stderr naming that risk the moment such a host is used, and keeps serving —
the flag exists for a person who has already decided they need it (a remote
dev container, a VM), not to make the choice for them.

*Alternative considered:* require a token in the URL or a header for every
request, printed at startup like some local dev tools do. Rejected for this
change — it adds a credential to copy into the browser for a server that is
loopback-only by default and therefore already only as reachable as the
filesystem is, and it can be added later without breaking anything if `--host`
usage turns out to be common enough to need it.

### The frontend is vanilla HTML/CSS/JS with no build step

No framework, no bundler, no `package.json`. `internal/browse/web` holds
`index.html`, one stylesheet and one script, embedded with `go:embed` and
served as-is. `go build`/`go install` need nothing beyond the Go toolchain
already required, and there is no `node_modules` step to keep in sync with the
Go build.

*Alternative considered:* a small framework (Preact, htm) for the edit panel's
state handling. Rejected — the UI is a handful of views (list, create form,
edit panel) over one JSON API; the state involved does not justify a
dependency that would need a build step to produce the embedded assets, which
is exactly the property being protected.

### Editing is a full-record `PATCH`, validated independently of `set`

The edit panel sends whatever subset of `{title, description, tags, priority,
status, reason, refs}` changed. `internal/browse` validates it directly
against `internal/task`'s constants (`ValidStatus`, `ValidPriority`) and the
same reason-iff-declined rule `set` enforces, then loads the task, applies the
change, and calls `Store.Save` — the same write path `set` uses. `tags` and
`refs` are replaced wholesale when present in the request, matching a form
that shows and edits the whole list, unlike the CLI's repeatable `--tag`/
`--ref` flags which only ever add.

*Alternative considered:* have the handler shell out to (or call in-process)
the same flag-parsing code `runSet` uses. Rejected — `runSet` has no path for
title or description at all, so the handler would still need its own
validation for those two fields; splitting one edit across two validation
paths (one delegated, one direct) is worse than one direct path that mirrors
`add`'s and `set`'s rules explicitly, which is what this decision does.

### Task creation always records `author: human`

The UI has no author field. `browse.CreateTask` calls `task.New` with
`task.AuthorHuman` unconditionally, the same as `--author human` on `add`.

*Alternative considered:* expose an author selector in the create form.
Rejected — a person filling in a web form is definitionally a human; the field
exists to distinguish agent captures from person captures, and a selector
would just be a way to lie about which one this is.

### Visual design: a small token set translated for offline use

Claude's product identity is read as a small set of tokens rather than copied
asset-for-asset, since no font files or icon sets are available offline:

- **Type:** a serif stack (`ui-serif, Georgia, "Times New Roman", serif`) for
  headings, standing in for Claude's display serif; the system sans stack
  (`-apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif`) for
  everything else. No `@font-face`, no Google Fonts link.
- **Colour:** a warm paper background and dark warm-grey ink rather than pure
  white/black, and the signature terracotta as the one accent — primary
  buttons, focus rings, the high-priority indicator — used sparingly so it
  still reads as an accent once every status and priority also has a colour.
- **Shape:** generously rounded corners (12–16px) on cards and the edit panel,
  1px warm-grey borders instead of drop shadows as the primary way surfaces
  separate from the page, consistent with a calm, paper-like feel rather than
  a conventional dashboard's card shadows.
- **Dark mode:** a parallel dark palette (dark warm ink background, warm
  off-white text, the same terracotta accent) behind
  `prefers-color-scheme: dark`, applied automatically.

These are implemented as CSS custom properties in one stylesheet, so the two
palettes are two blocks of variable definitions rather than two copies of
every rule.

### Graceful shutdown, not a bare `ListenAndServe`

`browse.Serve` runs the HTTP server and also listens for `SIGINT`/`SIGTERM`,
calling `http.Server.Shutdown` with a bounded timeout on either, so `Ctrl+C`
in the terminal (or a supervisor sending `SIGTERM`) stops the server cleanly
and `backlog browse` exits 0 rather than being killed mid-request.

## Risks / Trade-offs

- **An unauthenticated local server is still an HTTP server.** → Loopback by
  default keeps it reachable only to processes already on the machine, which
  already have filesystem access to the same data; `--host` is opt-in and
  warned, per the Decisions above.
- **A second place (`internal/browse`) now validates task fields, alongside
  `internal/cli`'s `add`/`set` flag validation.** → Both call the same
  `internal/task` predicates (`ValidStatus`, `ValidPriority`) and the same
  reason rule, so the permitted values can't drift even though the two entry
  points are separate; a future change that wants one shared validator can
  factor it out of both without either changing behaviour.
- **The UI can go stale** if a task changes on disk while the page is open (an
  agent's `backlog add`, a hand edit, another browser tab). → Accepted per the
  Non-Goals above; the list is a snapshot, and saving an edit re-fetches that
  one task so at least the panel just used reflects the write that was made.
- **A hand-rolled vanilla-JS frontend is more code to write than reaching for
  a framework**, for a UI that will likely grow more views over time. →
  Accepted for this change given the no-build-step constraint; if `browse`
  grows enough to need real client-side state management, that is a decision
  for the change that needs it, not one to pre-pay for here.

## Migration Plan

None required. `browse` is a new command with no on-disk format change and no
effect on any existing command's behaviour or output. A project that upgrades
the binary and never runs `backlog browse` sees nothing different. Rollback is
reverting the binary; no task file is touched by this change in a way that an
older binary would need to undo.
