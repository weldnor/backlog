import type { DragEvent, KeyboardEvent } from "react";

import type { TaskView } from "../api";
import { descExcerpt, PRI_DOT, PRI_LABEL, tagColor } from "../constants";

// The most label bars a card draws; the rest are still listed in the modal.
const MAX_LABELS = 6;

export function PriorityDot({ priority }: { priority: string }) {
  return (
    <span
      className="pri-dot"
      aria-hidden="true"
      style={{ background: PRI_DOT[priority] ?? "var(--ink-20)" }}
    />
  );
}

interface TaskCardProps {
  task: TaskView;
  isDragging?: boolean;
  onOpen: (id: number) => void;
  onDragStart?: (id: number) => void;
  onDragEnd?: () => void;
}

// TaskCard is the board card of mockup 01 (design.md D6): label bars, title,
// the start of the description, then priority and assignee. It is focusable
// and opens on Enter or Space, and carries the task id through the drag.
export function TaskCard({ task, isDragging, onOpen, onDragStart, onDragEnd }: TaskCardProps) {
  const excerpt = descExcerpt(task.description);

  return (
    <article
      className={"card" + (isDragging ? " is-dragging" : "")}
      tabIndex={0}
      role="button"
      draggable
      data-task-id={task.id}
      onDragStart={(e: DragEvent<HTMLElement>) => {
        e.dataTransfer.setData("text/plain", String(task.id));
        e.dataTransfer.effectAllowed = "move";
        onDragStart?.(task.id);
      }}
      onDragEnd={() => onDragEnd?.()}
      onClick={() => onOpen(task.id)}
      onKeyDown={(e: KeyboardEvent<HTMLElement>) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onOpen(task.id);
        }
      }}
    >
      {task.tags.length > 0 ? (
        <div className="label-bars">
          {task.tags.slice(0, MAX_LABELS).map((g) => (
            <span key={g} className="label-bar" title={g} style={{ background: tagColor(g) }} />
          ))}
        </div>
      ) : null}
      <div className="card-title">{task.title}</div>
      {excerpt ? <p className="card-desc">{excerpt}</p> : null}
      <div className="card-foot">
        <PriorityDot priority={task.priority} />
        <span className="card-pri">{PRI_LABEL[task.priority] ?? task.priority}</span>
        <span className="spacer" />
        <span className="card-assignee">{task.assignee ? "@" + task.assignee : "unassigned"}</span>
      </div>
    </article>
  );
}
