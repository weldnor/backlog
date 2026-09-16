import { useEffect } from "react";

import type { ToastState } from "../useToast";

interface ToastHostProps {
  toast: ToastState | null;
  onDismiss: () => void;
}

// Six seconds, per design.md D9 — long enough to read and act on Undo before
// it fades on its own.
const DURATION_MS = 6000;

// ToastHost renders the single active toast (if any) bottom-left and clears
// it on its own after DURATION_MS; a new toast (a new `id`) restarts the
// timer, matching "creating replaces any previous one".
export function ToastHost({ toast, onDismiss }: ToastHostProps) {
  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(onDismiss, DURATION_MS);
    return () => window.clearTimeout(timer);
  }, [toast, onDismiss]);

  if (!toast) return null;

  return (
    <div className="toast" role="status">
      <span className="toast-text">{toast.text}</span>
      {toast.actions?.map((a) => (
        <button
          key={a.label}
          type="button"
          className="text-btn is-strong toast-action"
          onClick={() => {
            onDismiss();
            a.onClick();
          }}
        >
          {a.label}
        </button>
      ))}
    </div>
  );
}
