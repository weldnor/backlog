## 1. Scaffold the frontend project

- [x] 1.1 Create `frontend/` from a Vite React-TS setup: `package.json`, `vite.config.ts`, `tsconfig.json`, `src/main.tsx`, `index.html`. Verify `cd frontend && npm install && npm run build` succeeds.
- [x] 1.2 Set `build.outDir` to `../internal/browse/web` and `emptyOutDir: true` in `vite.config.ts`; use relative asset paths (`base: "./"`). Verify a build emits `internal/browse/web/index.html` and `internal/browse/web/assets/*`.
- [x] 1.3 Move `internal/browse/web/fonts/Archivo-Variable.ttf` and `OFL.txt` under `frontend/` (e.g. `frontend/public/fonts/`), referenced so Vite emits them into the output. Verify the built `index.html`/CSS references the font by a relative path and the font file is present in the output.
- [x] 1.4 Add `frontend/.nvmrc` (pinned Node LTS) and matching `engines.node` in `package.json`. Add `frontend/node_modules/` to `.gitignore`; confirm the Vite output under `internal/browse/web/` is NOT ignored. Commit `package-lock.json`.

## 2. Port shared modules

- [x] 2.1 Add `src/style.css` as a verbatim copy of the current `internal/browse/web/style.css` (font `src` path adjusted to the new location) and import it from `main.tsx`. Verify the built CSS contains the Modernist tokens and `@font-face`.
- [x] 2.2 Implement `src/api.ts`: `TaskView`/`MetaView`/`SourceView` interfaces matching `internal/taskview/view.go` field-for-field, plus `listTasks`, `getTask`, `createTask`, `patchTask`, `getRepo`, with the same non-2xx error behavior as the current `api()` helper (`Error(body.error || statusText)`, `.status`). Verify `tsc --noEmit` passes.
- [x] 2.3 Implement `src/markdown.ts` porting `md()`/`inline()`/`esc()`/`escAttr()` from `app.js` with identical output. Verify with a small unit test (Vitest) covering headings, lists, code fences, inline code/bold/links, and HTML escaping.

## 3. Port the main screens

- [x] 3.1 Implement the `useTasks` hook owning the `?all=1` and filtered fetches and a `refresh()`; implement the `App` reducer (view, filters, archive, dialog mode, open task, draft, error). Verify filter/archive/view state transitions with a component test or manual check against `app.js` behavior.
- [x] 3.2 Implement `TopBar` (wordmark + version, repo chip from `/api/repo`, search box, LIST/BOARD toggle, CAPTURE button) and `Sidebar` (status list, priority list, tag cloud with counts from `all`, click-to-toggle). Verify counts and active states match the current UI.
- [x] 3.3 Implement `ResultBar` (result count with singular/plural, archive toggle label states, live `backlog list …` command string) matching `currentCommand()`. Verify the command string for several filter combinations equals the old output.
- [x] 3.4 Implement `ListView` (table, 3-digit padded id, priority badge, title + source-file line, tag chips, status label, `is-open` highlight, empty-state note). Verify rows and empty state match the current markup's behavior.
- [x] 3.5 Implement `BoardView` (four fixed columns `todo`/`doing`/`done`/`declined`, per-column count, cards, high-priority rule, verbatim per-column empty notes, no drag-and-drop). Verify a filter applied in list carries to board.
- [x] 3.6 Implement free-text filtering as a derived computation over the loaded `visible` set (title + description + tags, case-insensitive, no request). Verify typing in the search box filters both views instantly with no network call.

## 4. Port the task dialog

- [x] 4.1 Implement `TaskDialog` as a portal + backdrop with `ReadView` (priority/status header, rendered markdown description, decline callout when declined) and `MetadataAside` (read-only id, created, author, branch/commit, source files, refs). Verify the read view and metadata match the current dialog.
- [x] 4.2 Implement `EditForm` shared by edit and create: title, status segmented control, priority segmented control, tags, description textarea; files/refs only in create; reason field shown only while status is `declined`; clearing `declined` clears the reason in the draft. Verify each field maps to the correct `POST`/`PATCH` body key.
- [x] 4.3 Wire open/read/edit-toggle/create/save/cancel/close and post-save/post-create `refresh()`; surface server validation errors in the dialog without closing it. Verify: empty-title create rejected, decline-without-reason rejected, successful edit returns to read view with refreshed metadata.
- [x] 4.4 Add dialog keyboard/focus behavior: `Escape` closes (same as close button, no save), focus moves into the dialog on open and is trapped while open, focus returns to the triggering row/card on close. Verify per the new `browse-ui` scenarios.

## 5. Build output and cleanup

- [x] 5.1 Run `npm run build`; delete the old `internal/browse/web/index.html`, `app.js`, `style.css` sources; commit the generated `index.html` + `assets/` + fonts. Verify `go build ./...` and `go run . browse` serve the React UI.
- [x] 5.2 Run the existing Go suites (`go test ./internal/browse/... ./internal/cli/...`); fix only tests that pin a literal string from the old HTML. Verify `go test ./...` passes.
- [x] 5.3 Manually verify offline operation: run `backlog browse`, load the UI with DevTools open, confirm zero requests to any non-localhost host and the page is fully styled in Archivo.

## 6. Tooling and CI

- [x] 6.1 Add `build-web` to the `justfile` (`cd frontend && npm ci && npm run build`). Verify `just build-web` regenerates the committed bundle with no diff on a clean tree.
- [x] 6.2 Add a `frontend` job to `.github/workflows/ci.yml`: `actions/setup-node` with `node-version-file: frontend/.nvmrc`, `npm ci`, `npm run build`, then `git diff --exit-code -- internal/browse/web`. Verify the job passes on a fresh build and fails when the committed bundle is stale.
- [x] 6.3 Update `README.md` (and any contributor note) to document that UI work needs Node + `just build-web`, and that the Go build/install path is unchanged. Verify the instructions produce a working build from a clean checkout.

## 7. Spec validation

- [x] 7.1 Run `openspec validate migrate-browse-ui-to-react --strict` and confirm it passes.
- [x] 7.2 Walk every scenario in `specs/browse-ui/spec.md` and `specs/ci-pipeline/spec.md` against the running UI and CI, confirming each holds.
