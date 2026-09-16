import type { Theme } from "../useTheme";
import { CloseIcon, MoonIcon, SearchIcon, SunIcon } from "./icons";

interface TopBarProps {
  query: string;
  onQuery: (q: string) => void;
  view: "list" | "board";
  onView: (v: "list" | "board") => void;
  theme: Theme;
  onToggleTheme: () => void;
}

// TopBar is the 58px chrome of the redesign (proposal — Chrome): wordmark,
// search pill, the List/Board toggle and the theme switch. The sidebar,
// result bar and CAPTURE button it used to carry are gone — filtering moved
// to FilterRow and capture moved to the column slots and floating button.
export function TopBar({ query, onQuery, view, onView, theme, onToggleTheme }: TopBarProps) {
  return (
    <header className="topbar">
      <span className="wordmark">Backlog</span>
      <span className="topbar-divider" aria-hidden="true" />

      <div className="search-pill">
        <SearchIcon size={15} />
        <input
          className="search-input"
          placeholder="Search tasks, tags, files"
          value={query}
          onChange={(e) => onQuery(e.target.value)}
        />
        {query ? (
          <button
            type="button"
            className="search-clear"
            aria-label="Clear search"
            onClick={() => onQuery("")}
          >
            <CloseIcon size={12} />
          </button>
        ) : null}
      </div>

      <div className="topbar-spacer" />

      <div className="view-toggle" role="group" aria-label="View">
        <button
          type="button"
          className={"view-toggle-btn" + (view === "list" ? " is-active" : "")}
          aria-pressed={view === "list"}
          onClick={() => onView("list")}
        >
          List
        </button>
        <button
          type="button"
          className={"view-toggle-btn" + (view === "board" ? " is-active" : "")}
          aria-pressed={view === "board"}
          onClick={() => onView("board")}
        >
          Board
        </button>
      </div>

      <span className="topbar-divider" aria-hidden="true" />

      <button
        type="button"
        className="theme-toggle"
        aria-label={theme === "dark" ? "Switch to light theme" : "Switch to dark theme"}
        onClick={onToggleTheme}
      >
        {theme === "dark" ? <SunIcon /> : <MoonIcon />}
      </button>
    </header>
  );
}
