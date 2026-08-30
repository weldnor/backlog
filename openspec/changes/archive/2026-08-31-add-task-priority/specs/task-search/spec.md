## MODIFIED Requirements

### Requirement: Searchable content
The `search` command SHALL match a query against a task's title, description body and tags. It SHALL NOT match against status, priority, timestamps or any field inside the `metadata` block.

#### Scenario: Match in the title
- **WHEN** a query matches text in a task's title
- **THEN** that task is returned

#### Scenario: Match in the description
- **WHEN** a query matches text in a task's description body
- **THEN** that task is returned

#### Scenario: Match in a tag
- **WHEN** a query matches one of a task's tags
- **THEN** that task is returned

#### Scenario: Structural fields are not searched
- **WHEN** the query is a status value such as `todo`
- **THEN** tasks are returned only if that text appears in their title, description or tags, not because of their status field

#### Scenario: Priority is not searched
- **WHEN** the query is a priority value such as `high`
- **THEN** tasks are returned only if that text appears in their title, description or tags, not because of their priority field

#### Scenario: Metadata is not searched
- **WHEN** the query matches only a value recorded under `metadata`, such as a commit hash or a source file path
- **THEN** the task is not returned
