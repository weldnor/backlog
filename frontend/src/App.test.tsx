import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  act,
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
    assignee: "",
    tags: [],
    links: [],
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
let nextId = 100;
// The task set the fake server owns for a test; a PATCH/POST mutates it in
// place so a subsequent list reflects the change, the way the real server does.
let tasksState: TaskView[] = [];
// Every PATCH/POST body the fake server saw, in order — the drag and capture
// tests assert on them.
let patchBodies: { id: number; body: Record<string, unknown> }[] = [];
let postBodies: Record<string, unknown>[] = [];

function seedTasks() {
  tasksState = [
    task({ id: 1, title: "Alpha bug", status: "doing", priority: "high", tags: ["ui"] }),
    task({ id: 2, title: "Beta chore", status: "todo", priority: "low", tags: ["docs"] }),
  ];
}

function fakeFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const url = typeof input === "string" ? input : input.toString();
  const method = init?.method ?? "GET";
  // A real fetch always hands back a freshly parsed body, never the server's
  // own object; clone here too, so a test mutating tasksState after a request
  // was made can't retroactively change what an in-flight response resolves
  // to, and so React always sees a new array/object reference to diff against.
  const json = (body: unknown, status = 200) =>
    Promise.resolve({
      ok: status >= 200 && status < 300,
      status,
      statusText: "",
      json: () => Promise.resolve(JSON.parse(JSON.stringify(body))),
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
  if (url.startsWith("/api/tasks") && method === "POST") {
    const body = JSON.parse(String(init?.body));
    postBodies.push(body);
    const created = task({
      id: nextId++,
      title: body.title,
      status: "new",
      priority: body.priority || "medium",
      assignee: body.assignee || "",
      tags: body.tags || [],
      description: body.description || "",
      links: body.links || [],
    });
    tasksState.push(created);
    return json(created);
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
  await user.click(screen.getByRole("button", { name: "Board" }));
}

const card = (title: string) =>
  within(boardView()).getByText(title).closest(".card") as HTMLElement;
const column = (label: string) =>
  within(boardView()).getByText(label).closest(".board-col") as HTMLElement;

// Opens a task's detail from the list view by its title, and returns the
// resulting dialog.
async function openFromList(user: ReturnType<typeof userEvent.setup>, title: string) {
  const row = await within(listView()).findByRole("button", { name: new RegExp(title) });
  await user.click(row);
  return screen.findByRole("dialog");
}

beforeEach(() => {
  patchStatus = 200;
  listCalls = 0;
  nextId = 100;
  patchBodies = [];
  postBodies = [];
  seedTasks();
  vi.stubGlobal("fetch", vi.fn(fakeFetch));
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  try {
    window.localStorage.clear();
  } catch {
    // ignore
  }
});

describe("App", () => {
  it("renders the list with padded ids and the filter row count", async () => {
    render(<App />);
    expect(await within(listView()).findByText("Alpha bug")).toBeInTheDocument();
    expect(within(listView()).getByText("001")).toBeInTheDocument();
    expect(within(listView()).getByText("002")).toBeInTheDocument();

    expect(screen.getByText("2 of 2 tasks")).toBeInTheDocument();
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
    expect(screen.getByText("4 of 4 tasks")).toBeInTheDocument();
  });

  it("filters by free text over the loaded set with no extra request", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    const callsBefore = listCalls;

    await user.type(screen.getByPlaceholderText("Search tasks, tags, files"), "beta");

    expect(within(listView()).queryByText("Alpha bug")).not.toBeInTheDocument();
    expect(within(listView()).getByText("Beta chore")).toBeInTheDocument();
    expect(listCalls).toBe(callsBefore);
  });

  it("re-runs the filtered fetch and updates the count when a filter is toggled", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    const callsBefore = listCalls;

    await user.click(screen.getByRole("button", { name: "High" }));

    await waitFor(() => expect(listCalls).toBe(callsBefore + 1));
    await waitFor(() =>
      expect(within(listView()).queryByText("Beta chore")).not.toBeInTheDocument(),
    );
    expect(screen.getByText("1 of 2 tasks")).toBeInTheDocument();
  });

  it("clears the search text from the pill's clear control", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    const search = screen.getByPlaceholderText("Search tasks, tags, files");
    await user.type(search, "beta");
    expect(within(listView()).queryByText("Alpha bug")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear search" }));

    expect(search).toHaveValue("");
    expect(within(listView()).getByText("Alpha bug")).toBeInTheDocument();
  });

  it("resets every filter and the search text from the filter row's Reset", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    await user.click(screen.getByRole("button", { name: "High" }));
    await waitFor(() =>
      expect(within(listView()).queryByText("Beta chore")).not.toBeInTheDocument(),
    );

    await user.click(screen.getByRole("button", { name: "Reset" }));

    await waitFor(() =>
      expect(within(listView()).getByText("Beta chore")).toBeInTheDocument(),
    );
    expect(screen.getByRole("button", { name: "High" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
  });

  it("shows the empty-result notice when nothing matches and clears it from its control", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    await user.type(screen.getByPlaceholderText("Search tasks, tags, files"), "nonesuch");

    expect(
      await within(listView()).findByText("No task matches the current search and filters."),
    ).toBeInTheDocument();

    await user.click(within(listView()).getByRole("button", { name: "Clear everything" }));

    expect(await within(listView()).findByText("Alpha bug")).toBeInTheDocument();
  });

  it("switches the theme and stamps it on the document element", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    const initial = document.documentElement.getAttribute("data-theme");
    await user.click(screen.getByRole("button", { name: /Switch to (dark|light) theme/ }));

    expect(document.documentElement.getAttribute("data-theme")).not.toBe(initial);
  });

  it("remembers the chosen theme across a remount", async () => {
    const user = userEvent.setup();
    const { unmount } = render(<App />);
    await within(listView()).findByText("Alpha bug");

    const initial = document.documentElement.getAttribute("data-theme");
    await user.click(screen.getByRole("button", { name: /Switch to (dark|light) theme/ }));
    const chosen = document.documentElement.getAttribute("data-theme");
    unmount();

    render(<App />);
    await within(listView()).findByText("Alpha bug");
    expect(document.documentElement.getAttribute("data-theme")).toBe(chosen);
    expect(chosen).not.toBe(initial);
  });

  it("re-fetches tasks when the window regains focus", async () => {
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    const callsBefore = listCalls;

    tasksState.push(task({ id: 5, title: "Epsilon added", status: "todo" }));
    act(() => {
      window.dispatchEvent(new Event("focus"));
    });

    expect(await within(listView()).findByText("Epsilon added")).toBeInTheDocument();
    await waitFor(() => expect(listCalls).toBeGreaterThan(callsBefore));
  });

  it("refreshes an unfiltered view with a single list request", async () => {
    render(<App />);
    await within(listView()).findByText("Alpha bug");
    await waitFor(() => expect(listCalls).toBe(1));

    act(() => {
      window.dispatchEvent(new Event("focus"));
    });
    await waitFor(() => expect(listCalls).toBe(2));
    await new Promise((r) => setTimeout(r, 20));
    expect(listCalls).toBe(2);
  });

  it("shows a load failure instead of an empty result, and Retry reloads", async () => {
    const user = userEvent.setup();
    let fail = true;
    vi.stubGlobal(
      "fetch",
      vi.fn((input: RequestInfo | URL, init?: RequestInit) =>
        fail && String(input).startsWith("/api/tasks")
          ? Promise.resolve(
              new Response(JSON.stringify({ error: "backlog is unreadable" }), {
                status: 500,
                headers: { "Content-Type": "application/json" },
              }),
            )
          : fakeFetch(input, init),
      ),
    );
    render(<App />);

    expect(await screen.findByRole("alert")).toHaveTextContent("backlog is unreadable");
    expect(screen.queryByText(/Clear everything/)).not.toBeInTheDocument();

    fail = false;
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(await within(listView()).findByText("Alpha bug")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("opens a task, closes on Escape without saving, and restores focus to the row", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    const row = within(listView()).getByRole("button", { name: /Alpha bug/ });
    await user.click(row);

    const dialog = await screen.findByRole("dialog", { name: "Alpha bug" });
    expect(dialog).toContainElement(document.activeElement as HTMLElement | null);
    expect(within(dialog).getByRole("button", { name: "Title" })).toHaveTextContent("Alpha bug");

    await user.keyboard("{Escape}");

    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    expect(document.activeElement).toBe(row);
  });

  it("keeps focus inside the dialog when Tab would leave it", async () => {
    const user = userEvent.setup();
    render(<App />);
    await openFromList(user, "Alpha bug");

    const dialog = screen.getByRole("dialog");
    const focusables = dialog.querySelectorAll<HTMLElement>(
      "button, a[href], input, textarea",
    );
    focusables[focusables.length - 1].focus();

    await user.tab();

    expect(dialog).toContainElement(document.activeElement as HTMLElement | null);
  });

  it("sends only the assignee field when it is edited in place", async () => {
    const user = userEvent.setup();
    render(<App />);
    const dialog = await openFromList(user, "Alpha bug");

    await user.click(within(dialog).getByRole("button", { name: "Assignee" }));
    await user.type(within(dialog).getByLabelText("Assignee"), "weldnor{Enter}");

    await waitFor(() =>
      expect(patchBodies).toContainEqual({ id: 1, body: { assignee: "weldnor" } }),
    );
    expect(await within(dialog).findByRole("button", { name: "Assignee" })).toHaveTextContent(
      "weldnor",
    );
  });

  it("cancels an in-place edit with Escape without closing the dialog", async () => {
    const user = userEvent.setup();
    render(<App />);
    const dialog = await openFromList(user, "Alpha bug");

    await user.click(within(dialog).getByRole("button", { name: "Title" }));
    await user.type(within(dialog).getByLabelText("Title"), " changed{Escape}");

    expect(patchBodies).toEqual([]);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Title" })).toHaveTextContent("Alpha bug");
  });

  it("surfaces a server validation error in the dialog without closing it", async () => {
    const user = userEvent.setup();
    render(<App />);
    const dialog = await openFromList(user, "Alpha bug");

    patchStatus = 400;
    await user.click(within(dialog).getByRole("button", { name: "Assignee" }));
    await user.type(within(dialog).getByLabelText("Assignee"), "weldnor{Enter}");

    expect(
      await within(dialog).findByText(/declining a task requires a reason/),
    ).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("deletes a task from its detail after confirming, closing the dialog", async () => {
    const user = userEvent.setup();
    render(<App />);
    const dialog = await openFromList(user, "Alpha bug");

    await user.click(within(dialog).getByRole("button", { name: "Delete" }));

    const confirmDialog = await screen.findByRole("dialog", { name: "001 · Alpha bug" });
    await user.click(within(confirmDialog).getByRole("button", { name: "Delete" }));

    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(within(listView()).queryByText("Alpha bug")).not.toBeInTheDocument(),
    );
    expect(within(listView()).getByText("Beta chore")).toBeInTheDocument();
  });

  it("leaves the task and dialog in place when the delete confirmation is cancelled", async () => {
    const user = userEvent.setup();
    render(<App />);
    const dialog = await openFromList(user, "Alpha bug");

    await user.click(within(dialog).getByRole("button", { name: "Delete" }));
    const confirmDialog = await screen.findByRole("dialog", { name: "001 · Alpha bug" });
    await user.click(within(confirmDialog).getByRole("button", { name: "Cancel" }));

    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "001 · Alpha bug" })).not.toBeInTheDocument(),
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(within(listView()).getByText("Alpha bug")).toBeInTheDocument();
    expect(patchBodies).toEqual([]);
  });

  it("shows a task's links and jumps to the target when its title is clicked", async () => {
    tasksState[0].links = [{ type: "blocks", id: 2 }];
    const user = userEvent.setup();
    render(<App />);

    const dialog = await openFromList(user, "Alpha bug");
    const linkedTitle = within(dialog).getByRole("button", { name: "Beta chore" });
    expect(within(dialog).getByText("blocks")).toBeInTheDocument();

    await user.click(linkedTitle);

    expect(
      within(await screen.findByRole("dialog")).getByRole("button", { name: "Title" }),
    ).toHaveTextContent("Beta chore");
  });

  it("renders the board with a card per column and verbatim empty notes", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    await user.click(screen.getByRole("button", { name: "Board" }));

    // Alpha bug (doing) sits in its column; the terminal columns show their
    // verbatim empty notes.
    const doingCol = within(boardView()).getByText("Doing").closest(".board-col") as HTMLElement;
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
    fireEvent.dragOver(column("Doing"), { dataTransfer: dt });
    fireEvent.drop(column("Doing"), { dataTransfer: dt });

    await waitFor(() => expect(patchBodies).toEqual([{ id: 2, body: { status: "doing" } }]));
    await waitFor(() =>
      expect(within(column("Doing")).getByText("Beta chore")).toBeInTheDocument(),
    );
  });

  it("asks for a reason in an in-app dialog when a card is dropped on the declined column", async () => {
    const user = userEvent.setup();
    render(<App />);
    await showBoard(user);

    const dt = dataTransferStub();
    fireEvent.dragStart(card("Alpha bug"), { dataTransfer: dt });
    fireEvent.drop(column("Declined"), { dataTransfer: dt });

    const dialog = await screen.findByRole("dialog", { name: "Decline this task?" });
    await user.type(within(dialog).getByLabelText("Reason"), "out of scope");
    await user.click(within(dialog).getByRole("button", { name: "Decline" }));

    await waitFor(() =>
      expect(patchBodies).toEqual([
        { id: 1, body: { status: "declined", reason: "out of scope" } },
      ]),
    );
  });

  it("makes no request when the decline dialog is cancelled", async () => {
    const user = userEvent.setup();
    render(<App />);
    await showBoard(user);

    const dt = dataTransferStub();
    fireEvent.dragStart(card("Alpha bug"), { dataTransfer: dt });
    fireEvent.drop(column("Declined"), { dataTransfer: dt });

    const dialog = await screen.findByRole("dialog", { name: "Decline this task?" });
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Decline this task?" })).not.toBeInTheDocument(),
    );
    expect(patchBodies).toEqual([]);
  });

  it("makes no request when a card is dropped on its own column", async () => {
    const user = userEvent.setup();
    render(<App />);
    await showBoard(user);

    const dt = dataTransferStub();
    fireEvent.dragStart(card("Alpha bug"), { dataTransfer: dt });
    fireEvent.drop(column("Doing"), { dataTransfer: dt });

    await new Promise((r) => setTimeout(r, 0));
    expect(patchBodies).toEqual([]);
  });

  it("remembers a collapsed column across a remount", async () => {
    const user = userEvent.setup();
    const { unmount } = render(<App />);
    await showBoard(user);

    await user.click(within(column("Done")).getByRole("button", { name: "Done column menu" }));
    await user.click(screen.getByRole("menuitem", { name: "Collapse list" }));
    expect(within(boardView()).getByLabelText("Done, collapsed")).toBeInTheDocument();
    unmount();

    render(<App />);
    await showBoard(user);

    expect(within(boardView()).getByLabelText("Done, collapsed")).toBeInTheDocument();
  });

  it("creates a task from a column's draft, sending a create then a status PATCH", async () => {
    const user = userEvent.setup();
    render(<App />);
    await showBoard(user);

    await user.click(within(column("Todo")).getByRole("button", { name: "+ Add new task" }));
    await user.type(
      within(column("Todo")).getByPlaceholderText(/Title, #tag/),
      "#docs Ship the release notes",
    );
    await user.keyboard("{Enter}");

    await waitFor(() => expect(postBodies).toHaveLength(1));
    expect(postBodies[0]).toMatchObject({ title: "Ship the release notes", tags: ["docs"] });
    await waitFor(() =>
      expect(patchBodies.some((p) => p.body.status === "todo")).toBe(true),
    );
    expect(
      await within(column("Todo")).findByText("Ship the release notes"),
    ).toBeInTheDocument();
  });

  it("rejects an empty title from a draft without creating anything", async () => {
    const user = userEvent.setup();
    render(<App />);
    await showBoard(user);

    await user.click(within(column("New")).getByRole("button", { name: "+ Add new task" }));
    await user.keyboard("{Enter}");

    expect(within(column("New")).getByText("A title is required.")).toBeInTheDocument();
    expect(postBodies).toEqual([]);
  });

  it("shows a duplicate hint when the draft's title matches an existing task", async () => {
    const user = userEvent.setup();
    render(<App />);
    await showBoard(user);

    await user.click(within(column("New")).getByRole("button", { name: "+ Add new task" }));
    await user.type(within(column("New")).getByPlaceholderText(/Title, #tag/), "Alpha bug{Enter}");

    expect(
      await within(column("New")).findByText(/Looks like an existing task/),
    ).toBeInTheDocument();
    expect(postBodies).toEqual([]);

    await user.click(within(column("New")).getByRole("button", { name: "Create anyway" }));

    await waitFor(() => expect(postBodies).toHaveLength(1));
  });

  it("undoes a creation from the toast, deleting the task", async () => {
    const user = userEvent.setup();
    render(<App />);
    await showBoard(user);

    await user.click(within(column("New")).getByRole("button", { name: "+ Add new task" }));
    await user.type(
      within(column("New")).getByPlaceholderText(/Title, #tag/),
      "Add fuzzy search{Enter}",
    );
    const createdId = await waitFor(() => {
      expect(postBodies).toHaveLength(1);
      return tasksState.find((t) => t.title === "Add fuzzy search")!.id;
    });

    const toast = await screen.findByRole("status");
    await user.click(within(toast).getByRole("button", { name: "Undo" }));

    await waitFor(() => expect(tasksState.some((t) => t.id === createdId)).toBe(false));
    expect(
      within(boardView()).queryByText("Add fuzzy search"),
    ).not.toBeInTheDocument();
  });

  it("opens a capture draft in the first column when 'n' is pressed on the board", async () => {
    const user = userEvent.setup();
    render(<App />);
    await showBoard(user);

    await user.keyboard("n");

    expect(
      within(column("New")).getByPlaceholderText(/Title, #tag/),
    ).toBeInTheDocument();
  });

  it("opens the full form when 'n' is pressed in list view", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    await user.keyboard("n");

    expect(await screen.findByRole("dialog")).toBeInTheDocument();
  });

  it("does nothing extra when 'n' is typed into the search field", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    await user.type(screen.getByPlaceholderText("Search tasks, tags, files"), "n");

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("opens capture from the floating button too", async () => {
    const user = userEvent.setup();
    render(<App />);
    await within(listView()).findByText("Alpha bug");

    await user.click(screen.getByRole("button", { name: "Add new task" }));

    expect(await screen.findByRole("dialog")).toBeInTheDocument();
  });
});
