import { useCallback, useRef, useState } from "react";

export interface ToastAction {
  label: string;
  onClick: () => void;
}

export interface ToastSpec {
  text: string;
  actions?: ToastAction[];
}

export interface ToastState extends ToastSpec {
  id: number;
}

// useToast holds at most one toast at a time (design.md D9): showing a new
// one replaces whatever was already there. `ToastHost` owns the 6s auto-fade;
// this hook only owns the current toast value and how to change it.
export function useToast(): {
  toast: ToastState | null;
  show: (spec: ToastSpec) => void;
  dismiss: () => void;
} {
  const [toast, setToast] = useState<ToastState | null>(null);
  const nextId = useRef(0);

  const show = useCallback((spec: ToastSpec) => {
    nextId.current += 1;
    setToast({ ...spec, id: nextId.current });
  }, []);

  const dismiss = useCallback(() => setToast(null), []);

  return { toast, show, dismiss };
}
