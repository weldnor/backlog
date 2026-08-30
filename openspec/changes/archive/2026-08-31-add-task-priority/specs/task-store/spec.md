## MODIFIED Requirements

### Requirement: Author-owned frontmatter fields
Task frontmatter SHALL contain the top-level fields `id`, `title`, `status`, `priority` and `tags`, which represent deliberate authoring intent and are safe to edit by hand. `id` SHALL be a positive integer. `title` SHALL be a non-empty single-line string. `status` SHALL be exactly one of `todo`, `doing`, `done`. `priority` SHALL be exactly one of `high`, `medium`, `low`, which are ordered from most to least severe. `tags` SHALL be a possibly empty list of non-empty strings.

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
