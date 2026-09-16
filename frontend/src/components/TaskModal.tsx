import { useState, type ReactNode } from "react";

import type { LinkView, PatchTaskBody, TaskView } from "../api";
import { copyText } from "../clipboard";
import {
  padId,
  PRI_LABEL,
  PRI_ORDER,
  splitList,
  STATUS_LABEL,
  STATUS_ORDER,
  tagChipStyle,
} from "../constants";
import { md } from "../markdown";
import { ChevronDownIcon, CloseIcon } from "./icons";
import { InlineSelect, InlineText } from "./InlineFields";
import { LayeredDialog } from "./LayeredDialog";
import { LinkPicker } from "./LinkPicker";
import { MarkdownEditor } from "./MarkdownEditor";
import { PriorityDot } from "./TaskCard";

interface TaskModalProps {
  task: TaskView;
  // The full task set, threaded to the links picker.
  tasks: TaskView[];
  error: string;
  onClose: () => void;
  // Expected to reject on failure, after the caller has already surfaced the
  // error — every inline control's own catch only reverts its display.
  onPatch: (body: PatchTaskBody) => Promise<void>;
  onDelete: () => void;
  // Jumps to another task from a link row, replacing what is open.
  onOpenLink: (id: number) => void;
  // Reports a client-side validation problem (e.g. an empty title) the same
  // way a rejected PATCH would.
  onError: (message: string) => void;
  // The in-app decline-reason prompt, awaited before a status PATCH to
  // `declined` — mirrors the board drag's handleMove.
  askReason: () => Promise<string | null>;
}

interface SectionProps {
  title: string;
  defaultOpen?: boolean;
  children: ReactNode;
}

// Section is the modal's collapsible heading (design.md D11) — local state
// only, no persistence.
function Section({ title, defaultOpen = true, children }: SectionProps) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className="modal-section">
      <button
        type="button"
        className={"modal-section-head" + (open ? "" : " is-collapsed")}
        aria-expanded={open}
        onClick={() => setOpen((o) => !o)}
      >
        <ChevronDownIcon size={12} />
        {title}
      </button>
      {open ? <div className="modal-section-body">{children}</div> : null}
    </div>
  );
}

const PRIORITY_OPTIONS = PRI_ORDER.map((p) => ({
  value: p,
  label: PRI_LABEL[p],
  render: (
    <>
      <PriorityDot priority={p} />
      {PRI_LABEL[p]}
    </>
  ),
}));

const STATUS_OPTIONS = STATUS_ORDER.map((s) => ({ value: s, label: STATUS_LABEL[s] }));

