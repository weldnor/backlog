import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { MarkdownEditor } from "./MarkdownEditor";

afterEach(cleanup);

describe("MarkdownEditor", () => {
  it("wraps the selection in ** when B is clicked", async () => {
    const user = userEvent.setup();
    render(<MarkdownEditor value="hello world" onSave={vi.fn()} onCancel={vi.fn()} />);

    const textarea = screen.getByLabelText("Description") as HTMLTextAreaElement;
    textarea.setSelectionRange(0, 5);
    await user.click(screen.getByRole("button", { name: "Bold" }));

    expect(textarea).toHaveValue("**hello** world");
  });

  it("switches to Preview and renders the markdown", async () => {
    const user = userEvent.setup();
    render(<MarkdownEditor value="# Title" onSave={vi.fn()} onCancel={vi.fn()} />);

    await user.click(screen.getByRole("tab", { name: "Preview" }));

    expect(screen.getByRole("heading", { name: "Title" })).toBeInTheDocument();
  });

  it("saves the edited text on Ctrl+Enter", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<MarkdownEditor value="hello" onSave={onSave} onCancel={vi.fn()} />);

    const textarea = screen.getByLabelText("Description");
    await user.type(textarea, " world");
    await user.keyboard("{Control>}{Enter}{/Control}");

    expect(onSave).toHaveBeenCalledWith("hello world");
  });

  it("cancels on Escape without saving", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    const onCancel = vi.fn();
    render(<MarkdownEditor value="hello" onSave={onSave} onCancel={onCancel} />);

    await user.type(screen.getByLabelText("Description"), " world{Escape}");

    expect(onSave).not.toHaveBeenCalled();
    expect(onCancel).toHaveBeenCalled();
  });

  it("saves on the Save button and cancels on the Cancel button", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<MarkdownEditor value="hello" onSave={onSave} onCancel={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSave).toHaveBeenCalledWith("hello");
  });
});
