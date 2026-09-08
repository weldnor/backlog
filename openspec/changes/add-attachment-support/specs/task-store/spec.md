## MODIFIED Requirements

### Requirement: Tool-owned metadata block
Task frontmatter SHALL contain a `metadata` mapping holding fields recorded by the tool rather than authored by hand: `schema` (format version), `created` (RFC 3339 timestamp), `author` (`agent` or `human`), `source` (a mapping with optional `branch` and `commit`), and `attachments` (a possibly empty list of typed entries). The set of keys permitted under `metadata` SHALL be closed, and the set of keys permitted on each `attachments` entry — `type`, `path`, `url` — SHALL also be closed.

Each `attachments` entry SHALL declare `type` as exactly `file` or `link`. A `file` entry SHALL carry a non-empty `path` and SHALL NOT carry `url`; `path` SHALL be a slash-separated path relative to the project root. A `link` entry SHALL carry a non-empty `url` and SHALL NOT carry `path`; `url` is a free-form string, stored and reported verbatim, never interpreted or resolved by the CLI. A `path` under `.backlog/attachments/<id>/` for the task it is recorded on names a file the backlog itself physically stores, per the following requirement; any other `path` is a reference the CLI never reads or copies.

#### Scenario: Recording a file reference
- **WHEN** a task is created or an attachment is added with a `file`-type value that does not point under the task's own attachment directory
- **THEN** it is recorded under `metadata.attachments` as `{type: file, path: ...}`, and no file is read or copied

#### Scenario: Recording a link
- **WHEN** a free-form value is attached to a task as a link
- **THEN** it is recorded under `metadata.attachments` as `{type: link, url: ...}`, stored and later reported exactly as supplied

#### Scenario: Git provenance stays separate
- **WHEN** a task is created in a project that is a git repository
- **THEN** the current branch and commit are recorded under `metadata.source`, independently of any attachment

#### Scenario: Malformed attachment entry
- **WHEN** a hand-edited task file declares an `attachments` entry with both `path` and `url`, or with neither, or with a `type` outside `file`/`link`
- **THEN** commands that read the task report it as invalid rather than crashing

## ADDED Requirements

### Requirement: Attachment file storage
A backlog SHALL provide `.backlog/attachments/<id>/` as the directory a `file` attachment's physically stored copy lives in, where `<id>` is the task's identifier formatted the same way its file name's identifier is. Files under this directory belong to the tool, the same way `.backlog/tasks/` and `.backlog/hooks/` do, and SHALL NOT be required to exist ahead of time — the directory is created when a task's first stored attachment is written. Removing a task SHALL also remove its attachment directory, best-effort: a failure to remove the directory SHALL NOT fail the removal of the task itself.

#### Scenario: Storing a file against a task
- **WHEN** a local file is attached to a task with storage requested
- **THEN** a copy is written under that task's `.backlog/attachments/<id>/` directory and the task records a `file` attachment whose `path` points at the copy

#### Scenario: Name collision on storage
- **WHEN** a file is stored against a task whose attachment directory already contains a file of the same name
- **THEN** the new copy is stored under a name that does not overwrite the existing one, and both remain retrievable

#### Scenario: Removing the task removes its stored attachments
- **WHEN** a task with one or more stored attachments is removed
- **THEN** its `.backlog/attachments/<id>/` directory is also removed

#### Scenario: A stored file that no attachment references
- **WHEN** `backlog validate` runs and a file exists under a task's attachment directory that no attachment entry on that task points at
- **THEN** it is reported as a warning, the same way a stray file in the task directory is

#### Scenario: An attachment directory with no task
- **WHEN** `backlog validate` runs and `.backlog/attachments/<id>/` exists for an identifier no task carries
- **THEN** it is reported as a warning

#### Scenario: A missing stored file
- **WHEN** `backlog validate` runs and a task's `attachments` names a `path` under its own attachment directory that does not exist on disk
- **THEN** it is reported as an error, and not as something `--fix` can repair
