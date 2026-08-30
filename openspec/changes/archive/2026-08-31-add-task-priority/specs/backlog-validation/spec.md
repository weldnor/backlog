## MODIFIED Requirements

### Requirement: Per-file checks
For each task file, validation SHALL verify that the frontmatter parses, that `id`, `title` and `status` are present and well-formed, that `status` is one of the permitted values, that `priority`, when present, is one of the permitted values, that `tags` is a list of non-empty strings, that timestamps are valid RFC 3339 values, that every key under `metadata` belongs to the closed permitted set, and that the identifier in the frontmatter matches the identifier in the file name. A task file carrying no `priority` field SHALL be reported as a warning rather than an error, since the file is still fully readable and the omission has a single unambiguous correction.

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

#### Scenario: Identifier disagrees with file name
- **WHEN** a task file named for identifier 7 declares a different identifier in its frontmatter
- **THEN** validation reports an error

### Requirement: Automatic repair
Validation SHALL offer an opt-in repair mode that fixes only findings with a single unambiguous correction: renaming a file whose slug drifted from its title, moving a task to the directory its status requires, adding a missing format version, adding a missing priority as the default `medium`, normalising timestamp formatting, and de-duplicating tags. It SHALL NOT attempt to repair findings that require a judgement, such as duplicate identifiers, unparseable frontmatter, or a priority whose value is outside the permitted set.

#### Scenario: Repairing a drifted file name
- **WHEN** repair mode is run on a backlog whose task file name no longer matches its title
- **THEN** the file is renamed and the action is reported

#### Scenario: Adding a missing priority
- **WHEN** repair mode is run on a backlog containing a task file with no priority field
- **THEN** the file is rewritten declaring priority `medium` and the action is reported

#### Scenario: Repair leaves an unrecognised priority alone
- **WHEN** repair mode is run on a task whose priority is a value outside the permitted set
- **THEN** the value is not replaced with the default, the task is not modified, and the finding is still reported

#### Scenario: Repair leaves ambiguous findings alone
- **WHEN** repair mode is run on a backlog containing two tasks with the same identifier
- **THEN** neither task is modified and the finding is still reported

#### Scenario: Repair is opt-in
- **WHEN** validation is run without requesting repair
- **THEN** no file is modified
