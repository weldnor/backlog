import type { TaskView } from "../api";
import { PRI_ORDER, STATUS_ORDER } from "../constants";

type FilterKey = "status" | "priority" | "tag";

interface SidebarProps {
  all: TaskView[];
  status: string | null;
  priority: string | null;
  tag: string | null;
  onPick: (key: FilterKey, value: string) => void;
}

function countBy(tasks: TaskView[], pick: (t: TaskView) => string, keys: readonly string[]) {
  const counts: Record<string, number> = {};
  keys.forEach((k) => (counts[k] = 0));
  tasks.forEach((t) => {
    const k = pick(t);
    counts[k] = (counts[k] || 0) + 1;
  });
  return counts;
}

export function Sidebar({ all, status, priority, tag, onPick }: SidebarProps) {
  const statusCounts = countBy(all, (t) => t.status, STATUS_ORDER);
  const priCounts = countBy(all, (t) => t.priority, PRI_ORDER);

  const tagSet: string[] = [];
  all.forEach((t) =>
    t.tags.forEach((g) => {
      if (!tagSet.includes(g)) tagSet.push(g);
    }),
  );
  tagSet.sort();

  return (
    <aside className="sidebar">
      <div className="side-group">
        <div className="side-heading">Status</div>
        <div className="side-list">
          {STATUS_ORDER.map((k) => (
            <button
              key={k}
              className={"side-item" + (status === k ? " is-active" : "")}
              onClick={() => onPick("status", k)}
            >
              <span>{k}</span>
              <span className="count">{statusCounts[k]}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="side-group">
        <div className="side-heading">
          Priority <span className="note">— severity, not schedule</span>
        </div>
        <div className="side-list">
          {PRI_ORDER.map((k) => (
            <button
              key={k}
              className={"side-item" + (priority === k ? " is-active" : "")}
              onClick={() => onPick("priority", k)}
            >
              <span>{k}</span>
              <span className="count">{priCounts[k]}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="side-group">
        <div className="side-heading">Tags</div>
        <div className="tag-cloud">
          {tagSet.map((g) => {
            const value = g.toLowerCase();
            return (
              <button
                key={g}
                className={"tag-chip" + (tag === value ? " is-active" : "")}
                onClick={() => onPick("tag", value)}
              >
                {g}
              </button>
            );
          })}
        </div>
      </div>
    </aside>
  );
}
