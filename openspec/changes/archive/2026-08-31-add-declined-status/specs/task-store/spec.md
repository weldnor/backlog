## MODIFIED Requirements

### Requirement: Directory layout
A backlog SHALL consist of exactly two task directories: `.backlog/tasks/` holding tasks in status `todo` or `doing`, and `.backlog/archive/` holding tasks in status `done` or `declined`. The CLI SHALL NOT require or read any configuration file.

#### Scenario: Task reaches done
- **WHEN** a task's status is set to `done`
- **THEN** its file is moved from `.backlog/tasks/` to `.backlog/archive/` with its name unchanged

#### Scenario: Task is declined
- **WHEN** a task's status is set to `declined`
- **THEN** its file is moved from `.backlog/tasks/` to `.backlog/archive/` with its name unchanged

#### Scenario: Task leaves done
- **WHEN** an archived task's status is set to `todo` or `doing`
- **THEN** its file is moved from `.backlog/archive/` back to `.backlog/tasks/`

#### Scenario: Task leaves declined
- **WHEN** a task in status `declined` is set to `todo` or `doing`
- **THEN** its file is moved from `.backlog/archive/` back to `.backlog/tasks/`

#### Scenario: Moving between terminal statuses
- **WHEN** a task in status `done` is set to `declined`
- **THEN** its status is updated and the file stays in `.backlog/archive/`

### Requirement: Author-owned frontmatter fields
Task frontmatter SHALL contain the top-level fields `id`, `title`, `status`, `priority`, `reason` and `tags`, which represent deliberate authoring intent and are safe to edit by hand. `id` SHALL be a positive integer. `title` SHALL be a non-empty single-line string. `status` SHALL be exactly one of `todo`, `doing`, `done`, `declined`, of which `done` and `declined` are terminal. `priority` SHALL be exactly one of `high`, `medium`, `low`, which are ordered from most to least severe. `tags` SHALL be a possibly empty list of non-empty strings.

`reason` SHALL be a non-empty string recording why a finding was declined, and SHALL be present exactly when the status is `declined`: a task in status `declined` SHALL carry a `reason`, and a task in any other status SHALL NOT. A task that leaves status `declined` SHALL have its `reason` removed, since the field describes a state the task is no longer in and version control retains what it said.

A task file that carries no `priority` field SHALL be read as though it declared `medium`, so that a backlog written before the field existed stays fully operable. Every task file the CLI writes SHALL carry an explicit `priority`.

#### Scenario: Unknown top-level field
- **WHEN** a task file contains a top-level frontmatter field the CLI does not recognise
- **THEN** the field is preserved on write and the CLI continues to operate on the task

#### Scenario: Invalid status value
- **WHEN** a task file declares a status outside the permitted set
- **THEN** commands that read the task report it as invalid rather than crashing

#### Scenario: Invalid priority value
- **WHEN** a task file declares a priority outside the permitted set
- **THEN** commands that read the task report it as invalid rather than crashing, in the same way as an invalid status

#### Scenario: Priority absent from an older task file
- **WHEN** a task file written before this field existed is read
- **THEN** the task is treated as priority `medium` and every command operates on it normally

#### Scenario: Priority written on every new task
- **WHEN** the CLI creates a task file
- **THEN** the file declares an explicit `priority`, even when the author supplied none

#### Scenario: Priority survives an unrelated write
- **WHEN** a task whose priority is `high` has its status changed
- **THEN** the written file still declares priority `high`

#### Scenario: Declined task carries its reason
- **WHEN** a task is set to status `declined` with a reason supplied
- **THEN** the written file declares both `status: declined` and the `reason` text

#### Scenario: Reason is removed when the task is reopened
- **WHEN** a task in status `declined` is set to `todo`
- **THEN** the written file declares `status: todo` and carries no `reason` field

#### Scenario: Declined task with no reason
- **WHEN** a hand-edited task file declares status `declined` and no `reason`
- **THEN** commands that read the task report it as invalid rather than crashing

#### Scenario: Reason on a task that is not declined
- **WHEN** a hand-edited task file declares a `reason` and a status other than `declined`
- **THEN** commands that read the task report it as invalid rather than crashing

#### Scenario: Reason survives an unrelated write
- **WHEN** a task in status `declined` has its priority changed
- **THEN** the written file still declares its original `reason`
