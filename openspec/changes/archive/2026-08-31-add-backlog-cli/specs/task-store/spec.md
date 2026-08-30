## Purpose

Defines how a project's backlog is represented on disk: where it lives, how the CLI finds it, the format of a task file, and the rules that keep the store readable and safely hand-editable by both people and agents.

## ADDED Requirements

### Requirement: Backlog root discovery
The CLI SHALL locate the backlog by searching for a `.backlog/` directory in the current working directory and then in each ancestor directory, using the first one found. If no `.backlog/` directory is found, every command other than `init` SHALL fail with a non-zero exit code and a message explaining that no backlog was found.

#### Scenario: Command run from a nested directory
- **WHEN** a command is run from a subdirectory of a project whose root contains `.backlog/`
- **THEN** the CLI operates on that project's backlog

#### Scenario: No backlog in the directory tree
- **WHEN** `backlog list` is run in a directory that has no `.backlog/` in it or any ancestor
- **THEN** the command exits non-zero and reports that no backlog was found

#### Scenario: Nearest backlog wins
- **WHEN** a project containing `.backlog/` is nested inside another project that also contains `.backlog/`
- **THEN** the CLI operates on the nearest `.backlog/` directory

### Requirement: Directory layout
A backlog SHALL consist of exactly two task directories: `.backlog/tasks/` holding tasks in status `todo` or `doing`, and `.backlog/archive/` holding tasks in status `done`. The CLI SHALL NOT require or read any configuration file.

#### Scenario: Task reaches done
- **WHEN** a task's status is set to `done`
- **THEN** its file is moved from `.backlog/tasks/` to `.backlog/archive/` with its name unchanged

#### Scenario: Task leaves done
- **WHEN** an archived task's status is set to `todo` or `doing`
- **THEN** its file is moved from `.backlog/archive/` back to `.backlog/tasks/`

### Requirement: Task file format
A task SHALL be a single UTF-8 markdown file consisting of a YAML frontmatter block followed by a markdown body. The body SHALL be the task description and MAY be empty.

#### Scenario: Reading a task file
- **WHEN** the CLI reads a task file with valid frontmatter and a body
- **THEN** the frontmatter supplies the task's fields and the body supplies its description

#### Scenario: Description spanning multiple paragraphs
- **WHEN** a task body contains multiple markdown paragraphs
- **THEN** the full body is preserved verbatim on read and on any write that does not change the description

### Requirement: Author-owned frontmatter fields
Task frontmatter SHALL contain the top-level fields `id`, `title`, `status` and `tags`, which represent deliberate authoring intent and are safe to edit by hand. `id` SHALL be a positive integer. `title` SHALL be a non-empty single-line string. `status` SHALL be exactly one of `todo`, `doing`, `done`. `tags` SHALL be a possibly empty list of non-empty strings.

#### Scenario: Unknown top-level field
- **WHEN** a task file contains a top-level frontmatter field the CLI does not recognise
- **THEN** the field is preserved on write and the CLI continues to operate on the task

#### Scenario: Invalid status value
- **WHEN** a task file declares a status outside the permitted set
- **THEN** commands that read the task report it as invalid rather than crashing

### Requirement: Tool-owned metadata block
Task frontmatter SHALL contain a `metadata` mapping holding fields recorded by the tool rather than authored by hand: `schema` (format version), `created` (RFC 3339 timestamp), `author` (`agent` or `human`), `source` (a mapping with optional `files`, `branch` and `commit`), and `refs` (a possibly empty list of free-form strings). The set of keys permitted under `metadata` SHALL be closed.

#### Scenario: Recording where a finding was made
- **WHEN** a task is created with file references supplied
- **THEN** they are recorded under `metadata.source.files` together with the current branch and commit when the project is a git repository

#### Scenario: Linking a task to external work
- **WHEN** a free-form reference string is attached to a task
- **THEN** it is appended to `metadata.refs` and the CLI makes no attempt to interpret or resolve it

### Requirement: No stored modification time
Task files SHALL NOT store a last-modified timestamp. Modification history is supplied by version control.

#### Scenario: Editing a task description
- **WHEN** a task's description is changed
- **THEN** the resulting file diff contains only the changed description and no timestamp change

### Requirement: Identifier allocation and file naming
Each task SHALL have an identifier unique within the backlog, allocated as the lowest positive integer not already in use across both task directories. A task file SHALL be named `<id>-<slug>.md`, where `<id>` is the identifier zero-padded to at least three digits and `<slug>` is a kebab-case reduction of the title. Concurrent creation SHALL NOT produce two tasks with the same identifier.

#### Scenario: First task in an empty backlog
- **WHEN** a task is created in a backlog with no existing tasks
- **THEN** it receives identifier 1 and is stored as `001-<slug>.md`

#### Scenario: Identifier reuse after deletion
- **WHEN** a task is created in a backlog whose highest identifier is 9 and whose identifier 4 is unused
- **THEN** the new task receives identifier 4

#### Scenario: Two tasks created at the same moment
- **WHEN** two processes create a task simultaneously in the same backlog
- **THEN** each task receives a distinct identifier and neither creation overwrites the other

### Requirement: Durable writes
Any command that modifies a task SHALL leave the store in a consistent state: a task file is never observed partially written, and a failure during a write SHALL NOT destroy the previous content.

#### Scenario: Process interrupted mid-write
- **WHEN** the process is terminated while writing a task file
- **THEN** the task file on disk is either the complete previous version or the complete new version
