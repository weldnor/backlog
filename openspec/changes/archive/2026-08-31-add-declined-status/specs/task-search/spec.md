## MODIFIED Requirements

### Requirement: Searchable content
The `search` command SHALL match a query against a task's title, description body and tags. It SHALL NOT match against status, priority, the reason a task was declined, timestamps or any field inside the `metadata` block.

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

#### Scenario: The decline reason is not searched
- **WHEN** the query matches text that appears only in a declined task's reason
- **THEN** that task is not returned, because search asks whether a finding was recorded and the reason describes the disposition rather than the finding

#### Scenario: Metadata is not searched
- **WHEN** the query matches only a value recorded under `metadata`, such as a commit hash or a source file path
- **THEN** the task is not returned

### Requirement: Search scope and filters
Search SHALL cover tasks in status `todo` and `doing` by default, include tasks in status `done` when all tasks are requested, and support the same status and tag filters as listing.

Tasks in status `declined` SHALL be in scope by default, whether or not all tasks were requested. Search exists to answer whether a finding has already been recorded, and a decline is the most consequential form of having recorded one; a scope that hid it would let a duplicate hide behind the scope, in the same way a severity filter would let one hide behind a filter. Tasks in status `done` SHALL NOT be treated this way, because a fixed problem that reappears is a regression and is genuinely new information, whereas a declined problem that reappears is the same decision arriving a second time.

An explicit status filter SHALL be taken at face value, and SHALL therefore exclude declined tasks when it does not name `declined`.

#### Scenario: Archived tasks excluded by default
- **WHEN** a query matches only a task in status `done` and all tasks were not requested
- **THEN** no results are returned

#### Scenario: Searching the archive
- **WHEN** the same query is run requesting all tasks
- **THEN** the matching task in status `done` is returned

#### Scenario: Declined tasks are found without asking for all tasks
- **WHEN** a query matches a declined task and all tasks were not requested
- **THEN** that task is returned

#### Scenario: A declined result is identifiable as such
- **WHEN** a search returns a declined task
- **THEN** the result reports its status as `declined`, so that the caller can tell it apart from a live task

#### Scenario: An explicit status filter still excludes declined tasks
- **WHEN** a query is run with a status filter naming `todo` and a declined task matches the query
- **THEN** that task is not returned

#### Scenario: Selecting declined tasks explicitly
- **WHEN** a query is run with a status filter naming `declined`
- **THEN** only declined tasks are returned
