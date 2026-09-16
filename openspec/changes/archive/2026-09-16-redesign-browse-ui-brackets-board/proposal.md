## Why

The `browse` UI still wears the Modernist port — square corners, a fixed sidebar, uppercase segmented controls, a four-slot read/edit dialog — while the product design has moved on: the Claude Design project "Backlog · Brackets board" (`Mockups.dc.html`) defines a new board-first interface with inline capture, inline field editing and a lighter visual language (Poppins, black-plus-alpha ink ladder, 10px radii, no shadows). The mockups also describe features the backend cannot serve yet (covers, attachments, bulk selection). This change brings the UI in line with the mockups **for everything the existing `/api` already handles**, and explicitly leaves the rest out, so the visual refresh does not smuggle in backend work.

## What Changes

- **Visual system.** Replace the Modernist tokens and the vendored Archivo face with the Brackets board design-system tokens (colors, ink opacity ladder, label palette, radii, spacing, typography) and a locally vendored Poppins (300/400/500). Light theme by default with a dark theme matching mockup 02, toggled from the top bar and remembered per browser; the system preference is the initial value.
- **Chrome.** A 58px top bar: `Backlog` wordmark, search pill, `List` / `Board` text toggle, theme toggle. The left sidebar and the `backlog list …` result bar are removed; filtering moves to a single filter row (Priority `All/High/Medium/Low`, Tags chips, Assignee chips, Status in list view, `Reset`, `N of M tasks`). An empty result shows the mockup 03 notice with `Clear everything`.
- **Board.** Horizontally scrolling 362px columns for `new`, `todo`, `doing`, `done`, `declined`; 50px column header with name, count and a `…` menu (`Collapse list`, `Copy list as markdown`); a collapsed column becomes a 58px rail with a vertical label. Cards show tag label bars, title, a short description, a priority dot and the assignee. Drag-and-drop keeps its current semantics, with the mockup's drag visuals (dragged card at 40%, 1px inset ring on the target column).
- **Capture.** Every column except `declined` ends in a `+ Add new task` slot that turns into an inline draft. One-line token syntax — `!high|!med|!low`, `#tag`, `@assignee`, `>status` — is parsed live and shown under the line; `#` opens tag autocomplete over existing tags. `Enter` creates, `Shift+Enter` opens the full form prefilled, `Esc` cancels. A floating round capture button and the `n` key open a draft in the first column (or the full form in list view). An empty title is reported inline; a title matching an existing task shows a non-blocking "looks like an existing task" hint. After creating, a bottom-left toast `Task NNN created in <Status> · Open · Undo` stays for 6 seconds; `Undo` deletes the just-created task. A failed create keeps the draft text and offers `Retry`, `Copy text`, `Discard`.
- **Full create form** (mockup C5): title, description, status, priority, assignee, tags, plus source files, references and links (the existing create fields), `Create task` and `Create and add another`, `⌘/Ctrl+Enter` to submit.
- **Task modal** (mockup 05): large overlay window with id + title, `Copy command` and `Delete`, a `Details` grid (status, priority, assignee, tags, source, branch/commit, created, author, refs, decline reason), the description, and `Linked tasks` with remove and a `Link a task` picker. Every editable field is edited **inline** (mockup 09/08): title, assignee and tags as in-place inputs (`Enter` saves, `Esc` cancels), status and priority as inline selects, description through a `Write` / `Preview` markdown editor with a small insert toolbar. The separate EDIT mode and its form go away.
- **Confirmations.** The native `window.confirm` for delete and `window.prompt` for a decline reason are replaced by in-app nested dialogs (mockup 07). Short "Task id copied" / "Command copied" confirmations appear for 1.6s.
- **Freshness.** The UI re-fetches tasks when the window regains focus, so tasks added by the CLI appear without a reload.
- **Out of scope (no backend support):** card covers and attachments (mockups 05, 10), `Select all cards` and `Sort by priority` column actions, optimistic "Saving" cards, and any server-side change. The CLI token syntax shown in mockup C1 is not added to `backlog add`.
- **BREAKING** (UI only): the read/edit dialog toggle, the sidebar and the command-string result bar are removed. No API, data format or CLI change.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities

- `browse-ui`: Requirements for browsing/filtering, list/board views, task detail, creating, editing, deleting, dialog keyboard/focus handling and the offline bundle are restated for the new interface (filter row, collapsible columns, inline capture with token syntax and undo toast, inline field editing, in-app confirmation dialogs, vendored Poppins). A new requirement covers the light/dark theme. The API-level guarantees — same validation, same files written, parity with the CLI — are unchanged.

## Impact

- **Frontend:** `frontend/src/style.css` rewritten; `frontend/src/App.tsx`, `constants.ts`, `useTasks.ts` reworked; components `TopBar`, `BoardView`, `ListView`, `TaskDialog`, `LinksEditor` rewritten; `Sidebar`, `ResultBar`, `EditForm`, `ReadView`, `MetadataAside`, `TagChips` replaced by new components (filter row, board column, task card, capture draft, task form, task modal, inline fields, confirm dialog, toast); new `frontend/src/tokens.ts` (draft token parser) with unit tests; `App.test.tsx` rewritten against the new UI.
- **Assets:** `frontend/src/fonts/Archivo-Variable.ttf` removed; Poppins woff2 files (+ OFL) vendored under `frontend/src/fonts/`; rebuilt bundle committed under `internal/browse/web/`.
- **Backend:** none. Uses the existing `GET/POST /api/tasks`, `GET/PATCH/DELETE /api/tasks/{id}`, `GET /api/repo`.
- **Sequencing:** builds on `add-ui-task-deletion` (implemented, not yet archived); that change should be archived before this one so its "Deleting a task from the UI" requirement exists in the main spec when this delta modifies it.
