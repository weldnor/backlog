import { describe, expect, it } from "vitest";

import { esc, escAttr, inline, md } from "./markdown";

describe("esc / escAttr", () => {
  it("escapes the HTML metacharacters", () => {
    expect(esc('a & b < c > d')).toBe("a &amp; b &lt; c &gt; d");
    expect(esc(null)).toBe("");
    expect(escAttr('say "hi" <b>')).toBe("say &quot;hi&quot; &lt;b&gt;");
  });
});

describe("inline", () => {
  it("renders code, bold and links", () => {
    expect(inline("run `backlog list`")).toBe("run <code>backlog list</code>");
    expect(inline("**loud**")).toBe("<strong>loud</strong>");
    expect(inline("see [docs](https://x)")).toBe('see <a href="https://x">docs</a>');
  });
});

describe("md", () => {
  it("renders a heading", () => {
    expect(md("# Title")).toBe("<h3>Title</h3>");
    expect(md("### Also a title")).toBe("<h3>Also a title</h3>");
  });

  it("renders an unordered list", () => {
    expect(md("- one\n- two")).toBe("<ul><li>one</li><li>two</li></ul>");
    expect(md("* star")).toBe("<ul><li>star</li></ul>");
  });

  it("joins consecutive lines into one paragraph", () => {
    expect(md("first line\nsecond line")).toBe("<p>first line second line</p>");
  });

  it("separates paragraphs on a blank line", () => {
    expect(md("a\n\nb")).toBe("<p>a</p><p>b</p>");
  });

  it("renders a fenced code block without escaping-then-tagging its contents", () => {
    expect(md("```\nx := 1\n```")).toBe("<pre><code>x := 1</code></pre>");
  });

  it("closes an unterminated code fence at end of input", () => {
    expect(md("```\nunclosed")).toBe("<pre><code>unclosed</code></pre>");
  });

  it("escapes HTML in body text", () => {
    expect(md("<script>alert(1)</script>")).toBe(
      "<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>",
    );
  });

  it("renders inline code and bold inside a paragraph", () => {
    expect(md("a `b` and **c**")).toBe(
      "<p>a <code>b</code> and <strong>c</strong></p>",
    );
  });
});
