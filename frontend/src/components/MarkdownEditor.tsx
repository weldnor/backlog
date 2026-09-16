import { useLayoutEffect, useRef, useState, type KeyboardEvent } from "react";

import { MOD_KEY } from "../constants";
import { md } from "../markdown";

interface ToolbarButton {
  label: string;
  ariaLabel: string;
  run: (api: {
    wrap: (before: string, after?: string) => void;
    prefixLine: (prefix: string) => void;
    insertFence: () => void;
  }) => void;
}

const TOOLBAR: ToolbarButton[] = [
  { label: "B", ariaLabel: "Bold", run: (api) => api.wrap("**") },
  { label: "I", ariaLabel: "Italic", run: (api) => api.wrap("_") },
  { label: "`", ariaLabel: "Code", run: (api) => api.wrap("`") },
  { label: "H", ariaLabel: "Heading", run: (api) => api.prefixLine("### ") },
  { label: "—", ariaLabel: "Bulleted list", run: (api) => api.prefixLine("- ") },
  { label: "”", ariaLabel: "Quote", run: (api) => api.prefixLine("> ") },
  { label: "</>", ariaLabel: "Code block", run: (api) => api.insertFence() },
];

interface MarkdownEditorProps {
  value: string;
  ariaLabel?: string;
  onSave: (value: string) => Promise<void>;
  onCancel: () => void;
}

// MarkdownEditor is the task modal's description field (design.md D12): a
// Write/Preview toggle, an insert toolbar over the textarea, and Save/Cancel
// with the same commit contract as InlineText — a rejected save reverts the
// editor to the saved value, since the detail window's error line already
// carries the reason (design.md D11's "leave the displayed value at what is
// saved").
export function MarkdownEditor({ value, ariaLabel = "Description", onSave, onCancel }: MarkdownEditorProps) {
  const [draft, setDraft] = useState(value);
  const [tab, setTab] = useState<"write" | "preview">("write");
  const [saving, setSaving] = useState(false);
  const [caretTarget, setCaretTarget] = useState<number | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useLayoutEffect(() => {
    if (caretTarget !== null && textareaRef.current) {
      textareaRef.current.focus();
      textareaRef.current.setSelectionRange(caretTarget, caretTarget);
      setCaretTarget(null);
    }
  }, [caretTarget, draft]);

  function wrap(before: string, after: string = before) {
    const el = textareaRef.current;
    if (!el) return;
    const start = el.selectionStart ?? draft.length;
    const end = el.selectionEnd ?? draft.length;
    const selected = draft.slice(start, end);
    setDraft(draft.slice(0, start) + before + selected + after + draft.slice(end));
    setCaretTarget(selected ? start + before.length + selected.length + after.length : start + before.length);
  }

  function prefixLine(prefix: string) {
    const el = textareaRef.current;
    if (!el) return;
    const pos = el.selectionStart ?? draft.length;
    const lineStart = draft.lastIndexOf("\n", pos - 1) + 1;
    setDraft(draft.slice(0, lineStart) + prefix + draft.slice(lineStart));
    setCaretTarget(pos + prefix.length);
  }

  function insertFence() {
    const el = textareaRef.current;
    if (!el) return;
    const start = el.selectionStart ?? draft.length;
    const end = el.selectionEnd ?? draft.length;
    const selected = draft.slice(start, end);
    const block = selected ? "```\n" + selected + "\n```" : "```\n\n```";
    setDraft(draft.slice(0, start) + block + draft.slice(end));
    setCaretTarget(selected ? start + block.length : start + 4);
  }

  async function save() {
    setSaving(true);
    try {
      await onSave(draft);
      setSaving(false);
    } catch {
      setSaving(false);
      setDraft(value);
      setTab("write");
      onCancel();
    }
  }

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      onCancel();
      return;
    }
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      save();
    }
  }

  return (
    <div className="md-editor" onKeyDown={onKeyDown}>
      <div className="md-editor-tabs" role="tablist" aria-label={`${ariaLabel} mode`}>
        <button
          type="button"
          role="tab"
          aria-selected={tab === "write"}
          className={"text-toggle" + (tab === "write" ? " is-active" : "")}
          onClick={() => setTab("write")}
        >
          Write
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={tab === "preview"}
          className={"text-toggle" + (tab === "preview" ? " is-active" : "")}
          onClick={() => setTab("preview")}
        >
          Preview
        </button>
      </div>

      {tab === "write" ? (
        <>
          <div className="md-editor-toolbar" role="toolbar" aria-label="Formatting">
            {TOOLBAR.map((btn) => (
              <button
                key={btn.ariaLabel}
                type="button"
                className="md-editor-tool"
                aria-label={btn.ariaLabel}
                disabled={saving}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => btn.run({ wrap, prefixLine, insertFence })}
              >
                {btn.label}
              </button>
            ))}
          </div>
          <textarea
            ref={textareaRef}
            className="field-input md-editor-textarea"
            aria-label={ariaLabel}
            value={draft}
            disabled={saving}
            onChange={(e) => setDraft(e.target.value)}
          />
        </>
      ) : (
        <div className="md md-editor-preview" dangerouslySetInnerHTML={{ __html: md(draft) }} />
      )}

      <div className="md-editor-actions">
        <button type="button" className="pill-btn" disabled={saving} onClick={save}>
          Save
        </button>
        <button type="button" className="text-btn" disabled={saving} onClick={onCancel}>
          Cancel
        </button>
        <span className="form-hint">{MOD_KEY}↵ save · Esc cancel</span>
      </div>
    </div>
  );
}
