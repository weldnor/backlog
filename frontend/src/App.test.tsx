import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
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

const ALL: TaskView[] = [
  task({ id: 1, title: "Alpha bug", status: "doing", priority: "high", tags: ["ui"] }),
  task({ id: 2, title: "Beta chore", status: "todo", priority: "low", tags: ["docs"] }),
];

let patchStatus = 200;
let listCalls = 0;

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
  if (url.startsWith("/api/tasks/1") && method === "GET") {
    return json(ALL[0]);
  }
  if (url.startsWith("/api/tasks/1") && method === "PATCH") {
    const body = JSON.parse(String(init?.body));
    if (patchStatus !== 200) {
      return json({ error: "declining a task requires a reason" }, patchStatus);
    }
    return json({ ...ALL[0], ...body });
  }
  if (url.startsWith("/api/tasks") && method === "GET") {
    listCalls++;
    const u = new URL("http://x" + url);
    let out = ALL;
    if (!u.searchParams.has("all")) {
      out = ALL.filter((t) => t.status === "todo" || t.status === "doing");
    }
    const pri = u.searchParams.get("priority");
    if (pri) out = out.filter((t) => t.priority === pri);
    return json(out);
  }
  return json({ error: "unexpected " + method + " " + url }, 500);
}

const listView = () => document.getElementById("listView") as HTMLElement;
const boardView = () => document.getElementById("boardView") as HTMLElement;

beforeEach(() => {
  patchStatus = 200;
  listCalls = 0;
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
      screen.getByText("backlog list --priority high --json"),
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
      within(boardView()).getByText("Archive — acted on. Behind --all."),
    ).toBeInTheDocument();
    expect(
      within(boardView()).getByText(
        "Always in scope for search — a duplicate must not hide behind a filter.",
      ),
    ).toBeInTheDocument();
  });
});
