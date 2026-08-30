## Purpose

Defines the correctness check that makes hand-editing task files safe: it reports what is broken or untidy in a backlog, distinguishes fatal problems from cosmetic ones, and repairs the unambiguous cases on request.

## ADDED Requirements

### Requirement: Severity model
Validation SHALL classify each finding as either an error — the backlog cannot be read or operated on reliably — or a warning — the backlog is readable but violates a convention. A strict mode SHALL treat warnings as errors.

#### Scenario: Error and warning reported together
- **WHEN** a backlog contains one unreadable task and one task violating a naming convention
- **THEN** the first is reported as an error and the second as a warning

#### Scenario: Strict mode
- **WHEN** validation is run in strict mode on a backlog with warnings and no errors
- **THEN** the warnings are treated as errors

### Requirement: Exit status
Validation SHALL exit zero when no errors were found and non-zero when at least one error was found, so that it can gate a commit hook or an agent's own follow-up.

#### Scenario: Clean backlog
- **WHEN** validation finds nothing wrong
- **THEN** the command exits zero

#### Scenario: Warnings only
- **WHEN** validation finds warnings but no errors and strict mode was not requested
- **THEN** the command exits zero and still reports the warnings

#### Scenario: Errors present
- **WHEN** validation finds at least one error
- **THEN** the command exits non-zero

### Requirement: Structural checks
Validation SHALL verify that the backlog directory contains the expected task and archive directories and SHALL report any file inside them that is not a task file.

#### Scenario: Missing archive directory
- **WHEN** the archive directory is absent
- **THEN** validation reports it

#### Scenario: Stray file among tasks
- **WHEN** a file that is not a task markdown file is present in the task directory
- **THEN** validation reports it

### Requirement: Per-file checks
For each task file, validation SHALL verify that the frontmatter parses, that `id`, `title` and `status` are present and well-formed, that `status` is one of the permitted values, that `tags` is a list of non-empty strings, that timestamps are valid RFC 3339 values, that every key under `metadata` belongs to the closed permitted set, and that the identifier in the frontmatter matches the identifier in the file name.

#### Scenario: Unparseable frontmatter
- **WHEN** a task file's frontmatter is not valid YAML
- **THEN** validation reports an error naming the file

#### Scenario: Misspelled metadata key
- **WHEN** a task file contains an unrecognised key under `metadata`
- **THEN** validation reports an error naming the key and, where a permitted key is similar, suggests it

#### Scenario: Unrecognised top-level field
- **WHEN** a task file contains an unrecognised field at the top level of the frontmatter
- **THEN** validation reports a warning rather than an error

#### Scenario: Identifier disagrees with file name
- **WHEN** a task file named for identifier 7 declares a different identifier in its frontmatter
- **THEN** validation reports an error

### Requirement: Cross-file checks
Validation SHALL detect identifiers used by more than one task, task files whose name slug no longer matches their title, tasks in status `done` outside the archive directory, and tasks not in status `done` inside the archive directory.

#### Scenario: Duplicate identifier
- **WHEN** two task files declare the same identifier
- **THEN** validation reports an error naming both files

#### Scenario: Title changed by hand
- **WHEN** a task's title was edited so that the file name slug no longer matches it
- **THEN** validation reports a warning

#### Scenario: Completed task left among active tasks
- **WHEN** a task in status `done` is present in the task directory
- **THEN** validation reports a warning

### Requirement: References are not resolved
Validation SHALL verify only that entries in `metadata.refs` are non-empty strings. It SHALL NOT interpret their content or check whether they point at anything that exists.

#### Scenario: Arbitrary reference string
- **WHEN** a task carries a reference string naming an external work item
- **THEN** validation accepts it without attempting to resolve it

#### Scenario: Empty reference
- **WHEN** a task carries an empty string in its reference list
- **THEN** validation reports an error

### Requirement: Automatic repair
Validation SHALL offer an opt-in repair mode that fixes only findings with a single unambiguous correction: renaming a file whose slug drifted from its title, moving a task to the directory its status requires, adding a missing format version, normalising timestamp formatting, and de-duplicating tags. It SHALL NOT attempt to repair findings that require a judgement, such as duplicate identifiers or unparseable frontmatter.

#### Scenario: Repairing a drifted file name
- **WHEN** repair mode is run on a backlog whose task file name no longer matches its title
- **THEN** the file is renamed and the action is reported

#### Scenario: Repair leaves ambiguous findings alone
- **WHEN** repair mode is run on a backlog containing two tasks with the same identifier
- **THEN** neither task is modified and the finding is still reported

#### Scenario: Repair is opt-in
- **WHEN** validation is run without requesting repair
- **THEN** no file is modified

### Requirement: Validation output
Validation SHALL report findings grouped by file, each with its severity and an indication of whether repair mode can fix it, and SHALL support machine-readable output carrying the same information.

#### Scenario: Human-readable report
- **WHEN** validation is run without requesting JSON
- **THEN** findings are grouped by file with severities and a summary count

#### Scenario: Machine-readable report
- **WHEN** validation is run requesting JSON
- **THEN** the output is valid JSON listing each finding with its file, severity, message and repairability