// TaskModal is the task detail window (design.md D11, mockup 05): every field
// but links edits in place and PATCHes only what changed, replacing the old
// TaskDialog/ReadView/EditForm/MetadataAside read/edit shell.
export function TaskModal({
  task,
  tasks,
  error,
  onClose,
  onPatch,
  onDelete,
  onOpenLink,
  onError,
  askReason,
}: TaskModalProps) {
  const [copyMsg, setCopyMsg] = useState<string | null>(null);
  const [descEditing, setDescEditing] = useState(false);
  const { source, refs, created, author } = task.metadata;

  function flashCopy(msg: string) {
    setCopyMsg(msg);
    window.setTimeout(() => setCopyMsg((m) => (m === msg ? null : m)), 1600);
  }

  async function copyId() {
    const ok = await copyText(padId(task.id));
    flashCopy(ok ? "Task id copied" : "Could not copy to the clipboard");
  }

  async function copyCommand() {
    const ok = await copyText(`backlog show ${task.id}`);
    flashCopy(ok ? "Command copied" : "Could not copy to the clipboard");
  }

  async function commitTitle(v: string) {
    const t = v.trim();
    if (!t) {
      onError("A title is required.");
      throw new Error("empty title");
    }
    await onPatch({ title: t });
  }

  async function commitStatus(v: string) {
    if (v === task.status) return;
    if (v === "declined") {
      const reason = await askReason();
      if (reason === null) throw new Error("cancelled");
      await onPatch({ status: v, reason });
      return;
    }
    // Leaving `declined` clears the reason, matching `backlog set`.
    const body: PatchTaskBody = { status: v };
    if (task.status === "declined") body.reason = "";
    await onPatch(body);
  }

  async function saveDescription(v: string) {
    await onPatch({ description: v });
    setDescEditing(false);
  }

  function onLinksChange(links: LinkView[]) {
    onPatch({ links }).catch(() => {});
  }

  return (
    <LayeredDialog label={task.title} className="task-modal" onClose={onClose}>
      <div className="modal-head">
        <button type="button" className="modal-id" onClick={copyId}>
          {padId(task.id)}
        </button>
        <InlineText
          value={task.title}
          ariaLabel="Title"
          className="modal-title"
          onCommit={commitTitle}
        />
        <div className="modal-actions">
          <button type="button" className="text-btn" onClick={copyCommand}>
            Copy command
          </button>
          <button type="button" className="text-btn is-danger" onClick={onDelete}>
            Delete
          </button>
          <button type="button" className="icon-btn" aria-label="Close" onClick={onClose}>
            <CloseIcon />
          </button>
        </div>
      </div>

      {copyMsg ? <div className="modal-copy-msg">{copyMsg}</div> : null}
      {error ? <div className="modal-error">{error}</div> : null}

      <div className="modal-body">
        <Section title="Details">
          <div className="details-grid">
            <div className="details-key">Status</div>
            <div className="details-value">
              <InlineSelect
                value={task.status}
                options={STATUS_OPTIONS}
                ariaLabel="Status"
                onCommit={commitStatus}
              />
            </div>

            <div className="details-key">Priority</div>
            <div className="details-value">
              <InlineSelect
                value={task.priority}
                options={PRIORITY_OPTIONS}
                ariaLabel="Priority"
                onCommit={(v) => onPatch({ priority: v })}
              />
            </div>

            <div className="details-key">Assignee</div>
            <div className="details-value">
              <InlineText
                value={task.assignee}
                placeholder="unassigned"
                ariaLabel="Assignee"
                onCommit={(v) => onPatch({ assignee: v.trim() })}
              />
            </div>

            <div className="details-key">Tags</div>
            <div className="details-value">
              <InlineText
                value={task.tags.join(", ")}
                placeholder="no tags"
                ariaLabel="Tags"
                onCommit={(v) => onPatch({ tags: splitList(v) })}
                renderValue={() => (
                  <span className="tag-chips-rest">
                    {task.tags.map((g) => (
                      <span key={g} className="tag-chip-rest" style={tagChipStyle(g)}>
                        {g}
                      </span>
                    ))}
                  </span>
                )}
              />
            </div>

            <div className="details-key">Source</div>
            <div className="details-value is-muted">
              {source.files.length ? source.files.join("  ·  ") : "none"}
            </div>

            <div className="details-key">Branch · commit</div>
            <div className="details-value is-muted">
              {source.branch || "—"} · {source.commit ? source.commit.slice(0, 12) : "—"}
            </div>

            <div className="details-key">Created</div>
            <div className="details-value is-muted">{created}</div>

            <div className="details-key">Author</div>
            <div className="details-value is-muted">{author}</div>

            <div className="details-key">Refs</div>
            <div className="details-value is-muted">{refs.length ? refs.join("  ·  ") : "none"}</div>

            {task.status === "declined" ? (
              <>
                <div className="details-key">Reason</div>
                <div className="details-value">{task.reason}</div>
              </>
            ) : null}
          </div>
        </Section>

        <Section title="Description">
          {descEditing ? (
            <MarkdownEditor
              value={task.description}
              onSave={saveDescription}
              onCancel={() => setDescEditing(false)}
            />
          ) : (
            <>
              {task.description ? (
                <div className="md" dangerouslySetInnerHTML={{ __html: md(task.description) }} />
              ) : (
                <div className="md-empty">No description.</div>
              )}
              <button type="button" className="text-btn" onClick={() => setDescEditing(true)}>
                Edit
              </button>
            </>
          )}
        </Section>

        <Section title="Linked tasks">
          <LinkPicker
            links={task.links}
            tasks={tasks}
            excludeId={task.id}
            onChange={onLinksChange}
            onOpen={onOpenLink}
          />
        </Section>
      </div>
    </LayeredDialog>
  );
}
