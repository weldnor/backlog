import { useCallback, useEffect, useRef, useState } from "react";

import { listTasks, type TaskView } from "./api";

export interface VisibleFilter {
  status: string | null;
  priority: string | null;
  tag: string | null;
  assignee: string | null;
}

// visibleParams builds the query for the filtered fetch. Unlike `backlog list`,
// the UI shows every status by default, so it always asks for ?all=1 — an
// explicit status filter replaces that with a single-status request. Priority,
// tag and assignee narrow further.
function visibleParams(f: VisibleFilter): URLSearchParams {
  const p = new URLSearchParams();
  if (f.status) {
    p.set("status", f.status);
  } else {
    p.set("all", "1");
  }
  if (f.priority) p.set("priority", f.priority);
  if (f.tag) p.set("tag", f.tag);
  if (f.assignee) p.set("assignee", f.assignee);
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

  // With no filter active the visible set *is* the full set, so one request
  // feeds both instead of fetching ?all=1 twice.
  const unfiltered = visibleKey === "all=1";

  const fetchAll = useCallback(
    () => listTasks({ all: "1" }).then(setAll),
    [],
  );
  const fetchVisible = useCallback(
    () =>
      listTasks(Object.fromEntries(new URLSearchParams(visibleKey))).then(
        (tasks) => {
          setVisible(tasks);
          if (visibleKey === "all=1") setAll(tasks);
        },
      ),
    [visibleKey],
  );

  useEffect(() => {
    fetchVisible().catch((e) => setLoadError(message(e)));
  }, [fetchVisible]);

  // A filtered view still needs the full set for the chips and counts. It
  // only changes on a mutation (refresh), so it's fetched on its own just once,
  // and only when the hook starts filtered — otherwise the fetch above has it.
  const startedFiltered = useRef(!unfiltered);
  useEffect(() => {
    if (!startedFiltered.current) return;
    fetchAll().catch((e) => setLoadError(message(e)));
  }, [fetchAll]);

  const refresh = useCallback(async () => {
    setLoadError("");
    try {
      await Promise.all(unfiltered ? [fetchVisible()] : [fetchAll(), fetchVisible()]);
    } catch (e) {
      setLoadError(message(e));
    }
  }, [fetchAll, fetchVisible, unfiltered]);

  // Tasks added or changed outside the UI (the CLI, another tab) should show
  // up without a manual reload; refetching when the window regains focus is
  // the whole freshness mechanism (design.md D4).
  useEffect(() => {
    window.addEventListener("focus", refresh);
    return () => window.removeEventListener("focus", refresh);
  }, [refresh]);

  return { all, visible, loadError, refresh };
}
