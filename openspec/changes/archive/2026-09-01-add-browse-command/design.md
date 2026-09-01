## Context

See proposal.md — Why. The constraints that actually shape the approach are
properties of the existing code and of what the binary promises today:

- **One self-contained binary, no runtime dependencies.** `go install
  github.com/weldnor/backlog@latest` has to keep producing a single binary
  that needs nothing else on disk or on the network at run time. A web UI
  therefore has to be embedded (`go:embed`), and its assets cannot load fonts,
  scripts or styles from a CDN — the binary is expected to work offline, on an
  airgapped machine, the same as every other command.
- **The visual design already exists and is not this change's to invent.**
  The user supplied a Claude Design canvas built directly against this
  repository — `Backlog Web.dc.html`, together with a bundled "Modernist"
  design-system export (`styles.css`, `readme.md`, a component/foundation
  reference set) — covering a desktop list/board view, the task detail-and-edit
  dialog, and a mobile flow. Its own screen map (`github.md` in the export)
  records what each screen was built from: the desktop list, board and editor
  come from this README and `internal/task/task.go`; the mobile flow from the
  README's file-naming and directory rules. The canvas's own interactive
  prototype (artboard `1a`, with real filtering, sorting, read/edit and save
  logic in its `<script>`) is the authoritative, load-bearing screen; the
  supplementary artboards under sections "2" (a dark-ground illustration) and
  "3" ("mobile, reworked" — a more elaborate phone flow with a split
  write/preview editor, a decline sheet and a `···` overflow menu) are
  explorations the canvas itself frames as "try next" ideas, not committed
  screens — `github.md`'s own screen map only claims `1a` and `1b` as built
  from the repo. This change implements `1a` and `1b`; section 2's tokens are
  used only for what they actually specify (the dark-mode color values,
  applied automatically rather than as a manual toggle), and section 3 is left
  for a later change.
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
  prior UI, no CSS and no frontend tooling in the repository — the canvas
  described above is the only design source, and it has to be translated into
  a static, offline, dependency-free page rather than referenced live.

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
- No drag-and-drop, no multi-task/bulk actions. The board view (artboard
  `1a`) is a second way to *look at* the same filtered list, one column per
  status; moving a task between columns happens through the edit dialog like
  any other field change, never by dragging a card.
- No bespoke mobile screens beyond a responsive baseline built on the same
  tokens: the canvas's "mobile, reworked" flow (section 3 — a split
  write/preview editor, a dedicated decline bottom sheet, a `···` overflow
  menu) is a further exploration the canvas itself did not commit to
  (see Context), and is left for a later change. `1b`'s simpler mobile list
  and read-only detail sheet describes the baseline this change targets.
- No manual light/dark toggle. The palette follows `prefers-color-scheme`
  automatically, per section 2 of the canvas, applied as an automatic media
  query rather than the manually-switched illustration the canvas shows it
  as.
- No authentication, no HTTPS, no CSRF protection. The trust boundary is the
  same one the CLI already has — whoever can run `backlog` already has full
  read/write access to `.backlog/` on disk — and is discussed under Decisions.
- No regex search and no server-side text search endpoint. The UI's text
  filter runs client-side over the already-fetched list, matching the same
  substring-only spirit as `search` but not calling into `internal/search` or
  exposing its ranking; a person who needs `search`'s regex mode or its
  declined-always-in-scope behaviour still reaches for the CLI.
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

### Visual design: the canvas's tokens, verbatim, self-hosting the one font

The canvas's own `styles.css` is the specification, not a mood board — its
`:root` custom properties and component classes are ported into
`internal/browse/web/style.css` unchanged in value, with only the font's
source changed for offline use:

- **Type — Archivo, self-hosted.** `--font-heading` and `--font-body` are
  both `"Archivo", system-ui, sans-serif`, heading weight 800, body 400/600.
  The canvas's stylesheet pulls it from Google Fonts
  (`@import url('https://fonts.googleapis.com/css2?family=Archivo…')`); this
  change instead vendors the OFL-licensed Archivo variable font into
  `internal/browse/web/fonts/` and declares it with a local `@font-face`, so
  the page needs no network access — the one substantive deviation from the
  canvas file, made for the offline constraint above, not a stylistic choice.
  Heading sizes follow the canvas scale: h1 42px down to h6 13px (uppercase,
  tracked), line-height 1.12, letter-spacing −0.015em.
