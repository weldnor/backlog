import { useMemo, useRef, useState } from "react";

import type { LinkView, TaskView } from "../api";
import { LINK_TYPES, padId } from "../constants";
import { CloseIcon, PlusIcon } from "./icons";

interface LinksEditorProps {
  links: LinkView[];
  // The full task set, used to search by id/title and to label existing
  // links with a title instead of a bare id.
  tasks: TaskView[];
  // Omitted for a task still being created, which has no id yet to exclude.
  excludeId?: number;
  onChange: (links: LinkView[]) => void;
}

function taskLabel(tasks: TaskView[], id: number): string {
  const t = tasks.find((x) => x.id === id);
  return t ? t.title : "unknown task";
}

// LinksEditor replaces the old "type:id" free-text field with a typed
// picker: choose a link type, then search the task list by id or title and
// pick a target. Existing links render as removable rows labelled with the
// target's current title, so a stale id is obvious at a glance.
export function LinksEditor({ links, tasks, excludeId, onChange }: LinksEditorProps) {
  const [type, setType] = useState<string>(LINK_TYPES[0]);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [activeIdx, setActiveIdx] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const candidates = useMemo(() => {
    const q = query.trim().toLowerCase();
    return tasks
      .filter((t) => t.id !== excludeId)
      .filter((t) => !links.some((l) => l.type === type && l.id === t.id))
      .filter((t) => {
        if (!q) return true;
        return (
          t.title.toLowerCase().includes(q) ||
          String(t.id).includes(q) ||
          padId(t.id).includes(q)
        );
      })
      .slice(0, 8);
  }, [tasks, excludeId, links, type, query]);

  function addLink(id: number) {
    onChange([...links, { type, id }]);
    setQuery("");
    setOpen(false);
    setActiveIdx(0);
    inputRef.current?.focus();
  }

  function removeLink(l: LinkView) {
    onChange(links.filter((x) => !(x.type === l.type && x.id === l.id)));
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (!open || candidates.length === 0) return;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActiveIdx((i) => (i + 1) % candidates.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActiveIdx((i) => (i - 1 + candidates.length) % candidates.length);
    } else if (e.key === "Enter") {
      e.preventDefault();
      addLink(candidates[activeIdx].id);
    } else if (e.key === "Escape") {
      setOpen(false);
    }
  }

  return (
    <div className="field" style={{ marginBottom: 16 }}>
      <label>
        Links — typed references to other tasks, recorded on this task only,
        never mirrored back
      </label>

      {links.length > 0 ? (
        <div className="links-editor-list">
          {links.map((l) => (
            <div className="links-editor-row" key={l.type + ":" + l.id}>
              <span className="tag tag-neutral links-editor-type">{l.type}</span>
              <span className="links-editor-target">
                #{padId(l.id)} · {taskLabel(tasks, l.id)}
              </span>
              <button
                type="button"
                className="btn-icon-plain"
                aria-label={`Remove ${l.type} link to #${padId(l.id)}`}
                onClick={() => removeLink(l)}
              >
                <CloseIcon size={12} />
              </button>
            </div>
          ))}
        </div>
      ) : null}

      <div className="links-editor-add">
        <select
          className="input links-editor-type-select"
          value={type}
          onChange={(e) => setType(e.target.value)}
          aria-label="Link type"
        >
          {LINK_TYPES.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>

        <div className="links-editor-combobox">
          <input
            ref={inputRef}
            className="input"
            value={query}
            placeholder="search by title or #id…"
            onChange={(e) => {
              setQuery(e.target.value);
              setOpen(true);
              setActiveIdx(0);
            }}
            onFocus={() => setOpen(true)}
            onBlur={() => setTimeout(() => setOpen(false), 120)}
            onKeyDown={onKeyDown}
          />
          {open && candidates.length > 0 ? (
            <ul className="links-editor-options" role="listbox">
              {candidates.map((t, i) => (
                <li
                  key={t.id}
                  role="option"
                  aria-selected={i === activeIdx}
                  className={"links-editor-option" + (i === activeIdx ? " is-active" : "")}
                  onMouseDown={(e) => {
                    // Beat the input's blur so the click still registers.
                    e.preventDefault();
                    addLink(t.id);
                  }}
                  onMouseEnter={() => setActiveIdx(i)}
                >
                  <span className="links-editor-option-id">#{padId(t.id)}</span>
                  <span className="links-editor-option-title">{t.title}</span>
                </li>
              ))}
            </ul>
          ) : null}
        </div>

        <button
          type="button"
          className="btn btn-secondary"
          disabled={candidates.length === 0}
          onClick={() => candidates[0] && addLink(candidates[0].id)}
        >
          Add
          <PlusIcon size={12} />
        </button>
      </div>
    </div>
  );
}
