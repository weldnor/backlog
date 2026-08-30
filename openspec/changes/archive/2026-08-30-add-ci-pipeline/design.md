## Context

See proposal.md - Why. The module already defines `fmt`, `vet`, `test` and `build` as separate `justfile` recipes; the pipeline mirrors that separation as GitHub Actions jobs rather than introducing a different grouping.

## Goals / Non-Goals

**Goals:**
- One workflow file, three independently reported jobs (lint, test, build), so a contributor can see at a glance which concern broke.
- Fast, cheap runs on GitHub's hosted runners with dependency caching.

**Non-Goals:**
- Branch protection configuration (requiring the check before merge) - that's a repository-settings decision for the owner, not part of this workflow file.
- Cross-platform (Windows/macOS) or multi-version Go matrix builds - the project targets one Go version, declared in `go.mod`.
- Release/publish automation (tagging, building release binaries) - out of scope for a lint/test/build pipeline.

## Decisions

- **Single workflow file (`.github/workflows/ci.yml`) with three jobs** (`lint`, `test`, `build`) rather than three separate workflow files. All three share the same trigger conditions, so one file avoids duplicating the `on:` block, while separate jobs still give independent pass/fail status checks and run in parallel.
- **`actions/setup-go` with `go-version-file: go.mod`** rather than a hardcoded version string. This keeps CI automatically in sync with the version pinned in `go.mod` (per the spec's Go toolchain requirement) with no manual bump when the module's Go version changes.
- **Built-in caching via `actions/setup-go`'s `cache: true`** rather than a hand-rolled `actions/cache` step. The module's dependency graph is small (a single indirect dependency), so the built-in module-cache support is sufficient and simpler to maintain.
- **`gofmt -l` with a non-empty-output check, run as a step in the `lint` job**, rather than `gofmt -w` (which would silently rewrite files) or a separate formatting workflow. `gofmt -l` lists non-conforming files without modifying them, matching the spec's "fail and identify the file" scenario.
- **`go vet ./...` as a second step in the same `lint` job** rather than a separate job, since both are static, fast checks that share the same failure mode (a problem in source, not runtime behavior).
- **Checkout via `actions/checkout`** (standard for all jobs) with no depth or submodule options, since the repository has no submodules and jobs only need the current commit.

## Risks / Trade-offs

- [Pinned third-party action versions can drift out of date] → Pin `actions/checkout` and `actions/setup-go` to specific major versions (e.g. `@v4`, `@v5`) and let Dependabot or manual review bump them; this is a one-line maintenance cost, not a design flaw.
- [`gofmt -l` only covers formatting, not import grouping] → Acceptable for this project; `goimports` or a stricter linter (e.g. `golangci-lint`) can be layered in later as a separate change if needed, without altering this workflow's structure.
- [No branch protection means a red check doesn't actually block a merge yet] → Called out explicitly in the proposal's Impact section as a follow-up left to the repository owner; this change only makes the signal exist.
