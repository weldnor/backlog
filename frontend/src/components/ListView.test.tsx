import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import type { TaskView } from "../api";
import { ListView } from "./ListView";

afterEach(cleanup);

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

describe("ListView", () => {
  it("shows a padded id, title and status for each task", () => {
    render(
      <ListView
        tasks={[task({ id: 7, title: "Add fuzzy search", status: "doing" })]}
        openId={null}
        onOpen={vi.fn()}
      />,
    );

    expect(screen.getByText("007")).toBeInTheDocument();
    expect(screen.getByText("Add fuzzy search")).toBeInTheDocument();
    expect(screen.getByText("Doing")).toBeInTheDocument();
  });

  it("opens a row on click and on Enter", async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    render(
      <ListView tasks={[task({ id: 7, title: "Add fuzzy search" })]} openId={null} onOpen={onOpen} />,
    );

    await user.click(screen.getByRole("button", { name: /Add fuzzy search/ }));
    expect(onOpen).toHaveBeenCalledWith(7);

    onOpen.mockClear();
    screen.getByRole("button", { name: /Add fuzzy search/ }).focus();
    await user.keyboard("{Enter}");
    expect(onOpen).toHaveBeenCalledWith(7);
  });
});
