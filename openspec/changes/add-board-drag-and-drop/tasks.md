## 1. Move handler in App

- [x] 1.1 Add `handleMove(id: number, status: string)` to `frontend/src/App.tsx`: look the task up in the loaded set, no-op if `status` matches its current status; when `status === "declined"` read a reason via `window.prompt` and abort on null/blank, sending `{ status, reason }`, otherwise send `{ status }`; call `patchTask` then `refresh()` on success and `dispatch({ type: "set_error", ... })` on failure. Verify by unit test that the correct PATCH body is sent for a plain move and for a declined move.
- [x] 1.2 Pass `onMove={handleMove}` into `<BoardView>` in `App.tsx`. Verify the app still type-checks (`npm run typecheck`).

## 2. Drag-and-drop in BoardView

- [x] 2.1 Add an `onMove: (id: number, status: string) => void` prop to `BoardView` and a transient `dragOverStatus` state. Verify with `npm run typecheck`.
- [x] 2.2 Make each card `draggable`, setting `dataTransfer.setData("text/plain", String(t.id))` and `effectAllowed = "move"` on `dragStart`; keep the existing click/keydown open behavior intact. Verify existing board test ("renders the board with a card per column") still passes.
- [x] 2.3 On each column, handle `dragOver` (preventDefault, set `dropEffect = "move"`, record `dragOverStatus`), `dragLeave`/`drop`/`dragEnd` (clear `dragOverStatus`), and `drop` (read the id, call `onMove(id, columnStatus)`). Add a CSS class while `dragOverStatus === k`.
- [x] 2.4 Add the drop-target highlight and a grab cursor for `.board-card` to `frontend/src/style.css`.

## 3. Tests

- [x] 3.1 In `frontend/src/App.test.tsx`, add a test: switch to the board, fire `dragStart` on a card and `drop` on another column with a stub `DataTransfer`, and assert the `PATCH /api/tasks/{id}` body is `{ status: <column> }` and the card renders in the target column after refresh.
- [x] 3.2 Add a test that dropping on the `declined` column with `window.prompt` stubbed to return a reason sends `{ status: "declined", reason: "<text>" }`, and that a stubbed `prompt` returning `null` sends no PATCH.
- [x] 3.3 Add a test that dropping a card on its own column sends no PATCH request.
- [x] 3.4 Run `npm test` in `frontend/` and confirm the whole suite passes.

## 4. Rebuild the embedded bundle

- [x] 4.1 Run `just build-web` (or `cd frontend && npm ci && npm run build`) and commit the regenerated files under `internal/browse/web/`.
- [x] 4.2 Run `just` (fmt, vet, `go test ./...`) and confirm the Go suite, including `internal/browse` and `e2e_test.go`, still passes.

## 5. Docs

- [x] 5.1 Update `README.md` where it describes the browse board view to mention that cards can be dragged between columns to change status.
