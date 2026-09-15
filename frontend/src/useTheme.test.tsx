import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { useTheme } from "./useTheme";

function Probe() {
  const { theme, toggle } = useTheme();
  return <button onClick={toggle}>{theme}</button>;
}

const stamped = () => document.documentElement.getAttribute("data-theme");

beforeEach(() => {
  window.localStorage.clear();
  document.documentElement.removeAttribute("data-theme");
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("useTheme", () => {
  it("follows the system preference when nothing is remembered", () => {
    vi.stubGlobal("matchMedia", (q: string) => ({ matches: q.includes("dark") }));
    render(<Probe />);

    expect(stamped()).toBe("dark");
  });

  it("stamps the toggled theme and keeps it across a remount", async () => {
    const user = userEvent.setup();
    const { unmount } = render(<Probe />);
    expect(stamped()).toBe("light");

    await user.click(screen.getByRole("button"));
    expect(stamped()).toBe("dark");

    unmount();
    document.documentElement.removeAttribute("data-theme");
    render(<Probe />);

    expect(stamped()).toBe("dark");
    expect(screen.getByRole("button")).toHaveTextContent("dark");
  });

  it("still toggles when storage throws", async () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("blocked");
    });
    const user = userEvent.setup();
    render(<Probe />);

    await user.click(screen.getByRole("button"));

    expect(stamped()).toBe("dark");
    vi.restoreAllMocks();
  });
});
