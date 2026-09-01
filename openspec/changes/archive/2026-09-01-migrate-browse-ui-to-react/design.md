## Context

See proposal.md — Why. The constraints that shape the approach:

- **`go:embed all:web` and the offline guarantee are fixed points.**
  `internal/browse/assets.go` embeds `internal/browse/web/` and serves it at
  `/`. `browse`'s spec requires the binary to build with no JS toolchain and
  the served page to make zero network requests. Whatever the new front-end
  looks like, its build output has to land back in `internal/browse/web/` as
  plain files, committed to git.
- **The `/api` contract is already the right one.** `taskview.TaskView` /
  `MetaView` / `SourceView` (`internal/taskview/view.go`) is the exact JSON
  the UI consumes today — `GET/POST/PATCH /api/tasks`, `GET /api/repo`. This
  change does not touch any handler; it re-types the client against the same
  shapes.
- **The visual design is already ported.** `internal/browse/web/style.css` is
  the Modernist design-system export, tokens and component classes carried
  over by value, with Archivo vendored locally. The React migration reuses
  these styles rather than re-deriving them.
- **The current app is a single IIFE** (`app.js`) with one `state` object,
  a `render()` that rewrites `#statusList`, `#listBody`, `#boardView`,
  `#dialog` etc. wholesale, and document-level delegated `click`/`change`/
  `input` listeners. Every behavior to preserve is visible there.
- **CI is three parallel jobs** (lint, test, build) on `ubuntu-latest`, Go
  from `go.mod`. There is no Node in CI today.

## Goals / Non-Goals

**Goals:**

- A `frontend/` React + TypeScript + Vite source tree that builds to static
  files committed under `internal/browse/web/`, embedded unchanged.
- Typed API client mirroring `taskview` shapes; one place that can drift is
  caught by code review, not runtime.
- Component decomposition of the four screens (list, board, detail/read,
  edit/create dialog) with the current behavior preserved exactly.
- The dialog gains `Escape`-to-close, a focus trap, and focus restore.
- `just build-web` regenerates the bundle; CI fails on a stale bundle.

**Non-Goals:**

- No change to any `/api` handler, the `browse` command, or `assets.go`'s
  embed directive.
- No visual redesign — same layout, same Modernist tokens/classes.
- No client-side router, no state-management library, no server-side
  rendering, no PWA/service worker.
- No drag-and-drop on the board (still forbidden by the spec).
- No Go-side build orchestration: `go build` never shells out to npm.

## Decisions

### D1: Vite + React 18 + TypeScript, output committed under `internal/browse/web/`

`frontend/` holds `package.json`, `vite.config.ts`, `tsconfig.json`, `src/`.
`vite build` emits to `internal/browse/web/` (configured via `build.outDir`
with `emptyOutDir: true`), producing `index.html` plus `assets/<name>-<hash>.js`
and `.css`. The Archivo `.ttf` stays vendored — imported from `src/` so Vite
fingerprints and copies it, or kept as a `public/` asset; either way it is
emitted into the committed output and referenced relatively.

`emptyOutDir` wiping `internal/browse/web/` on each build is why the vendored
font must live under `frontend/` (in `src/` or `public/`), not only in the
output directory — otherwise a rebuild deletes it. The old hand-written
`index.html`, `app.js`, `style.css` are removed from that directory; from then
on everything there is generated.

*Alternatives considered:* (a) esbuild alone — smaller, but we would hand-roll
HTML templating and asset hashing that Vite gives for free. (b) Keeping output
in a separate `internal/browse/web/dist/` and pointing `go:embed` at it —
avoids mixing source and output, but there is no source left in that directory
after this change, so the extra level is noise. (c) A Go `embed` of a `.tar` —
pointless indirection.

### D2: Rebuild-and-diff freshness check, not a checked-in hash

CI job `frontend`: `actions/setup-node` (version from `frontend/.nvmrc`),
`npm ci`, `npm run build`, then `git diff --exit-code -- internal/browse/web`.
If a fresh build changes the committed files, the job fails. This is the same
pattern as the existing `gofmt -l` check — the repo must already contain the
generated result.

*Alternatives considered:* storing a content hash and comparing — extra
machinery, and the diff output is less useful than seeing the actual file
change. Building the front-end in CI and uploading it as a release artifact
instead of committing — breaks `go install ...@latest`, rejected in the
proposal.

### D3: Typed API client in `src/api.ts`

Hand-written `interface TaskView`, `MetaView`, `SourceView` matching
`internal/taskview/view.go` field-for-field (snake_case JSON keys). Functions:
`listTasks(params)`, `getTask(id)`, `createTask(body)`, `patchTask(id, body)`,
`getRepo()`. Error handling mirrors the current `api()` helper: parse JSON,
throw `Error(body.error || statusText)` with `.status` on non-2xx.

*Alternatives considered:* generating types from Go via a schema — worth doing
if the API grows, overkill for five endpoints; a comment in both files
pointing at each other is enough for now.

### D4: State with `useState`/`useReducer` in a top-level `<App>`, no context library

The current single `state` object becomes a `useReducer` in `<App>` (filters,
view, dialog mode, open task, draft, error). Server data (`all`, `visible`)
moves to a small `useTasks` hook that owns the two fetches (`?all=1` for
sidebar counts + the filtered query) and exposes a `refresh()`. Free-text
filtering stays a pure derived computation over `visible` — no request, matching
the spec's "evaluated locally" requirement. View toggle and filter clicks
re-run only the affected fetch, as today.

