## 1. Server package skeleton

- [x] 1.1 Create `internal/browse` with a `Server` (or `Options`) type carrying the `*store.Store`, host, port and open-browser flag, and a `Serve(store *store.Store, opts Options) error` entry point that builds an `http.Server` and its `http.ServeMux`. Verify `go build ./...` succeeds with the new, still-empty-of-routes package wired in.
- [x] 1.2 Implement graceful shutdown: `Serve` listens for `SIGINT`/`SIGTERM` and calls `http.Server.Shutdown` with a bounded timeout, returning nil on a clean interrupt. Verify with a test that starts the server on `:0`, sends the process a simulated shutdown trigger (or calls the shutdown path directly), and asserts `Serve` returns nil within the timeout.
- [x] 1.3 Implement cross-platform browser launching (`xdg-open` / `open` / the Windows equivalent) as a small helper that returns an error rather than panicking when no opener is found, so the caller can log and continue. Verify with a unit test that stubs the command runner and asserts the right command is chosen per `runtime.GOOS`.

## 2. Reusing the CLI's task JSON shape

- [x] 2.1 Export what `internal/browse` needs from `internal/cli/output.go` (`TaskView`, `view`, `views`, or move them to a small shared location both packages import) so the HTTP handlers and `backlog --json` produce identical JSON for the same task. Verify `go build ./...` and that existing `internal/cli` tests asserting JSON shape still pass unchanged.
- [x] 2.2 Add a JSON error envelope helper (`{"error": "..."}`) used by every handler that fails, with the status code chosen per failure (400 for validation, 404 for an unknown identifier, 500 for anything else). Verify with a table test covering one case of each status.

## 3. Listing and reading tasks

- [x] 3.1 Implement `GET /api/tasks`, reusing the same status/tag/priority/all selection rules `backlog list --all` uses and the same descending-priority-then-ascending-identifier order. Verify with a test seeding a store with tasks across every status and priority, and asserting the endpoint's JSON matches `views(...)` on the equivalent `list --all --json` selection.
- [x] 3.2 Implement `GET /api/tasks/{id}`, returning 404 with the JSON error envelope for an unknown identifier. Verify with tests for an existing and a missing identifier.

## 4. Creating a task

- [x] 4.1 Implement `POST /api/tasks` accepting `{title, description, tags, priority, files, refs}`, rejecting an empty or whitespace-only title and an invalid priority with 400, defaulting priority to `medium`, and always recording `author: human`. Reuse `task.New` and `Store.Create`, and record source provenance (branch/commit) the same way `add` does. Verify with tests for the minimal case, the full-context case, an empty title, and an invalid priority, asserting the resulting task file on disk and the response body.
- [x] 4.2 Verify the created task's identifier allocation races safely under concurrent creation from the UI: a test firing several concurrent `POST /api/tasks` requests at one running server and asserting every task receives a distinct identifier.

## 5. Editing a task

- [x] 5.1 Implement `PATCH /api/tasks/{id}` accepting any subset of `{title, description, tags, priority, status, reason, refs}`, loading the task, applying only the fields present in the request, and calling `Store.Save`. `tags` and `refs`, when present, replace the task's existing lists rather than appending. Verify with a test that a partial request touching only `description` leaves every other field, including `tags`, unchanged.
- [x] 5.2 Implement the validation rules mirroring `add`/`set`: reject an empty title, reject an invalid status or priority, require a reason when the resulting status is `declined`, reject a reason supplied for a resulting status other than `declined`, and clear the reason when a declined task's status changes to anything else. Verify with a table test covering: declining with a reason, declining without one, reopening a declined task, revising the reason alone on an already-declined task, and a reason submitted alongside a non-declined status.
- [x] 5.3 Verify cross-consistency with the CLI: a test that edits a task via `PATCH /api/tasks/{id}` and then asserts `backlog show <id> --json`, run against the same store, reports exactly the saved fields.

## 6. The `browse` command

- [x] 6.1 Add `browsecmd.go` to `internal/cli` registering `browse` in `commands()`, parsing `--port`, `--host` (default `127.0.0.1`), `--no-open` and `--json`, opening the store the same way every other command does, and calling `browse.Serve`. Print the URL (or, with `--json`, `{"url": "..."}`) before blocking. Verify with a CLI test that starts the server against a temporary backlog, asserts the printed URL is well-formed, and that the command exits non-zero with no backlog present.
- [x] 6.2 Print the loopback-exposure warning to stderr when `--host` is anything other than a loopback address. Verify with a test asserting the warning's presence for a non-default host and its absence for the default.
- [x] 6.3 Wire `--no-open` to skip the browser-launch call from task 1.3, and confirm a browser-launch failure is logged to stderr rather than failing the command. Verify with a test stubbing the opener to fail and asserting the server still starts and the command does not exit non-zero because of it.

## 7. The web UI — ported from the `Backlog Web.dc.html` canvas

The canvas the user supplied (its `1a`/`1b` artboards, per design.md's Context
and "Screen architecture" decision) is the layout and copy source for this
section, not a reference to work from loosely — column widths, class names,
filter/empty-state copy and the CLI-hint strings below are taken from it.

