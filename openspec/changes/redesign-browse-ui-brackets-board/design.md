## Context

See proposal.md — Why. The shape of the current code and the constraints the redesign works within:

- **The API is fixed.** `internal/browse/handlers.go` exposes `GET /api/repo`, `GET/POST /api/tasks`, `GET/PATCH/DELETE /api/tasks/{id}`. `POST` accepts title, description, tags, priority, assignee, files, refs, links — **not status**; every created task starts as `new`. `PATCH` takes any subset of title, description, tags, priority, status, reason, assignee, refs, links, runs pre-hooks, validates (`applyPatch`) and refuses `done` while a blocked-by target is unfinished (409). `DELETE` runs the `rm` pre-hook and can return 409. No endpoint streams changes.
- **The frontend is a small React 18 + TS + Vite app** (`frontend/src`): one reducer in `App.tsx`, `useTasks` owning an `?all=1` fetch and a filtered fetch, components per screen, a hand-rolled `markdown.ts`, and `App.test.tsx` driving the whole app against a fake `fetch`. Output is committed into `internal/browse/web/` and embedded; `just build-web` rebuilds.
- **The design source** is the Claude Design project's `Mockups.dc.html` plus the "Brackets board" design-system tokens (`tokens/{colors,typography,spacing,effects,base}.css`) and a `_ds_bundle.js` of reference JSX components (`TaskCard`, `ListHeader`, `Label`, `SearchField`, `AddButton`, …). The mockups are static inline-styled frames, several of which depict unsupported features (proposal — Out of scope).
- **Offline guarantee.** The DS `fonts.css` imports Poppins from Google Fonts; the spec forbids any network request, so the face has to be vendored.

## Goals / Non-Goals

**Goals:**
- Pixel-level fidelity to the mockup frames for every supported surface, expressed as CSS classes over DS tokens rather than inline styles.
- Keep all server interaction inside `api.ts`; no new endpoint, no handler change.
- Keep the app state understandable in one reducer; new transient UI state (drafts, toasts, confirm dialogs) lives next to the component that owns it.
- Keep the whole-app test style (`App.test.tsx` + fake fetch) and add focused unit tests for pure logic (token parser, tag color, duplicate match).

**Non-Goals:**
- Importing or shipping `_ds_bundle.js` at runtime — it depends on a global `React` and inline styles; we port its geometry into CSS instead.
- Optimistic rendering: every write waits for the server and then `refresh()`es, as today.
- Real-time sync (websocket/polling). Focus-refetch is the only freshness mechanism.
- Mobile board layout beyond "columns scroll horizontally"; phones keep a usable list.

## Decisions

### D1: Port DS tokens into `style.css`, drop Modernist entirely
`style.css` starts with the DS token blocks verbatim (`--ink-*`, `--surface-*`, `--label-*`, `--radius-*`, `--space-*`, `--text-*` composites), followed by component classes named after the mockup parts (`.topbar`, `.filter-row`, `.col`, `.col-head`, `.card`, `.label-bar`, `.slot`, `.draft`, `.modal`, `.details`, `.pill-btn`, `.text-btn`, `.popover`, `.toast`). No CSS-in-JS, no Tailwind: the existing build already handles one stylesheet and the mockups are regular enough for plain classes.
*Alternative:* inline styles copied from the mockups — rejected: unthemeable (dark theme must redefine tokens) and untestable by class.

### D2: Poppins vendored from `@fontsource/poppins`
Add `@fontsource/poppins` as a devDependency and `@import` its `300.css`, `400.css`, `500.css` (latin + latin-ext subsets) from `style.css`; Vite fingerprints the woff2 files into `internal/browse/web/assets/`. The OFL licence text is committed next to the old font location. Archivo and its `@font-face` are removed. The `--font-core` stack keeps the system fallbacks, so Cyrillic titles (Poppins has no Cyrillic) fall back cleanly.
*Alternative:* copy woff2 files by hand into `src/fonts/` — equivalent output, but a versioned package is easier to update and audit.

