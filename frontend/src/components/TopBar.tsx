import type { RepoInfo } from "../api";
import { PlusIcon, SearchIcon } from "./icons";

interface TopBarProps {
  repo: RepoInfo | null;
  query: string;
  onQuery: (q: string) => void;
  view: "list" | "board";
  onView: (v: "list" | "board") => void;
  onCapture: () => void;
}

export function TopBar({
  repo,
  query,
  onQuery,
  view,
  onView,
  onCapture,
}: TopBarProps) {
  return (
    <header className="topbar">
      <div className="wordmark-row">
        <span>
          <span className="wordmark">backlog</span>
          <span className="version">
            {repo?.version ? " v" + repo.version : ""}
          </span>
        </span>
      </div>

      {repo?.name ? (
        <div className="repo-chip">
          <span>{repo.name}</span>
          <span className="sep">/</span>
          <span>.backlog</span>
          <span className="sep">·</span>
          <span className="branch">{repo.branch || "—"}</span>
        </div>
      ) : null}

      <div className="topbar-spacer" />

      <div className="searchbox">
        <SearchIcon />
        <input
          className="input"
          placeholder="title, description, tags"
          value={query}
          onChange={(e) => onQuery(e.target.value)}
        />
      </div>

      <div className="seg seg-view">
        <label className="seg-opt">
          <input
            type="radio"
            name="view"
            value="list"
            checked={view === "list"}
            onChange={() => onView("list")}
          />
          LIST
        </label>
        <label className="seg-opt">
          <input
            type="radio"
            name="view"
            value="board"
            checked={view === "board"}
            onChange={() => onView("board")}
          />
          BOARD
        </label>
      </div>

      <button className="btn btn-primary btn-capture" onClick={onCapture}>
        CAPTURE
        <PlusIcon />
      </button>
    </header>
  );
}
