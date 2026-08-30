## Purpose

Covers the commands that create a backlog and move tasks through their lifecycle — recording a finding, reviewing what has accumulated, inspecting one task, changing its status and removing it — for both a person at a terminal and an agent consuming machine-readable output.

## Requirements

### Requirement: Initialising a backlog
The CLI SHALL provide an `init` command that creates the backlog directory structure in the current directory. Running `init` in a directory that already has a backlog SHALL NOT destroy existing tasks.

#### Scenario: Initialising a fresh project
- **WHEN** `backlog init` is run in a project with no backlog
- **THEN** the task and archive directories are created and the command reports where the backlog was created

#### Scenario: Re-running init
- **WHEN** `backlog init` is run in a project that already has a backlog containing tasks
- **THEN** all existing tasks are left untouched

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
The CLI SHALL provide a `list` command that shows tasks in status `todo` and `doing` by default, includes archived tasks — those in status `done` or `declined` — when asked for all tasks, and supports filtering by status, by tag and by priority. A priority filter MAY be given more than once, in which case a task matches when its priority is any of those given; it combines with the status and tag filters so that a task is shown only when it satisfies all of them. Results SHALL be ordered deterministically by descending priority, and by ascending identifier among tasks of equal priority. Human-readable output SHALL additionally group the results by status in the order `todo`, `doing`, `done`, `declined`, preserving that order within each group; machine-readable output SHALL present them as a single sequence in that order.

#### Scenario: Default listing
- **WHEN** `backlog list` is run in a backlog containing tasks in every status
- **THEN** only tasks in status `todo` and `doing` are shown

#### Scenario: Declined tasks are excluded by default
- **WHEN** `backlog list` is run in a backlog containing a declined task
- **THEN** that task is not shown

#### Scenario: Listing everything
- **WHEN** `backlog list` is run requesting all tasks
- **THEN** archived tasks are included

#### Scenario: Listing all tasks includes declined ones
- **WHEN** `backlog list` is run requesting all tasks in a backlog containing a declined task
- **THEN** that task is shown, in the group after the tasks in status `done`

#### Scenario: Filtering to declined tasks
- **WHEN** `backlog list` is run with a status filter naming `declined`
- **THEN** only declined tasks are shown, whether or not all tasks were requested

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

#### Scenario: Declined tasks keep their priority
- **WHEN** a backlog contains a declined task whose priority is `high`
- **THEN** that priority is shown unchanged and does not affect the task's disposition

#### Scenario: Order of machine-readable results
- **WHEN** `backlog list` is run requesting JSON in a backlog with tasks of mixed priority
- **THEN** the tasks appear as one sequence ordered by descending priority, then by ascending identifier

#### Scenario: Empty backlog
- **WHEN** `backlog list` is run and no task matches
- **THEN** the command exits zero and reports that nothing matched

### Requirement: Viewing a task
The CLI SHALL provide a `show` command that displays a single task by identifier, including its priority, its description, its reason when it is declined, and its recorded metadata.

#### Scenario: Showing an existing task
- **WHEN** `backlog show` is invoked with the identifier of an existing task
- **THEN** the task's identifier, title, status, priority, tags, description and metadata are displayed

#### Scenario: Showing an archived task
- **WHEN** `backlog show` is invoked with the identifier of a task in status `done`
- **THEN** the task is found and displayed

#### Scenario: Showing a declined task
- **WHEN** `backlog show` is invoked with the identifier of a task in status `declined`
- **THEN** the task is found and displayed together with the reason it was declined

#### Scenario: Unknown identifier
- **WHEN** `backlog show` is invoked with an identifier that does not exist
- **THEN** the command exits non-zero and reports that no such task exists

### Requirement: Changing task status
The CLI SHALL provide a `set` command that changes a task's status to one of `todo`, `doing`, `done` or `declined`, that changes its priority to one of `high`, `medium` or `low`, that records the reason a task was declined, and that can attach a free-form reference string to the task. Any combination SHALL be permitted in one invocation, and the command SHALL fail when none is supplied. Changing only the priority SHALL leave the status and therefore the task's directory untouched.

Setting a task to `declined` SHALL require a reason and SHALL fail without one, so that no decline can be recorded that a later reader cannot audit. Supplying a reason for any status other than `declined` SHALL fail. Supplying a reason alone SHALL be permitted only for a task already in status `declined`, and SHALL replace the recorded text. Setting a declined task to any other status SHALL remove its reason.

