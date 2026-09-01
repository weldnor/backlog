export function TagChips({ tags }: { tags: string[] }) {
  return (
    <>
      {tags.map((g) => (
        <span key={g} className="tag tag-neutral">
          {g}
        </span>
      ))}
    </>
  );
}
