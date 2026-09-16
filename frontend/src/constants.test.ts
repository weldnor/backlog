import { describe, expect, it } from "vitest";

import { descExcerpt, LABEL_COLORS, tagChipStyle, tagColor } from "./constants";

describe("tagColor", () => {
  it("gives a tag the same label colour every time, ignoring case", () => {
    expect(tagColor("ui")).toBe(tagColor("ui"));
    expect(tagColor("UI")).toBe(tagColor("ui"));
  });

  it("draws from the ten label tokens and spreads different tags", () => {
    const tags = ["ui", "board", "cli", "docs", "api", "bug", "infra", "search"];
    const colours = new Set(tags.map(tagColor));
    colours.forEach((c) => expect(LABEL_COLORS).toContain(c));
    expect(LABEL_COLORS).toHaveLength(10);
    expect(colours.size).toBeGreaterThan(1);
  });
});

describe("tagChipStyle", () => {
  it("puts dark text on the light labels and white text on the rest", () => {
    const light = ["blue", "green", "yellow", "cyan"].map((n) => `var(--label-${n})`);
    const seen = new Set<string>();
    for (let i = 0; i < 200; i++) {
      const tag = `tag${i}`;
      const style = tagChipStyle(tag);
      expect(style.background).toBe(tagColor(tag));
      expect(style.color).toBe(light.includes(style.background) ? "#131211" : "#ffffff");
      seen.add(style.background);
    }
    expect(seen.size).toBe(LABEL_COLORS.length);
  });
});

describe("descExcerpt", () => {
  it("skips headings and strips inline markup from the first paragraph", () => {
    const src =
      "# Context\n\nFirst **bold** part with `code` and [a link](http://x).\nsecond _line_\n\nNext paragraph";
    expect(descExcerpt(src)).toBe("First bold part with code and a link. second line");
  });

  it("skips a leading code block and drops list and quote markers", () => {
    expect(descExcerpt("```\nx := 1\n```\n- one\n- two")).toBe("one two");
    expect(descExcerpt("> quoted")).toBe("quoted");
  });

  it("keeps underscores inside words", () => {
    expect(descExcerpt("rename snake_case_name")).toBe("rename snake_case_name");
  });

  it("is empty for a description with only headings", () => {
    expect(descExcerpt("## Only a heading")).toBe("");
    expect(descExcerpt("")).toBe("");
  });
});
