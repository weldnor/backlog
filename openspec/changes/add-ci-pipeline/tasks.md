## 1. Workflow scaffolding

- [ ] 1.1 Create `.github/workflows/ci.yml` with `name:` and an `on:` block triggering on push to the default branch and on pull requests, and verify the file is valid YAML (`yamllint` or a local Actions syntax check)
- [ ] 1.2 Add a shared checkout step (`actions/checkout@v4`) and a Go setup step (`actions/setup-go@v5` with `go-version-file: go.mod` and `cache: true`) to each job, and verify the resolved Go version in a run log matches the version declared in `go.mod`

## 2. Lint job

- [ ] 2.1 Add the `lint` job running `gofmt -l .` and failing the step if it prints any file path, and verify locally by running the same command against the current tree (expect no output) and against a deliberately misformatted file (expect the file listed and a non-zero exit)
- [ ] 2.2 Add a `go vet ./...` step to the `lint` job, and verify it passes locally against the current tree

## 3. Test job

- [ ] 3.1 Add the `test` job running `go test ./...`, and verify it passes locally, exercising both the unit tests under `internal/` and the root `e2e_test.go`

## 4. Build job

- [ ] 4.1 Add the `build` job running `go build ./...`, and verify it passes locally and produces no compiled artifacts left uncommitted (temp/build output stays out of git status)

## 5. Verification

- [ ] 5.1 Push the branch and confirm in the GitHub Actions run that `lint`, `test`, and `build` appear as three independent, parallel job results on the commit
- [ ] 5.2 Open (or update) a pull request from the branch and confirm the same three checks report status on the pull request
- [ ] 5.3 Deliberately introduce a failure in each job in turn (an unformatted file, a failing test, a compile error) on a scratch commit, confirm the corresponding job goes red while the others stay green, then revert the scratch commit
