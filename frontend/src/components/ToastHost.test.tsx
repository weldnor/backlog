import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

import { ToastHost } from "./ToastHost";
import type { ToastState } from "../useToast";

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

function toast(over: Partial<ToastState> = {}): ToastState {
  return { id: 1, text: "Task 007 created in New", ...over };
}

describe("ToastHost", () => {
  it("renders nothing when there is no toast", () => {
    const { container } = render(<ToastHost toast={null} onDismiss={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("shows the text and its actions", () => {
    const openSpy = vi.fn();
    render(
      <ToastHost
        toast={toast({ actions: [{ label: "Open", onClick: openSpy }] })}
        onDismiss={() => {}}
      />,
    );
    expect(screen.getByText("Task 007 created in New")).toBeInTheDocument();
    screen.getByRole("button", { name: "Open" }).click();
    expect(openSpy).toHaveBeenCalled();
  });

  it("dismisses itself after 6 seconds", () => {
    const onDismiss = vi.fn();
    render(<ToastHost toast={toast()} onDismiss={onDismiss} />);

    vi.advanceTimersByTime(5999);
    expect(onDismiss).not.toHaveBeenCalled();

    vi.advanceTimersByTime(1);
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("restarts the timer when a new toast (a new id) replaces the current one", () => {
    const onDismiss = vi.fn();
    const { rerender } = render(<ToastHost toast={toast({ id: 1 })} onDismiss={onDismiss} />);

    vi.advanceTimersByTime(4000);
    rerender(<ToastHost toast={toast({ id: 2, text: "Task 008 created in New" })} onDismiss={onDismiss} />);
    vi.advanceTimersByTime(4000);

    // 8s of wall time passed but the second toast has only been up for 4s.
    expect(onDismiss).not.toHaveBeenCalled();

    vi.advanceTimersByTime(2000);
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });
});
