import type { TaskView } from "../api";
import { padId, priBadge, statusMeta } from "../constants";
import { md } from "../markdown";

interface ReadViewProps {
  task: TaskView;
  // Opens another task's dialog when its link chip is clicked. Optional so a
  // caller with nowhere to navigate to can simply omit it.
  onOpenLink?: (id: number) => void;
}

export function ReadView({ task, onOpenLink }: ReadViewProps) {
  const p = priBadge(task);
  const st = statusMeta(task);

  return (
    <>
      <div className="read-top">
        <span className={p.cls}>{p.label}</span>
        <span className={"read-status " + st.fg}>{st.headLabel}</span>
      </div>
      <h2 className="read-title">{task.title}</h2>
      <div className="hr" />
      <div
        className="md"
        dangerouslySetInnerHTML={{ __html: md(task.description) }}
      />
      {task.status === "declined" && task.reason ? (
        <div className="decline-callout">
          <div className="heading">DECLINE REASON — REQUIRED, AUDITABLE</div>
          <div className="text">{task.reason}</div>
        </div>
      ) : null}
      {task.links.length > 0 ? (
        <div className="links-block">
          <div className="heading">LINKS — RECORDED HERE ONLY, NOT MIRRORED BACK</div>
          <div className="links-list">
            {task.links.map((l) => (
              <button
                key={l.type + ":" + l.id}
                type="button"
                className="tag-chip"
                onClick={() => onOpenLink?.(l.id)}
              >
                {l.type} · #{padId(l.id)}
              </button>
            ))}
          </div>
        </div>
      ) : null}
    </>
  );
}
