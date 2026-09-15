import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";

import { hasOpenLayer, LayeredDialog } from "./LayeredDialog";

afterEach(cleanup);

describe("LayeredDialog", () => {
  it("closes only the topmost layer on Escape", async () => {
    const user = userEvent.setup();
    const outer = vi.fn();
    const inner = vi.fn();
    render(
      <LayeredDialog label="Outer" onClose={outer}>
        <button>outer action</button>
        <LayeredDialog label="Inner" variant="nested" onClose={inner}>
          <button>inner action</button>
        </LayeredDialog>
      </LayeredDialog>,
    );

    await user.keyboard("{Escape}");

    expect(inner).toHaveBeenCalledTimes(1);
    expect(outer).not.toHaveBeenCalled();
  });

  it("wraps Tab from the last control inside the topmost layer", async () => {
    const user = userEvent.setup();
    render(
      <LayeredDialog label="Outer" onClose={() => {}}>
        <button>outer action</button>
        <LayeredDialog label="Inner" variant="nested" onClose={() => {}}>
          <button>first</button>
          <button>last</button>
        </LayeredDialog>
      </LayeredDialog>,
    );

    screen.getByRole("button", { name: "last" }).focus();
    await user.tab();

    expect(document.activeElement).toBe(screen.getByRole("button", { name: "first" }));
  });

  it("moves focus in on open and restores it to the opener on close", async () => {
    const user = userEvent.setup();
    function Harness() {
      const [open, setOpen] = useState(false);
      return (
        <>
          <button onClick={() => setOpen(true)}>open</button>
          {open ? (
            <LayeredDialog label="Dialog" onClose={() => setOpen(false)}>
              <button>inside</button>
            </LayeredDialog>
          ) : null}
        </>
      );
    }
    render(<Harness />);
    const opener = screen.getByRole("button", { name: "open" });

    await user.click(opener);
    expect(document.activeElement).toBe(screen.getByRole("button", { name: "inside" }));
    expect(hasOpenLayer()).toBe(true);

    await user.keyboard("{Escape}");

    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(document.activeElement).toBe(opener);
    expect(hasOpenLayer()).toBe(false);
  });

  it("leaves the layer open when a control inside stops Escape", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <LayeredDialog label="Dialog" onClose={onClose}>
        <input
          aria-label="inline"
          onKeyDown={(e) => {
            if (e.key === "Escape") e.stopPropagation();
          }}
        />
      </LayeredDialog>,
    );

    await user.keyboard("{Escape}");

    expect(onClose).not.toHaveBeenCalled();
  });
});
