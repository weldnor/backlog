import { forwardRef, useImperativeHandle, useRef, useState } from "react";

import type { TaskView } from "../api";
import { STATUS_ORDER } from "../constants";
import type { ParsedDraft } from "../tokens";
import type { ToastSpec } from "../useToast";
import { BoardColumn } from "./BoardColumn";
import type { CaptureHandle } from "./CaptureSlot";

interface BoardViewProps {
  tasks: TaskView[];
  onOpen: (id: number) => void;
  onMove: (id: number, status: string) => void;
  onMenu?: (id: number, x: number, y: number) => void;
  /** Every task — threaded to each column's capture draft. */
  all: TaskView[];
  onCreated: () => void;
  onOpenFullForm: (parsed: ParsedDraft) => void;
  showToast: (spec: ToastSpec) => void;
}

export interface BoardViewHandle {
  /** Opens a capture draft in the first non-collapsed column (design.md D14). */
  openCapture: () => void;
}

const COLLAPSED_KEY = "backlog.collapsed";

// Storage can be unavailable (private windows, blocked site data); the
// collapsed set then simply isn't remembered, matching useTheme's fallback.
function loadCollapsed(): Set<string> {
  try {
    const raw = window.localStorage.getItem(COLLAPSED_KEY);
    const arr = raw ? JSON.parse(raw) : [];
    return new Set(Array.isArray(arr) ? arr.filter((x) => typeof x === "string") : []);
  } catch {
    return new Set();
  }
}

function saveCollapsed(collapsed: Set<string>) {
  try {
    window.localStorage.setItem(COLLAPSED_KEY, JSON.stringify([...collapsed]));
  } catch {
    // Not persisted; the collapse still applies for this page.
  }
}

// BoardView lays out one BoardColumn per status (design.md D5). Collapsed
// state is remembered per browser; which card is mid-drag is tracked here so
// it keeps its `.is-dragging` look as it crosses from one column to another.
export const BoardView = forwardRef<BoardViewHandle, BoardViewProps>(function BoardView(
  { tasks, onOpen, onMove, onMenu, all, onCreated, onOpenFullForm, showToast },
  ref,
) {
  const [collapsed, setCollapsed] = useState<Set<string>>(() => loadCollapsed());
  const [draggingId, setDraggingId] = useState<number | null>(null);
  const captureHandles = useRef<Partial<Record<string, CaptureHandle>>>({});

  function toggleCollapse(status: string) {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(status)) {
        next.delete(status);
      } else {
        next.add(status);
      }
      saveCollapsed(next);
      return next;
    });
  }

  useImperativeHandle(
    ref,
    () => ({
      openCapture: () => {
        const first = STATUS_ORDER.find((s) => s !== "declined" && !collapsed.has(s));
        if (first) captureHandles.current[first]?.open();
      },
    }),
    [collapsed],
  );

  return (
    <>
      {STATUS_ORDER.map((status) => (
        <BoardColumn
          key={status}
          status={status}
          tasks={tasks.filter((t) => t.status === status)}
          collapsed={collapsed.has(status)}
          onToggleCollapse={() => toggleCollapse(status)}
          onOpen={onOpen}
          onMove={onMove}
          onMenu={onMenu}
          draggingId={draggingId}
          onDragStart={setDraggingId}
          onDragEnd={() => setDraggingId(null)}
          all={all}
          onCreated={onCreated}
          onOpenFullForm={onOpenFullForm}
          showToast={showToast}
          onCaptureReady={(s, handle) => {
            if (handle) captureHandles.current[s] = handle;
            else delete captureHandles.current[s];
          }}
        />
      ))}
    </>
  );
});