- **Colour — one accent, tonal ramps, real values.** Light ground
  `--color-bg` `#f3f2f2`, surface `--color-surface` `#eae9e9`, ink text
  `--color-text` `#201e1d`, and a single accent `--color-accent` `#ec3013` — a
  saturated red-orange, not Claude's own palette. It is a mono scheme (the
  canvas's `--color-accent-2-*` ramp is a machine-derived stand-in kept only
  so both resolve; treat it as the same role as `--color-accent-*`). Each
  role carries a 100–900 OKLCH tonal ramp; light steps (100–300) tint fills
  and hovers, 500 is the base, dark steps (700–900) sit on tinted fills and
  mark pressed/selected states. Priority reuses these ramps rather than
  inventing new colours: `high` is `.tag-outline` (accent border and text),
  `medium` is `.tag-accent` (`--color-accent-100` fill, `--color-accent-800`
  text), `low` is `.tag-neutral`.
- **Shape — zero radius, everywhere, on purpose.** `--radius-sm/md/lg` are
  all `0px`; the canvas's `readme.md` states this as a rule ("do not round a
  corner anywhere"), which rules out the rounded-card look this design.md
  described before the canvas was supplied. Separation between surfaces comes
  from strong 1–2px ink rules (`--color-divider`, a 40%-opacity mix of the
  text colour) far more than from the `--shadow-sm/md/lg` tokens, which are
  reserved for genuinely floating surfaces — the detail dialog's backdrop, the
  "browser frame" treatment around a whole screen.
- **Components — the canvas's own classes, not new ones.** `.btn` /
  `.btn-primary` / `.btn-secondary` / `.btn-icon`; `.tag` /
  `.tag-accent` / `.tag-neutral` / `.tag-outline`; `.field` + `.input`,
  `.seg` + `.seg-opt` (the segmented control used for LIST/BOARD and for
  status/priority in the edit form); `.table`; `.dialog-backdrop` + `.dialog`.
  `internal/browse/web/style.css` carries these classes forward from the
  export rather than re-deriving equivalents, so a future retouch of the
  canvas's tokens can be re-ported by diffing one file.
- **Dark mode — the canvas's section 2 values, applied automatically.** The
  same structure inverted: ground `#201e1d`, rules and text in `#f3f2f2`,
  the accent held at full strength for priority and the primary action, and
  `--color-accent-400` (`#ff9783`) wherever the accent has to carry running
  text on the dark ground, exactly as the canvas's own dark artboards use it.
  Behind `prefers-color-scheme: dark`, not a manually-switched illustration.
- **Icons — Lucide, inline SVG, `currentColor`.** Matches what is already
  inline in the canvas's markup (search, plus, edit-pencil, save, archive,
  close); no icon font, no external icon request.

### Screen architecture matches artboard `1a`/`1b`, not a fresh layout

The canvas's interactive prototype is implemented close to structurally
as-is, because it already answers the layout questions a UI like this raises
and answering them differently would be a second, unreviewed design:

