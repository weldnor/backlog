// backlog browse — vanilla JS, no build step (design.md, "The frontend is
// vanilla HTML/CSS/JS with no build step"). Ported from the layout and
// interaction logic of the supplied Claude Design canvas's artboard 1a/1b.
(function () {
  "use strict";

  var STATUS_ORDER = ["todo", "doing", "done", "declined"];
  var PRI_ORDER = ["high", "medium", "low"];

  var STATUS_META = {
    todo: { headLabel: "TODO", fg: "status-todo" },
    doing: { headLabel: "DOING", fg: "status-doing" },
    done: { headLabel: "DONE", fg: "status-done" },
    declined: { headLabel: "DECLINED", fg: "status-declined" }
  };
  // The board's own empty-column copy, kept verbatim from the canvas: it is
  // teaching the same distinctions the README does (design.md).
  var BOARD_EMPTY_NOTE = {
    todo: "Nothing captured here yet.",
    doing: "Nothing in flight.",
    done: "Archive — acted on. Behind --all.",
    declined: "Always in scope for search — a duplicate must not hide behind a filter."
  };
  var PRI_META = {
    high: { label: "HIGH", cls: "tag tag-outline", fg: "pri-high" },
    medium: { label: "MEDIUM", cls: "tag tag-accent", fg: "pri-medium" },
    low: { label: "LOW", cls: "tag tag-neutral", fg: "pri-low" }
  };

  var state = {
    view: "list",
    query: "",
    status: null,
    priority: null,
    tag: null,
    archive: false,
    all: [],        // every task in scope of ?all=1 — for sidebar counts and the tag cloud
    visible: [],     // the server-filtered set for the current filters, pre free-text
    dialogMode: null, // null | "read" | "edit" | "create"
    openId: null,
    openTask: null,
    draft: null,
    error: ""
  };

  // ---------------- markdown, ported from the canvas's md()/inline()/esc() ----------------

  function esc(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }
  function escAttr(s) { return esc(s).replace(/"/g, "&quot;"); }
  function inline(s) {
    return s
      .replace(/`([^`]+)`/g, "<code>$1</code>")
      .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
      .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');
  }
  function md(src) {
    var lines = esc(src || "").split("\n");
    var out = [], para = [], list = [], code = null;
    function flushP() { if (para.length) { out.push("<p>" + inline(para.join(" ")) + "</p>"); para = []; } }
    function flushL() { if (list.length) { out.push("<ul>" + list.map(function (i) { return "<li>" + inline(i) + "</li>"; }).join("") + "</ul>"); list = []; } }
    lines.forEach(function (raw) {
      var line = raw.replace(/\s+$/, "");
      if (code !== null) {
        if (/^```/.test(line)) { out.push("<pre><code>" + code.join("\n") + "</code></pre>"); code = null; }
        else code.push(line);
        return;
      }
      if (/^```/.test(line)) { flushP(); flushL(); code = []; return; }
      if (/^#{1,6}\s/.test(line)) { flushP(); flushL(); out.push("<h3>" + inline(line.replace(/^#{1,6}\s+/, "")) + "</h3>"); return; }
      if (/^[-*]\s+/.test(line)) { flushP(); list.push(line.replace(/^[-*]\s+/, "")); return; }
      if (line === "") { flushP(); flushL(); return; }
      flushL(); para.push(line);
    });
    if (code !== null) out.push("<pre><code>" + code.join("\n") + "</code></pre>");
    flushP(); flushL();
    return out.join("");
  }

  // ---------------- fetch helpers ----------------

  function api(path, opts) {
    return fetch(path, opts).then(function (res) {
      return res.json().catch(function () { return {}; }).then(function (body) {
        if (!res.ok) {
          var err = new Error((body && body.error) || res.statusText);
          err.status = res.status;
          throw err;
        }
        return body;
      });
    });
  }

  function fetchAll() {
    return api("/api/tasks?all=1").then(function (tasks) { state.all = tasks; });
  }

  function fetchVisible() {
    var params = new URLSearchParams();
    if (state.status) {
      params.set("status", state.status);
    } else if (state.archive) {
      params.set("all", "1");
    }
    if (state.priority) params.set("priority", state.priority);
    if (state.tag) params.set("tag", state.tag);
    return api("/api/tasks?" + params.toString()).then(function (tasks) { state.visible = tasks; });
  }

  function refresh() {
    return Promise.all([fetchAll(), fetchVisible()]).then(render);
  }

  function visibleFiltered() {
    var q = state.query.trim().toLowerCase();
    if (!q) return state.visible;
    return state.visible.filter(function (t) {
      var hay = (t.title + " " + t.description + " " + t.tags.join(" ")).toLowerCase();
      return hay.indexOf(q) !== -1;
    });
  }

  // ---------------- rendering ----------------

  function render() {
    renderSidebar();
    renderResults();
    renderDialog();
  }

  function renderSidebar() {
    var counts = {};
    STATUS_ORDER.forEach(function (k) { counts[k] = 0; });
    state.all.forEach(function (t) { counts[t.status] = (counts[t.status] || 0) + 1; });
    document.getElementById("statusList").innerHTML = STATUS_ORDER.map(function (k) {
      var active = state.status === k;
      return '<button class="side-item' + (active ? " is-active" : "") + '" data-status="' + k + '">' +
        "<span>" + k + "</span><span class=\"count\">" + counts[k] + "</span></button>";
    }).join("");

    var priCounts = {};
    PRI_ORDER.forEach(function (k) { priCounts[k] = 0; });
    state.all.forEach(function (t) { priCounts[t.priority] = (priCounts[t.priority] || 0) + 1; });
    document.getElementById("priorityList").innerHTML = PRI_ORDER.map(function (k) {
      var active = state.priority === k;
      return '<button class="side-item' + (active ? " is-active" : "") + '" data-priority="' + k + '">' +
        "<span>" + k + "</span><span class=\"count\">" + priCounts[k] + "</span></button>";
    }).join("");

    var tagSet = [];
    state.all.forEach(function (t) { t.tags.forEach(function (g) { if (tagSet.indexOf(g) === -1) tagSet.push(g); }); });
    tagSet.sort();
    document.getElementById("tagCloud").innerHTML = tagSet.map(function (g) {
      var active = state.tag === g.toLowerCase();
      return '<button class="tag-chip' + (active ? " is-active" : "") + '" data-tag="' + escAttr(g.toLowerCase()) + '">' + esc(g) + "</button>";
    }).join("");
  }

  function currentCommand() {
    var parts = ["backlog", "list"];
    if (state.status) { parts.push("--status", state.status); }
    else if (state.archive) { parts.push("--all"); }
    if (state.priority) parts.push("--priority", state.priority);
    if (state.tag) parts.push("--tag", state.tag);
    parts.push("--json");
    return parts.join(" ");
  }

  function renderResults() {
    var vis = visibleFiltered();

    document.getElementById("resultCount").textContent = vis.length + (vis.length === 1 ? " task" : " tasks");
    document.getElementById("resultCmd").textContent = currentCommand();
    var archiveBtn = document.getElementById("archiveToggle");
    archiveBtn.textContent = state.archive ? "ARCHIVE SHOWN · --all" : "Show archive · --all";
    archiveBtn.classList.toggle("is-active", state.archive);

    document.getElementById("listView").hidden = state.view !== "list";
    document.getElementById("boardView").hidden = state.view !== "board";
    if (state.view === "list") renderList(vis); else renderBoard(vis);
  }

  function priBadge(t) { return PRI_META[t.priority] || { label: t.priority, cls: "tag tag-neutral", fg: "" }; }
  function statusMeta(t) { return STATUS_META[t.status] || { headLabel: t.status, fg: "" }; }
  function tagChips(tags) {
    return tags.map(function (g) { return '<span class="tag tag-neutral">' + esc(g) + "</span>"; }).join("");
  }

  function renderList(vis) {
    document.getElementById("listBody").innerHTML = vis.map(function (t) {
      var p = priBadge(t), st = statusMeta(t);
      var files = t.metadata.source.files.length ? esc(t.metadata.source.files.join("  ·  ")) : "no source location";
      return '<tr data-id="' + t.id + '" class="' + (t.id === state.openId ? "is-open" : "") + '">' +
        "<td>" + String(t.id).padStart(3, "0") + "</td>" +
        '<td><span class="' + p.cls + '">' + p.label + "</span></td>" +
        '<td><div class="row-title">' + esc(t.title) + '</div><div class="row-file">' + files + "</div></td>" +
        '<td><div class="row-tags">' + tagChips(t.tags) + "</div></td>" +
        '<td class="status-label ' + st.fg + '">' + st.headLabel + "</td>" +
        "</tr>";
    }).join("");
    document.getElementById("listEmpty").hidden = vis.length !== 0;
  }

  function renderBoard(vis) {
    document.getElementById("boardView").innerHTML = STATUS_ORDER.map(function (k) {
      var list = vis.filter(function (t) { return t.status === k; });
      var cards = list.map(function (t) {
        var p = priBadge(t);
        var file = t.metadata.source.files.length ? esc(t.metadata.source.files[0]) : "no source location";
        return '<div class="board-card' + (t.priority === "high" ? " pri-high-rule" : "") + '" data-id="' + t.id + '">' +
          '<div class="board-card-top"><span class="board-card-id">' + String(t.id).padStart(3, "0") +
          '</span><span class="pri-label ' + p.fg + '">' + p.label + "</span></div>" +
          '<div class="board-card-title">' + esc(t.title) + "</div>" +
          '<div class="board-card-file">' + file + "</div>" +
          '<div class="board-card-tags">' + tagChips(t.tags) + "</div>" +
          "</div>";
      }).join("");
      var empty = list.length === 0 ? '<div class="board-empty">' + esc(BOARD_EMPTY_NOTE[k]) + "</div>" : "";
      return '<div class="board-col"><div class="board-head"><span class="label">' + STATUS_META[k].headLabel +
        '</span><span class="count">' + list.length + "</span></div>" +
        '<div class="board-body">' + cards + empty + "</div></div>";
    }).join("");
  }

  function taskFilePath(t) {
    var dir = (t.status === "done" || t.status === "declined") ? ".backlog/archive/" : ".backlog/tasks/";
    return dir + t.file;
  }

  function renderReadBody(t) {
    var p = priBadge(t), st = statusMeta(t);
    var html = '<div class="read-top"><span class="' + p.cls + '">' + p.label +
      '</span><span class="read-status ' + st.fg + '">' + st.headLabel + "</span></div>";
    html += '<h2 class="read-title">' + esc(t.title) + "</h2>";
    html += '<div class="hr"></div>';
    html += '<div class="md">' + md(t.description) + "</div>";
    if (t.status === "declined" && t.reason) {
      html += '<div class="decline-callout"><div class="heading">DECLINE REASON — REQUIRED, AUDITABLE</div>' +
        '<div class="text">' + esc(t.reason) + "</div></div>";
    }
    return html;
  }

  function renderAside(t) {
    var files = t.metadata.source.files.length
      ? t.metadata.source.files.map(function (f) { return '<div class="meta-mono">' + esc(f) + "</div>"; }).join("")
      : '<div class="meta-none">none</div>';
    var refs = t.metadata.refs.length
      ? t.metadata.refs.map(function (r) { return '<div class="meta-ref">' + esc(r) + "</div>"; }).join("")
      : '<div class="meta-none">none</div>';
    var branch = t.metadata.source.branch || "—";
    var commit = t.metadata.source.commit ? t.metadata.source.commit.slice(0, 12) : "—";
    return '<aside class="dialog-aside"><div class="aside-heading">Metadata — tool-owned</div>' +
      '<div class="meta-list">' +
      '<div><div class="meta-key">ID</div>' + String(t.id).padStart(3, "0") + "</div>" +
      '<div><div class="meta-key">CREATED</div>' + esc(t.metadata.created) + "</div>" +
      '<div><div class="meta-key">AUTHOR</div>' + esc(t.metadata.author) + "</div>" +
      '<div><div class="meta-key">SOURCE · BRANCH / COMMIT</div><span class="meta-mono">' + esc(branch) + " · " + esc(commit) + "</span></div>" +
      '<div><div class="meta-key">SOURCE · FILES</div>' + files + "</div>" +
      '<div><div class="meta-key">REFS — VERBATIM, NEVER RESOLVED</div>' + refs + "</div>" +
      '<div class="hr" style="margin:2px 0"></div>' +
      '<div class="meta-note">The key set under <code>metadata</code> is closed — an unrecognised key is an error, ' +
      "which is what catches a typo like <code>creted</code>. This panel is read-only for that reason.</div>" +
      "</div></aside>";
  }

  function renderEditFields(d, isCreate) {
    var html = '<div class="field" style="margin-bottom:16px"><label>Title</label>' +
      '<input class="input" id="draftTitle" value="' + escAttr(d.title) + '" style="font-size:16px;font-weight:600;min-height:42px"></div>';

    html += '<div class="edit-row"><div class="field"><label>Status</label><div class="seg">' +
      STATUS_ORDER.map(function (k) {
        return '<label class="seg-opt"><input type="radio" name="draft-status" value="' + k + '"' +
          (d.status === k ? " checked" : "") + ">" + k.toUpperCase() + "</label>";
      }).join("") + "</div></div>";
    html += '<div class="field"><label>Priority — severity</label><div class="seg">' +
      PRI_ORDER.map(function (k) {
        return '<label class="seg-opt"><input type="radio" name="draft-priority" value="' + k + '"' +
          (d.priority === k ? " checked" : "") + ">" + PRI_META[k].label + "</label>";
      }).join("") + "</div></div></div>";

    html += '<div class="field" style="margin-bottom:16px"><label>Tags — comma separated</label>' +
      '<input class="input" id="draftTags" value="' + escAttr(d.tags) + '" style="font-family:var(--font-mono);font-size:13px"></div>';

    if (isCreate) {
      html += '<div class="edit-row"><div class="field"><label>Files — comma separated</label>' +
        '<input class="input" id="draftFiles" value="' + escAttr(d.files || "") + '" style="font-family:var(--font-mono);font-size:13px"></div>' +
        '<div class="field"><label>Refs — comma separated</label>' +
        '<input class="input" id="draftRefs" value="' + escAttr(d.refs || "") + '" style="font-family:var(--font-mono);font-size:13px"></div></div>';
    }

    if (d.status === "declined") {
      html += '<div class="field" style="margin-bottom:16px"><label style="color:var(--color-accent-700)">Reason — required when declining</label>' +
        '<input class="input has-error" id="draftReason" value="' + escAttr(d.reason) + '" placeholder="why this finding will not be acted on"></div>';
    }

    html += '<div class="desc-label"><span class="k">DESCRIPTION</span><span class="v">markdown · the file body, preserved verbatim</span></div>';
    html += '<textarea class="input" id="draftBody" rows="12" style="min-height:230px">' + esc(d.body) + "</textarea>";

    if (state.error) {
      html += '<div class="error-note">' + esc(state.error) + "</div>";
    }

    html += '<div class="edit-actions">' +
      '<button class="btn btn-primary" id="saveBtn">Save · writes one file' +
      '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path><path d="M17 21v-8H7v8"></path><path d="M7 3v5h8"></path></svg></button>' +
      '<button class="btn btn-secondary" id="cancelBtn">Cancel</button></div>';
    return html;
  }

  function renderDialogHead(t) {
    var editing = state.dialogMode === "edit";
    return '<div class="dialog-head"><span class="filename">' + esc(taskFilePath(t)) + "</span>" +
      '<div class="topbar-spacer"></div>' +
      '<button class="btn btn-secondary' + (editing ? " is-active" : "") + '" id="editToggle">' +
      (editing ? "READING VIEW" : "EDIT") +
      '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 20h9"></path><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4Z"></path></svg></button>' +
      '<button class="btn btn-secondary btn-icon" id="closeBtn" aria-label="Close"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"></path><path d="m6 6 12 12"></path></svg></button></div>';
  }

  function renderCreateHead() {
    return '<div class="dialog-head"><span class="filename">new task</span><div class="topbar-spacer"></div>' +
      '<button class="btn btn-secondary btn-icon" id="closeBtn" aria-label="Close"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"></path><path d="m6 6 12 12"></path></svg></button></div>';
  }

  function renderDialog() {
    var backdrop = document.getElementById("dialogBackdrop");
    var dialog = document.getElementById("dialog");

    if (state.dialogMode === "create") {
      backdrop.hidden = false;
      dialog.innerHTML = renderCreateHead() + '<div style="padding:20px 20px 24px">' + renderEditFields(state.draft, true) + "</div>";
      return;
    }
    var t = state.openTask;
    if (state.dialogMode === null || !t) {
      backdrop.hidden = true;
      dialog.innerHTML = "";
      return;
    }
    backdrop.hidden = false;
    if (state.dialogMode === "edit") {
      dialog.innerHTML = renderDialogHead(t) +
        '<div class="dialog-grid"><div class="dialog-left">' + renderEditFields(state.draft, false) + "</div>" + renderAside(t) + "</div>";
    } else {
      dialog.innerHTML = renderDialogHead(t) +
        '<div class="dialog-grid"><div class="dialog-left">' + renderReadBody(t) + "</div>" + renderAside(t) + "</div>";
    }
  }

  // ---------------- actions ----------------

  function pick(key, value) {
    state[key] = state[key] === value ? null : value;
    fetchVisible().then(render);
  }

  function openTask(id) {
    state.error = "";
    api("/api/tasks/" + id).then(function (t) {
      state.openTask = t;
      state.openId = id;
      state.dialogMode = "read";
      state.draft = null;
      render();
    });
  }

  function openCreate() {
    state.error = "";
    state.dialogMode = "create";
    state.draft = { title: "", status: "todo", priority: "medium", tags: "", body: "", reason: "", files: "", refs: "" };
    render();
  }

  function closeDialog() {
    state.dialogMode = null;
    state.openId = null;
    state.openTask = null;
    state.draft = null;
    state.error = "";
    render();
  }

  function toggleEdit() {
    if (state.dialogMode === "edit") {
      state.dialogMode = "read";
      state.draft = null;
    } else {
      var t = state.openTask;
      state.draft = {
        title: t.title, status: t.status, priority: t.priority,
        tags: t.tags.join(", "), body: t.description, reason: t.reason || ""
      };
      state.dialogMode = "edit";
    }
    state.error = "";
    render();
  }

  function cancelEdit() {
    if (state.dialogMode === "create") { closeDialog(); return; }
    state.dialogMode = "read";
    state.draft = null;
    state.error = "";
    render();
  }

  function splitList(s) {
    return (s || "").split(",").map(function (x) { return x.trim(); }).filter(Boolean);
  }

  function save() {
    var d = state.draft;
    state.error = "";

    if (state.dialogMode === "create") {
      var createBody = {
        title: d.title,
        description: d.body,
        tags: splitList(d.tags),
        priority: d.priority,
        files: splitList(d.files),
        refs: splitList(d.refs)
      };
      api("/api/tasks", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(createBody) })
        .then(function () { closeDialog(); return refresh(); })
        .catch(function (err) { state.error = err.message; render(); });
      return;
    }

    var patchBody = {
      title: d.title,
      description: d.body,
      tags: splitList(d.tags),
      priority: d.priority,
      status: d.status,
      // Only carried when the resulting status is declined; leaving declined
      // clears it, matching `backlog set`.
      reason: d.status === "declined" ? d.reason : ""
    };
    api("/api/tasks/" + state.openId, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patchBody) })
      .then(function (t) {
        state.openTask = t;
        state.dialogMode = "read";
        state.draft = null;
        return refresh();
      })
      .catch(function (err) { state.error = err.message; render(); });
  }

  // ---------------- event wiring (delegated: the DOM under these ids is replaced wholesale on every render) ----------------

  document.addEventListener("click", function (e) {
    var el;
    if ((el = e.target.closest("#statusList .side-item"))) { pick("status", el.dataset.status); return; }
    if ((el = e.target.closest("#priorityList .side-item"))) { pick("priority", el.dataset.priority); return; }
    if ((el = e.target.closest("#tagCloud .tag-chip"))) { pick("tag", el.dataset.tag); return; }
    if (e.target.closest("#archiveToggle")) {
      state.archive = !state.archive;
      state.status = null;
      fetchVisible().then(render);
      return;
    }
    if (e.target.closest("#captureBtn")) { openCreate(); return; }
    if ((el = e.target.closest("#listBody tr"))) { openTask(Number(el.dataset.id)); return; }
    if ((el = e.target.closest(".board-card"))) { openTask(Number(el.dataset.id)); return; }
    if (e.target.closest("#editToggle")) { toggleEdit(); return; }
    if (e.target.closest("#closeBtn")) { closeDialog(); return; }
    if (e.target.closest("#cancelBtn")) { cancelEdit(); return; }
    if (e.target.closest("#saveBtn")) { save(); return; }
    if (e.target.id === "dialogBackdrop") { closeDialog(); return; }
  });

  document.addEventListener("change", function (e) {
    if (e.target.name === "view") { state.view = e.target.value; renderResults(); return; }
    if (e.target.name === "draft-status") {
      state.draft.status = e.target.value;
      if (state.draft.status !== "declined") state.draft.reason = "";
      renderDialog();
      return;
    }
    if (e.target.name === "draft-priority") { state.draft.priority = e.target.value; renderDialog(); return; }
  });

  document.addEventListener("input", function (e) {
    if (e.target.id === "queryInput") { state.query = e.target.value; renderResults(); return; }
    if (!state.draft) return;
    if (e.target.id === "draftTitle") state.draft.title = e.target.value;
    else if (e.target.id === "draftTags") state.draft.tags = e.target.value;
    else if (e.target.id === "draftFiles") state.draft.files = e.target.value;
    else if (e.target.id === "draftRefs") state.draft.refs = e.target.value;
    else if (e.target.id === "draftReason") state.draft.reason = e.target.value;
    else if (e.target.id === "draftBody") state.draft.body = e.target.value;
  });

  // ---------------- boot ----------------

  api("/api/repo").then(function (info) {
    if (info.version) document.getElementById("version").textContent = " v" + info.version;
    if (info.name) {
      document.getElementById("repoChip").hidden = false;
      document.getElementById("repoName").textContent = info.name;
      document.getElementById("repoBranch").textContent = info.branch || "—";
    }
  }).catch(function () { /* the chip is decorative; a failure here is not fatal */ });

  refresh();
})();
