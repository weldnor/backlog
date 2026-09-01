import type { KeyboardEvent } from "react";

import type { TaskView } from "../api";
import { padId, priBadge, statusMeta } from "../constants";
import { TagChips } from "./TagChips";

interface ListViewProps {
  tasks: TaskView[];
  openId: number | null;
  onOpen: (id: number) => void;
}

function activateOnKey(e: KeyboardEvent<HTMLTableRowElement>, run: () => void) {
  if (e.key === "Enter" || e.key === " ") {
    e.preventDefault();
    run();
  }
}

export function ListView({ tasks, openId, onOpen }: ListViewProps) {
  return (
    <>
      <table className="table">
        <colgroup>
          <col style={{ width: "70px" }} />
          <col className="col-priority" style={{ width: "104px" }} />
          <col />
          <col className="col-tags" style={{ width: "210px" }} />
          <col className="col-status" style={{ width: "110px" }} />
        </colgroup>
        <thead>
          <tr>
            <th>ID</th>
            <th>Priority</th>
            <th>Title</th>
            <th>Tags</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {tasks.map((t) => {
            const p = priBadge(t);
            const st = statusMeta(t);
            const files = t.metadata.source.files.length
              ? t.metadata.source.files.join("  ·  ")
              : "no source location";
            return (
              <tr
                key={t.id}
                className={t.id === openId ? "is-open" : ""}
                tabIndex={0}
                role="button"
                onClick={() => onOpen(t.id)}
                onKeyDown={(e) => activateOnKey(e, () => onOpen(t.id))}
              >
                <td>{padId(t.id)}</td>
                <td>
                  <span className={p.cls}>{p.label}</span>
                </td>
                <td>
                  <div className="row-title">{t.title}</div>
                  <div className="row-file">{files}</div>
                </td>
                <td>
                  <div className="row-tags">
                    <TagChips tags={t.tags} />
                  </div>
                </td>
                <td className={"status-label " + st.fg}>{st.headLabel}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
      {tasks.length === 0 ? (
        <div className="empty-note">
          No task matches. An empty result is an answer, not a failure —{" "}
          <code>exit 0</code>.
        </div>
      ) : null}
    </>
  );
}
