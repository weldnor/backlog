## Why

The `backlog` CLI is a Go module with `fmt`, `vet`, `test` and `build` steps already codified in the `justfile`, but nothing enforces them automatically. A broken build, a failing test, or unformatted code can land on the default branch or in a pull request unnoticed until someone runs `just` locally. A GitHub Actions pipeline that runs lint, test and build on every push and pull request closes that gap.

## What Changes

- New GitHub Actions workflow at `.github/workflows/ci.yml` that runs on pushes to the default branch and on every pull request.
- A **lint** job that runs `gofmt -l` (failing if any file is unformatted) and `go vet ./...`.
- A **test** job that runs `go test ./...`, covering both unit tests and the repository's `e2e_test.go`.
- A **build** job that runs `go build ./...` to confirm the module compiles.
- The Go toolchain version used in CI is pinned to the version declared in `go.mod`.
- Module dependencies are cached between runs to keep the pipeline fast.
- The workflow fails (non-zero) if any job fails, so a red CI run is visible on the PR and on the commit.

## Capabilities

### New Capabilities
- `ci-pipeline`: The GitHub Actions workflow that lints, tests and builds the module on push and pull request — its triggers, jobs, Go toolchain version, caching and pass/fail behavior.

### Modified Capabilities
<!-- None: no existing runtime capability's requirements change. -->

## Impact

- New file: `.github/workflows/ci.yml`. No application code, CLI behavior, or on-disk task format is affected.
- Contributors and pull requests now get automated feedback; a red check blocks silent regressions from merging unnoticed (branch protection enforcing that is a separate, later decision left to the repository owner).
- No new runtime dependencies; the workflow uses the Go toolchain and commands already defined in the `justfile`.
