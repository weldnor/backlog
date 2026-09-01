import { useCallback, useEffect, useState } from "react";

import { listTasks, type TaskView } from "./api";

export interface VisibleFilter {
  status: string | null;
  priority: string | null;
  tag: string | null;
}

// visibleParams builds the query for the filtered fetch. Unlike `backlog list`,
// the UI shows every status by default, so it always asks for ?all=1 — an
// explicit status filter replaces that with a single-status request. Priority
// and tag narrow further.
function visibleParams(f: VisibleFilter): URLSearchParams {
  const p = new URLSearchParams();
  if (f.status) {
    p.set("status", f.status);
  } else {
    p.set("all", "1");
  }
  if (f.priority) p.set("priority", f.priority);
  if (f.tag) p.set("tag", f.tag);
  return p;
}

function message(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export interface Tasks {
  /** Every task in scope of ?all=1 — drives the sidebar counts and tag cloud. */
  all: TaskView[];
  /** The server-filtered set for the current filters, before free-text search. */
  visible: TaskView[];
  loadError: string;
  /** Re-run both fetches — call after a create or an edit. */
  refresh: () => Promise<void>;
}

export function useTasks(filter: VisibleFilter): Tasks {
  const [all, setAll] = useState<TaskView[]>([]);
  const [visible, setVisible] = useState<TaskView[]>([]);
  const [loadError, setLoadError] = useState("");

  const visibleKey = visibleParams(filter).toString();

  const fetchAll = useCallback(
    () => listTasks({ all: "1" }).then(setAll),
    [],
  );
  const fetchVisible = useCallback(
    () =>
      listTasks(Object.fromEntries(new URLSearchParams(visibleKey))).then(
        setVisible,
      ),
    [visibleKey],
  );

  useEffect(() => {
    fetchAll().catch((e) => setLoadError(message(e)));
  }, [fetchAll]);

  useEffect(() => {
    fetchVisible().catch((e) => setLoadError(message(e)));
  }, [fetchVisible]);

  const refresh = useCallback(async () => {
    setLoadError("");
    try {
      await Promise.all([fetchAll(), fetchVisible()]);
    } catch (e) {
      setLoadError(message(e));
    }
  }, [fetchAll, fetchVisible]);

  return { all, visible, loadError, refresh };
}
