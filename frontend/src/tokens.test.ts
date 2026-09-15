import { describe, expect, it } from "vitest";

import { parseDraft } from "./tokens";

describe("parseDraft", () => {
  it("returns a plain line as the title", () => {
    expect(parseDraft("Add fuzzy search")).toEqual({ title: "Add fuzzy search", tags: [] });
  });

  it("parses every token kind out of the spec example", () => {
    expect(parseDraft("Drop zone highlight !high #ui @ann >doing")).toEqual({
      title: "Drop zone highlight",
      priority: "high",
      tags: ["ui"],
      assignee: "ann",
      status: "doing",
    });
  });

  it("maps each priority spelling", () => {
    expect(parseDraft("x !high").priority).toBe("high");
    expect(parseDraft("x !med").priority).toBe("medium");
    expect(parseDraft("x !medium").priority).toBe("medium");
    expect(parseDraft("x !low").priority).toBe("low");
    expect(parseDraft("x !HIGH").priority).toBe("high");
  });

  it("collects multiple tags and keeps a repeated tag once", () => {
    expect(parseDraft("Fix parser #cli #ui #CLI").tags).toEqual(["cli", "ui"]);
  });

  it("accepts any @name as the assignee, the last one winning", () => {
    expect(parseDraft("x @weldnor").assignee).toBe("weldnor");
    expect(parseDraft("x @ann @bob").assignee).toBe("bob");
  });

  it("accepts >status only for new, todo, doing and done", () => {
    for (const s of ["new", "todo", "doing", "done"]) {
      expect(parseDraft("x >" + s).status).toBe(s);
    }
    expect(parseDraft("x >declined")).toEqual({ title: "x >declined", tags: [] });
  });

  it("keeps unknown tokens and bare sigils in the title", () => {
    expect(parseDraft("a !urgent b >later c # d @ e >")).toEqual({
      title: "a !urgent b >later c # d @ e >",
      tags: [],
    });
  });

  it("collapses whitespace in the title", () => {
    expect(parseDraft("  Fix   the\tparser  #cli  ").title).toBe("Fix the parser");
  });

  it("yields an empty title when the line holds only tokens", () => {
    expect(parseDraft("!low #cli").title).toBe("");
  });
});
