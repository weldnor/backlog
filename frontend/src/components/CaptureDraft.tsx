import { useEffect, useLayoutEffect, useRef, useState } from "react";

import {
  ApiError,
  createTaskWithStatus,
  CreateStatusError,
  deleteTask,
  type TaskView,
} from "../api";
import { copyText } from "../clipboard";
import { padId, PRI_DOT, PRI_LABEL, STATUS_LABEL, tagChipStyle } from "../constants";
import type { ToastSpec } from "../useToast";
import { parseDraft, type ParsedDraft } from "../tokens";

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

function wordAt(line: string, caret: number): { start: number; end: number; word: string } {
  let start = caret;
  while (start > 0 && !/\s/.test(line[start - 1])) start--;
  let end = caret;
  while (end < line.length && !/\s/.test(line[end])) end++;
  return { start, end, word: line.slice(start, end) };
}

interface TagCount {
  name: string;
  count: number;
}

// tagSuggestions ranks existing tags containing `query` prefix-first, then by
// usage count (design.md D7). Exported for direct unit testing.
export function tagSuggestions(query: string, all: TaskView[]): TagCount[] {
  const counts = new Map<string, TagCount>();
  for (const t of all) {
    for (const g of t.tags) {
      const key = g.toLowerCase();
      const cur = counts.get(key);
      if (cur) cur.count++;
      else counts.set(key, { name: g, count: 1 });
    }
  }
  const q = query.toLowerCase();
  const list = [...counts.values()].filter((x) => (q ? x.name.toLowerCase().includes(q) : true));
  list.sort((a, b) => {
    const aPrefix = a.name.toLowerCase().startsWith(q) ? 0 : 1;
    const bPrefix = b.name.toLowerCase().startsWith(q) ? 0 : 1;
    if (aPrefix !== bPrefix) return aPrefix - bPrefix;
    if (a.count !== b.count) return b.count - a.count;
    return a.name.localeCompare(b.name);
  });
  return list.slice(0, 5);
}

type AcItem = { kind: "tag"; name: string } | { kind: "create"; name: string };

interface AcState {
  start: number;
  end: number;
  items: AcItem[];
  activeIdx: number;
}

export interface CaptureDraftProps {
  /** The column this draft belongs to; also the status a plain Enter creates into. */
  status: string;
  /** Every task, used for the duplicate check and the tag autocomplete. */
  all: TaskView[];
  /** Call after any successful mutation (create, undo) so the board refetches. */
  onCreated: () => void;
  /** Opens a task's detail — used by the duplicate hint's "Open" and the toast's "Open". */
  onOpen: (id: number) => void;
  /** Shift+Enter: hand the parsed line to the full form. */
  onOpenFullForm: (parsed: ParsedDraft) => void;
  /** Esc: collapse back to the rest "+ Add new task" slot. */
  onClose: () => void;
  /** Re-expand the draft holding `line` (Undo restores a draft-origin line,
   *  even when this draft was closed in the meantime). */
  onReopen: (line: string) => void;
  showToast: (spec: ToastSpec) => void;
  autoFocus?: boolean;
  /** The text the draft opens with. */
  initialLine?: string;
}