- **Top bar:** wordmark + version, a repo/branch chip (`weldnor/backlog /
  .backlog · main`, read from git the way `add`'s provenance already is),
  the free-text filter input, the LIST/BOARD `.seg` toggle, and the primary
  `CAPTURE` button that opens the create dialog.
- **Left sidebar (232px):** three filter groups over `.field`-style
  buttons — STATUS and PRIORITY each list every value with a live count and
  toggle on click (clicking the active one clears it, matching `pick()` in
  the prototype), TAGS is a chip cloud of every tag currently in use.
  Multiple groups combine as a conjunction, matching `scope.apply`; within a
  group only one value is selected at a time in this first version, which is
  narrower than the CLI's repeatable `--tag`/`--priority` but matches what
  the canvas actually renders (a single active filter per group, not a
  multi-select list) — expanding a sidebar group to multi-select is a natural
  follow-up, not part of this change.
- **Result bar:** a count and the fixed order note ("descending priority,
  then ascending identifier"), a "Show archive · --all" toggle, and — a
  detail worth keeping, not decoration — the equivalent CLI invocation for
  the current view (`backlog list --tag bug`, `backlog list --json`) printed
  in monospace. It costs one string per view and keeps the UI honest that it
  is the same store and the same commands, not a parallel system.
- **List view:** one `.table`, columns ID / Priority / Title (with its
  source file path as a second line) / Tags / Status.
- **Board view:** four columns, fixed order `todo`/`doing`/`done`/`declined`,
  each a header with a live count and a stack of cards; an empty column shows
  a one-line explanatory note rather than nothing (the canvas's own copy —
  "Archive — acted on. Behind `--all`.", "Always in scope for `search` — a
  duplicate must not hide behind a filter." — is worth keeping verbatim,
  since it is teaching the same distinctions the README does).
- **Detail dialog:** opened by clicking any row or card, `840px`, two panes.
  The left pane defaults to a **read view** — priority tag, status, title,
  rendered markdown body, and a decline-reason callout when applicable — with
  an `Edit` toggle in the dialog header that switches the same pane to the
  **edit form**: title input, a status `.seg` (four options), a priority
  `.seg` (three), a tags input, a reason input that appears only while the
  selected status is `declined` (mirroring the `needsReason` binding in the
  prototype), and the description as a plain textarea — raw markdown, no
  split source/preview pane; that richer editor is section 2b's exploration,
  left for later per the Non-Goals. The right pane (264px) is the read-only
  metadata sidebar — identifier, created timestamp, author, branch/commit,
  source files, refs — with the canvas's own explanation of why it is
  read-only ("the key set under `metadata` is closed... this panel is
  read-only for that reason"), which doubles as user-facing documentation of
  a rule this project already enforces server-side.
- **Create dialog:** the canvas's working prototype wires `CAPTURE` to open
  the dialog with no task selected, but does not finish rendering a blank
  form in that state — the one gap in an otherwise complete interactive
  spec. This change fills it the only way consistent with everything else
  the prototype does: the same edit-form layout, opened directly (no read
  view to toggle from), fields blank except `status: todo` and
  `priority: medium`, and `Save` calling `POST /api/tasks` instead of
  `PATCH /api/tasks/{id}`.
- **Mobile (`1b`, ≤480px):** the sidebar collapses and the top bar's search
  and view toggle stack under the wordmark; the list becomes the single
  full-width column the canvas draws, and tapping a task opens the same
  detail dialog full-screen rather than the desktop's centred overlay. The
  board view and its column layout are not adapted for mobile in this change
  — a person on a phone gets the list.

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
- **Vendoring the Archivo font changes what the canvas actually renders**
  (a Google Fonts request) rather than reproducing it byte-for-byte. → Archivo
  is SIL Open Font License 1.1, which permits bundling and redistribution;
  the OFL text is vendored alongside the font files. The rendered typeface is
  unchanged, only its source is, which is the offline constraint's cost, made
  explicit here rather than silently reverting to a system font.
- **Two sections of the supplied canvas (dark-ground detail, "mobile,
  reworked") were read as exploratory rather than committed**, which is a
  judgement call this change made rather than one the user confirmed
  artboard-by-artboard. → The canvas's own `github.md` screen map supports it
  (only `1a`/`1b` are listed as built from the repo), and dark mode's *values*
  from section 2 are still used, just applied automatically instead of as the
  manual toggle the illustration shows. If this reading is wrong, it is easy
  to correct in an update to this change before implementation starts.

## Migration Plan

None required. `browse` is a new command with no on-disk format change and no
effect on any existing command's behaviour or output. A project that upgrades
the binary and never runs `backlog browse` sees nothing different. Rollback is
reverting the binary; no task file is touched by this change in a way that an
older binary would need to undo.
