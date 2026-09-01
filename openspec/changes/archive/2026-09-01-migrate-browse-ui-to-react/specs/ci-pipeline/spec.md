## ADDED Requirements

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

### Requirement: Overall pipeline result
The pipeline's overall status SHALL be a failure if any of the lint, test,
build, or frontend build jobs fails, and a success only if all of them pass.
The result SHALL be visible as a status check on the commit and, when
applicable, on the pull request.

#### Scenario: One job fails
- **WHEN** the lint, test, build, or frontend build job fails
- **THEN** the pipeline reports an overall failing status on the commit and pull request

#### Scenario: All jobs pass
- **WHEN** the lint, test, build, and frontend build jobs all pass
- **THEN** the pipeline reports an overall successful status on the commit and pull request
