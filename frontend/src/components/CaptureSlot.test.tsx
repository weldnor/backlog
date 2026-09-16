import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import type { TaskView } from "../api";
import { ApiError } from "../api";
import { CaptureSlot } from "./CaptureSlot";

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

const all = [
  task({ id: 11, title: "Existing task", status: "todo", tags: ["ui", "urgent"] }),
];

function setup() {
  const onCreated = vi.fn();
  const onOpen = vi.fn();
  const onOpenFullForm = vi.fn();
  const showToast = vi.fn();
  render(
    <CaptureSlot
      status="todo"
      all={all}
      onCreated={onCreated}
      onOpen={onOpen}
      onOpenFullForm={onOpenFullForm}
      showToast={showToast}
    />,
  );
  return { onCreated, onOpen, onOpenFullForm, showToast };
}

beforeEach(() => {
  createTaskWithStatus.mockReset();
  deleteTask.mockReset();
});

afterEach(cleanup);

describe("CaptureSlot / CaptureDraft", () => {
  it("opens the draft from the rest slot and creates from tokens in the column's status", async () => {
    const user = userEvent.setup();
    createTaskWithStatus.mockResolvedValue(task({ id: 20, title: "Drop zone highlight", status: "doing" }));
    const { onCreated, showToast } = setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    const input = screen.getByPlaceholderText(/Title, #tag/);
    await user.type(input, "Drop zone highlight !high #ui @ann >doing");
    await user.keyboard("{Enter}");

    expect(createTaskWithStatus).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "Drop zone highlight",
        priority: "high",
        tags: ["ui"],
        assignee: "ann",
      }),
      "doing",
    );
    expect(onCreated).toHaveBeenCalled();
    expect(showToast).toHaveBeenCalledWith(
      expect.objectContaining({ text: "Task 020 created in Doing" }),
    );
    expect(input).toHaveValue("");
  });

  it("defaults to the column's own status when no >status token is given", async () => {
    const user = userEvent.setup();
    createTaskWithStatus.mockResolvedValue(task({ id: 21, title: "Add fuzzy search", status: "todo" }));
    setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    await user.type(screen.getByPlaceholderText(/Title, #tag/), "Add fuzzy search");
    await user.keyboard("{Enter}");

    expect(createTaskWithStatus).toHaveBeenCalledWith(expect.anything(), "todo");
  });

  it("rejects an empty title without a request and shows the problem", async () => {
    const user = userEvent.setup();
    setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    await user.keyboard("{Enter}");

    expect(createTaskWithStatus).not.toHaveBeenCalled();
    expect(screen.getByText("A title is required.")).toBeInTheDocument();
  });

  it("closes the draft on Esc without making a request", async () => {
    const user = userEvent.setup();
    setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    await user.type(screen.getByPlaceholderText(/Title, #tag/), "Something");
    await user.keyboard("{Escape}");

    expect(createTaskWithStatus).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "+ Add new task" })).toBeInTheDocument();
  });

  it("opens the full form on Shift+Enter with the parsed line", async () => {
    const user = userEvent.setup();
    const { onOpenFullForm } = setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    await user.type(screen.getByPlaceholderText(/Title, #tag/), "Fix parser !low #cli");
    await user.keyboard("{Shift>}{Enter}{/Shift}");

    expect(onOpenFullForm).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Fix parser", priority: "low", tags: ["cli"] }),
    );
    expect(createTaskWithStatus).not.toHaveBeenCalled();
  });

  it("offers tag autocomplete after #, and Tab accepts the highlighted tag", async () => {
    const user = userEvent.setup();
    setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    const input = screen.getByPlaceholderText(/Title, #tag/);
    await user.type(input, "Something #u");

    expect(screen.getByRole("option", { name: "ui" })).toBeInTheDocument();

    await user.keyboard("{Tab}");

    expect(input).toHaveValue("Something #ui");
  });

  it("closes only the autocomplete list on Esc, leaving the draft open", async () => {
    const user = userEvent.setup();
    setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    const input = screen.getByPlaceholderText(/Title, #tag/);
    await user.type(input, "Something #u");
    expect(screen.getByRole("listbox")).toBeInTheDocument();

    await user.keyboard("{Escape}");

    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    expect(input).toHaveValue("Something #u");
    expect(screen.queryByRole("button", { name: "+ Add new task" })).not.toBeInTheDocument();
  });

  it("blocks creation on a matching title until Create anyway, and Open opens the existing task", async () => {
    const user = userEvent.setup();
    const { onOpen } = setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    await user.type(screen.getByPlaceholderText(/Title, #tag/), "  existing TASK  ");
    await user.keyboard("{Enter}");

    expect(createTaskWithStatus).not.toHaveBeenCalled();
    expect(screen.getByText(/Looks like an existing task/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Open" }));
    expect(onOpen).toHaveBeenCalledWith(11);

    createTaskWithStatus.mockResolvedValue(task({ id: 30, title: "existing TASK", status: "todo" }));
    await user.click(screen.getByRole("button", { name: "Create anyway" }));
    expect(createTaskWithStatus).toHaveBeenCalled();
  });

  it("keeps the line and offers Retry/Copy text/Discard when creation fails", async () => {
    const user = userEvent.setup();
    createTaskWithStatus.mockRejectedValueOnce(new ApiError("boom", 500));
    setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    const input = screen.getByPlaceholderText(/Title, #tag/);
    await user.type(input, "Retry me");
    await user.keyboard("{Enter}");

    expect(await screen.findByText("boom")).toBeInTheDocument();
    expect(input).toHaveValue("Retry me");

    createTaskWithStatus.mockResolvedValueOnce(task({ id: 40, title: "Retry me", status: "todo" }));
    await user.click(screen.getByRole("button", { name: "Retry" }));

    expect(createTaskWithStatus).toHaveBeenCalledTimes(2);
    expect(await screen.findByPlaceholderText(/Title, #tag/)).toHaveValue("");
  });

  it("discards the failed draft's text", async () => {
    const user = userEvent.setup();
    createTaskWithStatus.mockRejectedValueOnce(new ApiError("boom", 500));
    setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    const input = screen.getByPlaceholderText(/Title, #tag/);
    await user.type(input, "Discard me");
    await user.keyboard("{Enter}");
    await screen.findByText("boom");

    await user.click(screen.getByRole("button", { name: "Discard" }));

    expect(input).toHaveValue("");
    expect(screen.queryByText("boom")).not.toBeInTheDocument();
  });

  it("Undo deletes the created task and restores the line into the draft", async () => {
    const user = userEvent.setup();
    createTaskWithStatus.mockResolvedValue(task({ id: 50, title: "Undo me", status: "todo" }));
    deleteTask.mockResolvedValue(task({ id: 50, title: "Undo me", status: "todo" }));
    const { onCreated, showToast } = setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    await user.type(screen.getByPlaceholderText(/Title, #tag/), "Undo me");
    await user.keyboard("{Enter}");

    const toastCall = showToast.mock.calls.find((c) => c[0].text.includes("Task 050"));
    expect(toastCall).toBeTruthy();
    const undo = toastCall![0].actions.find((a: { label: string }) => a.label === "Undo");

    await act(async () => {
      undo.onClick();
    });

    expect(deleteTask).toHaveBeenCalledWith(50);
    await vi.waitFor(() => expect(onCreated).toHaveBeenCalledTimes(2));
    expect(screen.getByPlaceholderText(/Title, #tag/)).toHaveValue("Undo me");
  });
  it("Undo restores the line even after the draft was closed", async () => {
    const user = userEvent.setup();
    createTaskWithStatus.mockResolvedValue(task({ id: 51, title: "Closed then undone", status: "todo" }));
    deleteTask.mockResolvedValue(task({ id: 51, title: "Closed then undone", status: "todo" }));
    const { showToast } = setup();

    await user.click(screen.getByRole("button", { name: "+ Add new task" }));
    await user.type(screen.getByPlaceholderText(/Title, #tag/), "Closed then undone !high");
    await user.keyboard("{Enter}");
    await user.keyboard("{Escape}");
    expect(screen.queryByPlaceholderText(/Title, #tag/)).not.toBeInTheDocument();

    const undo = showToast.mock.calls
      .find((c) => c[0].text.includes("Task 051"))![0]
      .actions.find((a: { label: string }) => a.label === "Undo");
    await act(async () => {
      undo.onClick();
    });

    expect(deleteTask).toHaveBeenCalledWith(51);
    expect(screen.getByPlaceholderText(/Title, #tag/)).toHaveValue("Closed then undone !high");
  });
});
