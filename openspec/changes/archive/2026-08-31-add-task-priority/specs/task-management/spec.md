## MODIFIED Requirements

### Requirement: Recording a task
The CLI SHALL provide an `add` command that creates a task from a required title, with optional description, tags, priority and source file references. The command SHALL complete in a single invocation without prompting for input. When no priority is supplied the task SHALL be created with priority `medium`.

#### Scenario: Minimal capture
- **WHEN** `backlog add` is invoked with only a title
- **THEN** a task is created with status `todo`, priority `medium`, no tags, an empty description, and its identifier is reported

#### Scenario: Capture with full context
- **WHEN** `backlog add` is invoked with a title, a description, tags, a priority and one or more file references
- **THEN** all supplied values are stored on the new task

#### Scenario: Missing title
- **WHEN** `backlog add` is invoked without a title or with an empty title
- **THEN** the command exits non-zero, reports the problem, and creates no task

#### Scenario: Rejecting an invalid priority
- **WHEN** `backlog add` is invoked with a priority outside the permitted set
- **THEN** the command exits non-zero, lists the permitted values, and creates no task

#### Scenario: Never interactive
- **WHEN** `backlog add` is invoked with no terminal attached to standard input
- **THEN** the command completes without waiting for input

### Requirement: Listing tasks
The CLI SHALL provide a `list` command that shows tasks in status `todo` and `doing` by default, includes archived tasks when asked for all tasks, and supports filtering by status, by tag and by priority. A priority filter MAY be given more than once, in which case a task matches when its priority is any of those given; it combines with the status and tag filters so that a task is shown only when it satisfies all of them. Results SHALL be ordered deterministically by descending priority, and by ascending identifier among tasks of equal priority. Human-readable output SHALL additionally group the results by status, preserving that order within each group; machine-readable output SHALL present them as a single sequence in that order.

#### Scenario: Default listing
- **WHEN** `backlog list` is run in a backlog containing tasks in every status
- **THEN** only tasks in status `todo` and `doing` are shown

#### Scenario: Listing everything
- **WHEN** `backlog list` is run requesting all tasks
- **THEN** archived tasks are included

#### Scenario: Filtering
- **WHEN** `backlog list` is run with a status filter and a tag filter
- **THEN** only tasks matching both filters are shown

#### Scenario: Filtering by priority
- **WHEN** `backlog list` is run with a priority filter naming `high`
- **THEN** only tasks whose priority is `high` are shown

#### Scenario: Filtering by several priorities
- **WHEN** `backlog list` is run with a priority filter naming `high` and again naming `medium`
- **THEN** tasks of either priority are shown and tasks of priority `low` are not

#### Scenario: Priority and tag filters combine
- **WHEN** `backlog list` is run with a priority filter and a tag filter
- **THEN** only tasks satisfying both are shown

#### Scenario: Rejecting an invalid priority filter
- **WHEN** `backlog list` is run with a priority filter outside the permitted set
- **THEN** the command exits non-zero and lists the permitted values

#### Scenario: Order within a status group
- **WHEN** a status group contains tasks of mixed priority
- **THEN** they are shown highest priority first, and in ascending identifier order among tasks of equal priority

#### Scenario: Order of machine-readable results
- **WHEN** `backlog list` is run requesting JSON in a backlog with tasks of mixed priority
- **THEN** the tasks appear as one sequence ordered by descending priority, then by ascending identifier

#### Scenario: Empty backlog
- **WHEN** `backlog list` is run and no task matches
- **THEN** the command exits zero and reports that nothing matched

### Requirement: Viewing a task
The CLI SHALL provide a `show` command that displays a single task by identifier, including its priority, its description and its recorded metadata.

#### Scenario: Showing an existing task
- **WHEN** `backlog show` is invoked with the identifier of an existing task
- **THEN** the task's identifier, title, status, priority, tags, description and metadata are displayed

#### Scenario: Showing an archived task
- **WHEN** `backlog show` is invoked with the identifier of a task in status `done`
- **THEN** the task is found and displayed

#### Scenario: Unknown identifier
- **WHEN** `backlog show` is invoked with an identifier that does not exist
- **THEN** the command exits non-zero and reports that no such task exists

### Requirement: Changing task status
The CLI SHALL provide a `set` command that changes a task's status to one of `todo`, `doing` or `done`, that changes its priority to one of `high`, `medium` or `low`, and that can attach a free-form reference string to the task. Any combination of the three SHALL be permitted in one invocation, and the command SHALL fail when none of them is supplied. Changing only the priority SHALL leave the status and therefore the task's directory untouched.

#### Scenario: Starting work
- **WHEN** a task in status `todo` is set to `doing`
- **THEN** its status is updated and the task remains in the active task directory

#### Scenario: Completing a task with a reference
- **WHEN** a task is set to `done` with a reference string supplied
- **THEN** the status is updated, the reference is recorded on the task, and the file is moved to the archive

#### Scenario: Raising a task's priority
- **WHEN** `backlog set` is invoked with a priority and no status
- **THEN** the priority is updated, the status is unchanged, and the file stays in the directory it was in

#### Scenario: Changing status and priority together
- **WHEN** `backlog set` is invoked with both a status and a priority
- **THEN** both are updated in a single write

#### Scenario: Rejecting an invalid status
- **WHEN** `backlog set` is invoked with a status outside the permitted set
- **THEN** the command exits non-zero, lists the permitted values, and the task is unchanged

#### Scenario: Rejecting an invalid priority
- **WHEN** `backlog set` is invoked with a priority outside the permitted set
- **THEN** the command exits non-zero, lists the permitted values, and the task is unchanged

#### Scenario: Nothing to change
- **WHEN** `backlog set` is invoked with neither a status, nor a priority, nor a reference
- **THEN** the command exits non-zero and reports that there is nothing to do

#### Scenario: Setting the status a task already has
- **WHEN** a task already in status `doing` is set to `doing`
- **THEN** the command succeeds and the task is left in a valid state

### Requirement: Dual-audience output
Every command that reports tasks SHALL produce human-readable output by default and machine-readable JSON when JSON output is requested. Human-readable listings SHALL group tasks by status and SHALL show each task's priority. JSON output SHALL expose the task's identifier, title, status, priority, tags, description and metadata, with the priority as a top-level string alongside the status rather than inside the metadata block.

#### Scenario: Human reads the backlog
- **WHEN** `backlog list` is run without requesting JSON
- **THEN** tasks are presented grouped by status, each showing its priority, in a form intended for reading in a terminal

#### Scenario: Agent reads the backlog
- **WHEN** `backlog list` is run requesting JSON
- **THEN** the output is valid JSON on standard output with no human-oriented decoration, and each task carries its priority as a top-level field

#### Scenario: Errors stay off the data stream
- **WHEN** a command requesting JSON fails
- **THEN** the diagnostic message is written to standard error and the exit code is non-zero
