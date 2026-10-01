import { useEffect, useRef } from "react";

import type { TaskView } from "../api";
import { STATUS_LABEL, STATUS_ORDER } from "../constants";

interface TaskMenuProps {
  task: TaskView;
  x: number;
  y: number;
  onClose: () => void;
  onOpen: () => void;
  onCopyId: () => void;
  onMove: (status: string) => void;
  onDelete: () => void;
}

// TaskMenu is the right-click menu on a card or list row. It reuses the column
// menu's popover look and closes the same way: outside mousedown or Escape.
export function TaskMenu({ task, x, y, onClose, onOpen, onCopyId, onMove, onDelete }: TaskMenuProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onPointerDown(e: MouseEvent) {
      if (!ref.current?.contains(e.target as Node)) onClose();
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    window.addEventListener("scroll", onClose, true);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("scroll", onClose, true);
    };
  }, [onClose]);

  const item = (label: string, run: () => void, className = "") => (
    <button
      key={label}
      type="button"
      role="menuitem"
      className={"popover-item " + className}
      onClick={() => {
        onClose();
        run();
      }}
    >
      {label}
    </button>
  );

  return (
    <div ref={ref} className="popover task-menu" role="menu" style={{ left: x, top: y }}>
      {item("Open", onOpen)}
      {item("Copy ID", onCopyId)}
      {STATUS_ORDER.filter((s) => s !== task.status).map((s) =>
        item(`Move to ${STATUS_LABEL[s]}`, () => onMove(s)),
      )}
      {item("Delete", onDelete, "is-danger")}
    </div>
  );
}
