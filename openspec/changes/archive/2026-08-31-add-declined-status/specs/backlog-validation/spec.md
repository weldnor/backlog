## MODIFIED Requirements

### Requirement: Per-file checks
For each task file, validation SHALL verify that the frontmatter parses, that `id`, `title` and `status` are present and well-formed, that `status` is one of the permitted values, that `priority`, when present, is one of the permitted values, that `reason` is present when and only when the status is `declined`, that `tags` is a list of non-empty strings, that timestamps are valid RFC 3339 values, that every key under `metadata` belongs to the closed permitted set, and that the identifier in the frontmatter matches the identifier in the file name. A task file carrying no `priority` field SHALL be reported as a warning rather than an error, since the file is still fully readable and the omission has a single unambiguous correction.

A declined task with no `reason`, and a `reason` on a task in any other status, SHALL each be reported as an error rather than a warning: the pairing is what makes a decline auditable, and neither case has a correction that does not require writing or deleting prose.

#### Scenario: Unparseable frontmatter
- **WHEN** a task file's frontmatter is not valid YAML
- **THEN** validation reports an error naming the file

#### Scenario: Misspelled metadata key
- **WHEN** a task file contains an unrecognised key under `metadata`
- **THEN** validation reports an error naming the key and, where a permitted key is similar, suggests it

#### Scenario: Unrecognised top-level field
- **WHEN** a task file contains an unrecognised field at the top level of the frontmatter
- **THEN** validation reports a warning rather than an error

#### Scenario: Priority outside the permitted set
- **WHEN** a task file declares a priority that is not `high`, `medium` or `low`
- **THEN** validation reports an error naming the value and listing the permitted ones

#### Scenario: Priority is missing
- **WHEN** a task file declares no priority at all
- **THEN** validation reports a warning marked as repairable

#### Scenario: Priority is not a string
- **WHEN** a task file's priority is a list or a mapping rather than a string
- **THEN** validation reports an error

#### Scenario: Declined task with no reason
- **WHEN** a task file declares status `declined` and carries no `reason`
- **THEN** validation reports an error marked as not repairable

#### Scenario: Reason on a task that is not declined
- **WHEN** a task file declares a `reason` and a status other than `declined`
- **THEN** validation reports an error marked as not repairable

#### Scenario: Empty reason on a declined task
- **WHEN** a task file declares status `declined` and a `reason` that is empty or only whitespace
- **THEN** validation reports an error, since a reason nobody can read is the state the field exists to prevent

#### Scenario: Identifier disagrees with file name
- **WHEN** a task file named for identifier 7 declares a different identifier in its frontmatter
- **THEN** validation reports an error

### Requirement: Cross-file checks
Validation SHALL detect identifiers used by more than one task, task files whose name slug no longer matches their title, tasks in a terminal status — `done` or `declined` — outside the archive directory, and tasks in a non-terminal status inside the archive directory.

#### Scenario: Duplicate identifier
- **WHEN** two task files declare the same identifier
- **THEN** validation reports an error naming both files

#### Scenario: Title changed by hand
- **WHEN** a task's title was edited so that the file name slug no longer matches it
- **THEN** validation reports a warning

#### Scenario: Completed task left among active tasks
- **WHEN** a task in status `done` is present in the task directory
- **THEN** validation reports a warning

#### Scenario: Declined task left among active tasks
- **WHEN** a task in status `declined` is present in the task directory
- **THEN** validation reports a warning

#### Scenario: Active task left in the archive
- **WHEN** a task in status `todo` is present in the archive directory
- **THEN** validation reports a warning

### Requirement: Automatic repair
Validation SHALL offer an opt-in repair mode that fixes only findings with a single unambiguous correction: renaming a file whose slug drifted from its title, moving a task to the directory its status requires, adding a missing format version, adding a missing priority as the default `medium`, normalising timestamp formatting, and de-duplicating tags. It SHALL NOT attempt to repair findings that require a judgement, such as duplicate identifiers, unparseable frontmatter, a priority whose value is outside the permitted set, a declined task with no reason, or a reason recorded on a task that is not declined.

#### Scenario: Repairing a drifted file name
- **WHEN** repair mode is run on a backlog whose task file name no longer matches its title
- **THEN** the file is renamed and the action is reported

#### Scenario: Moving a declined task to the archive
- **WHEN** repair mode is run on a backlog containing a declined task in the task directory
- **THEN** the file is moved to the archive and the action is reported

#### Scenario: Adding a missing priority
- **WHEN** repair mode is run on a backlog containing a task file with no priority field
- **THEN** the file is rewritten declaring priority `medium` and the action is reported

#### Scenario: Repair leaves an unrecognised priority alone
- **WHEN** repair mode is run on a task whose priority is a value outside the permitted set
- **THEN** the value is not replaced with the default, the task is not modified, and the finding is still reported

#### Scenario: Repair does not invent a reason
- **WHEN** repair mode is run on a task in status `declined` that carries no reason
- **THEN** the task is not modified and the finding is still reported

#### Scenario: Repair does not delete a misplaced reason
- **WHEN** repair mode is run on a task in status `todo` that carries a reason
- **THEN** the reason is not removed, the task is not modified, and the finding is still reported

#### Scenario: Repair leaves ambiguous findings alone
- **WHEN** repair mode is run on a backlog containing two tasks with the same identifier
- **THEN** neither task is modified and the finding is still reported

#### Scenario: Repair is opt-in
- **WHEN** validation is run without requesting repair
- **THEN** no file is modified
