## Why

The board view groups tasks into one column per status but is purely a read layout: moving a task forward means opening it, entering edit mode, changing the status field and saving. On a board, the natural gesture for "this is done now" is to drag the card into the next column. Supporting that gesture removes several clicks from the single most common board interaction without adding a new way for a change to bypass validation — the drop performs exactly the same status edit the dialog already performs.

## What Changes

- The board view gains drag-and-drop: a task card can be dragged from its status column and dropped onto another status column, which sets the task's status to that column's status.
- A drop is equivalent to editing the task's status and saving: it goes through the same `PATCH /api/tasks/{id}` request, the same server-side validation, and the same list refresh.
- Dropping a card onto the `declined` column prompts for a decline reason before the request is sent; cancelling the prompt, or supplying an empty reason, leaves the task unchanged. This mirrors the edit form, where the reason field is required exactly when the resulting status is `declined`.
- Dropping a card back onto its own column, or a failed drop, is a no-op that leaves the board as it was.
- **BREAKING** (spec-level only): the current "No drag-and-drop" guarantee in the board-view requirement is removed. No API or data format changes.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `browse-ui`: The "Switching between a list and a board view" requirement is reworded to allow status changes by dragging a card between columns. Its "No drag-and-drop" scenario is narrowed to "status is the only field a drag can change", and new scenarios cover a successful drag, a drag onto `declined` (with and without a reason), a no-op drop onto the same column, and consistency with the CLI.

## Impact

- Frontend: `frontend/src/components/BoardView.tsx` (drag sources and drop targets), `frontend/src/App.tsx` (a move handler that reuses the existing patch/refresh path), `frontend/src/style.css` (drag affordance and drop-target highlight), and the board tests in `frontend/src/App.test.tsx`. The committed embedded bundle under `internal/browse/web/assets/` is rebuilt.
- Backend: none. `PATCH /api/tasks/{id}` already accepts a `status` (and `reason`) change and validates it.
