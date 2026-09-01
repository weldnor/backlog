## Why

The `browse` web UI is a single 512-line `app.js` that builds every screen by
concatenating HTML strings and re-rendering the whole document on each state
change. It works, but it has no component boundaries, no types against the JSON
the server returns, and no room to grow the detail/edit dialog without the
string-templating getting worse. Moving it to a real React + TypeScript build
gives the UI a structure to grow into while keeping the binary exactly as
self-contained as it is today.

## What Changes

- The `browse` frontend is rewritten as a React + TypeScript single-page app
  built with Vite. `internal/browse/web/{index.html,app.js,style.css}` are
  replaced by a `frontend/` source tree (components, hooks, typed API client)
  and a committed build output.
- The compiled bundle (hashed JS/CSS plus the vendored Archivo font) is
  committed under `internal/browse/web/` and embedded with the existing
  `go:embed`. `go build ./...` and `go install github.com/weldnor/backlog@latest`
  keep working with no Node toolchain and no network — nothing about the Go
  build changes.
- A `just build-web` recipe (and the underlying npm scripts) regenerates the
  committed bundle from `frontend/`. Rebuilding is a developer/CI step, never a
  `go build` step.
- CI gains a job that installs the pinned Node toolchain, rebuilds the bundle
  from source, and fails if the committed output is not in sync — so a stale
  bundle cannot land on the default branch.
- The UI's behavior is preserved: same list/board/detail/edit/create flows,
  same `/api` calls, same Modernist visual design ported to CSS modules or an
  equivalent scoped-styling approach, same offline guarantee.
- Small improvements the component model makes cheap: the task dialog closes on
  `Escape`, traps focus while open, and restores focus to the row that opened
  it. The free-text filter and view toggle keep their current instant,
  request-free behavior.
- **BREAKING** for contributors only: working on the `browse` UI now requires
  Node and `npm install` in `frontend/`. End users and anyone building the Go
  binary are unaffected.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities

- `browse-ui`: The "Offline, self-contained UI" requirement is restated so it
  is satisfied by a compiled, version-controlled bundle rather than
  hand-written static files, keeping the no-network guarantee explicit. A new
  requirement covers keyboard and focus handling for the task dialog
  (`Escape` to close, focus trap, focus restore).
- `ci-pipeline`: A new "Frontend build job" requirement: CI builds the web UI
  from its source with a pinned Node toolchain and fails if the committed
  bundle under `internal/browse/web/` does not match a fresh build.

## Impact

- **Replaced**: `internal/browse/web/app.js`, `internal/browse/web/index.html`,
  `internal/browse/web/style.css`.
- **New**: `frontend/` (React + TypeScript + Vite source, `package.json`,
  `package-lock.json`, `tsconfig.json`, `vite.config.ts`); committed build
  output under `internal/browse/web/` (e.g. `web/index.html` +
  `web/assets/*`); `.gitignore` entries for `frontend/node_modules` and
  `frontend/dist`.
- **Unchanged**: `internal/browse/assets.go` (`go:embed all:web` still points
  at the same directory), all `/api` handlers and their JSON shapes
  (`internal/browse/handlers.go`, `internal/taskview`), the `browse` command
  and its tests' expectations about served content.
- **Tooling**: `justfile` gains `build-web`; `.github/workflows/ci.yml` gains a
  frontend job; a Node version file (`frontend/.nvmrc` or `package.json`
  `engines`) pins the toolchain.
- **Dependencies**: adds a dev-only Node dependency tree (React, Vite,
  TypeScript) scoped to `frontend/`; no new Go dependencies.
