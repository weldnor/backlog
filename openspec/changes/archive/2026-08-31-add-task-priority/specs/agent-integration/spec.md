## ADDED Requirements

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