### D3: Theme via token override on `:root[data-theme]`
The dark palette is the exact override set from mockup 02 (`--ink-100:#f6f4f1`, `--surface-page:#131211`, …). Light tokens live on bare `:root`; dark tokens are applied under `:root[data-theme="dark"]` and under `@media (prefers-color-scheme: dark)` for `:root:not([data-theme="light"])`. A `useTheme` hook reads/writes `localStorage["backlog.theme"]` (wrapped in try/catch) and stamps `data-theme` on `<html>`. The top bar toggle shows a moon in light and a sun in dark, as in the mockups.

### D4: Filter row replaces sidebar and result bar; filters stay server-side
`FilterRow` renders Priority (text toggles, `All` clears), Tags chips, Assignee chips (only if any), Status text toggles (list view only — on the board the columns already are statuses), `Reset`, and `N of M tasks`. Filter state keeps its current reducer keys and `useTasks` still sends them as query params, so the existing `selectTasks` logic stays the source of truth. `M` is `all.length`, `N` is the free-text-filtered visible count. Switching to board clears an active status filter (otherwise four columns would be empty for no visible reason). `useTasks` gains a `focus` listener that calls `refresh()`.
*Alternative:* do all filtering client-side over `all` — simpler, but would drift from `backlog list` selection rules that the spec ties the UI to.

### D5: Board columns — five statuses, collapsible, menu
`BoardView` renders `BoardColumn` per `STATUS_ORDER` inside a horizontally scrolling row with a 14px styled scrollbar (`::-webkit-scrollbar` + `scrollbar-color`). `ColumnHeader` shows `Name count …`; the `…` opens a `Popover` with `Collapse list` / `Expand list` and `Copy list as markdown` (`- [NNN] Title (priority) #tags @assignee` per visible task, via `navigator.clipboard.writeText`). Collapsed ids are a `Set<string>` persisted to `localStorage["backlog.collapsed"]`; a collapsed column is the 58px rail from mockup 02 and remains a drop target. Drag keeps the existing `dataTransfer` protocol; the dragged card gets `.is-dragging` (opacity .4) and the hovered column `.is-drop` (inset 1px ring).

### D6: Card anatomy and tag colours
`TaskCard` = label bars (one 60×8 bar per tag, max 6, `title` = tag name) → title → description excerpt (first non-heading paragraph of the markdown, stripped of markup, clamped to 3 lines) → footer (priority dot + label, spacer, `@assignee` / `unassigned`). Tag colour is a stable hash of the lowercase tag name into the ten `--label-*` tokens, so a tag has the same colour on cards, chips in the modal and in autocomplete (mockup C3: "цвет назначается по кругу"). Priority dots: high `--label-red`, medium `--label-yellow`, low `--label-blue`.

### D7: Inline capture draft with a pure token parser
`parseDraft(line, knownStatuses)` in `tokens.ts` returns `{ title, priority?, tags[], assignee?, status? }`: whitespace-split words; `!high|!med|!medium|!low` → priority; `#x` → tag; `@x` → assignee; `>new|todo|doing|done` → status; anything else (including unknown `!foo`, `>foo`, or a bare `#`) stays in the title. The mockup's "unknown name stays in the title" for `@` cannot be applied — the backend has no user registry and assignees are free text — so any `@name` is accepted; recorded here as a deliberate deviation. `CaptureDraft` renders the line as an `<input>` plus a parsed-summary row (priority dot, tag chips, `@assignee`, `>status`) and the hint row (`↵ create · ⇧↵ details · Esc`). Tag autocomplete activates when the caret's current word starts with `#`: up to five existing tags (from `all`) containing the typed prefix, ranked prefix-first then by usage count, then a `Create tag «x»` row; ↑/↓ move, `Tab`/`Enter` accept (replacing the word), `Esc` closes only the list.

