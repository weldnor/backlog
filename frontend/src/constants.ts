import type { TaskView } from "./api";

// Fixed lifecycle order, matching internal/task and the old app.js.
export const STATUS_ORDER = ["todo", "doing", "done", "declined"] as const;
export const PRI_ORDER = ["high", "medium", "low"] as const;

export interface StatusMeta {
  headLabel: string;
  fg: string;
}

export const STATUS_META: Record<string, StatusMeta> = {
  todo: { headLabel: "TODO", fg: "status-todo" },
  doing: { headLabel: "DOING", fg: "status-doing" },
  done: { headLabel: "DONE", fg: "status-done" },
  declined: { headLabel: "DECLINED", fg: "status-declined" },
};

// The board's own empty-column copy, kept verbatim from the old UI: it teaches
// the same distinctions the README does.
export const BOARD_EMPTY_NOTE: Record<string, string> = {
  todo: "Nothing captured here yet.",
  doing: "Nothing in flight.",
  done: "Archive — acted on. Behind --all.",
  declined:
    "Always in scope for search — a duplicate must not hide behind a filter.",
};

export interface PriMeta {
  label: string;
  cls: string;
  fg: string;
}

export const PRI_META: Record<string, PriMeta> = {
  high: { label: "HIGH", cls: "tag tag-outline", fg: "pri-high" },
  medium: { label: "MEDIUM", cls: "tag tag-accent", fg: "pri-medium" },
  low: { label: "LOW", cls: "tag tag-neutral", fg: "pri-low" },
};

export function priBadge(t: TaskView): PriMeta {
  return PRI_META[t.priority] || { label: t.priority, cls: "tag tag-neutral", fg: "" };
}

export function statusMeta(t: TaskView): StatusMeta {
  return STATUS_META[t.status] || { headLabel: t.status, fg: "" };
}

export function padId(id: number): string {
  return String(id).padStart(3, "0");
}

// Where the task's file lives on disk — the archive directory once it reaches a
// terminal status, the working directory otherwise.
export function taskFilePath(t: TaskView): string {
  const dir =
    t.status === "done" || t.status === "declined"
      ? ".backlog/archive/"
      : ".backlog/tasks/";
  return dir + t.file;
}

export function splitList(s: string): string[] {
  return (s || "")
    .split(",")
    .map((x) => x.trim())
    .filter(Boolean);
}
