import { useState } from "react";

import type { TaskView } from "../api";
import type { ToastSpec } from "../useToast";
import type { ParsedDraft } from "../tokens";
import { CaptureDraft } from "./CaptureDraft";

interface CaptureSlotProps {
  status: string;
  all: TaskView[];
  onCreated: () => void;
  onOpen: (id: number) => void;
  onOpenFullForm: (parsed: ParsedDraft) => void;
  showToast: (spec: ToastSpec) => void;
}

// CaptureSlot is the rest "+ Add new task" affordance that every board column
// but `declined` ends in (proposal — Capture); clicking swaps it for a CaptureDraft.
export function CaptureSlot({
  status,
  all,
  onCreated,
  onOpen,
  onOpenFullForm,
  showToast,
}: CaptureSlotProps) {
  const [open, setOpen] = useState(false);
  // Undo reopens the draft with its original line; bumping the key remounts
  // the draft so the line is taken even if a draft is already open.
  const [seed, setSeed] = useState({ key: 0, line: "" });

  return (
    <div className="capture-slot">
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