#### Scenario: Starting work
- **WHEN** a task in status `todo` is set to `doing`
- **THEN** its status is updated and the task remains in the active task directory

#### Scenario: Completing a task with a reference
- **WHEN** a task is set to `done` with a reference string supplied
- **THEN** the status is updated, the reference is recorded on the task, and the file is moved to the archive

#### Scenario: Declining a task
- **WHEN** a task is set to `declined` with a reason supplied
- **THEN** the status is updated, the reason is recorded on the task, and the file is moved to the archive

#### Scenario: Declining without a reason
- **WHEN** a task is set to `declined` with no reason supplied
- **THEN** the command exits non-zero, reports that a reason is required, and the task is unchanged

#### Scenario: Revising the reason on a declined task
- **WHEN** `backlog set` is invoked with a reason and no status on a task already in status `declined`
- **THEN** the recorded reason is replaced, the status is unchanged, and the file stays in the archive

#### Scenario: Reason supplied for another status
- **WHEN** `backlog set` is invoked with a reason and a status of `done`
- **THEN** the command exits non-zero, reports that a reason applies only to a declined task, and the task is unchanged

#### Scenario: Reason supplied for a task that is not declined
- **WHEN** `backlog set` is invoked with a reason and no status on a task in status `todo`
- **THEN** the command exits non-zero and the task is unchanged

#### Scenario: Reopening a declined task
- **WHEN** a task in status `declined` is set to `todo`
- **THEN** the status is updated, the reason is removed, and the file is moved back to the active task directory

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
- **WHEN** `backlog set` is invoked with neither a status, nor a priority, nor a reason, nor a reference
- **THEN** the command exits non-zero and reports that there is nothing to do

#### Scenario: Setting the status a task already has
- **WHEN** a task already in status `doing` is set to `doing`
- **THEN** the command succeeds and the task is left in a valid state

### Requirement: Removing a task
The CLI SHALL provide an `rm` command that permanently deletes a task by identifier from either the active or the archive directory. Removal SHALL be reserved for a task that should never have been recorded — a duplicate, a mis-capture, an accidental entry — and SHALL NOT be the way a reviewer records a decision not to act on a finding, which is what status `declined` is for. The CLI SHALL enforce no such distinction; it is a matter of guidance, and `rm` SHALL delete whatever identifier it is given.

#### Scenario: Removing a task
- **WHEN** `backlog rm` is invoked with the identifier of an existing task
- **THEN** the task file is deleted and the removal is reported

#### Scenario: Removing a declined task
- **WHEN** `backlog rm` is invoked with the identifier of a task in status `declined`
- **THEN** the task file is deleted, since removal is available for any identifier

#### Scenario: Removing an unknown task
- **WHEN** `backlog rm` is invoked with an identifier that does not exist
- **THEN** the command exits non-zero and nothing is deleted

### Requirement: Dual-audience output
Every command that reports tasks SHALL produce human-readable output by default and machine-readable JSON when JSON output is requested. Human-readable listings SHALL group tasks by status and SHALL show each task's priority. JSON output SHALL expose the task's identifier, title, status, priority, reason, tags, description and metadata, with the priority and the reason as top-level strings alongside the status rather than inside the metadata block. The reason SHALL be an empty string for a task that is not declined, so that the shape of the JSON does not vary with status.

#### Scenario: Human reads the backlog
- **WHEN** `backlog list` is run without requesting JSON
- **THEN** tasks are presented grouped by status, each showing its priority, in a form intended for reading in a terminal

#### Scenario: Agent reads the backlog
- **WHEN** `backlog list` is run requesting JSON
- **THEN** the output is valid JSON on standard output with no human-oriented decoration, and each task carries its priority as a top-level field

#### Scenario: Agent reads a declined task
- **WHEN** a command reports a declined task as JSON
- **THEN** the task carries `declined` as its top-level status and the reason as a top-level string

#### Scenario: Reason is present but empty on a live task
- **WHEN** a command reports a task in status `todo` as JSON
- **THEN** the reason field is present and empty rather than absent

#### Scenario: Errors stay off the data stream
- **WHEN** a command requesting JSON fails
- **THEN** the diagnostic message is written to standard error and the exit code is non-zero
