import { useEffect, useMemo, useRef } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";

import type { CreateTaskBody, PatchTaskBody, TaskView } from "../api";
import { taskFilePath } from "../constants";
import { EditForm } from "./EditForm";
import { CloseIcon, EditIcon } from "./icons";
import { MetadataAside } from "./MetadataAside";
import { ReadView } from "./ReadView";

interface TaskDialogProps {
  mode: "read" | "edit" | "create";
  task: TaskView | null;
  error: string;
  onClose: () => void;
  onToggleEdit: () => void;
  onCancelEdit: () => void;
  onCreate: (body: CreateTaskBody) => void;
  onPatch: (body: PatchTaskBody) => void;
  onDelete: () => void;
}

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

export function TaskDialog(props: TaskDialogProps) {
  const { mode, task, error, onClose, onToggleEdit, onCancelEdit } = props;

  const portalEl = useMemo(() => document.createElement("div"), []);
  const dialogRef = useRef<HTMLDivElement>(null);
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  useEffect(() => {
    document.body.appendChild(portalEl);
    return () => {
      portalEl.remove();
    };
  }, [portalEl]);

  // D6: record what had focus, move focus into the dialog, trap Tab within it
  // while open, and restore focus to the recorded element on close. Escape is
  // wired to the same handler as the close control.
  useEffect(() => {
    const node = dialogRef.current;
    if (!node) return;
    const previouslyFocused = document.activeElement as HTMLElement | null;

    const focusables = () =>
      Array.from(node.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
        (el) => !el.hasAttribute("hidden") && el.getAttribute("aria-hidden") !== "true",
      );

    (focusables()[0] ?? node).focus();

    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        e.preventDefault();
        onCloseRef.current();
        return;
      }
      if (e.key !== "Tab") return;
      const items = focusables();
      if (items.length === 0) {
        e.preventDefault();
        node!.focus();
        return;
      }
      const first = items[0];
      const last = items[items.length - 1];
      const active = document.activeElement;
      if (!node!.contains(active)) {
        e.preventDefault();
        first.focus();
      } else if (e.shiftKey && active === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && active === last) {
        e.preventDefault();
        first.focus();
      }
    }

    document.addEventListener("keydown", onKeyDown, true);
    return () => {
      document.removeEventListener("keydown", onKeyDown, true);
      previouslyFocused?.focus?.();
    };
  }, []);

  const editing = mode === "edit";

  const head =
    mode === "create" ? (
      <div className="dialog-head">
        <span className="filename">new task</span>
        <div className="topbar-spacer" />
        <button
          className="btn btn-secondary btn-icon"
          aria-label="Close"
          onClick={onClose}
        >
          <CloseIcon />
        </button>
      </div>
    ) : (
      <div className="dialog-head">
        <span className="filename">{task ? taskFilePath(task) : ""}</span>
        <div className="topbar-spacer" />
        {editing ? null : (
          <button className="btn btn-danger" onClick={props.onDelete}>
            Delete
          </button>
        )}
        <button
          className={"btn btn-secondary" + (editing ? " is-active" : "")}
          onClick={onToggleEdit}
        >
          {editing ? "READING VIEW" : "EDIT"}
          <EditIcon />
        </button>
        <button
          className="btn btn-secondary btn-icon"
          aria-label="Close"
          onClick={onClose}
        >
          <CloseIcon />
        </button>
      </div>
    );

  let body: ReactNode = null;
  if (mode === "create") {
    body = (
      <div style={{ padding: "20px 20px 24px" }}>
        <EditForm
          mode="create"
          error={error}
          onCancel={onClose}
          onCreate={props.onCreate}
          onPatch={props.onPatch}
        />
      </div>
    );
  } else if (task) {
    body = (
      <div className="dialog-grid">
        <div className="dialog-left">
          {editing ? (
            <EditForm
              mode="edit"
              task={task}
              error={error}
              onCancel={onCancelEdit}
              onCreate={props.onCreate}
              onPatch={props.onPatch}
              onDelete={props.onDelete}
            />
          ) : (
            <ReadView task={task} />
          )}
        </div>
        <MetadataAside task={task} />
      </div>
    );
  }

  return createPortal(
    <div
      className="dialog-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="dialog" ref={dialogRef} role="dialog" aria-modal="true" tabIndex={-1}>
        {head}
        {body}
      </div>
    </div>,
    portalEl,
  );
}
