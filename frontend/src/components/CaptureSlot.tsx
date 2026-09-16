import { useEffect, useRef, useState } from "react";

import type { TaskView } from "../api";
import type { ToastSpec } from "../useToast";
import type { ParsedDraft } from "../tokens";
import { CaptureDraft } from "./CaptureDraft";

export interface CaptureHandle {
  open: () => void;
}

interface CaptureSlotProps {
  status: string;
  all: TaskView[];
  onCreated: () => void;
  onOpen: (id: number) => void;
  onOpenFullForm: (parsed: ParsedDraft) => void;
  showToast: (spec: ToastSpec) => void;
  /** Registers this slot's imperative handle with a parent (e.g. the board's
   *  own openCapture(), or the "n" shortcut) — called with null on unmount. */
  onReady?: (handle: CaptureHandle | null) => void;
}

// CaptureSlot is the rest "+ Add new task" affordance that every board column
// but `declined` ends in (proposal — Capture); clicking or an external
// `open()` call swaps it for a CaptureDraft.
export function CaptureSlot({
  status,
  all,
  onCreated,
  onOpen,
  onOpenFullForm,
  showToast,
  onReady,
}: CaptureSlotProps) {
  const [open, setOpen] = useState(false);
  // Undo reopens the draft with its original line; bumping the key remounts
  // the draft so the line is taken even if a draft is already open.
  const [seed, setSeed] = useState({ key: 0, line: "" });
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    onReady?.({
      open: () => {
        // jsdom (tests) has no scrollIntoView; guard so it's a no-op there.
        rootRef.current?.scrollIntoView?.({ behavior: "smooth", block: "nearest", inline: "start" });
        setOpen(true);
      },
    });
    return () => onReady?.(null);
    // onReady is expected to be referentially stable for the slot's lifetime.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div ref={rootRef} className="capture-slot">
      {open ? (
        <CaptureDraft
          key={seed.key}
          initialLine={seed.line}
          status={status}
          all={all}
          autoFocus
          onCreated={onCreated}
          onOpen={onOpen}
          onOpenFullForm={onOpenFullForm}
          onClose={() => {
            setOpen(false);
            setSeed((s) => ({ key: s.key, line: "" }));
          }}
          onReopen={(line) => {
            setSeed((s) => ({ key: s.key + 1, line }));
            setOpen(true);
          }}
          showToast={showToast}
        />
      ) : (
        <button type="button" className="capture-slot-btn" onClick={() => setOpen(true)}>
          + Add new task
        </button>
      )}
    </div>
  );
}
