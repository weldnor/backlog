interface ResultBarProps {
  count: number;
  command: string;
}

export function ResultBar({ count, command }: ResultBarProps) {
  return (
    <div className="resultbar">
      <span className="result-count">
        {count} {count === 1 ? "task" : "tasks"}
      </span>
      <span className="result-note">
        descending priority, then ascending identifier
      </span>
      <div className="topbar-spacer" />
      <span className="result-cmd">{command}</span>
    </div>
  );
}
