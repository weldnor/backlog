import type { TaskView } from "../api";
import { padId } from "../constants";

export function MetadataAside({ task }: { task: TaskView }) {
  const { source, refs, created, author } = task.metadata;
  const branch = source.branch || "—";
  const commit = source.commit ? source.commit.slice(0, 12) : "—";

  return (
    <aside className="dialog-aside">
      <div className="aside-heading">Metadata — tool-owned</div>
      <div className="meta-list">
        <div>
          <div className="meta-key">ID</div>
          {padId(task.id)}
        </div>
        <div>
          <div className="meta-key">CREATED</div>
          {created}
        </div>
        <div>
          <div className="meta-key">AUTHOR</div>
          {author}
        </div>
        <div>
          <div className="meta-key">SOURCE · BRANCH / COMMIT</div>
          <span className="meta-mono">
            {branch} · {commit}
          </span>
        </div>
        <div>
          <div className="meta-key">SOURCE · FILES</div>
          {source.files.length ? (
            source.files.map((f) => (
              <div key={f} className="meta-mono">
                {f}
              </div>
            ))
          ) : (
            <div className="meta-none">none</div>
          )}
        </div>
        <div>
          <div className="meta-key">REFS — VERBATIM, NEVER RESOLVED</div>
          {refs.length ? (
            refs.map((r) => (
              <div key={r} className="meta-ref">
                {r}
              </div>
            ))
          ) : (
            <div className="meta-none">none</div>
          )}
        </div>
        <div className="hr" style={{ margin: "2px 0" }} />
        <div className="meta-note">
          The key set under <code>metadata</code> is closed — an unrecognised key
          is an error, which is what catches a typo like <code>creted</code>.
          This panel is read-only for that reason.
        </div>
      </div>
    </aside>
  );
}
