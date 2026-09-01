## Context

See proposal.md - Why. The board view (`frontend/src/components/BoardView.tsx`) renders one column per status from `STATUS_ORDER` and filters the already-loaded task set into each. Status changes today go through `App.handlePatch`, which calls `patchTask(id, body)` (`PATCH /api/tasks/{id}`) and then `refresh()`. That handler is bound to `state.openTask` — the task open in the dialog — so the board cannot reuse it as-is. The backend `applyPatch` already validates a `status` change and requires a non-empty `reason` exactly when the resulting status is `declined`; leaving `declined` clears the reason server-side.

The UI is a committed, embedded bundle: source under `frontend/`, build output under `internal/browse/web/` (rebuilt with `just build-web`, never by `go build`). No network, no runtime dependencies beyond React.

## Goals / Non-Goals

**Goals:**
- Dragging a card between columns changes the task's status through the existing patch + refresh path.
- A drop onto `declined` collects a reason first and aborts cleanly if none is given.
- Keep the board otherwise unchanged: click-to-open, keyboard activation, empty-column notes.

**Non-Goals:**
- Reordering cards within a column (there is no manual order; sort is priority then id).
- Changing priority, tags, or any other field by dragging.
- A custom drag library or animated card movement — the refresh re-renders the board.
- Touch / mobile drag: the board is already hidden below the mobile breakpoint.
- Optimistic UI: the card moves when `refresh()` completes, matching how the dialog edit behaves.

## Decisions

### Native HTML5 drag-and-drop, no dependency
Use `draggable`, `onDragStart`/`onDragOver`/`onDrop`. The offline, self-contained-UI requirement and the tiny dependency list make a drag library (dnd-kit, react-dnd) a poor trade for a single drag interaction. The card carries `dataTransfer.setData("text/plain", String(id))`; the column reads it on drop. A React state holding the drag-over column id drives the drop-target highlight, cleared on `dragend`/`drop`/`dragleave`.

Alternative considered: pointer-event drag from scratch — more control over the drag image but far more code for no requirement we have.

### A dedicated `onMove(id, status)` on BoardView, handled in App
`App` gains `handleMove(id, status)`, independent of `state.openTask`:
- If `status` equals the task's current status, do nothing.
- If `status === "declined"`, call `window.prompt` for a reason; if the result is null or blank, abort. Otherwise send `{ status, reason }`.
- Otherwise send `{ status }`.
- On success `refresh()`; on failure dispatch `set_error` (surfaced by `ResultBar` the same way load errors are — the dialog is not open).

`window.prompt` is a deliberately minimal choice: it keeps the drop synchronous-feeling and avoids a second dialog state machine for what is a rare path. The edit dialog remains the full-featured way to decline. This is called out as a risk below.

Alternative considered: open the edit dialog pre-filled with `declined` on a drop onto that column. Rejected for this change as more moving parts; can be revisited if `prompt` proves too blunt.

### BoardView stays presentational
BoardView receives `onOpen` and `onMove` and owns only the transient `dragOverStatus` state. No data fetching moves into it.

## Risks / Trade-offs

- **`window.prompt` for the decline reason is spartan** (no multiline, no markdown affordance, blockable by the browser) → the edit dialog still offers the full decline flow; the prompt is a shortcut, not the only path. Revisit if it proves inadequate.
- **A drag onto `declined` that the user then cancels still fired `dragOver` highlights** → purely visual; state is cleared on drop/dragend.
- **Accidental drags on a touchpad** → the drop still goes through server validation, and the reverse drag restores the status; `done`/`declined` moves are the only ones that leave the default view, and `refresh()` makes the outcome visible immediately.
- **Test coverage of native DnD in jsdom** → jsdom does not implement a real drag; tests fire `dragStart`/`drop` with a stub `DataTransfer` (or call the handlers via `fireEvent`). The existing board test file already stubs `fetch`; the new tests assert the `PATCH` body and the post-refresh column.
