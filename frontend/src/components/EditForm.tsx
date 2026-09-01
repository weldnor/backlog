import { useState } from "react";

import type { CreateTaskBody, PatchTaskBody, TaskView } from "../api";
import { PRI_META, PRI_ORDER, splitList, STATUS_ORDER } from "../constants";
import { SaveIcon } from "./icons";

interface Draft {
  title: string;
  status: string;
  priority: string;
  tags: string;
  body: string;
  reason: string;
  files: string;
  refs: string;
}

function draftFor(task?: TaskView): Draft {
  if (!task) {
    return {
      title: "",
      status: "todo",
      priority: "medium",
      tags: "",
      body: "",
      reason: "",
      files: "",
      refs: "",
    };
  }
  return {
    title: task.title,
    status: task.status,
    priority: task.priority,
    tags: task.tags.join(", "),
    body: task.description,
    reason: task.reason || "",
    files: "",
    refs: "",
  };
}

interface EditFormProps {
  mode: "edit" | "create";
  task?: TaskView;
  error: string;
  onCancel: () => void;
  onCreate: (body: CreateTaskBody) => void;
  onPatch: (body: PatchTaskBody) => void;
  onDelete?: () => void;
}

export function EditForm({
  mode,
  task,
  error,
  onCancel,
  onCreate,
  onPatch,
  onDelete,
}: EditFormProps) {
  const isCreate = mode === "create";
  const [draft, setDraft] = useState<Draft>(() => draftFor(task));

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) =>
    setDraft((d) => ({ ...d, [key]: value }));

  function setStatus(value: string) {
    setDraft((d) => ({
      ...d,
      status: value,
      // Leaving declined clears the reason in the draft, matching the server
      // and `backlog set`.
      reason: value === "declined" ? d.reason : "",
    }));
  }

  function submit() {
    if (isCreate) {
      onCreate({
        title: draft.title,
        description: draft.body,
        tags: splitList(draft.tags),
        priority: draft.priority,
        files: splitList(draft.files),
        refs: splitList(draft.refs),
      });
    } else {
      onPatch({
        title: draft.title,
        description: draft.body,
        tags: splitList(draft.tags),
        priority: draft.priority,
        status: draft.status,
        reason: draft.status === "declined" ? draft.reason : "",
      });
    }
  }

  return (
    <>
      <div className="field" style={{ marginBottom: 16 }}>
        <label>Title</label>
        <input
          className="input"
          value={draft.title}
          onChange={(e) => set("title", e.target.value)}
          style={{ fontSize: 16, fontWeight: 600, minHeight: 42 }}
        />
      </div>

      <div className="edit-row">
        <div className="field">
          <label>Status</label>
          <div className="seg">
            {STATUS_ORDER.map((k) => (
              <label key={k} className="seg-opt">
                <input
                  type="radio"
                  name="draft-status"
                  value={k}
                  checked={draft.status === k}
                  onChange={() => setStatus(k)}
                />
                {k.toUpperCase()}
              </label>
            ))}
          </div>
        </div>
        <div className="field">
          <label>Priority — severity</label>
          <div className="seg">
            {PRI_ORDER.map((k) => (
              <label key={k} className="seg-opt">
                <input
                  type="radio"
                  name="draft-priority"
                  value={k}
                  checked={draft.priority === k}
                  onChange={() => set("priority", k)}
                />
                {PRI_META[k].label}
              </label>
            ))}
          </div>
        </div>
      </div>

      <div className="field" style={{ marginBottom: 16 }}>
        <label>Tags — comma separated</label>
        <input
          className="input"
          value={draft.tags}
          onChange={(e) => set("tags", e.target.value)}
          style={{ fontFamily: "var(--font-mono)", fontSize: 13 }}
        />
      </div>

      {isCreate ? (
        <div className="edit-row">
          <div className="field">
            <label>Files — comma separated</label>
            <input
              className="input"
              value={draft.files}
              onChange={(e) => set("files", e.target.value)}
              style={{ fontFamily: "var(--font-mono)", fontSize: 13 }}
            />
          </div>
          <div className="field">
            <label>Refs — comma separated</label>
            <input
              className="input"
              value={draft.refs}
              onChange={(e) => set("refs", e.target.value)}
              style={{ fontFamily: "var(--font-mono)", fontSize: 13 }}
            />
          </div>
        </div>
      ) : null}

      {draft.status === "declined" ? (
        <div className="field" style={{ marginBottom: 16 }}>
          <label style={{ color: "var(--color-accent-700)" }}>
            Reason — required when declining
          </label>
          <input
            className="input has-error"
            value={draft.reason}
            onChange={(e) => set("reason", e.target.value)}
            placeholder="why this finding will not be acted on"
          />
        </div>
      ) : null}

      <div className="desc-label">
        <span className="k">DESCRIPTION</span>
        <span className="v">markdown · the file body, preserved verbatim</span>
      </div>
      <textarea
        className="input"
        rows={12}
        value={draft.body}
        onChange={(e) => set("body", e.target.value)}
        style={{ minHeight: 230 }}
      />

      {error ? <div className="error-note">{error}</div> : null}

      <div className="edit-actions">
        <button className="btn btn-primary" onClick={submit}>
          Save · writes one file
          <SaveIcon />
        </button>
        <button className="btn btn-secondary" onClick={onCancel}>
          Cancel
        </button>
        {!isCreate && onDelete ? (
          <button className="btn btn-danger" onClick={onDelete}>
            Delete
          </button>
        ) : null}
      </div>
    </>
  );
}
