## REMOVED Requirements

### Requirement: Directory layout
**Reason**: The two-directory split stored the same fact — terminal versus non-terminal status — that every task file already carries in its `status` field, at the cost of moving files on every status change and of drift checks that only existed because a file's directory could disagree with its status. Replaced by "Single task directory" below.
**Migration**: Move every file under `.backlog/archive/` into `.backlog/tasks/` (`git mv .backlog/archive/* .backlog/tasks/`, then delete the empty `.backlog/archive/`). File names do not change and no task content is affected. The new binary reads only `.backlog/tasks/`.

## ADDED Requirements

### Requirement: Single task directory
A backlog SHALL consist of exactly one task directory, `.backlog/tasks/`, holding every task regardless of status. The CLI SHALL NOT require or read any configuration file, and a change of a task's status SHALL NOT move its file to a different directory.

#### Scenario: A fresh backlog has one task directory
- **WHEN** `backlog init` is run in a project with no backlog
- **THEN** `.backlog/tasks/` is created and no other task directory is created

#### Scenario: Task reaches a terminal status
- **WHEN** a task's status is set to `done` or `declined`
- **THEN** its status is updated and its file stays in `.backlog/tasks/`, its name changing only if its title changed

#### Scenario: Task leaves a terminal status
- **WHEN** a task in status `done` or `declined` is set to `todo` or `doing`
- **THEN** its status is updated and its file stays in `.backlog/tasks/`

#### Scenario: Moving between terminal statuses
- **WHEN** a task in status `done` is set to `declined`
- **THEN** its status is updated and its file stays in `.backlog/tasks/`

## MODIFIED Requirements

### Requirement: Identifier allocation and file naming
Each task SHALL have an identifier unique within the backlog, allocated as the lowest positive integer not already in use in the task directory. A task file SHALL be named `<id>-<slug>.md`, where `<id>` is the identifier zero-padded to at least three digits and `<slug>` is a kebab-case reduction of the title. Concurrent creation SHALL NOT produce two tasks with the same identifier.

#### Scenario: First task in an empty backlog
- **WHEN** a task is created in a backlog with no existing tasks
- **THEN** it receives identifier 1 and is stored as `001-<slug>.md`

#### Scenario: Identifier reuse after deletion
- **WHEN** a task is created in a backlog whose highest identifier is 9 and whose identifier 4 is unused
- **THEN** the new task receives identifier 4

#### Scenario: Two tasks created at the same moment
- **WHEN** two processes create a task simultaneously in the same backlog
- **THEN** each task receives a distinct identifier and neither creation overwrites the other
