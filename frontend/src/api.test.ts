import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CreateStatusError, createTaskWithStatus, type CreateTaskBody, type TaskView } from "./api";

const body: CreateTaskBody = {
  title: "Drop zone highlight",
  description: "",
  tags: ["ui"],
  priority: "high",
  assignee: "ann",
  files: [],
  refs: [],
  links: [],
};

let calls: { method: string; url: string; body: unknown }[] = [];
let patchStatus = 200;

function fakeFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const url = String(input);
  const method = init?.method ?? "GET";
  const sent = init?.body ? JSON.parse(String(init.body)) : undefined;
  calls.push({ method, url, body: sent });
  const created = { id: 7, status: "new", ...body } as unknown as TaskView;
  const respond = (payload: unknown, status = 200) =>
    Promise.resolve({
      ok: status < 300,
      status,
      statusText: "",
      json: () => Promise.resolve(payload),
    } as Response);

  if (method === "POST") return respond(created);
  if (method === "PATCH") {
    return patchStatus === 200
      ? respond({ ...created, ...sent })
      : respond({ error: "hook refused" }, patchStatus);
  }
  return respond({ error: "unexpected" }, 500);
}

beforeEach(() => {
  calls = [];
  patchStatus = 200;
  vi.stubGlobal("fetch", vi.fn(fakeFetch));
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("createTaskWithStatus", () => {
  it("posts, then patches only the status for a non-new status", async () => {
    const task = await createTaskWithStatus(body, "todo");

    expect(calls).toEqual([
      { method: "POST", url: "/api/tasks", body },
      { method: "PATCH", url: "/api/tasks/7", body: { status: "todo" } },
    ]);
    expect(task.status).toBe("todo");
  });

  it("only posts when the status is new", async () => {
    const task = await createTaskWithStatus(body, "new");

    expect(calls).toEqual([{ method: "POST", url: "/api/tasks", body }]);
    expect(task.status).toBe("new");
  });

  it("reports the created task when the status change fails", async () => {
    patchStatus = 409;

    const err = await createTaskWithStatus(body, "doing").catch((e) => e);

    expect(err).toBeInstanceOf(CreateStatusError);
    expect(err.message).toBe("hook refused");
    expect(err.status).toBe(409);
    expect(err.task.id).toBe(7);
  });
});
