## REMOVED Requirements

### Requirement: Structural checks
**Reason**: One directory replaces two; there is no archive directory to check for. Restated as "Directory structure checks" below.
**Migration**: None. `validate` no longer reports a missing `.backlog/archive/` directory.

### Requirement: Cross-file checks
**Reason**: Two of its checks — a terminal task outside `archive/`, a non-terminal task inside it — only made sense when a task's directory could disagree with its status. With one directory neither can occur. Restated as "Cross-file consistency checks" below.
**Migration**: None. After `.backlog/archive/*` is moved into `.backlog/tasks/`, these findings no longer arise, and `validate --fix` no longer relocates files by status.

### Requirement: Automatic repair
**Reason**: The "move a task to the directory its status requires" repair has no second directory to move to. Restated as "Repair mode" below without it.
**Migration**: None. `validate --fix` still repairs drifted file names, missing format versions, missing priorities, timestamp formatting and duplicate tags.

## ADDED Requirements

### Requirement: Directory structure checks
Validation SHALL verify that the backlog directory contains the expected `.backlog/tasks/` directory and SHALL report any file inside it that is not a task file.

#### Scenario: Missing task directory
- **WHEN** the `.backlog/tasks/` directory is absent
- **THEN** validation reports it as an error

#### Scenario: Stray file among tasks
- **WHEN** a file that is not a task markdown file is present in the task directory
- **THEN** validation reports it as a warning

### Requirement: Cross-file consistency checks
Validation SHALL detect identifiers used by more than one task and task files whose name slug no longer matches their title. Validation SHALL NOT report a task as misplaced on account of its status, since every task lives in the one directory regardless of status.

#### Scenario: Duplicate identifier
- **WHEN** two task files declare the same identifier
- **THEN** validation reports an error naming both files

#### Scenario: Title changed by hand
- **WHEN** a task's title was edited so that the file name slug no longer matches it
- **THEN** validation reports a warning

#### Scenario: A completed task is not misplaced
- **WHEN** a task in status `done` or `declined` sits in `.backlog/tasks/` alongside active tasks
- **THEN** validation does not report it, since there is only one task directory

### Requirement: Repair mode
Validation SHALL offer an opt-in repair mode that fixes only findings with a single unambiguous correction: renaming a file whose slug drifted from its title, adding a missing format version, adding a missing priority as the default `medium`, normalising timestamp formatting, and de-duplicating tags. It SHALL NOT attempt to repair findings that require a judgement, such as duplicate identifiers, unparseable frontmatter, a priority whose value is outside the permitted set, a declined task with no reason, or a reason recorded on a task that is not declined.

#### Scenario: Repairing a drifted file name
- **WHEN** repair mode is run on a backlog whose task file name no longer matches its title
- **THEN** the file is renamed and the action is reported

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
