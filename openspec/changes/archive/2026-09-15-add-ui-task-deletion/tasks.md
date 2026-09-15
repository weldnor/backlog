## 1. Backend: DELETE endpoint

- [x] 1.1 Add `handleDeleteTask(st *store.Store)` in `internal/browse/handlers.go`: parse the id with `pathID`, call `st.Remove(id)`, return `404` with the error message when it fails and `200` with `taskview.View(t)` for the removed task on success.
- [x] 1.2 Register `mux.HandleFunc("DELETE /api/tasks/{id}", handleDeleteTask(st))` in `newMux`.
- [x] 1.3 Add handler tests in `internal/browse/handlers_test.go`: deleting an existing task returns `200` with the removed task and the task file is gone from the store; deleting an unknown id returns `404` and no file is removed; a following `GET /api/tasks` does not list the deleted task. Verify with `go test ./internal/browse/`.

## 2. Frontend: API client

- [x] 2.1 Add `export function deleteTask(id: number): Promise<TaskView>` to `frontend/src/api.ts`, calling `api<TaskView>("/api/tasks/" + id, { method: "DELETE" })`.

## 3. Frontend: delete action in the dialog

- [x] 3.1 Add `handleDelete()` to `frontend/src/App.tsx` closing over `state.openTask`: call `window.confirm` with a message naming the task (`#<id> "<title>"`), and on accept call `deleteTask(id)`, then `dispatch({ type: "close" })` and `refresh()`, routing errors to `dispatch({ type: "set_error", ... })`.
- [x] 3.2 Thread an `onDelete` prop through `TaskDialog` and render a Delete button in the read view (in `dialog-head` or as a footer control in `ReadView`), shown only when `mode !== "create"`.
- [x] 3.3 Pass `onDelete` into `EditForm` and render a Delete button next to its Cancel button in edit mode (not in create mode).
- [x] 3.4 Add destructive-button styling to `frontend/src/style.css` (e.g. `.btn-danger`) for the Delete controls.

## 4. Frontend: tests

- [x] 4.1 In `frontend/src/App.test.tsx`, add tests: confirming delete from the read view removes the task from the list and closes the dialog; cancelling the confirm leaves the task and dialog in place; the Delete button is present in both the read and edit views. Stub `window.confirm`. Verify with the frontend test command.

## 5. Rebuild embedded bundle & full verification

- [x] 5.1 Run `just build-web` to regenerate `internal/browse/web/assets/` and commit the rebuilt bundle.
- [x] 5.2 Run `just` (fmt, vet, test) and the frontend tests; confirm all pass.
- [x] 5.3 Manually verify end to end: `backlog browse`, open a task, delete it with confirm, see it disappear from list and board, and confirm `backlog list` no longer shows it.
- [x] 5.4 Run `openspec validate add-ui-task-deletion --strict` and confirm it passes.
