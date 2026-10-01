import { useEffect, useMemo, useReducer, useState } from "react";

import { deleteTask, getTask, patchTask, type PatchTaskBody, type TaskView } from "./api";
import { BoardView } from "./components/BoardView";
import { EmptyResult } from "./components/EmptyResult";
import { FilterRow } from "./components/FilterRow";
import { hasOpenLayer } from "./components/LayeredDialog";
import { ListView } from "./components/ListView";
import { TaskForm } from "./components/TaskForm";
import { TaskMenu } from "./components/TaskMenu";
import { TaskModal } from "./components/TaskModal";
import { ToastHost } from "./components/ToastHost";
import { TopBar } from "./components/TopBar";
import { copyText } from "./clipboard";
import { padId } from "./constants";
import type { ParsedDraft } from "./tokens";
import { useDialogs } from "./useDialogs";
import { useTasks } from "./useTasks";
import { useTheme } from "./useTheme";
import { useToast } from "./useToast";

type View = "list" | "board";
type DialogMode = "read" | "create";
type FilterKey = "status" | "priority" | "tag" | "assignee";

interface State {
  view: View;
  query: string;
  status: string | null;
  priority: string | null;
  tag: string | null;
  assignee: string | null;
  dialogMode: DialogMode | null;
  openTask: TaskView | null;
  // Prefill for the full form when it's opened from a draft's Shift+Enter
  // (design.md D14) — title/priority/tags/assignee only; see App's capture
  // wiring for why status prefill isn't included yet.
  createPrefill: Partial<ParsedDraft> | null;
  error: string;
}

type Action =
  | { type: "set_view"; view: View }
  | { type: "set_query"; query: string }
  | { type: "toggle_filter"; key: FilterKey; value: string }
  | { type: "clear_filter"; key: FilterKey }
  | { type: "reset_all" }
  | { type: "open_read"; task: TaskView }
  | { type: "open_create"; prefill?: Partial<ParsedDraft> }
  | { type: "close" }
  | { type: "set_error"; error: string }
  | { type: "saved"; task: TaskView };

