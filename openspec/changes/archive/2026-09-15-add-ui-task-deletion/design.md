## Context

See proposal.md — Why. The browse server (`internal/browse`) exposes a small JSON API under `/api` and serves an embedded React bundle. The API already has `GET/POST /api/tasks` and `GET/PATCH /api/tasks/{id}`; the store already has `Store.Remove(id)`, which `backlog rm` calls — it does `Find` then `os.Remove(t.Path)` and returns the removed task. The React app routes every write through `frontend/src/api.ts` and refreshes via `useTasks`'s `refresh()` after a create or edit.

## Goals / Non-Goals

**Goals:**
- One new endpoint, `DELETE /api/tasks/{id}`, that reuses `Store.Remove` and returns the removed task's `taskview.View`.
- A Delete control in both the read and edit views of the task dialog, guarded by a confirmation that names the task.
- After a confirmed delete: close the dialog, run the same `refresh()` a create/edit runs.

**Non-Goals:**
- Undo / soft delete / trash. Git history is the recovery path, matching `backlog rm`.
- Bulk delete or a delete affordance on list rows / board cards. Delete stays inside the detail dialog.
- Any change to `backlog rm` or the store.

## Decisions

### Reuse `Store.Remove`, no new store method
`handleDeleteTask` mirrors `handleGetTask`: parse the id with `pathID`, call `st.Remove(id)`, map its error to `404` (id not found) and write `taskview.View(t)` for the removed task on success. `Store.Remove` already calls `Find` internally, so an unknown id returns the same not-found error `handleGetTask` surfaces. Alternative — a dedicated `DELETE` store path with extra guards (e.g. refuse to delete `done` tasks) — was rejected: `backlog rm` has no such guard and the spec's consistency scenario requires parity.

### Native `window.confirm` for the prompt
The confirmation is `window.confirm(\`Delete task #\${id} "\${title}"? This cannot be undone.\`)`. It needs no focus-trap work (it is browser-modal), keeps the change small, and matches the existing `window.prompt` used for the board's decline-reason flow in `App.tsx`. Trade-off: a native dialog is not styled to match the UI. Acceptable for a destructive, low-frequency action; a custom in-dialog confirmation can come later without a spec change since the requirement only says "a prompt that names the task".

### Delete handler lives in `App.tsx`, button in the dialog components
`App.tsx` gains `handleDelete()` closing over `state.openTask`: it runs `window.confirm`, and on accept calls `deleteTask(id)`, then `dispatch({ type: "close" })` and `refresh()`, routing errors to `set_error` like `handlePatch`. `TaskDialog` takes an `onDelete` prop and renders a Delete button in the read view (near the EDIT/close controls in `dialog-head`, or as a footer action in `ReadView`) and passes it into `EditForm`, which renders it alongside its existing Cancel button. Placing the handler in `App.tsx` keeps it independent of edit-mode draft state, the same way `handleMove` is.

### `deleteTask` in `api.ts`
`export function deleteTask(id: number): Promise<TaskView>` calling `api<TaskView>("/api/tasks/" + id, { method: "DELETE" })`. The existing `api()` helper already parses a JSON body and throws `ApiError` with the status on non-2xx, so a `404` surfaces as a message in the dialog.

## Risks / Trade-offs

- **A stale dialog re-opens a deleted task** → After delete the dialog is closed and `refresh()` re-fetches; opening a task that was deleted out-of-band already fails today via `getTask`'s `404` path, unchanged here.
- **Native confirm looks out of place / is suppressible by the browser** → Accepted for now; the spec permits a nicer prompt later. The server still requires an explicit `DELETE` request, so a suppressed confirm cannot delete without the button press that triggered it.
- **Binding beyond loopback exposes an unauthenticated destructive endpoint** → The `browse` command already prints a warning when bound to a non-loopback host that the write API is reachable without auth; `DELETE` is covered by that existing warning and needs no new one.

## Migration Plan

Additive. New endpoint and UI control; no data format or CLI change. Rollback is reverting the change and rebuilding the embedded bundle.
