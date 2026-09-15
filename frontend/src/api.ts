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

// LinkView is a typed reference to another task in the same backlog — see
// internal/task's Link. Recorded on one side only: this task's list, never
// mirrored automatically onto the target's.
export interface LinkView {
  type: string;
  id: number;
}

export interface TaskView {
  id: number;
  title: string;
  status: string;
  priority: string;
  reason: string;
  assignee: string;
  tags: string[];
  links: LinkView[];
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
  assignee: string;
  files: string[];
  refs: string[];
  links: LinkView[];
}

export interface PatchTaskBody {
  title?: string;
  description?: string;
  tags?: string[];
  priority?: string;
  status?: string;
  reason?: string;
  assignee?: string;
  refs?: string[];
  links?: LinkView[];
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

// CreateStatusError reports a create whose follow-up status change failed: the
// task exists, still in `new`, and `task` carries it so the caller can refresh
// and still name the task it created (design.md D8).
export class CreateStatusError extends ApiError {
  task: TaskView;
  constructor(message: string, status: number, task: TaskView) {
    super(message, status);
    this.name = "CreateStatusError";
    this.task = task;
  }
}

// createTaskWithStatus creates a task in the given status. POST cannot set a
// status, so anything other than `new` is applied afterwards with a PATCH
// {status} — the same request an edit of the status sends.
export async function createTaskWithStatus(
  body: CreateTaskBody,
  status: string,
): Promise<TaskView> {
  const created = await createTask(body);
  if (status === "new") return created;
  try {
    return await patchTask(created.id, { status });
  } catch (err) {
    throw new CreateStatusError(
      err instanceof Error ? err.message : String(err),
      err instanceof ApiError ? err.status : 0,
      created,
    );
  }
}

export function patchTask(id: number, body: PatchTaskBody): Promise<TaskView> {
  return api<TaskView>("/api/tasks/" + id, jsonBody("PATCH", body));
}

export function deleteTask(id: number): Promise<TaskView> {
  return api<TaskView>("/api/tasks/" + id, { method: "DELETE" });
}

export function getRepo(): Promise<RepoInfo> {
  return api<RepoInfo>("/api/repo");
}
