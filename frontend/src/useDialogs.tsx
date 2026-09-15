import { useCallback, useState, type ReactNode } from "react";

import { ConfirmDialog, ReasonDialog } from "./components/ConfirmDialog";
import { hasOpenLayer } from "./components/LayeredDialog";

export interface ConfirmOptions {
  title: string;
  description?: ReactNode;
  confirmLabel?: string;
}

export interface ReasonOptions {
  title?: string;
  description?: ReactNode;
}

type Variant = "top" | "nested";

type Pending =
  | { kind: "confirm"; opts: ConfirmOptions; variant: Variant; resolve: (ok: boolean) => void }
  | {
      kind: "reason";
      opts: ReasonOptions;
      variant: Variant;
      resolve: (reason: string | null) => void;
    };

export interface Dialogs {
  confirm: (opts: ConfirmOptions) => Promise<boolean>;
  askReason: (opts?: ReasonOptions) => Promise<string | null>;
  /** Render once; holds whichever dialog is waiting for an answer. */
  dialogs: ReactNode;
}

// A dialog asked for over an open layer (the task modal) takes the darker
// nested backdrop; one asked for from the page (a board drop) the top one.
const variantNow = (): Variant => (hasOpenLayer() ? "nested" : "top");

// useDialogs turns the in-app confirmation and reason dialogs into promises,
// so callers can `await` them where window.confirm / window.prompt used to be.
export function useDialogs(): Dialogs {
  const [pending, setPending] = useState<Pending | null>(null);

  const confirm = useCallback(
    (opts: ConfirmOptions) =>
      new Promise<boolean>((resolve) =>
        setPending({ kind: "confirm", opts, variant: variantNow(), resolve }),
      ),
    [],
  );

  const askReason = useCallback(
    (opts: ReasonOptions = {}) =>
      new Promise<string | null>((resolve) =>
        setPending({ kind: "reason", opts, variant: variantNow(), resolve }),
      ),
    [],
  );

  let dialogs: ReactNode = null;
  if (pending?.kind === "confirm") {
    const p = pending;
    dialogs = (
      <ConfirmDialog
        {...p.opts}
        variant={p.variant}
        onResult={(ok) => {
          setPending(null);
          p.resolve(ok);
        }}
      />
    );
  } else if (pending?.kind === "reason") {
    const p = pending;
    dialogs = (
      <ReasonDialog
        {...p.opts}
        variant={p.variant}
        onResult={(reason) => {
          setPending(null);
          p.resolve(reason);
        }}
      />
    );
  }

  return { confirm, askReason, dialogs };
}
