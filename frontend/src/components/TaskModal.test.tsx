import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import type { TaskView } from "../api";
import { TaskModal } from "./TaskModal";

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

const otherTasks = [task({ id: 8, title: "Add fuzzy search", status: "done" })];

function setup(over: Partial<TaskView> = {}) {
  const t = task({ id: 17, title: "Card covers on the board view", ...over });
  const onPatch = vi.fn().mockResolvedValue(undefined);
  const onDelete = vi.fn();
  const onOpenLink = vi.fn();
  const onError = vi.fn();
  const askReason = vi.fn();
  render(
    <TaskModal
      task={t}
      tasks={[t, ...otherTasks]}
      error=""
      onClose={vi.fn()}
      onPatch={onPatch}
      onDelete={onDelete}
      onOpenLink={onOpenLink}
      onError={onError}
      askReason={askReason}
    />,
  );
  return { task: t, onPatch, onDelete, onOpenLink, onError, askReason };
}

afterEach(cleanup);

describe("TaskModal", () => {
  it("sends only the assignee field when the assignee is edited", async () => {
    const user = userEvent.setup();
    const { onPatch } = setup();

    await user.click(screen.getByRole("button", { name: "Assignee" }));
    await user.type(screen.getByLabelText("Assignee"), "weldnor{Enter}");

    expect(onPatch).toHaveBeenCalledTimes(1);
    expect(onPatch).toHaveBeenCalledWith({ assignee: "weldnor" });
  });

  it("asks for a reason and PATCHes status and reason when declined is chosen", async () => {
    const user = userEvent.setup();
    const { onPatch, askReason } = setup();
    askReason.mockResolvedValue("out of scope");

    await user.click(screen.getByRole("button", { name: "Status" }));
    await user.click(screen.getByRole("option", { name: "Declined" }));

    expect(askReason).toHaveBeenCalled();
    expect(onPatch).toHaveBeenCalledWith({ status: "declined", reason: "out of scope" });
  });

  it("sends nothing when the decline reason prompt is cancelled", async () => {
    const user = userEvent.setup();
    const { onPatch, askReason } = setup();
    askReason.mockResolvedValue(null);

    await user.click(screen.getByRole("button", { name: "Status" }));
    await user.click(screen.getByRole("option", { name: "Declined" }));

    expect(onPatch).not.toHaveBeenCalled();
  });

  it("clears the reason when a declined task is set to another status", async () => {
    const user = userEvent.setup();
    const { onPatch } = setup({ status: "declined", reason: "dup" });

    await user.click(screen.getByRole("button", { name: "Status" }));
    await user.click(screen.getByRole("option", { name: "Todo" }));

    expect(onPatch).toHaveBeenCalledWith({ status: "todo", reason: "" });
  });

  it("reports an empty title without patching and keeps the old title shown", async () => {
    const user = userEvent.setup();
    const { onPatch, onError } = setup();

    await user.click(screen.getByRole("button", { name: "Title" }));
    await user.clear(screen.getByLabelText("Title"));
    await user.keyboard("{Enter}");

    expect(onPatch).not.toHaveBeenCalled();
    expect(onError).toHaveBeenCalledWith("A title is required.");
    expect(screen.getByRole("button", { name: "Title" })).toHaveTextContent(
      "Card covers on the board view",
    );
  });

  it("copies the show command and shows a brief confirmation", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn();
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    setup();

    await user.click(screen.getByRole("button", { name: "Copy command" }));

    expect(writeText).toHaveBeenCalledWith("backlog show 17");
    expect(await screen.findByText("Command copied")).toBeInTheDocument();
  });

  it("adds a link and PATCHes the full links array", async () => {
    const user = userEvent.setup();
    const { onPatch } = setup();

    await user.click(screen.getByRole("button", { name: "+ Link a task" }));
    await user.type(screen.getByLabelText("Search tasks to link"), "fuzzy");
    await user.click(screen.getByRole("option", { name: /Add fuzzy search/ }));

    expect(onPatch).toHaveBeenCalledWith({ links: [{ type: "related", id: 8 }] });
  });

  it("opens a linked task when its title is clicked", async () => {
    const user = userEvent.setup();
    const { onOpenLink } = setup({ links: [{ type: "blocks", id: 8 }] });

    await user.click(screen.getByRole("button", { name: "Add fuzzy search" }));

    expect(onOpenLink).toHaveBeenCalledWith(8);
  });

  it("removes a link and PATCHes without it", async () => {
    const user = userEvent.setup();
    const { onPatch } = setup({ links: [{ type: "blocks", id: 8 }] });

    await user.click(screen.getByRole("button", { name: /Remove blocks link/ }));

    expect(onPatch).toHaveBeenCalledWith({ links: [] });
  });

  it("calls onDelete when Delete is chosen", async () => {
    const user = userEvent.setup();
    const { onDelete } = setup();

    await user.click(screen.getByRole("button", { name: "Delete" }));

    expect(onDelete).toHaveBeenCalled();
  });
});
