// Typed client for `backlog browse`'s HTTP API. The interfaces below mirror
// internal/taskview/taskview.go field-for-field (snake_case JSON keys); if the
// Go shape changes, this file has to change with it. Error handling matches
// the old vanilla api() helper: parse the JSON body, and on a non-2xx status
// throw an Error carrying body.error (or the status text) with the numeric
// status attached.

export interface SourceView {
  files: string[];
  branch: string;
  commit: string;
}

export interface MetaView {
  schema: number;
  created: string;
  author: string;
  source: SourceView;
  refs: string[];
}

export interface TaskView {
  id: number;
  title: string;
  status: string;
  priority: string;
  reason: string;
  tags: string[];
  description: string;
  metadata: MetaView;
  file: string;
}

export interface RepoInfo {
  name: string;
  branch: string;
  version: string;
}

export interface CreateTaskBody {
  title: string;
  description: string;
  tags: string[];
  priority: string;
  files: string[];
  refs: string[];
}

export interface PatchTaskBody {
  title?: string;
  description?: string;
  tags?: string[];
  priority?: string;
  status?: string;
  reason?: string;
  refs?: string[];
}

export class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function api<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(path, opts);
  let body: unknown = {};
  try {
    body = await res.json();
  } catch {
    body = {};
  }
  if (!res.ok) {
    const message =
      (body && typeof body === "object" && "error" in body && typeof body.error === "string"
        ? body.error
        : "") || res.statusText;
    throw new ApiError(message, res.status);
  }
  return body as T;
}

function jsonBody(method: string, body: unknown): RequestInit {
  return {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

export function listTasks(params: Record<string, string> = {}): Promise<TaskView[]> {
  const qs = new URLSearchParams(params).toString();
  return api<TaskView[]>("/api/tasks" + (qs ? "?" + qs : ""));
}

export function getTask(id: number): Promise<TaskView> {
  return api<TaskView>("/api/tasks/" + id);
}

export function createTask(body: CreateTaskBody): Promise<TaskView> {
  return api<TaskView>("/api/tasks", jsonBody("POST", body));
}

export function patchTask(id: number, body: PatchTaskBody): Promise<TaskView> {
  return api<TaskView>("/api/tasks/" + id, jsonBody("PATCH", body));
}

export function getRepo(): Promise<RepoInfo> {
  return api<RepoInfo>("/api/repo");
}
