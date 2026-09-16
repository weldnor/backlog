import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import type { TaskView } from "../api";
import { TaskForm } from "./TaskForm";

const { createTaskWithStatus, deleteTask } = vi.hoisted(() => ({
  createTaskWithStatus: vi.fn(),
  deleteTask: vi.fn(),
}));

vi.mock("../api", async () => {
  const actual = await vi.importActual<typeof import("../api")>("../api");
  return { ...actual, createTaskWithStatus, deleteTask };
});

function task(over: Partial<TaskView> & { id: number; title: string }): TaskView {
  return {
    status: "new",
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

const tasks = [task({ id: 11, title: "Existing task", tags: ["ui", "urgent"] })];

function setup(prefill?: Partial<import("../tokens").ParsedDraft> | null) {
  const onClose = vi.fn();
  const onCreated = vi.fn();
  const onOpen = vi.fn();
  const showToast = vi.fn();
  render(
    <TaskForm
      tasks={tasks}
      prefill={prefill}
      onClose={onClose}
      onCreated={onCreated}
      onOpen={onOpen}
      showToast={showToast}
    />,
  );
  return { onClose, onCreated, onOpen, showToast };
}

beforeEach(() => {
  createTaskWithStatus.mockReset();
  deleteTask.mockReset();
});

afterEach(cleanup);

describe("TaskForm", () => {
  it("sends every field on a full-context create and closes on success", async () => {
    const user = userEvent.setup();
    createTaskWithStatus.mockResolvedValue(task({ id: 60, title: "Full context", status: "todo" }));
    const { onClose, onCreated, showToast } = setup();

    await user.type(screen.getByPlaceholderText("Task title"), "Full context");
    await user.click(screen.getByRole("button", { name: "Todo" }));
    await user.click(screen.getByRole("button", { name: "High" }));
    await user.type(screen.getByLabelText("Assignee"), "ann");
    await user.type(screen.getByPlaceholderText("+ tag"), "ui{Enter}");
    await user.type(screen.getByLabelText("Description"), "the body");
    await user.click(screen.getByRole("button", { name: /More/ }));
    await user.type(screen.getByLabelText("Source files"), "a.go, b.go");
    await user.type(screen.getByLabelText("References"), "PR-1");

    await user.click(screen.getByRole("button", { name: "Create task" }));

    expect(createTaskWithStatus).toHaveBeenCalledWith(
      {
        title: "Full context",
        description: "the body",
        tags: ["ui"],
        priority: "high",
        assignee: "ann",
        files: ["a.go", "b.go"],
        refs: ["PR-1"],
        links: [],
      },
      "todo",
    );
    expect(onCreated).toHaveBeenCalled();
    expect(showToast).toHaveBeenCalledWith(
      expect.objectContaining({ text: "Task 060 created in Todo" }),
    );
    expect(onClose).toHaveBeenCalled();
  });

  it("keeps the form open and empty after Create and add another", async () => {
    const user = userEvent.setup();
    createTaskWithStatus.mockResolvedValue(task({ id: 61, title: "First one" }));
    const { onClose } = setup();

    await user.type(screen.getByPlaceholderText("Task title"), "First one");
    await user.click(screen.getByRole("button", { name: "Create and add another" }));

    expect(createTaskWithStatus).toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByPlaceholderText("Task title")).toHaveValue("");
  });

  it("rejects an empty title without creating anything", async () => {
    const user = userEvent.setup();
    setup();

    await user.click(screen.getByRole("button", { name: "Create task" }));

    expect(createTaskWithStatus).not.toHaveBeenCalled();
    expect(screen.getByText("A title is required.")).toBeInTheDocument();
  });

  it("prefills title, priority and tags from a draft's Shift+Enter", () => {
    setup({ title: "Fix parser", priority: "low", tags: ["cli"] });

    expect(screen.getByPlaceholderText("Task title")).toHaveValue("Fix parser");
    expect(screen.getByRole("button", { name: "Low" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText("cli")).toBeInTheDocument();
  });
});
