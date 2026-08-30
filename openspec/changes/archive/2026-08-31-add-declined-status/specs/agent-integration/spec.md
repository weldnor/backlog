## ADDED Requirements

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

## MODIFIED Requirements

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