- [x] 7.0 Vendor the OFL-licensed Archivo variable font into `internal/browse/web/fonts/` and port the canvas's `styles.css` into `internal/browse/web/style.css`: same custom properties and component classes (`.btn`, `.tag`, `.field`/`.input`, `.seg`/`.seg-opt`, `.table`, `.dialog-backdrop`/`.dialog`), with the `@import` of Google Fonts replaced by a local `@font-face` pointing at the vendored files, and the dark-mode values (design.md's Decisions) added behind `@media (prefers-color-scheme: dark)`. Verify by starting the server locally and confirming the page loads styled with the browser's network panel showing no requests leaving the page's own origin, including no request to `fonts.googleapis.com`/`fonts.gstatic.com`.
- [x] 7.1 Build `internal/browse/web/index.html` and `app.js` around the canvas's top bar (wordmark + version, repo/branch chip, free-text filter input, LIST/BOARD `.seg` toggle, `CAPTURE` button) and left sidebar (STATUS and PRIORITY as single-select `.field` lists with live counts, TAGS as a chip cloud), fetching `GET /api/tasks` and re-fetching on every filter change. Embed the `internal/browse/web` directory with `go:embed` and serve it from `Serve`'s mux. Verify against a seeded backlog covering every status and priority that each sidebar control narrows the fetched set correctly, and that the count and order-note in the result bar match what task 3.1 returns.
- [x] 7.2 Implement the list view: the canvas's `.table` with columns ID / Priority / Title (with its source file path as a second line) / Tags / Status. Verify by hand against a seeded backlog that column content and priority/status styling match the canvas.
- [x] 7.3 Implement the board view: four columns in the fixed order `todo`/`doing`/`done`/`declined`, each with a header showing its label and live count and a stack of cards (id, priority, title, first source file, tags), and the canvas's own empty-column copy ("Archive — acted on. Behind `--all`.", "Always in scope for `search` — a duplicate must not hide behind a filter.", etc. — one per status). Verify by hand that switching LIST/BOARD preserves the active filters and that an empty column shows its note instead of nothing.
- [x] 7.4 Implement the client-side free-text filter over the already-fetched list (title, description, tags; case-insensitive substring; no request to the server), matching the canvas's `visible()` filter logic. Verify by hand that typing narrows both the list and board views without a network request, observable in the browser's network panel.
- [x] 7.5 Implement the detail dialog's read view: `840px`, two panes — the left showing priority tag, status, title, rendered markdown body (a minimal markdown-to-HTML pass covering the subset the canvas's own `md()` helper handles: paragraphs, `#`/`##`/`###` headings, `-`/`*` lists, `` ``` `` code fences, `` `code` ``, `**bold**`, `[text](url)` links) and a decline-reason callout when applicable; the right (264px) showing the read-only metadata — id, created, author, branch/commit, source files, refs — with the canvas's explanatory copy about `metadata`'s closed key set. Verify by hand against a task of each status, including one declined and one with no refs.
- [x] 7.6 Implement the `Edit` toggle in the dialog header switching the left pane to the edit form: title input, status `.seg` (four options), priority `.seg` (three options), tags input, a reason input shown only while the selected status is `declined`, and the description as a plain textarea, `Save` sending `PATCH /api/tasks/{id}` and `Cancel` discarding the draft and returning to the read view. A validation error from the server is shown inline without discarding what was typed; a successful save returns to the read view showing the refreshed task and metadata. Confirm no delete action exists anywhere in the dialog. Verify by hand against every scenario in the `browse-ui` "Editing a task from the UI" and "No deletion from the UI" delta spec requirements.
- [x] 7.7 Implement the create dialog: `CAPTURE` opens the same edit-form layout directly (no read view), fields blank except `status: todo` and `priority: medium`, plus files and refs inputs; `Save` sends `POST /api/tasks` and closes to the refreshed list on success. Verify by hand: successful creation appears in both the list and board views; an empty title shows the error and creates nothing.
- [x] 7.8 Adapt the layout for narrow viewports per `1b`: the sidebar collapses and the top bar's search and view toggle stack under the wordmark, the list becomes a single full-width column, and the detail dialog opens full-screen rather than as a centred overlay. Verify by hand at a mobile viewport width that every control from 7.1–7.7 remains reachable.

## 8. Documentation

- [x] 8.1 Add a `### backlog browse` section to the README's command reference, describing the flags, the default binding and browser-opening behaviour, and the fact that title/description/tag editing is only available here, not from `set`.
- [x] 8.2 Verify the README's `## Commands` intro line ("Every command that reports tasks accepts `--json`...") still holds, or is qualified, now that `browse` is a command that does not report tasks in the same sense; adjust the wording if it now reads as inaccurate.

## 9. End-to-end verification

- [x] 9.1 Add an end-to-end test that starts `backlog browse` against a temporary backlog, drives the HTTP API directly (create, list, edit through every status transition, edit title/description/tags), and asserts the resulting task files match what an equivalent sequence of `add`/`set` CLI invocations would have produced.
- [x] 9.2 Run `go build ./...`, `go vet ./...` and `go test ./...` and confirm the whole suite passes, including the new `internal/browse` package and its embedded assets.