// CaptureDraft is the one-line inline capture (design.md D7/D8/D9): token
// parsing, tag autocomplete, the duplicate hint, a failed-create block and the
// success toast are all local to this component — nothing here is lifted into
// the app reducer.
export function CaptureDraft({
  status,
  all,
  onCreated,
  onOpen,
  onOpenFullForm,
  onClose,
  onReopen,
  showToast,
  autoFocus,
  initialLine = "",
}: CaptureDraftProps) {
  const [line, setLine] = useState(initialLine);
  const [error, setError] = useState<string | null>(null);
  const [duplicate, setDuplicate] = useState<TaskView | null>(null);
  const [failure, setFailure] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [ac, setAc] = useState<AcState | null>(null);
  const [caretTarget, setCaretTarget] = useState<number | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useLayoutEffect(() => {
    if (caretTarget !== null && inputRef.current) {
      inputRef.current.setSelectionRange(caretTarget, caretTarget);
      setCaretTarget(null);
    }
  }, [caretTarget, line]);

  // The input is disabled while a create is in flight, which drops focus;
  // hand it back so the next line can be typed straight away.
  const wasSubmitting = useRef(false);
  useEffect(() => {
    if (wasSubmitting.current && !submitting) inputRef.current?.focus();
    wasSubmitting.current = submitting;
  }, [submitting]);

  const parsed = parseDraft(line);

  function updateAutocomplete(el: HTMLInputElement) {
    const caret = el.selectionStart ?? el.value.length;
    const { start, end, word } = wordAt(el.value, caret);
    if (word[0] !== "#") {
      setAc(null);
      return;
    }
    const query = word.slice(1);
    const suggestions = tagSuggestions(query, all);
    const items: AcItem[] = suggestions.map((s) => ({ kind: "tag", name: s.name }));
    if (query && !suggestions.some((s) => s.name.toLowerCase() === query.toLowerCase())) {
      items.push({ kind: "create", name: query });
    }
    if (items.length === 0) {
      setAc(null);
      return;
    }
    setAc({ start, end, items, activeIdx: 0 });
  }

  function moveActive(delta: number) {
    setAc((a) => {
      if (!a || a.items.length === 0) return a;
      const next = (a.activeIdx + delta + a.items.length) % a.items.length;
      return { ...a, activeIdx: next };
    });
  }

  function acceptItem(item: AcItem) {
    if (!ac) return;
    const insertion = "#" + item.name;
    setLine(line.slice(0, ac.start) + insertion + line.slice(ac.end));
    setCaretTarget(ac.start + insertion.length);
    setAc(null);
    inputRef.current?.focus();
  }

  function acceptActive() {
    const item = ac?.items[ac.activeIdx];
    if (item) acceptItem(item);
  }

  async function submit(force: boolean) {
    const title = parsed.title.trim();
    if (!title) {
      setError("A title is required.");
      return;
    }
    setError(null);
    if (!force) {
      const dup = all.find((t) => t.title.trim().toLowerCase() === title.toLowerCase());
      if (dup) {
        setDuplicate(dup);
        return;
      }
    }
    setDuplicate(null);
    setFailure(null);
    setSubmitting(true);
    const targetStatus = parsed.status ?? status;
    const originalLine = line;

    try {
      const created = await createTaskWithStatus(
        {
          title,
          description: "",
          tags: parsed.tags,
          priority: parsed.priority ?? "medium",
          assignee: parsed.assignee ?? "",
          files: [],
          refs: [],
          links: [],
        },
        targetStatus,
      );
      setLine("");
      setSubmitting(false);
      onCreated();
      announceCreated(created, originalLine);
    } catch (err) {
      setSubmitting(false);
      if (err instanceof CreateStatusError) {
        // The task was created (still `new`); only the follow-up status PATCH
        // failed. Refresh and still name the task, per design.md D8.
        setLine("");
        onCreated();
        showToast({
          text: `Task ${padId(err.task.id)} created in ${STATUS_LABEL[err.task.status] ?? err.task.status} — ${err.message}`,
          actions: [{ label: "Open", onClick: () => onOpen(err.task.id) }],
        });
        return;
      }
      setFailure(err instanceof ApiError ? err.message : message(err));
    }
  }

  function announceCreated(created: TaskView, originalLine: string) {
    showToast({
      text: `Task ${padId(created.id)} created in ${STATUS_LABEL[created.status] ?? created.status}`,
      actions: [
        { label: "Open", onClick: () => onOpen(created.id) },
        {
          label: "Undo",
          onClick: () => {
            deleteTask(created.id)
              .then(() => onCreated())
              .catch(() => {});
            onReopen(originalLine);
          },
        },
      ],
    });
  }

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (ac && !e.shiftKey && ["ArrowDown", "ArrowUp", "Tab", "Enter"].includes(e.key)) {
      e.preventDefault();
      if (e.key === "ArrowDown") moveActive(1);
      else if (e.key === "ArrowUp") moveActive(-1);
      else acceptActive();
      return;
    }
    if (ac && e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      setAc(null);
      return;
    }
    if (e.key === "Enter" && e.shiftKey) {
      e.preventDefault();
      onOpenFullForm(parsed);
      return;
    }
    if (e.key === "Enter") {
      e.preventDefault();
      submit(false);
      return;
    }
    if (e.key === "Escape") {
      e.preventDefault();
      onClose();
    }
  }

  return (
    <div className="draft">
      <input
        ref={inputRef}
        className="draft-input"
        autoFocus={autoFocus}
        value={line}
        disabled={submitting}
        placeholder="Title, #tag, @assignee, !priority, >status"
        onChange={(e) => {
          setLine(e.target.value);
          setError(null);
          setFailure(null);
          setDuplicate(null);
          updateAutocomplete(e.target);
        }}
        onClick={(e) => updateAutocomplete(e.currentTarget)}
        onKeyUp={(e) => {
          if (!["ArrowDown", "ArrowUp", "Tab", "Enter", "Escape"].includes(e.key)) {
            updateAutocomplete(e.currentTarget);
          }
        }}
        onKeyDown={onKeyDown}
      />

      {ac ? (
        <ul className="autocomplete-list" role="listbox">
          {ac.items.map((item, i) => (
            <li
              key={item.kind + ":" + item.name}
              role="option"
              aria-selected={i === ac.activeIdx}
              className={"autocomplete-item" + (i === ac.activeIdx ? " is-active" : "")}
              onMouseDown={(e) => {
                e.preventDefault();
                acceptItem(item);
              }}
              onMouseEnter={() => setAc((a) => a && { ...a, activeIdx: i })}
            >
              {item.kind === "create" ? `Create tag «${item.name}»` : item.name}
            </li>
          ))}
        </ul>
      ) : null}

      {!ac && (parsed.priority || parsed.tags.length > 0 || parsed.assignee || parsed.status) ? (
        <div className="draft-summary">
          {parsed.priority ? (
            <>
              <span className="pri-dot" style={{ background: PRI_DOT[parsed.priority] }} />
              <span>{PRI_LABEL[parsed.priority]}</span>
            </>
          ) : null}
          {parsed.tags.map((g) => (
            <span key={g} className="draft-tag" style={tagChipStyle(g)}>
              {g}
            </span>
          ))}
          {parsed.assignee ? <span>@{parsed.assignee}</span> : null}
          {parsed.status ? <span>&gt;{STATUS_LABEL[parsed.status] ?? parsed.status}</span> : null}
        </div>
      ) : null}

      {error ? <div className="draft-error">{error}</div> : null}

      {duplicate ? (
        <div className="draft-duplicate">
          <span>
            Looks like an existing task: {padId(duplicate.id)} · {duplicate.title}
          </span>
          <button type="button" className="text-btn" onClick={() => onOpen(duplicate.id)}>
            Open
          </button>
          <button type="button" className="text-btn is-strong" onClick={() => submit(true)}>
            Create anyway
          </button>
        </div>
      ) : null}

      {failure ? (
        <div className="draft-failure">
          <span>{failure}</span>
          <button type="button" className="text-btn" onClick={() => submit(false)}>
            Retry
          </button>
          <button
            type="button"
            className="text-btn"
            onClick={() => void copyText(line)}
          >
            Copy text
          </button>
          <button
            type="button"
            className="text-btn"
            onClick={() => {
              setLine("");
              setFailure(null);
            }}
          >
            Discard
          </button>
        </div>
      ) : null}

      {!duplicate && !failure ? (
        <div className="draft-hint">↵ create · ⇧↵ details · Esc cancel</div>
      ) : null}
    </div>
  );
}
