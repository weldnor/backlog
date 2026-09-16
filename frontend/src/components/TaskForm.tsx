import { useRef, useState, type KeyboardEvent } from "react";

import {
  ApiError,
  createTaskWithStatus,
  CreateStatusError,
  deleteTask,
  type CreateTaskBody,
  type LinkView,
  type TaskView,
} from "../api";
import { MOD_KEY, padId, PRI_DOT, PRI_LABEL, PRI_ORDER, splitList, STATUS_LABEL, tagChipStyle } from "../constants";
import { DRAFT_STATUSES, type ParsedDraft } from "../tokens";
import type { ToastSpec } from "../useToast";
import { tagSuggestions } from "./CaptureDraft";
import { ChevronDownIcon, CloseIcon } from "./icons";
import { LayeredDialog } from "./LayeredDialog";
import { LinksEditor } from "./LinksEditor";

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

type AcItem = { kind: "tag"; name: string } | { kind: "create"; name: string };

interface TaskFormProps {
  /** The full task set, threaded to the links picker and the tag autocomplete. */
  tasks: TaskView[];
  /** Prefill from a draft's Shift+Enter (design.md D10) — title/priority/tags/assignee. */
  prefill?: Partial<ParsedDraft> | null;
  onClose: () => void;
  /** Call after any successful create so the board/list refetches. */
  onCreated: () => void;
  /** Opens a task's detail — used by the create toast's "Open". */
  onOpen: (id: number) => void;
  showToast: (spec: ToastSpec) => void;
}

