import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { App } from "./App";
import type { TaskView } from "./api";

function task(over: Partial<TaskView> & { id: number; title: string }): TaskView {
  return {
    status: "todo",
    priority: "medium",
    reason: "",
    tags: [],
    description: "",
    file: `00${over.id}-x.md`,
    metadata: {
      schema: 1,
      created: "2026-01-01T00:00:00Z",
      author: "human",
      source: { files: [], branch: "", commit: "" },
      refs: [],
    },
    ...over,
  };
}

let patchStatus = 200;
let listCalls = 0;
// The task set the fake server owns for a test; a PATCH mutates it in place so a
// subsequent list reflects the change, the way the real server does.
let tasksState: TaskView[] = [];
// Every PATCH body the fake server saw, in order — the drag tests assert on it.
let patchBodies: { id: number; body: Record<string, unknown> }[] = [];

function seedTasks() {
  tasksState = [
    task({ id: 1, title: "Alpha bug", status: "doing", priority: "high", tags: ["ui"] }),
    task({ id: 2, title: "Beta chore", status: "todo", priority: "low", tags: ["docs"] }),
  ];
}

function fakeFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const url = typeof input === "string" ? input : input.toString();
  const method = init?.method ?? "GET";
  const json = (body: unknown, status = 200) =>
    Promise.resolve({
      ok: status >= 200 && status < 300,
      status,
      statusText: "",
      json: () => Promise.resolve(body),
    } as Response);

  if (url.startsWith("/api/repo")) {
    return json({ name: "demo", branch: "main", version: "9.9" });
  }
  const idMatch = url.match(/^\/api\/tasks\/(\d+)/);
  if (idMatch) {
    const id = Number(idMatch[1]);
    const t = tasksState.find((x) => x.id === id);
    if (method === "GET") {
      return t ? json(t) : json({ error: "not found" }, 404);
    }
    if (method === "PATCH") {
      const body = JSON.parse(String(init?.body));
      patchBodies.push({ id, body });
      if (patchStatus !== 200) {
        return json({ error: "declining a task requires a reason" }, patchStatus);
      }
      if (t) Object.assign(t, body);
      return json(t);
    }
    if (method === "DELETE") {
      if (!t) return json({ error: "not found" }, 404);
      tasksState = tasksState.filter((x) => x.id !== id);
      return json(t);
    }
  }
  if (url.startsWith("/api/tasks") && method === "GET") {
    listCalls++;
    const u = new URL("http://x" + url);
    let out = tasksState;
    if (!u.searchParams.has("all")) {
      out = tasksState.filter(
        (t) => t.status === "new" || t.status === "todo" || t.status === "doing",
      );
    }
    const pri = u.searchParams.get("priority");
    if (pri) out = out.filter((t) => t.priority === pri);
    return json(out);
  }
  return json({ error: "unexpected " + method + " " + url }, 500);
}

const listView = () => document.getElementById("listView") as HTMLElement;
const boardView = () => document.getElementById("boardView") as HTMLElement;

// jsdom has no real drag, so a shared object stands in for the drag's
// DataTransfer: the card writes the id on dragStart, the column reads it on drop.
function dataTransferStub(): DataTransfer {
  const store: Record<string, string> = {};
  return {
    setData: (type: string, value: string) => {
      store[type] = value;
    },
    getData: (type: string) => store[type] ?? "",
    effectAllowed: "",
    dropEffect: "",
  } as unknown as DataTransfer;
}

async function showBoard(user: ReturnType<typeof userEvent.setup>) {
  await within(listView()).findByText("Alpha bug");
  await user.click(screen.getByLabelText("BOARD"));
}

const card = (title: string) =>
  within(boardView()).getByText(title).closest(".board-card") as HTMLElement;
const column = (label: string) =>
  within(boardView()).getByText(label).closest(".board-col") as HTMLElement;