const initialState: State = {
  view: "list",
  query: "",
  status: null,
  priority: null,
  tag: null,
  assignee: null,
  dialogMode: null,
  openTask: null,
  createPrefill: null,
  error: "",
};

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case "set_view":
      // The board groups by status itself, so an active status filter would
      // just hide whole columns for no visible reason (design.md D4).
      return {
        ...state,
        view: action.view,
        status: action.view === "board" ? null : state.status,
      };
    case "set_query":
      return { ...state, query: action.query };
    case "toggle_filter":
      return {
        ...state,
        [action.key]:
          state[action.key] === action.value ? null : action.value,
      };
    case "clear_filter":
      return { ...state, [action.key]: null };
    case "reset_all":
      return {
        ...state,
        query: "",
        status: null,
        priority: null,
        tag: null,
        assignee: null,
      };
    case "open_read":
      return {
        ...state,
        dialogMode: "read",
        openTask: action.task,
        error: "",
      };
    case "open_create":
      return {
        ...state,
        dialogMode: "create",
        openTask: null,
        error: "",
        createPrefill: action.prefill ?? null,
      };
    case "close":
      return {
        ...state,
        dialogMode: null,
        openTask: null,
        error: "",
      };
    case "set_error":
      return { ...state, error: action.error };
    case "saved":
      return {
        ...state,
        dialogMode: "read",
        openTask: action.task,
        error: "",
      };
  }
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function App() {
  const [state, dispatch] = useReducer(reducer, initialState);
  const { theme, toggle: toggleTheme } = useTheme();
  const { confirm, askReason, dialogs } = useDialogs();
  const { toast, show: showToast, dismiss: dismissToast } = useToast();
  const [menu, setMenu] = useState<{ id: number; x: number; y: number } | null>(null);

  const { all, visible, loadError, refresh } = useTasks({
    status: state.status,
    priority: state.priority,
    tag: state.tag,
    assignee: state.assignee,
  });

  // The floating capture button and the "n" shortcut open the full form in
  // both views. "n" is ignored while a text field has focus or a dialog layer is open.
  function openCapture() {
    if (state.dialogMode || hasOpenLayer()) return;
    dispatch({ type: "open_create" });
  }

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key !== "n" || e.ctrlKey || e.metaKey || e.altKey) return;
      const target = e.target as HTMLElement | null;
      if (target?.closest("input, textarea, select, [contenteditable='true']")) return;
      if (state.dialogMode || hasOpenLayer()) return;
      e.preventDefault();
      openCapture();
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.view, state.dialogMode]);

  function openFullFormFromDraft(parsed: ParsedDraft) {
    dispatch({
      type: "open_create",
      prefill: { title: parsed.title, priority: parsed.priority, tags: parsed.tags, assignee: parsed.assignee },
    });
  }

  // Free-text search is a pure derived filter over the loaded set — title,
  // description and tags, case-insensitive, no request.
  const filtered = useMemo(() => {
    const q = state.query.trim().toLowerCase();
    if (!q) return visible;
    return visible.filter((t) => {
      const hay = (
        t.title +
        " " +
        t.description +
        " " +
        t.tags.join(" ")
      ).toLowerCase();
      return hay.includes(q);
    });
  }, [visible, state.query]);

  function openTask(id: number) {
    getTask(id)
      .then((task) => dispatch({ type: "open_read", task }))
      .catch((err) => dispatch({ type: "set_error", error: message(err) }));
  }

  // handlePatch is threaded into the task modal as onPatch: every inline
  // control awaits it, so it has to reject on failure (after the error is
  // surfaced here) for the control's own revert-on-error to run.
  function handlePatch(body: PatchTaskBody): Promise<void> {
    if (!state.openTask) return Promise.resolve();
    return patchTask(state.openTask.id, body)
      .then((task) => {
        dispatch({ type: "saved", task });
        return refresh();
      })
      .catch((err) => {
        dispatch({ type: "set_error", error: message(err) });
        throw err;
      });
  }

  // handleDelete removes the open task after an in-app confirmation naming it
  // (design.md D13, spec: "Confirmation names the task").
  async function handleDelete() {
    if (state.openTask) await deleteWithConfirm(state.openTask);
  }

  async function deleteWithConfirm(t: TaskView) {
    const ok = await confirm({
      title: `${padId(t.id)} · ${t.title}`,
      description: "Delete this task? This cannot be undone.",
    });
    if (!ok) return;
    deleteTask(t.id)
      .then(() => {
        if (state.openTask?.id === t.id) dispatch({ type: "close" });
        return refresh();
      })
      .catch((err) => dispatch({ type: "set_error", error: message(err) }));
  }

  // handleMove applies a board drag-and-drop: it performs the same status edit
  // the dialog performs, independent of whichever task the dialog has open.
  async function handleMove(id: number, status: string) {
    const t = all.find((x) => x.id === id);
    if (!t || t.status === status) return;
    let body: PatchTaskBody;
    if (status === "declined") {
      // The reason is required exactly when the resulting status is `declined`,
      // mirroring the edit form; a cancelled or blank dialog leaves the task be.
      const reason = await askReason();
      if (reason === null) return;
      body = { status, reason };
    } else {
      body = { status };
    }
    patchTask(id, body)
      .then(() => refresh())
      .catch((err) => dispatch({ type: "set_error", error: message(err) }));
  }

  const menuTask = menu ? all.find((t) => t.id === menu.id) : undefined;
  const openId = state.dialogMode === "read" ? (state.openTask?.id ?? null) : null;

  return (
    <div className="app">
      <TopBar
        query={state.query}
        onQuery={(query) => dispatch({ type: "set_query", query })}
        view={state.view}
        onView={(view) => dispatch({ type: "set_view", view })}
        theme={theme}
        onToggleTheme={toggleTheme}
      />

      <main className="main">
        <FilterRow
          all={all}
          view={state.view}
          status={state.status}
          priority={state.priority}
          tag={state.tag}
          assignee={state.assignee}
          query={state.query}
          visibleCount={filtered.length}
          onToggle={(key, value) => dispatch({ type: "toggle_filter", key, value })}
          onClear={(key) => dispatch({ type: "clear_filter", key })}
          onReset={() => dispatch({ type: "reset_all" })}
        />

        {loadError ? (
          <div className="load-error" role="alert">
            Could not load tasks: {loadError}{" "}
            <button type="button" className="text-btn is-strong" onClick={() => refresh()}>
              Retry
            </button>
          </div>
        ) : null}

        <div id="listView" hidden={state.view !== "list"}>
          {loadError && all.length === 0 ? null : filtered.length === 0 ? (
            <EmptyResult onClear={() => dispatch({ type: "reset_all" })} />
          ) : (
            <ListView tasks={filtered} openId={openId} onOpen={openTask} onMenu={(id, x, y) => setMenu({ id, x, y })} />
          )}
        </div>
        <div className="board" id="boardView" hidden={state.view !== "board"}>
          {loadError && all.length === 0 ? null : filtered.length === 0 ? (
            <EmptyResult onClear={() => dispatch({ type: "reset_all" })} />
          ) : (
            <BoardView
              tasks={filtered}
              onOpen={openTask}
              onMove={handleMove}
              onMenu={(id, x, y) => setMenu({ id, x, y })}
              all={all}
              onCreated={refresh}
              onOpenFullForm={openFullFormFromDraft}
              showToast={showToast}
            />
          )}
        </div>
      </main>

      <button
        type="button"
        className="capture-fab"
        aria-label="Add new task"
        onClick={openCapture}
      >
        +
      </button>
      <ToastHost toast={toast} onDismiss={dismissToast} />

      {state.dialogMode === "create" ? (
        <TaskForm
          tasks={all}
          prefill={state.createPrefill}
          onClose={() => dispatch({ type: "close" })}
          onCreated={refresh}
          onOpen={openTask}
          showToast={showToast}
        />
      ) : null}
      {state.dialogMode === "read" && state.openTask ? (
        <TaskModal
          task={state.openTask}
          tasks={all}
          error={state.error}
          onClose={() => dispatch({ type: "close" })}
          onPatch={handlePatch}
          onDelete={handleDelete}
          onOpenLink={openTask}
          onError={(err) => dispatch({ type: "set_error", error: err })}
          askReason={askReason}
        />
      ) : null}
      {menuTask ? (
        <TaskMenu
          task={menuTask}
          x={menu!.x}
          y={menu!.y}
          onClose={() => setMenu(null)}
          onOpen={() => openTask(menuTask.id)}
          onCopyId={() => void copyText(padId(menuTask.id))}
          onMove={(status) => void handleMove(menuTask.id, status)}
          onDelete={() => void deleteWithConfirm(menuTask)}
        />
      ) : null}
      {dialogs}
    </div>
  );
}
