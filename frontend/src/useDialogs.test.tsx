import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { useDialogs } from "./useDialogs";

function Harness({
  onConfirm,
  onReason,
}: {
  onConfirm: (ok: boolean) => void;
  onReason: (reason: string | null) => void;
}) {
  const { confirm, askReason, dialogs } = useDialogs();
  return (
    <>
      <button
        onClick={() =>
          confirm({
            title: "Delete this task?",
            description: "017 · Card covers on the board view — this cannot be undone.",
          }).then(onConfirm)
        }
      >
        ask confirm
      </button>
      <button onClick={() => askReason().then(onReason)}>ask reason</button>
      {dialogs}
    </>
  );
}

afterEach(cleanup);

async function openConfirm() {
  const user = userEvent.setup();
  const onConfirm = vi.fn();
  render(<Harness onConfirm={onConfirm} onReason={() => {}} />);
  await user.click(screen.getByRole("button", { name: "ask confirm" }));
  const dialog = await screen.findByRole("dialog", { name: "Delete this task?" });
  return { user, onConfirm, dialog };
}

describe("useDialogs confirm", () => {
  it("resolves true when Delete is chosen", async () => {
    const { user, onConfirm, dialog } = await openConfirm();
    expect(dialog).toHaveTextContent("017 · Card covers on the board view");

    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith(true));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("resolves false when Cancel is chosen", async () => {
    const { user, onConfirm } = await openConfirm();

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith(false));
  });

  it("resolves false on Escape", async () => {
    const { user, onConfirm } = await openConfirm();

    await user.keyboard("{Escape}");

    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith(false));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});

describe("useDialogs askReason", () => {
  it("keeps Decline disabled while the reason is blank and resolves the reason", async () => {
    const user = userEvent.setup();
    const onReason = vi.fn();
    render(<Harness onConfirm={() => {}} onReason={onReason} />);
    await user.click(screen.getByRole("button", { name: "ask reason" }));
    await screen.findByRole("dialog");
    const decline = screen.getByRole("button", { name: "Decline" });

    expect(decline).toBeDisabled();
    await user.type(screen.getByLabelText("Reason"), "   ");
    expect(decline).toBeDisabled();

    await user.type(screen.getByLabelText("Reason"), "out of scope");
    expect(decline).toBeEnabled();
    await user.click(decline);

    await waitFor(() => expect(onReason).toHaveBeenCalledWith("out of scope"));
  });

  it("resolves null when cancelled", async () => {
    const user = userEvent.setup();
    const onReason = vi.fn();
    render(<Harness onConfirm={() => {}} onReason={onReason} />);
    await user.click(screen.getByRole("button", { name: "ask reason" }));
    await screen.findByRole("dialog");

    await user.click(screen.getByRole("button", { name: "Cancel" }));

    await waitFor(() => expect(onReason).toHaveBeenCalledWith(null));
  });
});
