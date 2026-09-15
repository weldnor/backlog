## Why

The browse UI can create and edit tasks but deliberately cannot delete one — removing a task means dropping to a terminal and running `backlog rm`. For a human skimming what has accumulated, the task that should just be gone is a common case, and switching tools to act on it breaks the loop the UI exists to serve. The delete is a plain file removal that git already keeps history for, so the only thing missing is a safe way to trigger it from the browser.

## What Changes

- The JSON API gains `DELETE /api/tasks/{id}`, which removes the task's file exactly as `backlog rm` does and returns the removed task's view.
- The task detail dialog gains a **Delete** action, present in both the read view and the edit view.
- Triggering delete opens a confirmation prompt naming the task; the task is removed only on confirm. Cancelling leaves the task untouched.
- On a confirmed delete the dialog closes and the list and board refresh, with the task gone.
- **BREAKING** (spec-level only): the current "No deletion from the UI" guarantee in the `browse-ui` spec is removed. No data format changes; `backlog rm` is unchanged.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `browse-ui`: The "No deletion from the UI" requirement is removed and replaced by a "Deleting a task from the UI" requirement covering the `DELETE /api/tasks/{id}` endpoint, the Delete action in both the read and edit views, the naming confirmation prompt, the no-op on cancel, and the list/board refresh after a confirmed delete. The "Keyboard and focus handling" requirement is unaffected — the confirmation prompt is a native `window.confirm`.

## Impact

- Backend: `internal/browse/handlers.go` (new `handleDeleteTask`, route registration in `newMux`), `internal/browse/handlers_test.go`. `store.Remove` already performs the file deletion and is reused as-is.
- Frontend: `frontend/src/api.ts` (`deleteTask`), `frontend/src/App.tsx` (a delete handler that confirms, calls the API, closes the dialog and refreshes), `frontend/src/components/TaskDialog.tsx` and `frontend/src/components/EditForm.tsx` (the Delete button in each view), `frontend/src/style.css` (destructive-button styling), and `frontend/src/App.test.tsx`. The committed embedded bundle under `internal/browse/web/assets/` is rebuilt.
