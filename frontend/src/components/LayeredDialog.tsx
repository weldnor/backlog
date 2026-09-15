import { createContext, useContext, useEffect, useRef, type ReactNode, type RefObject } from "react";
import { createPortal } from "react-dom";

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

interface Layer {
  depth: number;
}

// The open layers, bottom to top. Only the last one answers Escape, Tab and a
// backdrop click, so a confirmation stacked over the task modal closes on its
// own and leaves the modal open (design.md D13).
const layers: Layer[] = [];

// How many layers enclose a component in the React tree. A layer rendered
// inside another mounts in the same commit and runs its effect before its
// parent's, so the stack is ordered by depth rather than by mount order.
const LayerDepth = createContext(0);

export function hasOpenLayer(): boolean {
  return layers.length > 0;
}

const isTop = (layer: Layer) => layers[layers.length - 1] === layer;

interface LayeredDialogProps {
  /** Accessible name of the dialog. */
  label: string;
  /** "top" dims the page with a light ink wash; "nested" darkens over a layer. */
  variant?: "top" | "nested";
  className?: string;
  onClose: () => void;
  /** Receives focus on open; defaults to the first focusable control. */
  initialFocus?: RefObject<HTMLElement | null>;
  children: ReactNode;
}

// LayeredDialog is the shared shell of every overlay: portal, backdrop, focus
// moved in on open and restored on close, Tab kept inside, Escape closing.
// Escape is heard on document in the bubble phase, so a control that handles
// Escape itself (an inline edit) and stops propagation keeps the layer open.
export function LayeredDialog({
  label,
  variant = "top",
  className,
  onClose,
  initialFocus,
  children,
}: LayeredDialogProps) {
  const depth = useContext(LayerDepth) + 1;
  const nodeRef = useRef<HTMLDivElement>(null);
  const layerRef = useRef<Layer>({ depth });
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  useEffect(() => {
    const node = nodeRef.current;
    if (!node) return;
    const layer = layerRef.current;
    layers.push(layer);
    layers.sort((a, b) => a.depth - b.depth);
    const previouslyFocused = document.activeElement as HTMLElement | null;

    const focusables = () =>
      Array.from(node.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
        (el) => !el.closest("[hidden]"),
      );

    if (isTop(layer)) {
      (initialFocus?.current ?? focusables()[0] ?? node).focus();
    }

    function onKeyDown(e: KeyboardEvent) {
      if (!isTop(layer)) return;
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

    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      layers.splice(layers.indexOf(layer), 1);
      previouslyFocused?.focus?.();
    };
    // Mount-only: the layer's place in the stack and the element to restore
    // focus to are fixed when it opens.
  }, []);

  return createPortal(
    <div
      className={"layer-backdrop is-" + variant}
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && isTop(layerRef.current)) onCloseRef.current();
      }}
    >
      <div
        ref={nodeRef}
        className={"layer" + (className ? " " + className : "")}
        role="dialog"
        aria-modal="true"
        aria-label={label}
        tabIndex={-1}
      >
        <LayerDepth.Provider value={depth}>{children}</LayerDepth.Provider>
      </div>
    </div>,
    document.body,
  );
}
