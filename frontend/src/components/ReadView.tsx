import type { TaskView } from "../api";
import { priBadge, statusMeta } from "../constants";
import { md } from "../markdown";

export function ReadView({ task }: { task: TaskView }) {
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
    </>
  );
}