beforeEach(() => {
  patchStatus = 200;
  listCalls = 0;
  patchBodies = [];
  seedTasks();
  vi.stubGlobal("fetch", vi.fn(fakeFetch));
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("App", () => {
  it("renders the list with padded ids and sidebar counts", async () => {
    render(<App />);
    expect(await within(listView()).findByText("Alpha bug")).toBeInTheDocument();
    expect(within(listView()).getByText("001")).toBeInTheDocument();
    expect(within(listView()).getByText("002")).toBeInTheDocument();

    // Sidebar status counts come from the ?all=1 set.
    const sidebar = document.querySelector(".sidebar") as HTMLElement;
    const doing = within(sidebar).getByRole("button", { name: /doing/i });
    expect(within(doing).getByText("1")).toBeInTheDocument();
  });

  it("shows done and declined tasks by default without an archive toggle", async () => {
    tasksState.push(
      task({ id: 3, title: "Gamma shipped", status: "done" }),
      task({ id: 4, title: "Delta dropped", status: "declined", reason: "dup" }),
    );
    render(<App />);

    expect(await within(listView()).findByText("Gamma shipped")).toBeInTheDocument();
    expect(within(listView()).getByText("Delta dropped")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /archive/i }),
    ).not.toBeInTheDocument();
    expect(screen.getByText("backlog list --all --json")).toBeInTheDocument();
  });

  it("filters by free text over the loaded set with no extra request", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    const callsBefore = listCalls;

    await user.type(screen.getByPlaceholderText("title, description, tags"), "beta");

    expect(within(listView()).queryByText("Alpha bug")).not.toBeInTheDocument();
    expect(within(listView()).getByText("Beta chore")).toBeInTheDocument();
    expect(listCalls).toBe(callsBefore);
  });

  it("re-runs the filtered fetch and updates the command when a filter is toggled", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    const callsBefore = listCalls;

    await user.click(screen.getByRole("button", { name: /^high/i }));

    await waitFor(() => expect(listCalls).toBe(callsBefore + 1));
    expect(
      screen.getByText("backlog list --all --priority high --json"),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(within(listView()).queryByText("Beta chore")).not.toBeInTheDocument(),
    );
  });

  it("opens a task, closes on Escape without saving, and restores focus to the row", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    const row = within(listView()).getByText("Alpha bug").closest("tr") as HTMLTableRowElement;
    await user.click(row);

    const dialog = await screen.findByRole("dialog");
    expect(dialog).toContainElement(document.activeElement as HTMLElement | null);
    expect(within(dialog).getByRole("heading", { name: "Alpha bug" })).toBeInTheDocument();

    await user.keyboard("{Escape}");

    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    expect(document.activeElement).toBe(row);
  });

  it("keeps focus inside the dialog when Tab would leave it", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    await user.click(within(listView()).getByText("Alpha bug").closest("tr") as HTMLTableRowElement);

    const dialog = await screen.findByRole("dialog");
    const focusables = dialog.querySelectorAll<HTMLElement>(
      "button, a[href], input, textarea",
    );
    focusables[focusables.length - 1].focus();

    await user.tab();

    expect(dialog).toContainElement(document.activeElement as HTMLElement | null);
  });

  it("surfaces a server validation error in the dialog without closing it", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    await user.click(within(listView()).getByText("Alpha bug").closest("tr") as HTMLTableRowElement);
    await screen.findByRole("dialog");

    await user.click(screen.getByRole("button", { name: /^EDIT/ }));
    patchStatus = 400;
    await user.click(screen.getByRole("button", { name: /^Save/ }));

    expect(
      await screen.findByText(/declining a task requires a reason/),
    ).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("deletes a task from the read view after confirming, closing the dialog", async () => {
    const user = userEvent.setup();
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(true);
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    await user.click(within(listView()).getByText("Alpha bug").closest("tr") as HTMLTableRowElement);
    await screen.findByRole("dialog");

    await user.click(screen.getByRole("button", { name: /^Delete/ }));

    expect(confirmSpy).toHaveBeenCalledWith(
      'Delete task #1 "Alpha bug"? This cannot be undone.',
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(within(listView()).queryByText("Alpha bug")).not.toBeInTheDocument(),
    );
    expect(within(listView()).getByText("Beta chore")).toBeInTheDocument();
  });

  it("leaves the task and dialog in place when the delete confirm is cancelled", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "confirm").mockReturnValue(false);
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    await user.click(within(listView()).getByText("Alpha bug").closest("tr") as HTMLTableRowElement);
    await screen.findByRole("dialog");

    await user.click(screen.getByRole("button", { name: /^Delete/ }));

    await new Promise((r) => setTimeout(r, 0));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(within(listView()).getByText("Alpha bug")).toBeInTheDocument();
  });

  it("offers a Delete action in both the read and the edit view", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    await user.click(within(listView()).getByText("Alpha bug").closest("tr") as HTMLTableRowElement);
    await screen.findByRole("dialog");

    expect(screen.getByRole("button", { name: /^Delete/ })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /^EDIT/ }));
    expect(screen.getByRole("button", { name: /^Delete/ })).toBeInTheDocument();
  });

  it("renders the board with a card per column and verbatim empty notes", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    await user.click(screen.getByLabelText("BOARD"));

    // Alpha bug (doing) sits in its column; the terminal columns show their
    // verbatim empty notes.
    const doingCol = within(boardView()).getByText("DOING").closest(".board-col") as HTMLElement;
    expect(within(doingCol).getByText("Alpha bug")).toBeInTheDocument();
    expect(
      within(boardView()).getByText(
        "Archive — acted on and moved out of the working set.",
      ),
    ).toBeInTheDocument();
    expect(
      within(boardView()).getByText(
        "Always in scope for search — a duplicate must not hide behind a filter.",
      ),
    ).toBeInTheDocument();
  });

  it("moves a task to another column when its card is dropped there", async () => {
    const user = userEvent.setup();
    render(<App />);
    await showBoard(user);

    const dt = dataTransferStub();
    fireEvent.dragStart(card("Beta chore"), { dataTransfer: dt });
    fireEvent.dragOver(column("DOING"), { dataTransfer: dt });
    fireEvent.drop(column("DOING"), { dataTransfer: dt });

    await waitFor(() => expect(patchBodies).toEqual([{ id: 2, body: { status: "doing" } }]));
    await waitFor(() =>
      expect(within(column("DOING")).getByText("Beta chore")).toBeInTheDocument(),
    );
  });

  it("prompts for a reason when a card is dropped on the declined column", async () => {
    const user = userEvent.setup();
    const promptSpy = vi.spyOn(window, "prompt").mockReturnValue("out of scope");
    render(<App />);
    await showBoard(user);

    const dt = dataTransferStub();
    fireEvent.dragStart(card("Alpha bug"), { dataTransfer: dt });
    fireEvent.drop(column("DECLINED"), { dataTransfer: dt });

    await waitFor(() =>
      expect(patchBodies).toEqual([
        { id: 1, body: { status: "declined", reason: "out of scope" } },
      ]),
    );
    expect(promptSpy).toHaveBeenCalledTimes(1);
  });

  it("makes no request when the decline prompt is cancelled", async () => {
    const user = userEvent.setup();
    vi.spyOn(window, "prompt").mockReturnValue(null);
    render(<App />);
    await showBoard(user);

    const dt = dataTransferStub();
    fireEvent.dragStart(card("Alpha bug"), { dataTransfer: dt });
    fireEvent.drop(column("DECLINED"), { dataTransfer: dt });

    await new Promise((r) => setTimeout(r, 0));
    expect(patchBodies).toEqual([]);
  });

  it("makes no request when a card is dropped on its own column", async () => {
    const user = userEvent.setup();
    render(<App />);
    await showBoard(user);

    const dt = dataTransferStub();
    fireEvent.dragStart(card("Alpha bug"), { dataTransfer: dt });
    fireEvent.drop(column("DOING"), { dataTransfer: dt });

    await new Promise((r) => setTimeout(r, 0));
    expect(patchBodies).toEqual([]);
  });
});
