import { useEffect, useRef, useState, type DragEvent } from "react";

import type { TaskView } from "../api";
import { copyText } from "../clipboard";
import { BOARD_EMPTY_NOTE, padId, STATUS_LABEL } from "../constants";
import type { ParsedDraft } from "../tokens";
import type { ToastSpec } from "../useToast";
import { CaptureSlot, type CaptureHandle } from "./CaptureSlot";
import { TaskCard } from "./TaskCard";

// columnMarkdown is the column menu's "Copy list as markdown" output
// (design.md D5): one line per visible task, `- [NNN] Title (priority)
// #tags @assignee`, tags and assignee omitted when the task has none.
export function columnMarkdown(tasks: TaskView[]): string {
  return tasks
    .map((t) => {
      const parts = [`- [${padId(t.id)}] ${t.title} (${t.priority})`];
      if (t.tags.length) parts.push(t.tags.map((g) => "#" + g).join(" "));
      if (t.assignee) parts.push("@" + t.assignee);
      return parts.join(" ");
    })
    .join("\n");
}

interface ColumnHeaderProps {
  status: string;
  count: number;
  collapsed: boolean;
  onToggleCollapse: () => void;
  onCopyMarkdown: () => void;
}

// ColumnHeader carries the status name, its task count and the "…" menu
// (design.md D5): Collapse/Expand list and Copy list as markdown.
function ColumnHeader({ status, count, collapsed, onToggleCollapse, onCopyMarkdown }: ColumnHeaderProps) {
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const label = STATUS_LABEL[status] ?? status;

  useEffect(() => {
    if (!open) return;
    function onPointerDown(e: MouseEvent) {
      const target = e.target as Node;
      if (menuRef.current?.contains(target) || triggerRef.current?.contains(target)) return;
      setOpen(false);
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  return (
    <div className="board-col-head">
      <span className="board-col-name">{label}</span>
      <span className="board-col-count">{count}</span>
      <span className="spacer" />
      <div className="col-menu">
        <button
          ref={triggerRef}
          type="button"
          className="icon-btn col-menu-trigger"
          aria-haspopup="menu"
          aria-expanded={open}
          aria-label={`${label} column menu`}
          onClick={() => setOpen((o) => !o)}
        >
          …
        </button>
        {open ? (
          <div ref={menuRef} className="popover col-menu-popover" role="menu">
            <button
              type="button"
              role="menuitem"
              className="popover-item"
              onClick={() => {
                setOpen(false);
                onToggleCollapse();
              }}
            >
              {collapsed ? "Expand list" : "Collapse list"}
            </button>
            <button
              type="button"
              role="menuitem"
              className="popover-item"
              onClick={() => {
                setOpen(false);
                onCopyMarkdown();
              }}
            >
              Copy list as markdown
            </button>
          </div>
        ) : null}
      </div>
    </div>
  );
}

interface BoardColumnProps {
  status: string;
  tasks: TaskView[];
  collapsed: boolean;
  onToggleCollapse: () => void;
  onOpen: (id: number) => void;
  onMove: (id: number, status: string) => void;
  draggingId: number | null;
  onDragStart: (id: number) => void;
  onDragEnd: () => void;
  onMenu?: (id: number, x: number, y: number) => void;
  /** Every task, threaded to the column's capture draft (duplicate check, tag autocomplete). */
  all: TaskView[];
  onCreated: () => void;
  onOpenFullForm: (parsed: ParsedDraft) => void;
  showToast: (spec: ToastSpec) => void;
  /** Registers this column's capture handle so the board can open it from outside (the "n" shortcut). */
  onCaptureReady?: (status: string, handle: CaptureHandle | null) => void;
}

// BoardColumn is one status lane of the board (design.md D5): a header with
// the menu, the cards, and — when collapsed — a narrow rail instead, which
// still accepts a drop so a card can be moved into a column without
// expanding it first.
export function BoardColumn({
  status,
  tasks,
  collapsed,
  onToggleCollapse,
  onOpen,
  onMove,
  draggingId,
  onDragStart,
  onDragEnd,
  onMenu,
  all,
  onCreated,
  onOpenFullForm,
  showToast,
  onCaptureReady,
}: BoardColumnProps) {
  const [isOver, setIsOver] = useState(false);
  const label = STATUS_LABEL[status] ?? status;

  function copyMarkdown() {
    void copyText(columnMarkdown(tasks));
  }

  const dropProps = {
    onDragOver: (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = "move";
      setIsOver(true);
    },
    onDragLeave: () => setIsOver(false),
    onDragEnd: () => setIsOver(false),
    onDrop: (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      setIsOver(false);
      const id = Number(e.dataTransfer.getData("text/plain"));
      if (Number.isFinite(id) && id > 0) onMove(id, status);
    },
  };

  if (collapsed) {
    return (
      <div
        className={"board-col board-col-rail" + (isOver ? " is-drop" : "")}
        role="region"
        aria-label={`${label}, collapsed`}
        {...dropProps}
      >
        <button
          type="button"
          className="board-col-rail-btn"
          aria-label={`Expand ${label} list`}
          onClick={onToggleCollapse}
        >
          <span className="board-col-rail-label">
            {label} · {tasks.length}
          </span>
        </button>
      </div>
    );
  }

  return (
    <div
      className={"board-col" + (isOver ? " is-drop" : "")}
      role="region"
      aria-label={label}
      {...dropProps}
    >
      <ColumnHeader
        status={status}
        count={tasks.length}
        collapsed={collapsed}
        onToggleCollapse={onToggleCollapse}
        onCopyMarkdown={copyMarkdown}
      />
      <div className="board-col-body">
        {tasks.map((t) => (
          <TaskCard
            key={t.id}
            task={t}
            isDragging={t.id === draggingId}
            onOpen={onOpen}
            onDragStart={onDragStart}
            onDragEnd={onDragEnd}
            onMenu={onMenu}
          />
        ))}
        {tasks.length === 0 ? <div className="board-empty">{BOARD_EMPTY_NOTE[status]}</div> : null}
        {status !== "declined" ? (
          <CaptureSlot
            status={status}
            all={all}
            onCreated={onCreated}
            onOpen={onOpen}
            onOpenFullForm={onOpenFullForm}
            showToast={showToast}
            onReady={onCaptureReady && ((handle) => onCaptureReady(status, handle))}
          />
        ) : null}
      </div>
    </div>
  );
}
