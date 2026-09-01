import type { KeyboardEvent } from "react";

import type { TaskView } from "../api";
import {
  BOARD_EMPTY_NOTE,
  padId,
  priBadge,
  STATUS_META,
  STATUS_ORDER,
} from "../constants";
import { TagChips } from "./TagChips";

interface BoardViewProps {
  tasks: TaskView[];
  onOpen: (id: number) => void;
}

function activateOnKey(e: KeyboardEvent<HTMLDivElement>, run: () => void) {
  if (e.key === "Enter" || e.key === " ") {
    e.preventDefault();
    run();
  }
}

export function BoardView({ tasks, onOpen }: BoardViewProps) {
  return (
    <>
      {STATUS_ORDER.map((k) => {
        const list = tasks.filter((t) => t.status === k);
        return (
          <div key={k} className="board-col">
            <div className="board-head">
              <span className="label">{STATUS_META[k].headLabel}</span>
              <span className="count">{list.length}</span>
            </div>
            <div className="board-body">
              {list.map((t) => {
                const p = priBadge(t);
                const file = t.metadata.source.files.length
                  ? t.metadata.source.files[0]
                  : "no source location";
                return (
                  <div
                    key={t.id}
                    className={
                      "board-card" +
                      (t.priority === "high" ? " pri-high-rule" : "")
                    }
                    tabIndex={0}
                    role="button"
                    onClick={() => onOpen(t.id)}
                    onKeyDown={(e) => activateOnKey(e, () => onOpen(t.id))}
                  >
                    <div className="board-card-top">
                      <span className="board-card-id">{padId(t.id)}</span>
                      <span className={"pri-label " + p.fg}>{p.label}</span>
                    </div>
                    <div className="board-card-title">{t.title}</div>
                    <div className="board-card-file">{file}</div>
                    <div className="board-card-tags">
                      <TagChips tags={t.tags} />
                    </div>
                  </div>
                );
              })}
              {list.length === 0 ? (
                <div className="board-empty">{BOARD_EMPTY_NOTE[k]}</div>
              ) : null}
            </div>
          </div>
        );
      })}
    </>
  );
}
