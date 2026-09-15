import { useRef, useState, type ReactNode } from "react";

import { LayeredDialog } from "./LayeredDialog";

type Variant = "top" | "nested";

interface ConfirmDialogProps {
  title: string;
  description?: ReactNode;
  confirmLabel?: string;
  variant?: Variant;
  onResult: (ok: boolean) => void;
}

// ConfirmDialog is mockup 07: a small window over the current layer, carrying
// the only destructive fill in the system. Focus starts on Cancel so a stray
// Enter never deletes.
export function ConfirmDialog({
  title,
  description,
  confirmLabel = "Delete",
  variant = "nested",
  onResult,
}: ConfirmDialogProps) {
  const cancelRef = useRef<HTMLButtonElement>(null);
  return (
    <LayeredDialog
      label={title}
      variant={variant}
      className="confirm"
      onClose={() => onResult(false)}
      initialFocus={cancelRef}
    >
      <div className="confirm-title">{title}</div>
      {description ? <div className="confirm-desc">{description}</div> : null}
      <div className="confirm-actions">
        <button type="button" className="pill-btn is-danger" onClick={() => onResult(true)}>
          {confirmLabel}
        </button>
        <button type="button" ref={cancelRef} className="text-btn" onClick={() => onResult(false)}>
          Cancel
        </button>
      </div>
    </LayeredDialog>
  );
}

interface ReasonDialogProps {
  title?: string;
  description?: ReactNode;
  variant?: Variant;
  /** The trimmed reason, or null when cancelled. */
  onResult: (reason: string | null) => void;
}

// ReasonDialog asks for the decline reason in the same shell; a reason is
// required exactly when the status becomes `declined`, so Decline stays
// disabled while the input is blank.
export function ReasonDialog({
  title = "Decline this task?",
  description = "The reason is saved with the task.",
  variant = "nested",
  onResult,
}: ReasonDialogProps) {
  const [reason, setReason] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const trimmed = reason.trim();
  const submit = () => {
    if (trimmed) onResult(trimmed);
  };

  return (
    <LayeredDialog
      label={title}
      variant={variant}
      className="confirm"
      onClose={() => onResult(null)}
      initialFocus={inputRef}
    >
      <div className="confirm-title">{title}</div>
      <div className="confirm-desc">{description}</div>
      <input
        ref={inputRef}
        className="field-input confirm-input"
        aria-label="Reason"
        placeholder="Reason"
        value={reason}
        onChange={(e) => setReason(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            submit();
          }
        }}
      />
      <div className="confirm-actions">
        <button type="button" className="pill-btn" disabled={!trimmed} onClick={submit}>
          Decline
        </button>
        <button type="button" className="text-btn" onClick={() => onResult(null)}>
          Cancel
        </button>
      </div>
    </LayeredDialog>
  );
}
