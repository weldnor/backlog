import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { InlineSelect, InlineText } from "./InlineFields";

afterEach(cleanup);

function deferred<T>() {
  let resolve!: (v: T) => void;
  let reject!: (e: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe("InlineText", () => {
  it("commits the typed value on Enter", async () => {
    const user = userEvent.setup();
    const onCommit = vi.fn().mockResolvedValue(undefined);
    render(<InlineText value="Alpha" ariaLabel="Title" onCommit={onCommit} />);

    await user.click(screen.getByRole("button", { name: "Title" }));
    const input = screen.getByLabelText("Title");
    await user.clear(input);
    await user.type(input, "Beta{Enter}");

    expect(onCommit).toHaveBeenCalledWith("Beta");
    expect(await screen.findByRole("button", { name: "Title" })).toHaveTextContent("Alpha");
  });

  it("cancels on Escape without committing, and stops the key from reaching the document", async () => {
    const user = userEvent.setup();
    const onCommit = vi.fn();
    const docSpy = vi.fn();
    document.addEventListener("keydown", docSpy);
    render(<InlineText value="Alpha" ariaLabel="Title" onCommit={onCommit} />);

    await user.click(screen.getByRole("button", { name: "Title" }));
    await user.type(screen.getByLabelText("Title"), "Changed{Escape}");

    expect(onCommit).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Title" })).toHaveTextContent("Alpha");
    expect(docSpy).not.toHaveBeenCalledWith(expect.objectContaining({ key: "Escape" }));
    document.removeEventListener("keydown", docSpy);
  });

  it("disables the input while a commit is in flight, then reverts on error", async () => {
    const user = userEvent.setup();
    const { promise, reject } = deferred<void>();
    const onCommit = vi.fn().mockReturnValue(promise);
    render(<InlineText value="Alpha" ariaLabel="Title" onCommit={onCommit} />);

    await user.click(screen.getByRole("button", { name: "Title" }));
    await user.clear(screen.getByLabelText("Title"));
    await user.type(screen.getByLabelText("Title"), "Beta{Enter}");

    expect(screen.getByLabelText("Title")).toBeDisabled();

    await act(async () => {
      reject(new Error("boom"));
      await promise.catch(() => {});
    });

    expect(screen.getByRole("button", { name: "Title" })).toHaveTextContent("Alpha");
  });

  it("shows the placeholder when the value is empty", () => {
    render(<InlineText value="" placeholder="unassigned" ariaLabel="Assignee" onCommit={vi.fn()} />);
    expect(screen.getByRole("button", { name: "Assignee" })).toHaveTextContent("unassigned");
  });
});

describe("InlineSelect", () => {
  const options = [
    { value: "todo", label: "Todo" },
    { value: "doing", label: "Doing" },
    { value: "done", label: "Done" },
  ];

  it("opens the listbox and commits the chosen option", async () => {
    const user = userEvent.setup();
    const onCommit = vi.fn().mockResolvedValue(undefined);
    render(<InlineSelect value="todo" options={options} ariaLabel="Status" onCommit={onCommit} />);

    await user.click(screen.getByRole("button", { name: "Status" }));
    await user.click(screen.getByRole("option", { name: "Doing" }));

    expect(onCommit).toHaveBeenCalledWith("doing");
  });

  it("closes on Escape without committing", async () => {
    const user = userEvent.setup();
    const onCommit = vi.fn();
    render(<InlineSelect value="todo" options={options} ariaLabel="Status" onCommit={onCommit} />);

    await user.click(screen.getByRole("button", { name: "Status" }));
    expect(screen.getByRole("listbox")).toBeInTheDocument();

    await user.keyboard("{Escape}");

    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    expect(onCommit).not.toHaveBeenCalled();
  });
});
