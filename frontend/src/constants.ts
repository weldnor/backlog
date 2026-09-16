// Fixed lifecycle order, matching internal/task and the old app.js.
export const STATUS_ORDER = ["new", "todo", "doing", "done", "declined"] as const;
export const PRI_ORDER = ["high", "medium", "low"] as const;

// The permitted link types, matching internal/task.LinkTypes.
export const LINK_TYPES = ["related", "blocks", "blocked-by", "duplicates", "duplicated-by"] as const;

// The board's own empty-column copy, kept verbatim from the old UI: it teaches
// the same distinctions the README does.
export const BOARD_EMPTY_NOTE: Record<string, string> = {
  new: "Nothing captured here yet, unreviewed.",
  todo: "Nothing approved and waiting.",
  doing: "Nothing in flight.",
  done: "Archive — acted on and moved out of the working set.",
  declined:
    "Always in scope for search — a duplicate must not hide behind a filter.",
};

export function padId(id: number): string {
  return String(id).padStart(3, "0");
}

// Display names for statuses and priorities, as the mockups spell them.
export const STATUS_LABEL: Record<string, string> = {
  new: "New",
  todo: "Todo",
  doing: "Doing",
  done: "Done",
  declined: "Declined",
};

export const PRI_LABEL: Record<string, string> = {
  high: "High",
  medium: "Medium",
  low: "Low",
};

// The priority dot colour on cards, rows and selects (design.md D6).
export const PRI_DOT: Record<string, string> = {
  high: "var(--label-red)",
  medium: "var(--label-yellow)",
  low: "var(--label-blue)",
};

// The ten label tokens a tag's colour is drawn from.
export const LABEL_COLORS = [
  "purple",
  "blue",
  "red",
  "green",
  "yellow",
  "indigo",
  "cyan",
  "blue-deep",
  "blue-ink",
  "red-deep",
].map((name) => `var(--label-${name})`);

// tagColor hashes the lowercase tag name into LABEL_COLORS, so a tag keeps one
// colour everywhere it is drawn without anyone choosing it.
export function tagColor(name: string): string {
  return LABEL_COLORS[tagIndex(name)];
}

function tagIndex(name: string): number {
  let h = 0;
  for (const ch of name.toLowerCase()) {
    h = (h * 31 + (ch.codePointAt(0) ?? 0)) >>> 0;
  }
  return h % LABEL_COLORS.length;
}

// Label colours are the same in both themes, so text on a tag chip needs a
// fixed colour too — dark on the light labels, white on the rest.
const LIGHT_LABELS = new Set(["blue", "green", "yellow", "cyan"]);
const LABEL_NAMES = LABEL_COLORS.map((c) => c.slice("var(--label-".length, -1));

// tagChipStyle is the inline style of a tag drawn as a text chip.
export function tagChipStyle(name: string): { background: string; color: string } {
  const i = tagIndex(name);
  return {
    background: LABEL_COLORS[i],
    color: LIGHT_LABELS.has(LABEL_NAMES[i]) ? "#131211" : "#ffffff",
  };
}

function stripInline(s: string): string {
  return s
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/(\*\*|__)(.+?)\1/g, "$2")
    .replace(/(^|\W)[*_](\S(?:[^*_]*\S)?)[*_](?=\W|$)/g, "$1$2");
}

// descExcerpt is the start of a description as a card shows it: the first
// paragraph that is not a heading or a code block, with list/quote markers and
// inline markup removed.
export function descExcerpt(markdown: string): string {
  const para: string[] = [];
  let inCode = false;
  for (const raw of (markdown || "").split("\n")) {
    const line = raw.trim();
    if (/^```/.test(line)) {
      if (para.length) break;
      inCode = !inCode;
      continue;
    }
    if (inCode) continue;
    if (line === "" || /^#{1,6}\s/.test(line)) {
      if (para.length) break;
      continue;
    }
    para.push(line.replace(/^(?:[-*]|>|\d+\.)\s+/, ""));
  }
  return stripInline(para.join(" "));
}

export function splitList(s: string): string[] {
  return (s || "")
    .split(",")
    .map((x) => x.trim())
    .filter(Boolean);
}

// MOD_KEY is how the save/create shortcut's modifier is spelled in hints:
// ⌘ on Apple platforms, Ctrl+ elsewhere (both keys work everywhere).
export const MOD_KEY =
  typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform) ? "⌘" : "Ctrl+";
