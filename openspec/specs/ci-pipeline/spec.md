## Purpose

Defines the automated GitHub Actions pipeline that lints, tests and builds the Go module on every push and pull request, so regressions and formatting drift are caught before they land on the default branch.

## Requirements

### Requirement: Pipeline triggers
The CI pipeline SHALL run automatically on every push to the repository's default branch and on every pull request targeting it.

#### Scenario: Push to default branch
- **WHEN** a commit is pushed directly to the default branch
- **THEN** the pipeline runs against that commit

#### Scenario: Pull request opened or updated
- **WHEN** a pull request is opened, or a new commit is pushed to an open pull request
- **THEN** the pipeline runs against the pull request's head commit

### Requirement: Lint job
The pipeline SHALL run a lint job that checks source formatting with `gofmt` and static analysis with `go vet`. The job SHALL fail if any tracked `.go` file is not `gofmt`-formatted, or if `go vet` reports any issue.

#### Scenario: Unformatted file
- **WHEN** a tracked `.go` file does not match `gofmt` output
- **THEN** the lint job fails and identifies the unformatted file

#### Scenario: Vet issue
- **WHEN** `go vet ./...` reports a problem in any package
- **THEN** the lint job fails and reports the vet output

#### Scenario: Clean source
- **WHEN** all tracked `.go` files are `gofmt`-formatted and `go vet ./...` reports no issues
- **THEN** the lint job passes

### Requirement: Test job
The pipeline SHALL run a test job that executes `go test ./...` across the module, including the repository's end-to-end test suite. The job SHALL fail if any test fails.

#### Scenario: All tests pass
- **WHEN** every test in the module succeeds
- **THEN** the test job passes

#### Scenario: A test fails
- **WHEN** any test in the module fails
- **THEN** the test job fails and reports which test failed

### Requirement: Build job
The pipeline SHALL run a build job that compiles the module with `go build ./...`. The job SHALL fail if compilation fails.

#### Scenario: Successful compilation
- **WHEN** the module compiles without errors
- **THEN** the build job passes

#### Scenario: Compilation error
- **WHEN** the module fails to compile
- **THEN** the build job fails and reports the compiler error

### Requirement: Frontend build job
The pipeline SHALL run a job that rebuilds the `browse` web UI from its
front-end source using a pinned JavaScript toolchain version, and SHALL fail if
the committed, embedded build output does not match the freshly built output.
The pinned toolchain version SHALL be declared in the repository (for example a
Node version file or a `package.json` engines field) so CI and contributors
build against the same version.

#### Scenario: Committed bundle is in sync
- **WHEN** the front-end source and the committed embedded build output describe the same UI
- **THEN** the frontend build job passes

#### Scenario: Committed bundle is stale
- **WHEN** the front-end source has changed but the committed embedded build output was not regenerated
- **THEN** the frontend build job fails and reports that the committed bundle is out of date

#### Scenario: Pinned toolchain version
- **WHEN** the pipeline sets up the JavaScript toolchain for the frontend build job
- **THEN** the toolchain version matches the version pinned in the repository

### Requirement: Go toolchain version
The pipeline SHALL use the Go toolchain version declared in the module's `go.mod` file, so CI compiles and tests against the same version contributors are expected to use locally.

#### Scenario: go.mod version is used
- **WHEN** the pipeline sets up the Go toolchain for any job
- **THEN** the toolchain version matches the version declared in `go.mod`

### Requirement: Overall pipeline result
The pipeline's overall status SHALL be a failure if any of the lint, test, build, or frontend build jobs fails, and a success only if all of them pass. The result SHALL be visible as a status check on the commit and, when applicable, on the pull request.

#### Scenario: One job fails
- **WHEN** the lint, test, build, or frontend build job fails
- **THEN** the pipeline reports an overall failing status on the commit and pull request

#### Scenario: All jobs pass
- **WHEN** the lint, test, build, and frontend build jobs all pass
- **THEN** the pipeline reports an overall successful status on the commit and pull request