Duplicate hint: before creating, if some task in `all` has `title.trim().toLowerCase()` equal to the parsed title's, the draft shows the mockup C6 block and waits for `Open` / `Create anyway`. Exact-match only, to avoid false positives; fuzzy matching is left out.

### D8: Creating with a status = `POST` then `PATCH`
`POST /api/tasks` cannot set status, so `createTaskWithStatus(body, status)` in `api.ts` posts, then — only if `status !== "new"` — patches `{status}` and returns the patched view. If the patch fails, the task exists as `new`; the UI refreshes, shows the server error in the draft/form, and the toast still reports the task (in `New`). `declined` is excluded from drafts and from the form's status choices because a reason would be needed; declining stays an edit.
*Alternative:* add `status` to the create request — a backend change, excluded by the proposal.

### D9: Toast with Undo, short confirmations
A tiny `ToastHost` at app level holds at most one toast `{ text, actions, until }`; creating replaces any previous one. The create toast lasts 6s and fades out (`opacity` transition); `Open` calls `openTask(id)`, `Undo` calls `deleteTask(id)` then `refresh()` and, for a draft-origin create, restores the original line into that column's draft. Copy confirmations ("Task id copied", "Command copied") are not toasts: they render as a 20%-ink line under the modal title for 1.6s, per mockup 11.

### D10: Full form (`TaskForm`) replaces `EditForm` for create only
Fields per mockup C5 — Title, Description (the same `MarkdownEditor` as D12), Status (text toggles New/Todo/Doing/Done), Priority (dot toggles), Assignee (text input with `unassigned` placeholder), Tags (chips + `+ tag` inline input with the same autocomplete), then a collapsed-by-default `More` section with Source files, References (comma-separated) and Links (`LinkPicker`). Footer: `Create task` pill, `Create and add another` text button, hint `⌘↵ create · Esc cancel`. Prefill comes from the draft's parse result.

### D11: Task modal with inline fields (`TaskModal`)
Layout per mockup 05: 1100×760 max `92vw × 90vh`, 10% ink overlay, 10px radius, close X. Header: `NNN` at 40% ink (click → copy id) + title as an `InlineText` at display size. Action row: `Copy command`, `Delete`. `Details` grid rows: Status (`InlineSelect`), Priority (`InlineSelect` with dots), Assignee (`InlineText`, empty → `unassigned`), Tags (`InlineText` with comma-separated value rendered as colour chips at rest), Source (files, read-only), Branch · commit (read-only), Created, Author, Refs (read-only, as today), Reason (read-only, declined only). Then `Description` (`MarkdownEditor`, preview at rest, `Edit` enters Write), then `Linked tasks` (rows: type chip, id, target title — clickable to open —, target status, `Remove`) and `Link a task` (`LinkPicker`: type select then search popover over `all`, mockup 08).

Each inline control calls `onPatch(partialBody)` with only its own field, so a PATCH never carries untouched fields (same effect the spec requires; it also means only the relevant hook family fires). Links edits send the full new `links` array, as today. Section headers are collapsible (chevron) with local state only.
*Alternative:* keep a whole-task edit mode — rejected, the mockups have no edit mode.

`InlineText` / `InlineSelect` are generic: rest state is text styled as the value; activation (click/Enter/Space when focused) swaps in an input or a `Popover` listbox; `Enter` commits, `Esc` cancels and stops propagation so the modal's Escape handler doesn't also fire. While a commit is in flight the control is disabled; on error it reverts to the saved value and the modal's error line shows the message.

### D12: `MarkdownEditor` — Write/Preview with an insert toolbar
Write mode: toolbar buttons B, I, `code`, H, —, “, `</>` that wrap the selection (or insert at caret) with `**…**`, `_…_`, `` `…` ``, `### `, `- `, `> `, fenced block; textarea below. Preview renders through the existing `md()` with new `.md` styles from mockup 06 (paragraphs at 40% ink, `h3` at title size, `—` bullets, blockquote/pre on sunken fill). `markdown.ts` gains `_italic_`/`*italic*` and `> quote` support, both escaping first as before; its tests are extended.

