interface EmptyResultProps {
  onClear: () => void;
}

// EmptyResult is shown in place of the list/board when the active search and
// filters match nothing (spec: "Browsing the task list" — Empty result).
export function EmptyResult({ onClear }: EmptyResultProps) {
  return (
    <div className="empty-result">
      <p>No task matches the current search and filters.</p>
      <button type="button" className="text-btn is-strong" onClick={onClear}>
        Clear everything
      </button>
    </div>
  );
}