*Alternatives considered:* React Query / SWR — real caching value, but adds a
dependency and the data set is tiny and fully refetched on every mutation
anyway.

### D5: Component tree

```
App
├─ TopBar            (wordmark, repo chip, search box, view toggle, Capture)
├─ Sidebar           (StatusList, PriorityList, TagCloud — counts from `all`)
├─ ResultBar         (count, archive toggle, current `backlog list …` command)
├─ ListView          (Table → Row[])
├─ BoardView         (Column[] → Card[])  — fixed todo/doing/done/declined order
└─ TaskDialog        (portal + backdrop)
   ├─ ReadView       (rendered markdown, decline callout)
   ├─ EditForm       (shared by edit + create; reason field only when declined)
   └─ MetadataAside  (read-only)
```

Markdown rendering (`md()`/`inline()`/`esc()` in `app.js`) moves to
`src/markdown.ts` unchanged in behavior — still a tiny hand-rolled renderer, no
library, so nothing new to embed and no XSS surface beyond what exists today
(it escapes first, then adds a fixed set of tags).

### D6: Dialog keyboard/focus behavior

`<TaskDialog>` renders through a portal into a container outside `#root`. On
open it records `document.activeElement`, moves focus to the dialog's first
control, and installs a `keydown` handler: `Escape` → same handler as the close
button (no save); `Tab`/`Shift+Tab` wrap within the dialog's focusable set. On
close it restores focus to the recorded element (the list row or board card).
Backdrop click still closes. This is the one user-visible behavior addition and
is covered by the new `browse-ui` requirement.

### D7: Tooling

- `frontend/.nvmrc` pins Node (current LTS, e.g. `22`). `package.json`
  `engines.node` mirrors it.
- `justfile`: `build-web` → `cd frontend && npm ci && npm run build`. `default`
  recipe left as Go-only (fmt/vet/test) so a Go contributor without Node is
  unaffected; `build-web` is run explicitly when the UI changes.
- `.gitignore`: `frontend/node_modules/`. The Vite output under
  `internal/browse/web/` is **not** ignored — it is committed.
- `package-lock.json` committed for reproducible `npm ci`.

## Risks / Trade-offs

- **Generated files in the repo produce noisy diffs on every UI change.** →
  Output is confined to `internal/browse/web/index.html` and
  `internal/browse/web/assets/`; reviewers look at `frontend/src/`. Hashed
  filenames mean each build replaces the whole `assets/` set — expected, not a
  smell.
- **A contributor edits the UI and forgets `just build-web`.** → CI `frontend`
  job's rebuild-and-diff fails the PR with the exact missing change.
- **`emptyOutDir` deletes the vendored font on a misconfigured build.** → Font
  lives in `frontend/` (source), copied on every build; a from-scratch build
  reproduces the full output directory. CI's clean checkout proves this.
- **Node/npm supply chain enters the project.** → Dev-and-CI only, never in the
  Go build or the shipped binary; `package-lock.json` pins the tree; the
  dependency set is small (React, Vite, TS, types).
- **The hand-rolled markdown renderer is carried forward as-is.** → Behavior
  parity is the goal of this change; replacing it with a library (and vetting
  that library for offline use) is a separate decision.
- **Bundle size vs. the current ~1 KB of inline JS.** → React + app is on the
  order of a few tens of KB gzipped, served from localhost only. Acceptable for
  a local developer tool; no budget set.

## Migration Plan

1. Scaffold `frontend/` (Vite React-TS template), point `build.outDir` at
   `../internal/browse/web`, move the Archivo `.ttf` + `OFL.txt` under
   `frontend/src/fonts/` (or `frontend/public/fonts/`).
2. Port `style.css` verbatim into the Vite pipeline (global stylesheet import
   in `main.tsx`).
3. Port `api()` → `src/api.ts` with types; port `md()` → `src/markdown.ts`.
4. Build components D5, wiring reducer state D4, preserving every behavior in
   `app.js` (filter toggles, archive toggle, view switch, free-text filter,
   open/read/edit/create/save/cancel/close, `backlog list …` command string,
   3-digit id padding, empty-state copy).
5. Add dialog keyboard/focus behavior D6.
6. `npm run build`; commit the output; delete old `index.html`/`app.js`/
   `style.css` source (they now only exist as build output).
7. `just build-web` recipe; `.nvmrc`; `.gitignore`; `package-lock.json`.
8. CI `frontend` job (D2); extend the "overall pipeline result" to include it.
9. Verify existing Go tests in `internal/browse` / `internal/cli` still pass —
   they assert on served routes and `/api`, not on markup, so they should be
   untouched; adjust only if a test pins a literal from the old HTML.
10. Manual check: `backlog browse` offline, DevTools Network shows no external
    request; every flow from `browse_test.go` exercised by hand.

**Rollback:** revert the change set — the old `index.html`/`app.js`/`style.css`
come back and `go:embed` picks them up with no other coordination. No data or
API migration is involved.

## Open Questions

- Exact Node LTS to pin (`22` vs `20`) — safely decided at implementation time;
  does not affect specs, approach, or tasks.
- Whether to split `style.css` into per-component CSS modules now or leave it a
  single global sheet — leaning global to keep the port mechanical; can be
  refactored later without spec impact.
