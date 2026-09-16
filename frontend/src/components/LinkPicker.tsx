import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";

import type { LinkView, TaskView } from "../api";
import { LINK_TYPES, padId, STATUS_LABEL } from "../constants";
import { CloseIcon } from "./icons";

interface LinkPickerProps {
  links: LinkView[];
  // The full task set, used to search targets and to label existing links
  // with the target's current title and status.
  tasks: TaskView[];
  // Omitted for a task still being created, which has no id yet to exclude.
  excludeId?: number;
  onChange: (links: LinkView[]) => void;
  // Opens a linked task's own detail, replacing what is open.
  onOpen: (id: number) => void;
}

function targetTask(tasks: TaskView[], id: number): TaskView | undefined {
  return tasks.find((t) => t.id === id);
}

// LinkPicker is the task modal's "Linked tasks" section (design.md D11,
// mockup 08): existing links as rows (type chip, id, a clickable title that
// opens the target, its status, and Remove), and a "Link a task" trigger that
// opens a popover — a link type select, then a search box over `tasks` — to
// add another. Every change is sent as the full new `links` array, the same
// PATCH shape an edit of links has always sent.
export function LinkPicker({ links, tasks, excludeId, onChange, onOpen }: LinkPickerProps) {
  const [pickerOpen, setPickerOpen] = useState(false);
  const [type, setType] = useState<string>(LINK_TYPES[0]);
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const rootRef = useRef<HTMLDivElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);

  const candidates = useMemo(() => {
    const q = query.trim().toLowerCase();
    return tasks
      .filter((t) => t.id !== excludeId)
      .filter((t) => !links.some((l) => l.type === type && l.id === t.id))
      .filter(
        (t) =>
          !q ||
          t.title.toLowerCase().includes(q) ||
          String(t.id).includes(q) ||
          padId(t.id).includes(q),
      )
      .slice(0, 8);
  }, [tasks, excludeId, links, type, query]);

  useEffect(() => {
    if (!pickerOpen) return;
    function onDocDown(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setPickerOpen(false);
    }
    document.addEventListener("mousedown", onDocDown);
    return () => document.removeEventListener("mousedown", onDocDown);
  }, [pickerOpen]);

  // The picker sits at the bottom of a scrolling modal; bring all of it —
  // including the result list — into view when it opens.
  useEffect(() => {
    if (pickerOpen) popoverRef.current?.scrollIntoView?.({ block: "nearest" });
  }, [pickerOpen]);

  function openPicker() {
    setQuery("");
    setActive(0);
    setPickerOpen(true);
  }

  function addLink(id: number) {
    onChange([...links, { type, id }]);
    setPickerOpen(false);
  }

  function removeLink(l: LinkView) {
    onChange(links.filter((x) => !(x.type === l.type && x.id === l.id)));
  }

  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (candidates.length) setActive((i) => (i + 1) % candidates.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      if (candidates.length) setActive((i) => (i - 1 + candidates.length) % candidates.length);
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (candidates[active]) addLink(candidates[active].id);
    } else if (e.key === "Escape") {
      // Close only the picker — stop the key from also reaching the task
      // modal's own Escape handler.
      e.preventDefault();
      e.stopPropagation();
      setPickerOpen(false);
    }
  }

  return (
    <div className="link-picker" ref={rootRef}>
      {links.length > 0 ? (
        <div className="linked-tasks-list">
          {links.map((l) => {
            const target = targetTask(tasks, l.id);
            return (
              <div className="linked-task-row" key={l.type + ":" + l.id}>
                <span className="link-type-chip">{l.type}</span>
                <span className="linked-task-id">#{padId(l.id)}</span>
                <button type="button" className="linked-task-title" onClick={() => onOpen(l.id)}>
                  {target ? target.title : "unknown task"}
                </button>
                <span className="linked-task-status">
                  {target ? (STATUS_LABEL[target.status] ?? target.status) : ""}
                </span>
                <button
                  type="button"
                  className="icon-btn"
                  aria-label={`Remove ${l.type} link to #${padId(l.id)}`}
                  onClick={() => removeLink(l)}
                >
                  <CloseIcon size={12} />
                </button>
              </div>
            );
          })}
        </div>
      ) : null}

      <button type="button" className="text-btn link-picker-trigger" onClick={openPicker}>
        + Link a task
      </button>

      {pickerOpen ? (
        <div className="popover link-picker-popover" ref={popoverRef}>
          <div className="link-picker-row">
            <select
              className="field-input link-picker-type"
              aria-label="Link type"
              value={type}
              onChange={(e) => setType(e.target.value)}
            >
              {LINK_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
            <input
              className="field-input"
              aria-label="Search tasks to link"
              placeholder="search by title or #id…"
              value={query}
              autoFocus
              onChange={(e) => {
                setQuery(e.target.value);
                setActive(0);
              }}
              onKeyDown={onKeyDown}
            />
          </div>
          <ul className="link-picker-results" role="listbox" aria-label="Matching tasks">
            {candidates.length === 0 ? (
              <li className="link-picker-empty">No matching tasks.</li>
            ) : (
              candidates.map((t, i) => (
                <li
                  key={t.id}
                  role="option"
                  aria-selected={i === active}
                  className={"link-picker-result" + (i === active ? " is-active" : "")}
                  onMouseEnter={() => setActive(i)}
                  onClick={() => addLink(t.id)}
                >
                  <span className="linked-task-id">#{padId(t.id)}</span>
                  <span>{t.title}</span>
                </li>
              ))
            )}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
