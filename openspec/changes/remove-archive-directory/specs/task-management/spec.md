## REMOVED Requirements

### Requirement: Listing tasks
**Reason**: The default scope (only `todo` and `doing`) and the `--all` / `--status` flags were shaped around the archive directory as a place tasks fall out of view into. With one directory and no archive, listing is redefined below: it shows every status by default and selects a single status through subcommands.
**Migration**: Replace `backlog list --all` with `backlog list`. Replace `backlog list --status <s>` — and the old bare `backlog list`, which implied `todo` and `doing` — with the subcommand `backlog list <s>`, run once per status wanted. `--tag` and `--priority` are unchanged.

## ADDED Requirements

### Requirement: Listing tasks by status
The CLI SHALL provide a `list` command that, invoked with no subcommand, shows tasks in every status. Four subcommands — `todo`, `doing`, `done`, `declined` — SHALL each narrow the listing to tasks in that one status. There SHALL be no flag for selecting status. The command SHALL support filtering by tag and by priority, applied equally to the bare command and to each subcommand. A priority filter MAY be given more than once, in which case a task matches when its priority is any of those given; the tag, priority and subcommand selectors combine so that a task is shown only when it satisfies all of them. Results SHALL be ordered deterministically by descending priority, and by ascending identifier among tasks of equal priority. Human-readable output SHALL additionally group the results by status in the order `todo`, `doing`, `done`, `declined`, preserving that order within each group; machine-readable output SHALL present them as a single sequence in that order.

#### Scenario: Default listing shows every status
- **WHEN** `backlog list` is run in a backlog containing tasks in every status
- **THEN** tasks in all four statuses are shown, grouped in the order `todo`, `doing`, `done`, `declined`

#### Scenario: Narrowing to one status
- **WHEN** `backlog list declined` is run
- **THEN** only tasks in status `declined` are shown

#### Scenario: Narrowing to completed tasks
- **WHEN** `backlog list done` is run
- **THEN** only tasks in status `done` are shown

#### Scenario: A status subcommand excludes other statuses
- **WHEN** `backlog list todo` is run in a backlog that also contains a declined task
- **THEN** the declined task is not shown

#### Scenario: Unknown subcommand
- **WHEN** `backlog list` is run with a word that is not one of the four status subcommands
- **THEN** the command exits non-zero and lists the permitted subcommands

#### Scenario: Filtering a subcommand by tag
- **WHEN** `backlog list todo --tag bug` is run
- **THEN** only tasks in status `todo` that carry the tag `bug` are shown

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

#### Scenario: Nothing matches
- **WHEN** `backlog list` is run and no task matches
- **THEN** the command exits zero and reports that nothing matched

## MODIFIED Requirements

### Requirement: Initialising a backlog
The CLI SHALL provide an `init` command that creates the backlog directory structure in the current directory. Running `init` in a directory that already has a backlog SHALL NOT destroy existing tasks.

#### Scenario: Initialising a fresh project
- **WHEN** `backlog init` is run in a project with no backlog
- **THEN** the task directory is created and the command reports where the backlog was created

#### Scenario: Re-running init
- **WHEN** `backlog init` is run in a project that already has a backlog containing tasks
- **THEN** all existing tasks are left untouched

### Requirement: Changing task status
The CLI SHALL provide a `set` command that changes a task's status to one of `todo`, `doing`, `done` or `declined`, that changes its priority to one of `high`, `medium` or `low`, that records the reason a task was declined, and that can attach a free-form reference string to the task. Any combination SHALL be permitted in one invocation, and the command SHALL fail when none is supplied. No status change SHALL move the task file to a different directory.

Setting a task to `declined` SHALL require a reason and SHALL fail without one, so that no decline can be recorded that a later reader cannot audit. Supplying a reason for any status other than `declined` SHALL fail. Supplying a reason alone SHALL be permitted only for a task already in status `declined`, and SHALL replace the recorded text. Setting a declined task to any other status SHALL remove its reason.

#### Scenario: Starting work
- **WHEN** a task in status `todo` is set to `doing`
- **THEN** its status is updated

#### Scenario: Completing a task with a reference
- **WHEN** a task is set to `done` with a reference string supplied
- **THEN** the status is updated and the reference is recorded on the task

#### Scenario: Declining a task
- **WHEN** a task is set to `declined` with a reason supplied
- **THEN** the status is updated and the reason is recorded on the task

#### Scenario: Declining without a reason
- **WHEN** a task is set to `declined` with no reason supplied
- **THEN** the command exits non-zero, reports that a reason is required, and the task is unchanged

#### Scenario: Revising the reason on a declined task
- **WHEN** `backlog set` is invoked with a reason and no status on a task already in status `declined`
- **THEN** the recorded reason is replaced and the status is unchanged

#### Scenario: Reason supplied for another status
- **WHEN** `backlog set` is invoked with a reason and a status of `done`
- **THEN** the command exits non-zero, reports that a reason applies only to a declined task, and the task is unchanged

#### Scenario: Reason supplied for a task that is not declined
- **WHEN** `backlog set` is invoked with a reason and no status on a task in status `todo`
- **THEN** the command exits non-zero and the task is unchanged

#### Scenario: Reopening a declined task
- **WHEN** a task in status `declined` is set to `todo`
- **THEN** the status is updated and the reason is removed

#### Scenario: Raising a task's priority
- **WHEN** `backlog set` is invoked with a priority and no status
- **THEN** the priority is updated and the status is unchanged

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
The CLI SHALL provide an `rm` command that permanently deletes a task by identifier. Removal SHALL be reserved for a task that should never have been recorded — a duplicate, a mis-capture, an accidental entry — and SHALL NOT be the way a reviewer records a decision not to act on a finding, which is what status `declined` is for. The CLI SHALL enforce no such distinction; it is a matter of guidance, and `rm` SHALL delete whatever identifier it is given.

#### Scenario: Removing a task
- **WHEN** `backlog rm` is invoked with the identifier of an existing task
- **THEN** the task file is deleted and the removal is reported

#### Scenario: Removing a declined task
- **WHEN** `backlog rm` is invoked with the identifier of a task in status `declined`
- **THEN** the task file is deleted, since removal is available for any identifier

#### Scenario: Removing an unknown task
- **WHEN** `backlog rm` is invoked with an identifier that does not exist
- **THEN** the command exits non-zero and nothing is deleted
