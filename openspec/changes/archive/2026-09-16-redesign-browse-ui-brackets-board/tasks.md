## 0. Prerequisite

- [x] 0.1 Archive `add-ui-task-deletion` (implemented, all tasks checked) so its "Deleting a task from the UI" requirement is in `openspec/specs/browse-ui/spec.md`; verify `openspec validate redesign-browse-ui-brackets-board --strict` still passes afterwards.

## 1. Visual foundation

- [x] 1.1 Add `@fontsource/poppins` as a devDependency; import weights 300/400/500 from `style.css`; remove `src/fonts/Archivo-Variable.ttf` and its `@font-face`; commit the OFL text. Verify `npm run build` emits Poppins woff2 files under `internal/browse/web/assets/` and the built CSS references them only by relative URL.
- [x] 1.2 Rewrite `frontend/src/style.css`: DS token blocks (colors, typography, spacing, effects, base) verbatim, dark overrides from mockup 02 under `:root[data-theme="dark"]` and the `prefers-color-scheme` guard, then an empty component section. Verify with `npm run typecheck && npm run build` and by grepping the built CSS for `--ink-100` and `--label-red-deep`.
- [x] 1.3 Add `useTheme` (localStorage `backlog.theme`, try/catch, stamps `data-theme` on `<html>`, system preference as default). Verify with a test that toggling sets `data-theme` and survives a remount.
- [x] 1.4 Add shared helpers to `constants.ts`: `STATUS_LABEL` (`New`…`Declined`), `PRI_DOT` colour tokens, `tagColor(name)` stable hash over the ten `--label-*` tokens, `descExcerpt(markdown)`. Verify with unit tests (same tag → same colour, case-insensitive; excerpt skips headings and strips markup).

## 2. Layered dialogs

- [x] 2.1 Implement `LayeredDialog` (portal, backdrop variant top/nested, layer stack so only the topmost handles Escape and Tab, focus in on open, focus restore on close). Verify with tests: Escape closes only the top layer; Tab from the last control wraps inside the top layer.
- [x] 2.2 Implement `ConfirmDialog` and `ReasonDialog` on top of it plus a `useDialogs()` hook returning `confirm(opts): Promise<boolean>` and `askReason(): Promise<string | null>`. Verify with tests: confirm resolves true/false on Delete/Cancel/Escape; reason's confirm button is disabled while the input is blank.

## 3. Chrome and filtering

- [x] 3.1 Rewrite `TopBar`: `Backlog` wordmark, divider, search pill (placeholder `Search tasks, tags, files`, clear X when non-empty), spacer, `List` / `Board` text toggles, divider, theme toggle icon. Verify in `App.test.tsx` that typing filters without a request and the toggle switches views.
- [x] 3.2 Implement `FilterRow` (Priority All/High/Medium/Low, Tags chips, Assignee chips when any, Status toggles in list view only, `Reset` when anything active, `N of M tasks`) and remove `Sidebar` and `ResultBar`. Switching to board clears a status filter. Verify tests: chip toggles the query param and toggles off; Reset clears filters and search; count text.
- [x] 3.3 Implement the empty-result notice with `Clear everything`. Verify a test: a search with no match shows the notice and the control restores all tasks.
- [x] 3.4 Add a window `focus` listener in `useTasks` that calls `refresh()`. Verify a test: firing `focus` issues a new list request and shows a task added to the fake server meanwhile.

## 4. Board

- [x] 4.1 Implement `TaskCard` (label bars, title, excerpt clamped to 3 lines, priority dot + label, `@assignee`/`unassigned`, hover ring, `.is-dragging`), focusable with Enter/Space to open. Verify a test asserting card content for a task with tags, priority and assignee.
- [x] 4.2 Implement `BoardColumn` + `ColumnHeader` for all five statuses in a horizontally scrolling row with the 14px scrollbar styling; empty columns show `BOARD_EMPTY_NOTE`; drop target gets `.is-drop`. Verify the existing drag tests (move, same-column no-op) pass against the new markup.
- [x] 4.3 Implement the column `…` `Popover` with `Collapse list` / `Expand list` (58px rail, vertical label, still a drop target, persisted in `backlog.collapsed`) and `Copy list as markdown`. Verify tests: collapse persists across remount; dropping onto a collapsed rail PATCHes status; copy writes the expected markdown to a stubbed clipboard.
- [x] 4.4 Replace `window.prompt` in `handleMove` with `askReason()`. Verify tests: decline-by-drag with a reason PATCHes `{status, reason}`; cancelling the reason dialog makes no request.

## 5. Capture

