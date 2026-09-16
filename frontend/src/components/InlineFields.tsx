import { useEffect, useRef, useState, type KeyboardEvent, type ReactNode } from "react";

interface InlineTextProps {
  value: string;
  placeholder?: string;
  ariaLabel: string;
  className?: string;
  // Overrides the rest-state display (e.g. tags as colour chips) while the
  // input still edits the plain `value` text; omitted, the rest state is
  // just `value` itself.
  renderValue?: () => ReactNode;
  // Expected to both apply the change and, on failure, surface the error to
  // the caller (e.g. the modal's error line) and reject — InlineText itself
  // only reverts the displayed value and leaves edit mode (design.md D11).
  onCommit: (value: string) => Promise<void>;
}

// InlineText is the task modal's rest-as-text field editor (design.md D11):
// click or Enter/Space (native button semantics) activates an input; Enter
// commits, Esc cancels and stops propagation so the modal's own Escape
// handler doesn't also fire on the same keypress.
export function InlineText({ value, placeholder, ariaLabel, className, renderValue, onCommit }: InlineTextProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const [saving, setSaving] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus();
      inputRef.current?.select();
    }
  }, [editing]);

  function activate() {
    setDraft(value);
    setEditing(true);
  }

  function cancel() {
    setDraft(value);
    setEditing(false);
  }

  async function commit() {
    const next = draft;
    setSaving(true);
    try {
      await onCommit(next);
      setSaving(false);
      setEditing(false);
    } catch {
      // The caller is expected to have already surfaced the error; this
      // control only reverts to the last saved value and leaves edit mode.
      setSaving(false);
      setDraft(value);
      setEditing(false);
    }
  }

  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault();
      commit();
    } else if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      cancel();
    }
  }

  if (editing) {
    return (
      <input
        ref={inputRef}
        className={"field-input inline-text-input" + (className ? " " + className : "")}
        aria-label={ariaLabel}
        value={draft}
        disabled={saving}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={onKeyDown}
        onBlur={() => {
          if (!saving) cancel();
        }}
      />
    );
  }

  return (
    <button
      type="button"
      className={"inline-text" + (className ? " " + className : "") + (value ? "" : " is-empty")}
      aria-label={ariaLabel}
      onClick={activate}
    >
      {value ? (renderValue ? renderValue() : value) : placeholder}
    </button>
  );
}

export interface InlineSelectOption {
  value: string;
  label: string;
  render?: ReactNode;
}

interface InlineSelectProps {
  value: string;
  options: InlineSelectOption[];
  ariaLabel: string;
  className?: string;
  onCommit: (value: string) => Promise<void>;
}

// InlineSelect is InlineText's counterpart for a fixed set of choices
// (design.md D11): the rest state shows the current option, activation opens
// a Popover listbox, and choosing an option commits it directly (there is no
// separate draft to revert — the trigger simply re-reads `value` once the
// caller re-renders with the saved result, or stays put on failure).
export function InlineSelect({ value, options, ariaLabel, className, onCommit }: InlineSelectProps) {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const [saving, setSaving] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const current = options.find((o) => o.value === value);

  function openList() {
    const idx = options.findIndex((o) => o.value === value);
    setActive(idx >= 0 ? idx : 0);
    setOpen(true);
  }

  async function choose(v: string) {
    setOpen(false);
    if (v === value) return;
    setSaving(true);
    try {
      await onCommit(v);
    } catch {
      // The caller surfaces the error; the trigger keeps showing `value`
      // since it never optimistically changed.
    } finally {
      setSaving(false);
    }
  }

  function onTriggerKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (!open) {
      if (e.key === "ArrowDown" || e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        openList();
      }
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((i) => (i + 1) % options.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((i) => (i - 1 + options.length) % options.length);
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (options[active]) choose(options[active].value);
    } else if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      setOpen(false);
    }
  }

  useEffect(() => {
    if (!open) return;
    function onDocDown(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onDocDown);
    return () => document.removeEventListener("mousedown", onDocDown);
  }, [open]);

  return (
    <div ref={rootRef} className={"inline-select" + (className ? " " + className : "")}>
      <button
        type="button"
        className="inline-select-value"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={ariaLabel}
        disabled={saving}
        onClick={() => (open ? setOpen(false) : openList())}
        onKeyDown={onTriggerKeyDown}
      >
        {current?.render ?? current?.label ?? value}
      </button>
      {open ? (
        <ul className="popover inline-select-popover" role="listbox" aria-label={ariaLabel}>
          {options.map((o, i) => (
            <li
              key={o.value}
              role="option"
              aria-selected={o.value === value}
              className={"popover-item" + (i === active ? " is-active" : "")}
              onMouseDown={(e) => {
                e.preventDefault();
                choose(o.value);
              }}
              onMouseEnter={() => setActive(i)}
            >
              {o.render ?? o.label}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
