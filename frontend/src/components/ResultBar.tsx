interface ResultBarProps {
  count: number;
  command: string;
  archive: boolean;
  onToggleArchive: () => void;
}

export function ResultBar({
  count,
  command,
  archive,
  onToggleArchive,
}: ResultBarProps) {
  return (
    <div className="resultbar">
      <span className="result-count">
        {count} {count === 1 ? "task" : "tasks"}
      </span>
      <span className="result-note">
        descending priority, then ascending identifier
      </span>
      <div className="topbar-spacer" />
      <button
        className={"btn btn-secondary" + (archive ? " is-active" : "")}
        onClick={onToggleArchive}
      >
        {archive ? "ARCHIVE SHOWN · --all" : "Show archive · --all"}
      </button>
      <span className="result-cmd">{command}</span>
    </div>
  );
}
