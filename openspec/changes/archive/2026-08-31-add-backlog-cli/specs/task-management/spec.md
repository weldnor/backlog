## Purpose

Covers the commands that create a backlog and move tasks through their lifecycle — recording a finding, reviewing what has accumulated, inspecting one task, changing its status and removing it — for both a person at a terminal and an agent consuming machine-readable output.

## ADDED Requirements

### Requirement: Initialising a backlog
The CLI SHALL provide an `init` command that creates the backlog directory structure in the current directory. Running `init` in a directory that already has a backlog SHALL NOT destroy existing tasks.

#### Scenario: Initialising a fresh project
- **WHEN** `backlog init` is run in a project with no backlog
- **THEN** the task and archive directories are created and the command reports where the backlog was created

#### Scenario: Re-running init
- **WHEN** `backlog init` is run in a project that already has a backlog containing tasks
- **THEN** all existing tasks are left untouched

### Requirement: Recording a task
The CLI SHALL provide an `add` command that creates a task from a required title, with optional description, tags and source file references. The command SHALL complete in a single invocation without prompting for input.

#### Scenario: Minimal capture
- **WHEN** `backlog add` is invoked with only a title
- **THEN** a task is created with status `todo`, no tags, an empty description, and its identifier is reported

#### Scenario: Capture with full context
- **WHEN** `backlog add` is invoked with a title, a description, tags and one or more file references
- **THEN** all supplied values are stored on the new task

#### Scenario: Missing title
- **WHEN** `backlog add` is invoked without a title or with an empty title
- **THEN** the command exits non-zero, reports the problem, and creates no task

#### Scenario: Never interactive
- **WHEN** `backlog add` is invoked with no terminal attached to standard input
- **THEN** the command completes without waiting for input

### Requirement: Listing tasks
The CLI SHALL provide a `list` command that shows tasks in status `todo` and `doing` by default, includes archived tasks when asked for all tasks, and supports filtering by status and by tag. Results SHALL be ordered deterministically.

#### Scenario: Default listing
- **WHEN** `backlog list` is run in a backlog containing tasks in every status
- **THEN** only tasks in status `todo` and `doing` are shown

#### Scenario: Listing everything
- **WHEN** `backlog list` is run requesting all tasks
- **THEN** archived tasks are included

#### Scenario: Filtering
- **WHEN** `backlog list` is run with a status filter and a tag filter
- **THEN** only tasks matching both filters are shown

#### Scenario: Empty backlog
- **WHEN** `backlog list` is run and no task matches
- **THEN** the command exits zero and reports that nothing matched

### Requirement: Viewing a task
The CLI SHALL provide a `show` command that displays a single task by identifier, including its description and its recorded metadata.

#### Scenario: Showing an existing task
- **WHEN** `backlog show` is invoked with the identifier of an existing task
- **THEN** the task's identifier, title, status, tags, description and metadata are displayed

#### Scenario: Showing an archived task
- **WHEN** `backlog show` is invoked with the identifier of a task in status `done`
- **THEN** the task is found and displayed

#### Scenario: Unknown identifier
- **WHEN** `backlog show` is invoked with an identifier that does not exist
- **THEN** the command exits non-zero and reports that no such task exists

### Requirement: Changing task status
The CLI SHALL provide a `set` command that changes a task's status to one of `todo`, `doing` or `done`, and that can additionally attach a free-form reference string to the task.

#### Scenario: Starting work
- **WHEN** a task in status `todo` is set to `doing`
- **THEN** its status is updated and the task remains in the active task directory

#### Scenario: Completing a task with a reference
- **WHEN** a task is set to `done` with a reference string supplied
- **THEN** the status is updated, the reference is recorded on the task, and the file is moved to the archive

#### Scenario: Rejecting an invalid status
- **WHEN** `backlog set` is invoked with a status outside the permitted set
- **THEN** the command exits non-zero, lists the permitted values, and the task is unchanged

#### Scenario: Setting the status a task already has
- **WHEN** a task already in status `doing` is set to `doing`
- **THEN** the command succeeds and the task is left in a valid state

### Requirement: Removing a task
The CLI SHALL provide an `rm` command that permanently deletes a task by identifier from either the active or the archive directory.

#### Scenario: Removing a task
- **WHEN** `backlog rm` is invoked with the identifier of an existing task
- **THEN** the task file is deleted and the removal is reported

#### Scenario: Removing an unknown task
- **WHEN** `backlog rm` is invoked with an identifier that does not exist
- **THEN** the command exits non-zero and nothing is deleted

### Requirement: Dual-audience output
Every command that reports tasks SHALL produce human-readable output by default and machine-readable JSON when JSON output is requested. Human-readable listings SHALL group tasks by status. JSON output SHALL expose the task's identifier, title, status, tags, description and metadata.

#### Scenario: Human reads the backlog
- **WHEN** `backlog list` is run without requesting JSON
- **THEN** tasks are presented grouped by status in a form intended for reading in a terminal

#### Scenario: Agent reads the backlog
- **WHEN** `backlog list` is run requesting JSON
- **THEN** the output is valid JSON on standard output with no human-oriented decoration

#### Scenario: Errors stay off the data stream
- **WHEN** a command requesting JSON fails
- **THEN** the diagnostic message is written to standard error and the exit code is non-zero