### D13: In-app `ConfirmDialog` and `ReasonDialog`
A shared `LayeredDialog` handles the portal, backdrop (`rgba(0,0,0,.35)` for nested, 10% ink for top-level), focus trap, focus restore and Escape — replacing the per-dialog code in `TaskDialog`. A module-level dialog stack (array of open layer ids) makes only the topmost layer respond to Escape/Tab. `ConfirmDialog` (mockup 07: title, description, red `Delete` pill `--label-red-deep`, `Cancel`) and `ReasonDialog` (same shell, a text input, `Decline` pill disabled while empty) are promise-returning via a `useDialogs()` hook, so `handleMove` and the status select can `await askReason()` in place of `window.prompt`.

### D14: Keyboard entry points
A document `keydown` listener in `App` opens capture on `n` (no modifiers) when the target isn't an input/textarea/contenteditable and no layer is open, and also on `Ctrl/⌘+N` where the browser lets it through (Chrome reserves it; not relied on). Board: draft opens in the first non-collapsed column (normally `new`) and scrolls it into view; list: full form.

### D15: List view in the same visual language
No mockup frame exists for the list. It becomes a hairline-separated table on white: id (20% ink, `--text-count`), title (`--text-title`) with the description excerpt below at 40%, tag label bars, priority dot + label, `@assignee`/`unassigned`, status text. Rows are focusable, `Enter` opens, hover gets the card hover ring treatment. Under 720px the tag and status columns hide.

### D16: Tests
`App.test.tsx` is rewritten around the new selectors (roles and accessible names first: `button "Board"`, `region` per column labelled by status, `dialog` for modal/confirm) and keeps every existing behavioural case — filters, free-text no-request, Escape/focus, server error surfaced, delete confirm/cancel, links jump, drag move/decline/cancel/same-column — translated to the new UI (`window.confirm`/`prompt` stubs become interactions with the in-app dialogs). New cases: draft create with tokens and non-`new` column (asserts POST then PATCH bodies), empty-title rejection, duplicate hint, undo toast issues DELETE, inline assignee edit sends `{assignee}` only, Escape cancels inline edit before closing, theme toggle persists, collapse persists. `tokens.test.ts` and `markdown.test.ts` cover the pure functions.

## Risks / Trade-offs

- **Create-with-status is two writes** → post-add and post-set hooks both fire and a failing PATCH leaves a `new` task. Mitigated by surfacing the error and refreshing so the task is visible where it actually is; documented in D8.
- **`n` shortcut collides with typing** → guarded by target/layer checks; covered by a test that typing `n` in the search field does not open a draft.
- **Tag colours are hashed, not chosen** → two tags may share a colour; names are available on hover and in the modal, and the palette has ten entries.
- **Duplicate hint is exact-match only** → misses near duplicates; accepted to avoid nagging on false positives.
- **Focus-refetch can replace data under an open inline edit** → inline controls keep their local draft until commit/cancel; the modal re-reads the open task from the refreshed set only when no inline edit is active.
- **Poppins lacks Cyrillic** → system fallback renders Cyrillic titles in a different face; accepted, same as the DS itself.
- **Bundle grows** (three woff2 weights, more components) → still a few hundred KB, embedded once; acceptable for a local tool.
- **Large test rewrite can hide regressions** → every existing test case is mapped one-to-one before new cases are added (tasks 9.x).

## Migration Plan

UI-only. Rebuild the embedded bundle with `just build-web` and commit it together with the source. Rollback is reverting the commit (source + bundle). Per-browser `localStorage` keys (`backlog.theme`, `backlog.collapsed`) are additive and harmless if the UI is rolled back. Archive `add-ui-task-deletion` before archiving this change so the modified "Deleting a task from the UI" requirement exists in the main spec.