// TaskForm is the full create form (design.md D10, mockup C5): every field
// `backlog add` accepts, plus an initial status. It manages its own
// submission (create-then-maybe-patch, like CaptureDraft) and its own layer,
// rather than living inside TaskDialog's read/edit shell.
export function TaskForm({ tasks, prefill, onClose, onCreated, onOpen, showToast }: TaskFormProps) {
  const [title, setTitle] = useState(prefill?.title ?? "");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<string>(() =>
    prefill?.status && (DRAFT_STATUSES as readonly string[]).includes(prefill.status)
      ? prefill.status
      : "new",
  );
  const [priority, setPriority] = useState(prefill?.priority ?? "medium");
  const [assignee, setAssignee] = useState(prefill?.assignee ?? "");
  const [tags, setTags] = useState<string[]>(prefill?.tags ?? []);
  const [tagInput, setTagInput] = useState("");
  const [tagActive, setTagActive] = useState(0);
  const [tagAcClosed, setTagAcClosed] = useState(false);
  const [files, setFiles] = useState("");
  const [refs, setRefs] = useState("");
  const [links, setLinks] = useState<LinkView[]>([]);
  const [moreOpen, setMoreOpen] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const titleRef = useRef<HTMLInputElement>(null);

  const trimmedTagInput = tagInput.trim();
  const tagSugg = trimmedTagInput ? tagSuggestions(trimmedTagInput, tasks) : [];
  const tagItems: AcItem[] = tagSugg
    .filter((s) => !tags.some((t) => t.toLowerCase() === s.name.toLowerCase()))
    .map((s) => ({ kind: "tag", name: s.name }));
  if (
    trimmedTagInput &&
    !tagSugg.some((s) => s.name.toLowerCase() === trimmedTagInput.toLowerCase()) &&
    !tags.some((t) => t.toLowerCase() === trimmedTagInput.toLowerCase())
  ) {
    tagItems.push({ kind: "create", name: trimmedTagInput });
  }
  const tagAcOpen = !tagAcClosed && tagItems.length > 0;

  function addTag(name: string) {
    setTags((ts) => (ts.some((t) => t.toLowerCase() === name.toLowerCase()) ? ts : [...ts, name]));
    setTagInput("");
    setTagActive(0);
  }

  function removeTag(name: string) {
    setTags((ts) => ts.filter((t) => t !== name));
  }

  function onTagKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (tagAcOpen && ["ArrowDown", "ArrowUp", "Tab", "Enter"].includes(e.key)) {
      e.preventDefault();
      if (e.key === "ArrowDown") setTagActive((i) => (i + 1) % tagItems.length);
      else if (e.key === "ArrowUp") setTagActive((i) => (i - 1 + tagItems.length) % tagItems.length);
      else if (tagItems[tagActive]) addTag(tagItems[tagActive].name);
      return;
    }
    if (e.key === "Escape" && tagAcOpen) {
      e.preventDefault();
      e.stopPropagation();
      setTagAcClosed(true);
      return;
    }
    if (e.key === "Enter" && trimmedTagInput) {
      e.preventDefault();
      addTag(trimmedTagInput);
    }
  }

  function resetForAnother() {
    setTitle("");
    setDescription("");
    setStatus("new");
    setPriority("medium");
    setAssignee("");
    setTags([]);
    setTagInput("");
    setFiles("");
    setRefs("");
    setLinks([]);
    setMoreOpen(false);
    titleRef.current?.focus();
  }

  function announce(created: TaskView, extra?: string) {
    showToast({
      text:
        `Task ${padId(created.id)} created in ${STATUS_LABEL[created.status] ?? created.status}` +
        (extra ? ` — ${extra}` : ""),
      actions: extra
        ? [{ label: "Open", onClick: () => onOpen(created.id) }]
        : [
            { label: "Open", onClick: () => onOpen(created.id) },
            {
              label: "Undo",
              onClick: () => {
                deleteTask(created.id)
                  .then(() => onCreated())
                  .catch(() => {});
              },
            },
          ],
    });
  }

  async function submit(addAnother: boolean) {
    const t = title.trim();
    if (!t) {
      setError("A title is required.");
      titleRef.current?.focus();
      return;
    }
    setError("");
    setSubmitting(true);
    const body: CreateTaskBody = {
      title: t,
      description,
      tags,
      priority,
      assignee: assignee.trim(),
      files: splitList(files),
      refs: splitList(refs),
      links,
    };
    try {
      const created = await createTaskWithStatus(body, status);
      setSubmitting(false);
      onCreated();
      announce(created);
      if (addAnother) resetForAnother();
      else onClose();
    } catch (err) {
      setSubmitting(false);
      if (err instanceof CreateStatusError) {
        onCreated();
        announce(err.task, err.message);
        if (addAnother) resetForAnother();
        else onClose();
        return;
      }
      setError(err instanceof ApiError ? err.message : message(err));
    }
  }

  function onFormKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      submit(false);
    }
  }

  return (
    <LayeredDialog
      label="New task"
      className="task-form-dialog"
      onClose={onClose}
      initialFocus={titleRef}
    >
      <div className="form" onKeyDown={onFormKeyDown}>
        <div className="form-head">
          <span className="form-title">New task</span>
          <button type="button" className="icon-btn" aria-label="Close" onClick={onClose}>
            <CloseIcon />
          </button>
        </div>

        <input
          ref={titleRef}
          className="field-input form-title-input"
          placeholder="Task title"
          value={title}
          disabled={submitting}
          onChange={(e) => {
            setTitle(e.target.value);
            setError("");
          }}
        />
        {error ? <div className="draft-error">{error}</div> : null}

        <div className="form-row">
          <div className="form-field">
            <span className="field-label">Status</span>
            <div className="text-toggle-group" role="group" aria-label="Status">
              {DRAFT_STATUSES.map((s) => (
                <button
                  key={s}
                  type="button"
                  className={"text-toggle" + (status === s ? " is-active" : "")}
                  aria-pressed={status === s}
                  onClick={() => setStatus(s)}
                >
                  {STATUS_LABEL[s]}
                </button>
              ))}
            </div>
          </div>
          <div className="form-field">
            <span className="field-label">Priority</span>
            <div className="text-toggle-group" role="group" aria-label="Priority">
              {PRI_ORDER.map((p) => (
                <button
                  key={p}
                  type="button"
                  className={"pri-toggle" + (priority === p ? " is-active" : "")}
                  aria-pressed={priority === p}
                  onClick={() => setPriority(p)}
                >
                  <span className="pri-dot" style={{ background: PRI_DOT[p] }} />
                  {PRI_LABEL[p]}
                </button>
              ))}
            </div>
          </div>
        </div>

        <div className="form-field">
          <label className="field-label" htmlFor="task-form-assignee">
            Assignee
          </label>
          <input
            id="task-form-assignee"
            className="field-input"
            placeholder="unassigned"
            value={assignee}
            onChange={(e) => setAssignee(e.target.value)}
          />
        </div>

        <div className="form-field">
          <span className="field-label">Tags</span>
          <div className="tag-editor">
            {tags.map((g) => (
              <span key={g} className="tag-editor-chip" style={tagChipStyle(g)}>
                {g}
                <button
                  type="button"
                  className="tag-editor-chip-remove"
                  aria-label={`Remove tag ${g}`}
                  onClick={() => removeTag(g)}
                >
                  <CloseIcon size={10} />
                </button>
              </span>
            ))}
            <div className="tag-editor-input-wrap">
              <input
                className="field-input tag-editor-input"
                placeholder="+ tag"
                value={tagInput}
                onChange={(e) => {
                  setTagInput(e.target.value);
                  setTagAcClosed(false);
                }}
                onKeyDown={onTagKeyDown}
              />
              {tagAcOpen ? (
                <ul className="autocomplete-list" role="listbox">
                  {tagItems.map((item, i) => (
                    <li
                      key={item.kind + ":" + item.name}
                      role="option"
                      aria-selected={i === tagActive}
                      className={"autocomplete-item" + (i === tagActive ? " is-active" : "")}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        addTag(item.name);
                      }}
                      onMouseEnter={() => setTagActive(i)}
                    >
                      {item.kind === "create" ? `Create tag «${item.name}»` : item.name}
                    </li>
                  ))}
                </ul>
              ) : null}
            </div>
          </div>
        </div>

        <div className="form-field">
          <label className="field-label" htmlFor="task-form-desc">
            Description
          </label>
          <textarea
            id="task-form-desc"
            className="field-input form-textarea"
            rows={8}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>

        <button
          type="button"
          className="text-btn more-toggle"
          aria-expanded={moreOpen}
          onClick={() => setMoreOpen((o) => !o)}
        >
          <ChevronDownIcon /> More
        </button>

        {moreOpen ? (
          <div className="more-section">
            <div className="form-field">
              <label className="field-label" htmlFor="task-form-files">
                Source files
              </label>
              <input
                id="task-form-files"
                className="field-input"
                placeholder="comma separated"
                value={files}
                onChange={(e) => setFiles(e.target.value)}
              />
            </div>
            <div className="form-field">
              <label className="field-label" htmlFor="task-form-refs">
                References
              </label>
              <input
                id="task-form-refs"
                className="field-input"
                placeholder="comma separated"
                value={refs}
                onChange={(e) => setRefs(e.target.value)}
              />
            </div>
            <LinksEditor links={links} tasks={tasks} onChange={setLinks} />
          </div>
        ) : null}

        <div className="form-footer">
          <button type="button" className="pill-btn" disabled={submitting} onClick={() => submit(false)}>
            Create task
          </button>
          <button
            type="button"
            className="text-btn"
            disabled={submitting}
            onClick={() => submit(true)}
          >
            Create and add another
          </button>
          <span className="form-hint">{MOD_KEY}↵ create · Esc cancel</span>
        </div>
      </div>
    </LayeredDialog>
  );
}
