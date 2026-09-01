## 1. Server package skeleton

- [ ] 1.1 Create `internal/browse` with a `Server` (or `Options`) type carrying the `*store.Store`, host, port and open-browser flag, and a `Serve(store *store.Store, opts Options) error` entry point that builds an `http.Server` and its `http.ServeMux`. Verify `go build ./...` succeeds with the new, still-empty-of-routes package wired in.
- [ ] 1.2 Implement graceful shutdown: `Serve` listens for `SIGINT`/`SIGTERM` and calls `http.Server.Shutdown` with a bounded timeout, returning nil on a clean interrupt. Verify with a test that starts the server on `:0`, sends the process a simulated shutdown trigger (or calls the shutdown path directly), and asserts `Serve` returns nil within the timeout.
- [ ] 1.3 Implement cross-platform browser launching (`xdg-open` / `open` / the Windows equivalent) as a small helper that returns an error rather than panicking when no opener is found, so the caller can log and continue. Verify with a unit test that stubs the command runner and asserts the right command is chosen per `runtime.GOOS`.

## 2. Reusing the CLI's task JSON shape

- [ ] 2.1 Export what `internal/browse` needs from `internal/cli/output.go` (`TaskView`, `view`, `views`, or move them to a small shared location both packages import) so the HTTP handlers and `backlog --json` produce identical JSON for the same task. Verify `go build ./...` and that existing `internal/cli` tests asserting JSON shape still pass unchanged.
- [ ] 2.2 Add a JSON error envelope helper (`{"error": "..."}`) used by every handler that fails, with the status code chosen per failure (400 for validation, 404 for an unknown identifier, 500 for anything else). Verify with a table test covering one case of each status.

## 3. Listing and reading tasks

- [ ] 3.1 Implement `GET /api/tasks`, reusing the same status/tag/priority/all selection rules `backlog list --all` uses and the same descending-priority-then-ascending-identifier order. Verify with a test seeding a store with tasks across every status and priority, and asserting the endpoint's JSON matches `views(...)` on the equivalent `list --all --json` selection.
- [ ] 3.2 Implement `GET /api/tasks/{id}`, returning 404 with the JSON error envelope for an unknown identifier. Verify with tests for an existing and a missing identifier.

## 4. Creating a task

- [ ] 4.1 Implement `POST /api/tasks` accepting `{title, description, tags, priority, files, refs}`, rejecting an empty or whitespace-only title and an invalid priority with 400, defaulting priority to `medium`, and always recording `author: human`. Reuse `task.New` and `Store.Create`, and record source provenance (branch/commit) the same way `add` does. Verify with tests for the minimal case, the full-context case, an empty title, and an invalid priority, asserting the resulting task file on disk and the response body.
- [ ] 4.2 Verify the created task's identifier allocation races safely under concurrent creation from the UI: a test firing several concurrent `POST /api/tasks` requests at one running server and asserting every task receives a distinct identifier.

## 5. Editing a task

- [ ] 5.1 Implement `PATCH /api/tasks/{id}` accepting any subset of `{title, description, tags, priority, status, reason, refs}`, loading the task, applying only the fields present in the request, and calling `Store.Save`. `tags` and `refs`, when present, replace the task's existing lists rather than appending. Verify with a test that a partial request touching only `description` leaves every other field, including `tags`, unchanged.
- [ ] 5.2 Implement the validation rules mirroring `add`/`set`: reject an empty title, reject an invalid status or priority, require a reason when the resulting status is `declined`, reject a reason supplied for a resulting status other than `declined`, and clear the reason when a declined task's status changes to anything else. Verify with a table test covering: declining with a reason, declining without one, reopening a declined task, revising the reason alone on an already-declined task, and a reason submitted alongside a non-declined status.
- [ ] 5.3 Verify cross-consistency with the CLI: a test that edits a task via `PATCH /api/tasks/{id}` and then asserts `backlog show <id> --json`, run against the same store, reports exactly the saved fields.

## 6. The `browse` command

- [ ] 6.1 Add `browsecmd.go` to `internal/cli` registering `browse` in `commands()`, parsing `--port`, `--host` (default `127.0.0.1`), `--no-open` and `--json`, opening the store the same way every other command does, and calling `browse.Serve`. Print the URL (or, with `--json`, `{"url": "..."}`) before blocking. Verify with a CLI test that starts the server against a temporary backlog, asserts the printed URL is well-formed, and that the command exits non-zero with no backlog present.
- [ ] 6.2 Print the loopback-exposure warning to stderr when `--host` is anything other than a loopback address. Verify with a test asserting the warning's presence for a non-default host and its absence for the default.
- [ ] 6.3 Wire `--no-open` to skip the browser-launch call from task 1.3, and confirm a browser-launch failure is logged to stderr rather than failing the command. Verify with a test stubbing the opener to fail and asserting the server still starts and the command does not exit non-zero because of it.

## 7. The web UI

- [ ] 7.1 Author `internal/browse/web/index.html`, `style.css` and `app.js` (or equivalent) as static, dependency-free assets: no `<link>`/`<script>` to any external host, using the serif-for-headings/system-sans-for-body font stacks and the warm-paper/terracotta-accent palette from design.md as CSS custom properties, including the `prefers-color-scheme: dark` variant. Embed the directory with `go:embed` and serve it from `Serve`'s mux. Verify by starting the server locally and confirming the page loads with the browser's network panel showing no requests leaving the page's own origin.
- [ ] 7.2 Implement the list view: fetch `/api/tasks`, group by status in the fixed order, render each task as a card showing title, priority and tags, and provide status/tag/priority filter controls that re-fetch with the matching query parameters. Verify by hand against a seeded backlog covering every status, and with a lightweight test (Go `httptest` server plus a scripted fetch, or manual verification) that the filters produce the requests task 3.1 expects.
- [ ] 7.3 Implement the client-side free-text filter over the already-fetched list (title, description, tags; case-insensitive substring; no request to the server). Verify by hand that typing narrows the visible cards without a network request, observable in the browser's network panel.
- [ ] 7.4 Implement the create form (title required, description, tags, priority, files, refs) posting to `POST /api/tasks`, surfacing a validation error inline without losing what was typed, and refreshing the list on success. Verify by hand: successful creation appears in the list; an empty title shows the error and creates nothing.
- [ ] 7.5 Implement the edit panel: opening a task loads its current fields, editing and saving sends `PATCH /api/tasks/{id}`, a validation error is shown inline without discarding the edit, and a successful save re-fetches that task and updates the card in place. Confirm no delete action exists anywhere in the panel. Verify by hand against every scenario in the `browse-ui` edit-task delta spec.

## 8. Documentation

- [ ] 8.1 Add a `### backlog browse` section to the README's command reference, describing the flags, the default binding and browser-opening behaviour, and the fact that title/description/tag editing is only available here, not from `set`.
- [ ] 8.2 Verify the README's `## Commands` intro line ("Every command that reports tasks accepts `--json`...") still holds, or is qualified, now that `browse` is a command that does not report tasks in the same sense; adjust the wording if it now reads as inaccurate.

## 9. End-to-end verification

- [ ] 9.1 Add an end-to-end test that starts `backlog browse` against a temporary backlog, drives the HTTP API directly (create, list, edit through every status transition, edit title/description/tags), and asserts the resulting task files match what an equivalent sequence of `add`/`set` CLI invocations would have produced.
- [ ] 9.2 Run `go build ./...`, `go vet ./...` and `go test ./...` and confirm the whole suite passes, including the new `internal/browse` package and its embedded assets.
