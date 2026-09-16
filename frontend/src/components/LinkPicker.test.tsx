import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import type { TaskView } from "../api";
import { LinkPicker } from "./LinkPicker";

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

const tasks = [
  task({ id: 1, title: "Fix parser" }),
  task({ id: 8, title: "Add fuzzy search", status: "done" }),
];

afterEach(cleanup);

describe("LinkPicker", () => {
  it("shows a linked task's type, id, title and status, and opens it on click", async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    render(
      <LinkPicker
        links={[{ type: "blocks", id: 8 }]}
        tasks={tasks}
        excludeId={1}
        onChange={vi.fn()}
        onOpen={onOpen}
      />,
    );

    expect(screen.getByText("blocks")).toBeInTheDocument();
    expect(screen.getByText("#008")).toBeInTheDocument();
    expect(screen.getByText("Done")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Add fuzzy search" }));

    expect(onOpen).toHaveBeenCalledWith(8);
  });

  it("removes a link without the caller needing to look it up", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <LinkPicker
        links={[{ type: "blocks", id: 8 }]}
        tasks={tasks}
        excludeId={1}
        onChange={onChange}
        onOpen={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: /Remove blocks link/ }));

    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("adds a link of the chosen type from the search popover", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <LinkPicker links={[]} tasks={tasks} excludeId={1} onChange={onChange} onOpen={vi.fn()} />,
    );

    await user.click(screen.getByRole("button", { name: "+ Link a task" }));
    await user.selectOptions(screen.getByLabelText("Link type"), "duplicates");
    await user.type(screen.getByLabelText("Search tasks to link"), "fuzzy");
    await user.click(screen.getByRole("option", { name: /Add fuzzy search/ }));

    expect(onChange).toHaveBeenCalledWith([{ type: "duplicates", id: 8 }]);
  });

  it("closes the popover on Escape without changing anything", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <LinkPicker links={[]} tasks={tasks} excludeId={1} onChange={onChange} onOpen={vi.fn()} />,
    );

    await user.click(screen.getByRole("button", { name: "+ Link a task" }));
    expect(screen.getByLabelText("Search tasks to link")).toBeInTheDocument();

    await user.keyboard("{Escape}");

    expect(screen.queryByLabelText("Search tasks to link")).not.toBeInTheDocument();
    expect(onChange).not.toHaveBeenCalled();
  });
});
