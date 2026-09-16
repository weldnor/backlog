import type { KeyboardEvent } from "react";

import type { TaskView } from "../api";
import { descExcerpt, padId, PRI_LABEL, STATUS_LABEL, tagColor } from "../constants";
import { PriorityDot } from "./TaskCard";

interface ListViewProps {
  tasks: TaskView[];
  openId: number | null;
  onOpen: (id: number) => void;
}

function activateOnKey(e: KeyboardEvent<HTMLDivElement>, run: () => void) {
  if (e.key === "Enter" || e.key === " ") {
    e.preventDefault();
    run();
  }
}

// ListView is the DS-styled list (design.md D15): no mockup frame exists for
// it, so it borrows the card's typography and hover ring directly. The tag
// and status columns hide under 720px (style.css media query).
export function ListView({ tasks, openId, onOpen }: ListViewProps) {
  return (
    <div className="list" role="list">
      {tasks.map((t) => {
        const excerpt = descExcerpt(t.description);
        return (
          <div
            key={t.id}
            role="button"
            tabIndex={0}
            className={"list-row" + (t.id === openId ? " is-open" : "")}
            onClick={() => onOpen(t.id)}
            onKeyDown={(e) => activateOnKey(e, () => onOpen(t.id))}
          >
            <span className="list-id">{padId(t.id)}</span>
            <div className="list-main">
              <div className="list-title">{t.title}</div>
              {excerpt ? <div className="list-desc">{excerpt}</div> : null}
            </div>
            <div className="list-tags">
              {t.tags.map((g) => (
                <span key={g} className="label-bar" title={g} style={{ background: tagColor(g) }} />
              ))}
            </div>
            <div className="list-pri">
              <PriorityDot priority={t.priority} />
              {PRI_LABEL[t.priority] ?? t.priority}
            </div>
            <div className="list-assignee">{t.assignee ? "@" + t.assignee : "unassigned"}</div>
            <div className="list-status">{STATUS_LABEL[t.status] ?? t.status}</div>
          </div>
        );
      })}
    </div>
  );
}