- [x] 5.1 Implement `parseDraft` in `src/tokens.ts` (D7). Verify `tokens.test.ts`: priorities incl. `!med`/`!medium`, multiple tags, `@assignee`, `>status` for new/todo/doing/done only, unknown tokens and bare `#`/`@`/`>` stay in the title, whitespace collapsing.
- [x] 5.2 Add `createTaskWithStatus(body, status)` to `api.ts` (POST, then PATCH `{status}` only when not `new`). Verify a test with the fake server asserting POST then PATCH bodies for `todo` and POST only for `new`.
- [x] 5.3 Implement `CaptureSlot` / `CaptureDraft` (rest `+ Add new task` slot on every column but `declined`; focused draft with hint row; parsed summary row; Enter create, Shift+Enter full form, Esc close; empty-title message; draft stays open and empty after create). Verify tests: token create from the `todo` column; empty title makes no request and shows the message; Esc closes without a request.
- [x] 5.4 Implement tag autocomplete in the draft (≤5 existing tags, prefix-first then usage, `Create tag «x»` row, ↑/↓, Tab/Enter accept, Esc closes list only). Verify a test: typing `#u` lists `ui`, Tab replaces the word with `#ui`, Esc keeps the draft open.
- [x] 5.5 Implement the duplicate hint (exact case-insensitive title match against `all`, `Open` / `Create anyway`). Verify a test: matching title blocks the POST until `Create anyway`; `Open` opens the existing task.
- [x] 5.6 Implement the draft failure block (`Retry`, `Copy text`, `Discard`, text kept). Verify a test with a failing POST: message shown, line preserved, Retry re-POSTs, Discard clears.
- [x] 5.7 Implement `ToastHost` and the create toast (`Task NNN created in <Status>`, `Open`, `Undo`, 6s fade). Undo issues DELETE, refreshes and restores a draft-origin line. Verify tests with fake timers: toast disappears after 6s; Undo sends DELETE and removes the task.
- [x] 5.8 Implement the floating capture button and the `n` shortcut (D14). Verify tests: `n` on the board opens the first column's draft; `n` in list view opens the full form; typing `n` in the search field does nothing extra.

## 6. Full create form

- [x] 6.1 Implement `TaskForm` (mockup C5 fields; `More` section with Source files, References, Links via the existing links picker; status New/Todo/Doing/Done; `Create task`, `Create and add another`, Ctrl/⌘+Enter; empty-title message) in a `LayeredDialog`. Per design.md D10, `EditForm` keeps serving the task detail's edit mode until task 7.4's `TaskModal` replaces it — only its create-mode usage is removed, from `TaskDialog`. Verify tests: full-context create sends every field; `Create and add another` keeps the form open and empty; Shift+Enter from a draft prefills title/priority/tags.

## 7. Task modal and inline editing

- [x] 7.1 Implement `InlineText` and `InlineSelect` (rest-as-text, activate by click/Enter, Enter commit, Esc cancel with stopPropagation, disabled while saving, revert on error). Verify unit-level tests of commit/cancel/revert.
- [x] 7.2 Implement `MarkdownEditor` (Write/Preview tabs, insert toolbar, Save/Cancel, Ctrl/⌘+Enter, Esc) and extend `markdown.ts` with italic and blockquote plus mockup 06 `.md` styles. Verify `markdown.test.ts` additions and a toolbar test that B wraps the selection in `**`.
- [x] 7.3 Implement `LinkPicker` (type select → search popover over other tasks by id/title, ↑/↓/Enter) and linked-task rows (type chip, id, clickable title, target status, `Remove`). Verify tests: adding a `blocks` link PATCHes the full links array; clicking a linked title opens that task; Remove PATCHes without it.
- [x] 7.4 Implement `TaskModal` (mockup 05 layout, header id + inline title, `Copy command` / copy id with 1.6s confirmation line, `Delete`, collapsible `Details` / `Description` / `Linked tasks`, read-only metadata rows, reason row when declined, error line) and remove `TaskDialog`, `EditForm`, `ReadView`, `MetadataAside`, `TagChips`. Each inline control PATCHes only its own field. Verify tests: assignee edit sends `{assignee}` only; status → declined asks for a reason and cancel sends nothing; empty title reports the server/client error and keeps the old title; copy command writes `backlog show <id>`.
- [x] 7.5 Wire delete through `ConfirmDialog` naming `NNN · Title`. Verify tests: confirm deletes, closes the modal, task gone; Cancel/Escape leaves modal open and no DELETE.

## 8. List view and responsive pass

- [x] 8.1 Rewrite `ListView` in the DS style (D15) with focusable rows and hidden tag/status columns under 720px. Verify tests: rows show padded ids and open the modal on Enter; focus returns to the row after Escape.
- [x] 8.2 Responsive/theme pass: board columns scroll at narrow widths, modal fits `92vw × 90vh`, dialogs full-screen under 480px, all components read correctly in dark theme. Verify manually with `npm run dev` against `backlog browse` at 1440px, 800px and 400px in both themes.

## 9. Tests, bundle, verification

- [x] 9.1 Rewrite `App.test.tsx`: map every pre-existing case to the new UI first (list render, all statuses shown, free-text no request, filter re-fetch, Escape + focus restore, focus trap, server error surfaced, delete confirm/cancel, link jump, board columns/empty notes, drag move/decline/cancel/same column), then the new cases from sections 3–7. Verify `npm test` passes.
- [x] 9.2 Run `npm run typecheck` and `npm run build` in `frontend/` (via `just build-web`) and commit the regenerated `internal/browse/web/`. Verify no stale Archivo asset remains and `go test ./internal/browse/` passes.
- [x] 9.3 Run `just` (fmt, vet, test). Verify all Go tests pass.
- [x] 9.4 Manual end-to-end with `backlog browse --no-open` in a scratch backlog: capture with tokens into `todo`, undo it, drag to `declined` with a reason, edit title/assignee/description inline, add and remove a link, delete with confirmation, toggle theme, collapse a column, reload; confirm each change with `backlog show --json` / `backlog list --all`, and confirm in the browser network panel that no request leaves the machine.
- [x] 9.5 Run `openspec validate redesign-browse-ui-brackets-board --strict` and confirm it passes.
