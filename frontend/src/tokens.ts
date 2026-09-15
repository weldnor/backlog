// The one-line capture syntax of the inline draft (design.md D7). A pure
// function so the draft, the summary row under it and the full form's prefill
// all read the same parse.

// The statuses a draft may target. `declined` is left out on purpose: it needs
// a reason, so declining stays an edit (design.md D8).
export const DRAFT_STATUSES = ["new", "todo", "doing", "done"] as const;

const PRIORITY_TOKENS: Record<string, string> = {
  "!high": "high",
  "!med": "medium",
  "!medium": "medium",
  "!low": "low",
};

export interface ParsedDraft {
  title: string;
  priority?: string;
  tags: string[];
  assignee?: string;
  status?: string;
}

// parseDraft splits the line on whitespace and pulls out `!priority`, `#tag`,
// `@assignee` and `>status` words. Anything else — an unknown `!foo` or
// `>foo`, a bare `#`, `@` or `>` — stays in the title. Assignees are free text
// on the server, so any `@name` is accepted. A later priority, assignee or
// status word replaces an earlier one; repeated tags are kept once, compared
// case-insensitively.
export function parseDraft(
  line: string,
  knownStatuses: readonly string[] = DRAFT_STATUSES,
): ParsedDraft {
  const out: ParsedDraft = { title: "", tags: [] };
  const title: string[] = [];

  for (const word of line.split(/\s+/)) {
    if (!word) continue;
    const lower = word.toLowerCase();
    const rest = word.slice(1);

    if (lower in PRIORITY_TOKENS) {
      out.priority = PRIORITY_TOKENS[lower];
    } else if (word[0] === "#" && rest) {
      if (!out.tags.some((t) => t.toLowerCase() === rest.toLowerCase())) {
        out.tags.push(rest);
      }
    } else if (word[0] === "@" && rest) {
      out.assignee = rest;
    } else if (word[0] === ">" && knownStatuses.includes(rest.toLowerCase())) {
      out.status = rest.toLowerCase();
    } else {
      title.push(word);
    }
  }

  out.title = title.join(" ");
  return out;
}
