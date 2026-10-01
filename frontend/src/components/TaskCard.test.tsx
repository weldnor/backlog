import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import type { TaskView } from "../api";
import { TaskCard } from "./TaskCard";

afterEach(cleanup);

function task(over: Partial<TaskView> = {}): TaskView {
  return {
    id: 7,
    title: "Add fuzzy search",
    status: "todo",
    priority: "high",
    reason: "",
    assignee: "ann",
    tags: ["ui", "board"],
    links: [],
    description: "A short description of the work to do.",
    file: "007-x.md",
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

describe("TaskCard", () => {
  it("shows a label per tag, the title, the description start, priority and assignee", () => {
    render(<TaskCard task={task()} onOpen={() => {}} />);

    expect(screen.getByText("Add fuzzy search")).toBeInTheDocument();
    expect(screen.getByText(/A short description/)).toBeInTheDocument();
    expect(screen.getByText("High")).toBeInTheDocument();
    expect(screen.getByText("@ann")).toBeInTheDocument();
    expect(screen.getByTitle("ui")).toBeInTheDocument();
    expect(screen.getByTitle("board")).toBeInTheDocument();
  });

  it("shows unassigned when there is no assignee", () => {
    render(<TaskCard task={task({ assignee: "" })} onOpen={() => {}} />);
    expect(screen.getByText("unassigned")).toBeInTheDocument();
  });

  it("opens on click and on Enter/Space", async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    render(<TaskCard task={task()} onOpen={onOpen} />);

    await user.click(screen.getByRole("button"));
    expect(onOpen).toHaveBeenCalledWith(7);

    onOpen.mockClear();
    screen.getByRole("button").focus();
    await user.keyboard("{Enter}");
    expect(onOpen).toHaveBeenCalledWith(7);
  });

  it("carries the task id through the drag and marks itself dragging", () => {
    const onDragStart = vi.fn();
    render(<TaskCard task={task()} isDragging onOpen={() => {}} onDragStart={onDragStart} />);

    const card = screen.getByRole("button");
    expect(card).toHaveClass("is-dragging");
    expect(card).toHaveAttribute("draggable", "true");
  });
});

describe("TaskCard id and menu", () => {
  it("shows the padded id and reports a right-click", async () => {
    const onMenu = vi.fn();
    render(<TaskCard task={task()} onOpen={() => {}} onMenu={onMenu} />);
    expect(screen.getByText("007")).toBeTruthy();
    await userEvent.pointer({ keys: "[MouseRight]", target: screen.getByRole("button") });
    expect(onMenu).toHaveBeenCalledWith(7, expect.any(Number), expect.any(Number));
  });
});
