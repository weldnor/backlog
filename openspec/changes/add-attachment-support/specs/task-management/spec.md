## MODIFIED Requirements

### Requirement: Recording a task
The CLI SHALL provide an `add` command that creates a task from a required title, with optional description, tags, priority and attachments. An attachment is supplied as `<kind>:<value>`, where `<kind>` is `file` (a path, recorded as a reference only — no file is read or copied) or `link` (a free-form string, stored verbatim). The command SHALL complete in a single invocation without prompting for input. When no priority is supplied the task SHALL be created with priority `medium`.

#### Scenario: Minimal capture
- **WHEN** `backlog add` is invoked with only a title
- **THEN** a task is created with status `todo`, priority `medium`, no tags, an empty description, and its identifier is reported

#### Scenario: Capture with full context
- **WHEN** `backlog add` is invoked with a title, a description, tags, a priority and one or more attachments
- **THEN** all supplied values are stored on the new task

#### Scenario: Missing title
- **WHEN** `backlog add` is invoked without a title or with an empty title
- **THEN** the command exits non-zero, reports the problem, and creates no task

#### Scenario: Rejecting an invalid priority
- **WHEN** `backlog add` is invoked with a priority outside the permitted set
- **THEN** the command exits non-zero, lists the permitted values, and creates no task

#### Scenario: Rejecting an invalid attachment
- **WHEN** `backlog add` is invoked with an attachment whose `<kind>` is not `file` or `link`, or whose `<value>` is empty
- **THEN** the command exits non-zero, reports the problem, and creates no task

#### Scenario: Never interactive
- **WHEN** `backlog add` is invoked with no terminal attached to standard input
- **THEN** the command completes without waiting for input

### Requirement: Changing task status
The CLI SHALL provide a `set` command that changes a task's status to one of `todo`, `doing`, `done` or `declined`, that changes its priority to one of `high`, `medium` or `low`, and that records the reason a task was declined. Any combination SHALL be permitted in one invocation, and the command SHALL fail when none is supplied. No status change SHALL move the task file to a different directory.

Setting a task to `declined` SHALL require a reason and SHALL fail without one, so that no decline can be recorded that a later reader cannot audit. Supplying a reason for any status other than `declined` SHALL fail. Supplying a reason alone SHALL be permitted only for a task already in status `declined`, and SHALL replace the recorded text. Setting a declined task to any other status SHALL remove its reason.

#### Scenario: Starting work
- **WHEN** a task in status `todo` is set to `doing`
- **THEN** its status is updated

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
- **WHEN** `backlog set` is invoked with neither a status, nor a priority, nor a reason
- **THEN** the command exits non-zero and reports that there is nothing to do

#### Scenario: Setting the status a task already has
- **WHEN** a task already in status `doing` is set to `doing`
- **THEN** the command succeeds and the task is left in a valid state

### Requirement: Removing a task
The CLI SHALL provide an `rm` command that permanently deletes a task by identifier, together with any attachments the backlog physically stores for it. Removal SHALL be reserved for a task that should never have been recorded — a duplicate, a mis-capture, an accidental entry — and SHALL NOT be the way a reviewer records a decision not to act on a finding, which is what status `declined` is for. The CLI SHALL enforce no such distinction; it is a matter of guidance, and `rm` SHALL delete whatever identifier it is given.

#### Scenario: Removing a task
- **WHEN** `backlog rm` is invoked with the identifier of an existing task
- **THEN** the task file is deleted and the removal is reported

#### Scenario: Removing a task with stored attachments
- **WHEN** `backlog rm` is invoked with the identifier of a task that has one or more backlog-owned stored attachments
- **THEN** the task file is deleted and its attachment directory is also removed

#### Scenario: Removing a declined task
- **WHEN** `backlog rm` is invoked with the identifier of a task in status `declined`
- **THEN** the task file is deleted, since removal is available for any identifier

#### Scenario: Removing an unknown task
- **WHEN** `backlog rm` is invoked with an identifier that does not exist
- **THEN** the command exits non-zero and nothing is deleted

## ADDED Requirements

### Requirement: Attaching evidence to a task
The CLI SHALL provide an `attach` command with `add` and `rm` subcommands that add or remove one attachment on an existing task, scoped to that one task the same way `link add`/`link rm` are. `attach add` SHALL accept either a positional `<kind>:<value>` — `<kind>` being `file` (a path reference, recorded without reading or copying anything) or `link` (a free-form string, stored verbatim) — or an `--upload <local-path>` option that reads the file at `<local-path>`, writes a copy into the task's `.backlog/attachments/<id>/` directory, and records a `file` attachment whose path points at that copy. Exactly one of the positional form or `--upload` SHALL be supplied per invocation. `attach rm` SHALL accept a `<kind>:<value>` naming an attachment exactly as recorded, remove it from the task, and, when its `path` names a file under the task's own attachment directory, also delete that file.

#### Scenario: Attaching a file reference
- **WHEN** `backlog attach add` is invoked with a task identifier and `file:<path>`
- **THEN** the task records a `file` attachment with that path and no file is read or copied

#### Scenario: Attaching a link
- **WHEN** `backlog attach add` is invoked with a task identifier and `link:<value>`
- **THEN** the task records a `link` attachment with that value, stored verbatim

#### Scenario: Uploading a file
- **WHEN** `backlog attach add` is invoked with a task identifier and `--upload <local-path>` naming a file that exists
- **THEN** a copy of the file is written under the task's attachment directory and the task records a `file` attachment pointing at the copy

#### Scenario: Uploading a file that does not exist
- **WHEN** `backlog attach add --upload` names a path that does not exist
- **THEN** the command exits non-zero, reports the problem, and the task is unchanged

#### Scenario: Both forms supplied at once
- **WHEN** `backlog attach add` is invoked with both a positional `<kind>:<value>` and `--upload`
- **THEN** the command exits non-zero and reports that only one may be given

#### Scenario: Removing a reference attachment
- **WHEN** `backlog attach rm` is invoked with a task identifier and a `<kind>:<value>` matching an attachment that is a plain reference
- **THEN** the attachment is removed from the task and no file on disk is affected

#### Scenario: Removing a stored attachment
- **WHEN** `backlog attach rm` is invoked with a task identifier and a `<kind>:<value>` matching a `file` attachment whose path is under the task's own attachment directory
- **THEN** the attachment is removed from the task and the stored file is deleted

#### Scenario: Removing an attachment that does not exist
- **WHEN** `backlog attach rm` is invoked with a `<kind>:<value>` the task does not carry
- **THEN** the command exits non-zero, reports that no such attachment exists, and the task is unchanged
