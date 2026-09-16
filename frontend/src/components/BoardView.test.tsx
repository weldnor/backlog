import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import type { TaskView } from "../api";
import { BoardView } from "./BoardView";

function task(over: Partial<TaskView> & { id: number; title: string; status: string }): TaskView {
  return {
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

const tasks = [
  task({ id: 1, title: "Alpha bug", status: "doing", priority: "high", tags: ["ui"], assignee: "ann" }),
  task({ id: 2, title: "Beta chore", status: "todo" }),
];

const column = (label: string) =>
  screen.getByText(label).closest(".board-col") as HTMLElement;

beforeEach(() => {
  window.localStorage.clear();
});

afterEach(cleanup);

describe("BoardView", () => {
  it("collapses a column from its menu and remembers it across a remount", async () => {
    const user = userEvent.setup();
    const { unmount } = render(<BoardView tasks={tasks} onOpen={() => {}} onMove={() => {}} all={tasks} onCreated={() => {}} onOpenFullForm={() => {}} showToast={() => {}} />);

    await user.click(within(column("Doing")).getByRole("button", { name: "Doing column menu" }));
    await user.click(screen.getByRole("menuitem", { name: "Collapse list" }));

    expect(screen.getByRole("button", { name: "Expand Doing list" })).toBeInTheDocument();
    expect(screen.queryByText("Alpha bug")).not.toBeInTheDocument();

    unmount();
    render(<BoardView tasks={tasks} onOpen={() => {}} onMove={() => {}} all={tasks} onCreated={() => {}} onOpenFullForm={() => {}} showToast={() => {}} />);

    expect(screen.getByRole("button", { name: "Expand Doing list" })).toBeInTheDocument();
  });

  it("still accepts a dropped card while collapsed", async () => {
    const user = userEvent.setup();
    const onMove = vi.fn();
    render(<BoardView tasks={tasks} onOpen={() => {}} onMove={onMove} all={tasks} onCreated={() => {}} onOpenFullForm={() => {}} showToast={() => {}} />);

    await user.click(within(column("Doing")).getByRole("button", { name: "Doing column menu" }));
    await user.click(screen.getByRole("menuitem", { name: "Collapse list" }));

    const rail = screen.getByRole("button", { name: "Expand Doing list" }).closest(".board-col") as HTMLElement;
    const dt = dataTransferStub();
    dt.setData("text/plain", "2");
    fireEvent.dragOver(rail, { dataTransfer: dt });
    fireEvent.drop(rail, { dataTransfer: dt });

    expect(onMove).toHaveBeenCalledWith(2, "doing");
  });

  it("copies the column's visible tasks as a markdown list", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn();
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });
    render(<BoardView tasks={tasks} onOpen={() => {}} onMove={() => {}} all={tasks} onCreated={() => {}} onOpenFullForm={() => {}} showToast={() => {}} />);

    await user.click(within(column("Doing")).getByRole("button", { name: "Doing column menu" }));
    await user.click(screen.getByRole("menuitem", { name: "Copy list as markdown" }));

    expect(writeText).toHaveBeenCalledWith("- [001] Alpha bug (high) #ui @ann");
  });
});
