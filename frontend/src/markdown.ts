// A tiny hand-rolled markdown renderer, ported verbatim in behavior from the
// old app.js md()/inline()/esc()/escAttr(). It escapes HTML first and then
// adds a fixed set of tags, so there is no XSS surface beyond what the old
// renderer had. No library, nothing new to embed.

export function esc(s: unknown): string {
  return String(s == null ? "" : s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

export function escAttr(s: unknown): string {
  return esc(s).replace(/"/g, "&quot;");
}

export function inline(s: string): string {
  return s
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');
}

export function md(src: string): string {
  const lines = esc(src || "").split("\n");
  const out: string[] = [];
  let para: string[] = [];
  let list: string[] = [];
  let code: string[] | null = null;

  function flushP() {
    if (para.length) {
      out.push("<p>" + inline(para.join(" ")) + "</p>");
      para = [];
    }
  }
  function flushL() {
    if (list.length) {
      out.push(
        "<ul>" + list.map((i) => "<li>" + inline(i) + "</li>").join("") + "</ul>",
      );
      list = [];
    }
  }

  lines.forEach((raw) => {
    const line = raw.replace(/\s+$/, "");
    if (code !== null) {
      if (/^```/.test(line)) {
        out.push("<pre><code>" + code.join("\n") + "</code></pre>");
        code = null;
      } else {
        code.push(line);
      }
      return;
    }
    if (/^```/.test(line)) {
      flushP();
      flushL();
      code = [];
      return;
    }
    if (/^#{1,6}\s/.test(line)) {
      flushP();
      flushL();
      out.push("<h3>" + inline(line.replace(/^#{1,6}\s+/, "")) + "</h3>");
      return;
    }
    if (/^[-*]\s+/.test(line)) {
      flushP();
      list.push(line.replace(/^[-*]\s+/, ""));
      return;
    }
    if (line === "") {
      flushP();
      flushL();
      return;
    }
    flushL();
    para.push(line);
  });

  if (code !== null) {
    out.push("<pre><code>" + (code as string[]).join("\n") + "</code></pre>");
  }
  flushP();
  flushL();
  return out.join("");
}
