import type { TaskView } from "../api";
import { PRI_LABEL, PRI_ORDER, STATUS_LABEL, STATUS_ORDER } from "../constants";

export type FilterKey = "status" | "priority" | "tag" | "assignee";

interface FilterRowProps {
  all: TaskView[];
  view: "list" | "board";
  status: string | null;
  priority: string | null;
  tag: string | null;
  assignee: string | null;
  query: string;
  /** Tasks left after the filters and the search text. */
  visibleCount: number;
  onToggle: (key: FilterKey, value: string) => void;
  onClear: (key: FilterKey) => void;
  onReset: () => void;
}

function uniqueSorted(values: string[]): string[] {
  const seen = new Map<string, string>();
  values.forEach((v) => {
    if (v && !seen.has(v.toLowerCase())) seen.set(v.toLowerCase(), v);
  });
  return [...seen.values()].sort((a, b) => a.localeCompare(b));
}

// FilterRow replaces the sidebar and the command bar (design.md D4). Every
// control toggles one of the reducer's server-side filter keys; the chosen
// value is sent lowercase, as `backlog list` compares it.
export function FilterRow({
  all,
  view,
  status,
  priority,
  tag,
  assignee,
  query,
  visibleCount,
  onToggle,
  onClear,
  onReset,
}: FilterRowProps) {
  const tags = uniqueSorted(all.flatMap((t) => t.tags));
  const assignees = uniqueSorted(all.map((t) => t.assignee));
  const anyActive = Boolean(status || priority || tag || assignee || query.trim());

  const toggle = (key: FilterKey, value: string, active: boolean, label: string) => (
    <button
      key={value}
      type="button"
      className={"text-toggle" + (active ? " is-active" : "")}
      aria-pressed={active}
      onClick={() => onToggle(key, value)}
    >
      {label}
    </button>
  );

  const chip = (key: FilterKey, value: string, active: boolean, label: string) => (
    <button
      key={value}
      type="button"
      className={"chip" + (active ? " is-active" : "")}
      aria-pressed={active}
      onClick={() => onToggle(key, value.toLowerCase())}
    >
      {label}
    </button>
  );

  return (
    <div className="filter-row" role="toolbar" aria-label="Filters">
      <div className="filter-group" role="group" aria-label="Priority">
        <span className="filter-label">Priority</span>
        <button
          type="button"
          className={"text-toggle" + (priority === null ? " is-active" : "")}
          aria-pressed={priority === null}
          onClick={() => onClear("priority")}
        >
          All
        </button>
        {PRI_ORDER.map((k) => toggle("priority", k, priority === k, PRI_LABEL[k]))}
      </div>

      {view === "list" ? (
        <>
          <span className="vdivider" />
          <div className="filter-group" role="group" aria-label="Status">
            <span className="filter-label">Status</span>
            {STATUS_ORDER.map((k) => toggle("status", k, status === k, STATUS_LABEL[k]))}
          </div>
        </>
      ) : null}

      {tags.length > 0 ? (
        <>
          <span className="vdivider" />
          <div className="filter-group" role="group" aria-label="Tags">
            <span className="filter-label">Tags</span>
            {tags.map((g) => chip("tag", g, tag === g.toLowerCase(), g))}
          </div>
        </>
      ) : null}

      {assignees.length > 0 ? (
        <>
          <span className="vdivider" />
          <div className="filter-group" role="group" aria-label="Assignee">
            <span className="filter-label">Assignee</span>
            {assignees.map((a) => chip("assignee", a, assignee === a.toLowerCase(), "@" + a))}
          </div>
        </>
      ) : null}

      <div className="spacer" />
      {anyActive ? (
        <button type="button" className="text-btn is-strong filter-reset" onClick={onReset}>
          Reset
        </button>
      ) : null}
      <span className="filter-count">
        {visibleCount} of {all.length} tasks
      </span>
    </div>
  );
}
