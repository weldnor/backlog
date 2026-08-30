## Purpose

Defines how a coding agent learns to use the backlog: the skill files the CLI installs into a project for Claude Code, what guidance they must carry so findings are captured well and sparingly, and how those files are kept in step with the binary.

## Requirements

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
The capture skill SHALL require a search of the existing backlog before a task is created, and SHALL direct the agent to extend or leave alone an existing task rather than create a near-duplicate. The search SHALL be the default one, which covers live and declined tasks alike, so that a previously rejected finding is seen without the agent having to ask for it. The CLI SHALL NOT perform automatic duplicate detection.

#### Scenario: Similar task already exists
- **WHEN** a search before recording returns a task describing the same problem
- **THEN** the skill directs the agent not to create a second task

#### Scenario: Nothing similar found
- **WHEN** a search before recording returns no relevant task
- **THEN** the skill directs the agent to create the task

#### Scenario: The search sees declined tasks
- **WHEN** the capture skill's search runs against a backlog containing a declined task covering the finding
- **THEN** that task appears in the results without any additional option being supplied

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

### Requirement: Choosing a priority at capture
The capture skill SHALL state how to choose a task's priority at the moment the finding is recorded, and SHALL describe what distinguishes each of the three permitted values in terms of the consequence of leaving the finding unfixed rather than in terms of when work should be scheduled. It SHALL direct the agent to record `medium` when it has no basis for a stronger judgement, so that the priority reflects what the agent actually knows rather than an invented estimate.

#### Scenario: Finding with a clear consequence
- **WHEN** an agent records a finding whose consequence it can judge from what it has just read
- **THEN** the skill directs it to supply the matching priority with the task

#### Scenario: Consequence not clear from what was observed
- **WHEN** an agent records a finding whose consequence it cannot judge without investigation it was not asked to do
- **THEN** the skill directs it to leave the priority at the default rather than guess

#### Scenario: Priority is not a scheduling decision
- **WHEN** the capture skill explains the permitted values
- **THEN** it describes them as severity of the finding and does not present them as a statement about when the work will be done

### Requirement: Revising priority during triage
The triage skill SHALL read each task's priority as part of the review and SHALL treat it as a provisional judgement open to revision, since a finding's severity is often clearer once the surrounding code has been re-examined. It SHALL direct the reviewer to record a corrected priority on a task that is kept, and SHALL NOT make a task's disposition follow from its priority alone.

#### Scenario: Severity turns out to be higher
- **WHEN** triage finds that a task recorded as low severity still holds and matters more than the capturing agent could tell
- **THEN** the skill directs the reviewer to raise the task's priority

#### Scenario: Priority informs but does not decide
- **WHEN** triage reviews a task carrying a high priority
- **THEN** the skill still requires the task to be checked against the current code before a disposition is chosen

#### Scenario: Reviewing what matters first
- **WHEN** triage begins on a backlog with many accumulated tasks
- **THEN** the skill directs the reviewer to use priority to order the review

### Requirement: Triage dispositions
The triage skill SHALL give every reviewed task exactly one of four dispositions, and SHALL describe them as distinct situations rather than as degrees of the same one: promoting it into the project's planning system, fixing it immediately and setting it to `done`, keeping it in `todo` for a later review, or declining it. The skill SHALL direct the reviewer to record a decline with status `declined` and a reason, and SHALL reserve deletion for a task that should never have been recorded — a duplicate, a mis-capture, an accidental entry — rather than for a finding the reviewer has decided not to act on. A task that has already been fixed SHALL be set to `done` rather than declined or deleted, since it was acted on.

#### Scenario: Finding is real but not worth doing
- **WHEN** triage judges that a finding still holds but the cost of fixing it outweighs the benefit
- **THEN** the skill directs the reviewer to decline the task with a reason rather than to delete it

#### Scenario: Finding no longer holds
- **WHEN** triage re-reads the code and the finding is no longer true
- **THEN** the skill directs the reviewer to decline the task, recording in the reason what changed

#### Scenario: Finding was already fixed
- **WHEN** triage finds that the problem a task describes has since been fixed
- **THEN** the skill directs the reviewer to set the task to `done` rather than to decline or delete it

#### Scenario: Entry should never have been recorded
- **WHEN** triage finds a duplicate of another task, or an entry recorded by mistake
- **THEN** the skill directs the reviewer to delete it, since there is no decision worth preserving

#### Scenario: Every decline is explained
- **WHEN** triage declines any task
- **THEN** the skill requires a reason to be recorded on the task itself rather than only reported in the reviewer's summary

### Requirement: Declined findings are not recorded again
The capture skill SHALL treat a search result in status `declined` as an answer rather than as an absence: the finding has already been recorded and deliberately rejected. The skill SHALL direct the agent not to create a new task for it, and SHALL direct the agent to report the existing decline and its reason to the user rather than to discard the observation silently, so that a decision made under different circumstances can be reconsidered by someone able to judge that.

#### Scenario: Search finds a declined task describing the same problem
- **WHEN** an agent's search before recording returns a task in status `declined` covering the finding
- **THEN** the skill directs the agent not to create a second task

#### Scenario: The decline is surfaced, not swallowed
- **WHEN** an agent declines to record a finding because a declined task already covers it
- **THEN** the skill directs the agent to tell the user which task covers it and why it was declined

#### Scenario: Reopening is a human decision
- **WHEN** an agent judges that circumstances have changed since a finding was declined
- **THEN** the skill directs it to raise that with the user rather than to reopen the task or record a duplicate on its own
