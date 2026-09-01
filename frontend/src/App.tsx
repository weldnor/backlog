import { useEffect, useMemo, useReducer, useState } from "react";

import {
  createTask,
  deleteTask,
  getRepo,
  getTask,
  patchTask,
  type CreateTaskBody,
  type PatchTaskBody,
  type RepoInfo,
  type TaskView,
} from "./api";
import { BoardView } from "./components/BoardView";
import { ListView } from "./components/ListView";
import { ResultBar } from "./components/ResultBar";
import { Sidebar } from "./components/Sidebar";
import { TaskDialog } from "./components/TaskDialog";
import { TopBar } from "./components/TopBar";
import { useTasks } from "./useTasks";

type View = "list" | "board";
type DialogMode = "read" | "edit" | "create";
type FilterKey = "status" | "priority" | "tag";

interface State {
  view: View;
  query: string;
  status: string | null;
  priority: string | null;
  tag: string | null;
  dialogMode: DialogMode | null;
  openTask: TaskView | null;
  error: string;
}

type Action =
  | { type: "set_view"; view: View }
  | { type: "set_query"; query: string }
  | { type: "toggle_filter"; key: FilterKey; value: string }
  | { type: "open_read"; task: TaskView }
  | { type: "open_create" }
  | { type: "enter_edit" }
  | { type: "leave_edit" }
  | { type: "close" }
  | { type: "set_error"; error: string }
  | { type: "saved"; task: TaskView };

const initialState: State = {
  view: "list",
  query: "",
  status: null,
  priority: null,
  tag: null,
  dialogMode: null,
  openTask: null,
  error: "",
};

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case "set_view":
      return { ...state, view: action.view };
    case "set_query":
      return { ...state, query: action.query };
    case "toggle_filter":
      return {
        ...state,
        [action.key]:
          state[action.key] === action.value ? null : action.value,
      };
    case "open_read":
      return {
        ...state,
        dialogMode: "read",
        openTask: action.task,
        error: "",
      };
    case "open_create":
      return { ...state, dialogMode: "create", openTask: null, error: "" };
    case "enter_edit":
      return { ...state, dialogMode: "edit", error: "" };
    case "leave_edit":
      return { ...state, dialogMode: "read", error: "" };
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

// currentCommand renders the live `backlog list …` invocation the current
// filters map to. The UI always shows every status, so the base command is
// `backlog list --all` unless an explicit status filter narrows it.
function currentCommand(state: State): string {
  const parts = ["backlog", "list"];
  if (state.status) {
    parts.push("--status", state.status);
  } else {
    parts.push("--all");
  }
  if (state.priority) parts.push("--priority", state.priority);
  if (state.tag) parts.push("--tag", state.tag);
  parts.push("--json");
  return parts.join(" ");
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function App() {
  const [state, dispatch] = useReducer(reducer, initialState);
  const [repo, setRepo] = useState<RepoInfo | null>(null);

  const { all, visible, refresh } = useTasks({
    status: state.status,
    priority: state.priority,
    tag: state.tag,
  });

  useEffect(() => {
    // The repo chip is decorative; a failure here is not fatal.
    getRepo()
      .then(setRepo)
      .catch(() => {});
  }, []);

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

  function handleCreate(body: CreateTaskBody) {
    createTask(body)
      .then(() => {
        dispatch({ type: "close" });
        return refresh();
      })
      .catch((err) => dispatch({ type: "set_error", error: message(err) }));
  }

  function handlePatch(body: PatchTaskBody) {
    if (!state.openTask) return;
    patchTask(state.openTask.id, body)
      .then((task) => {
        dispatch({ type: "saved", task });
        return refresh();
      })
      .catch((err) => dispatch({ type: "set_error", error: message(err) }));
  }

  // handleDelete removes the open task after a naming confirmation. Like
  // handleMove it closes over state.openTask rather than any edit-mode draft, so
  // it works identically from the read and edit views.
  function handleDelete() {
    const t = state.openTask;
    if (!t) return;
    if (!window.confirm(`Delete task #${t.id} "${t.title}"? This cannot be undone.`)) {
      return;
    }
    deleteTask(t.id)
      .then(() => {
        dispatch({ type: "close" });
        return refresh();
      })
      .catch((err) => dispatch({ type: "set_error", error: message(err) }));
  }

  // handleMove applies a board drag-and-drop: it performs the same status edit
  // the dialog performs, independent of whichever task the dialog has open.
  function handleMove(id: number, status: string) {
    const t = all.find((x) => x.id === id);
    if (!t || t.status === status) return;
    let body: PatchTaskBody;
    if (status === "declined") {
      // The reason is required exactly when the resulting status is `declined`,
      // mirroring the edit form; a cancelled or blank prompt leaves the task be.
      const reason = window.prompt("Reason for declining this task?");
      if (reason === null || reason.trim() === "") return;
      body = { status, reason };
    } else {
      body = { status };
    }
    patchTask(id, body)
      .then(() => refresh())
      .catch((err) => dispatch({ type: "set_error", error: message(err) }));
  }

  const openId =
    state.dialogMode === "read" || state.dialogMode === "edit"
      ? (state.openTask?.id ?? null)
      : null;

  return (
    <div className="app">
      <TopBar
        repo={repo}
        query={state.query}
        onQuery={(query) => dispatch({ type: "set_query", query })}
        view={state.view}
        onView={(view) => dispatch({ type: "set_view", view })}
        onCapture={() => dispatch({ type: "open_create" })}
      />

      <div className="layout">
        <Sidebar
          all={all}
          status={state.status}
          priority={state.priority}
          tag={state.tag}
          onPick={(key, value) =>
            dispatch({ type: "toggle_filter", key, value })
          }
        />

        <main className="main">
          <ResultBar
            count={filtered.length}
            command={currentCommand(state)}
          />

          <div id="listView" hidden={state.view !== "list"}>
            <ListView tasks={filtered} openId={openId} onOpen={openTask} />
          </div>
          <div className="board" id="boardView" hidden={state.view !== "board"}>
            <BoardView tasks={filtered} onOpen={openTask} onMove={handleMove} />
          </div>
        </main>
      </div>

      {state.dialogMode ? (
        <TaskDialog
          mode={state.dialogMode}
          task={state.openTask}
          error={state.error}
          onClose={() => dispatch({ type: "close" })}
          onToggleEdit={() =>
            dispatch({
              type: state.dialogMode === "edit" ? "leave_edit" : "enter_edit",
            })
          }
          onCancelEdit={() => dispatch({ type: "leave_edit" })}
          onCreate={handleCreate}
          onPatch={handlePatch}
          onDelete={handleDelete}
        />
      ) : null}
    </div>
  );
}
