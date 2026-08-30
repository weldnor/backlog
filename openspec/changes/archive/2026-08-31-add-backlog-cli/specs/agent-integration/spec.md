## Purpose

Defines how a coding agent learns to use the backlog: the skill files the CLI installs into a project for Claude Code, what guidance they must carry so findings are captured well and sparingly, and how those files are kept in step with the binary.

## ADDED Requirements

### Requirement: Skill installation
Initialising a backlog SHALL install two Claude Code skill files into the project: a capture skill covering the recording of a finding during work, and a triage skill covering the review of accumulated findings. The files SHALL be placed in the project so that they are committed alongside the backlog rather than installed for the user globally.

#### Scenario: Skills installed on init
- **WHEN** `backlog init` is run in a project
- **THEN** the capture skill and the triage skill are written into the project's Claude Code skill directory

#### Scenario: Skills travel with the repository
- **WHEN** a project containing a backlog is cloned
- **THEN** both skills are present without any further installation step

### Requirement: Separate triggers for capture and triage
The capture skill and the triage skill SHALL be distinct files with distinct descriptions, so that each is loaded only in its own situation. The capture skill SHALL describe recording a finding encountered during unrelated work. The triage skill SHALL describe reviewing and dispositioning accumulated tasks.

#### Scenario: Agent notices an unrelated problem mid-task
- **WHEN** an agent working on one task encounters a problem outside its scope
- **THEN** the capture skill is the one whose description matches the situation

#### Scenario: User asks what has accumulated
- **WHEN** a user asks for the backlog to be reviewed
- **THEN** the triage skill is the one whose description matches the situation

### Requirement: Capture threshold guidance
The capture skill SHALL state both when to record a finding and when not to. It SHALL require that a finding is recorded only when it is outside the scope of the current task, is not being fixed now, and concerns the repository rather than transient session state. It SHALL name categories that must not be recorded, including stylistic preferences, speculative refactoring, and problems already covered by work in progress.

#### Scenario: Finding qualifies
- **WHEN** an agent finds a defect it is not fixing and which is unrelated to its current task
- **THEN** the skill directs it to record the finding and return to its work

#### Scenario: Finding does not qualify
- **WHEN** an agent has a stylistic preference about code it is reading
- **THEN** the skill directs it not to record anything

### Requirement: Search before recording
The capture skill SHALL require a search of the existing backlog before a task is created, and SHALL direct the agent to extend or leave alone an existing task rather than create a near-duplicate. The CLI SHALL NOT perform automatic duplicate detection.

#### Scenario: Similar task already exists
- **WHEN** a search before recording returns a task describing the same problem
- **THEN** the skill directs the agent not to create a second task

#### Scenario: Nothing similar found
- **WHEN** a search before recording returns no relevant task
- **THEN** the skill directs the agent to create the task

### Requirement: Recording provenance
The capture skill SHALL require that source file references are supplied when the finding is tied to specific locations in the code, since that context is available to the agent at the moment of capture and expensive to reconstruct later.

#### Scenario: Finding tied to a code location
- **WHEN** an agent records a finding it observed in a specific file
- **THEN** the skill requires that file reference to be supplied with the task

### Requirement: Planning-system independence
The CLI SHALL contain no knowledge of any specific planning or issue-tracking system. Guidance about promoting a task into such a system SHALL live only in the triage skill, which SHALL detect at run time which system the project uses and SHALL record the resulting link as a free-form reference on the task.

#### Scenario: Project uses a planning system
- **WHEN** the triage skill runs in a project that has a planning system present
- **THEN** it promotes the selected task into that system and records the resulting identifier as a reference on the task

#### Scenario: Project uses no planning system
- **WHEN** the triage skill runs in a project with no planning system present
- **THEN** triage still works and the task's disposition is recorded without an external reference

#### Scenario: Binary stays neutral
- **WHEN** the CLI attaches a reference string to a task
- **THEN** it stores the string unchanged and applies no system-specific interpretation

### Requirement: Skill version stamping and staleness
Installed skill files SHALL record the version of the binary that produced them. Validation SHALL report a warning when an installed skill was produced by an older version than the binary being run.

#### Scenario: Skills are current
- **WHEN** validation runs and the installed skills carry the running binary's version
- **THEN** no staleness warning is reported

#### Scenario: Skills are behind the binary
- **WHEN** validation runs and an installed skill carries an older version
- **THEN** a warning names the skill and explains how to refresh it

### Requirement: Preserving customised skills
Initialisation SHALL NOT silently overwrite an installed skill file that has been modified since it was written. Overwriting SHALL require an explicit force option, and SHALL warn that local modifications will be lost.

#### Scenario: Unmodified skill is refreshed
- **WHEN** `backlog init` is re-run and an installed skill is unmodified
- **THEN** it is rewritten to the current version without warning

#### Scenario: Modified skill without force
- **WHEN** `backlog init` is re-run and an installed skill has been edited locally
- **THEN** the file is left unchanged and the command reports that it was skipped

#### Scenario: Modified skill with force
- **WHEN** `backlog init` is re-run with the force option and an installed skill has been edited locally
- **THEN** the file is overwritten and the command warns that local modifications were replaced
