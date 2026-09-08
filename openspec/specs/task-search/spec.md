## Purpose

Defines how tasks are found by content rather than by identifier, so that a person can locate a half-remembered note and an agent can check whether a finding has already been recorded before creating a duplicate.

## Requirements

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

### Requirement: Deterministic matching
Search SHALL match by case-insensitive substring by default, and SHALL support regular-expression matching when explicitly requested. For a given query and store contents, the set of results and their order SHALL be identical on every run. Approximate or fuzzy matching SHALL NOT be used.

#### Scenario: Case-insensitive substring
- **WHEN** the query `cache` is searched and a task's title contains `Cache`
- **THEN** that task is returned

#### Scenario: Repeated searches agree
- **WHEN** the same query is run twice against an unchanged backlog
- **THEN** both runs return the same results in the same order

#### Scenario: Regular expression search
- **WHEN** a query is searched with regular-expression matching requested
- **THEN** the query is interpreted as a regular expression

#### Scenario: Invalid regular expression
- **WHEN** an invalid regular expression is supplied with regular-expression matching requested
- **THEN** the command exits non-zero and reports the syntax error

### Requirement: Search status scope
Search SHALL cover tasks in every status unconditionally. A finding that has already been recorded SHALL be found whether the earlier task is still open, has been completed, or was declined, so that neither a duplicate nor a previously rejected finding can hide behind a scope. Search SHALL support the same tag filter as listing and SHALL have no status selector of its own.

#### Scenario: A completed task is in scope
- **WHEN** a query matches only a task in status `done`
- **THEN** that task is returned

#### Scenario: A declined task is in scope
- **WHEN** a query matches a declined task
- **THEN** that task is returned

#### Scenario: A declined result is identifiable as such
- **WHEN** a search returns a declined task
- **THEN** the result reports its status as `declined`, so that the caller can tell it apart from a live task

#### Scenario: Filtering by tag
- **WHEN** a query is run with a tag filter
- **THEN** only matching tasks that also carry the tag are returned

### Requirement: Result ranking
Results SHALL be ordered with tasks matching in the title before tasks matching only in the description or tags, and by ascending identifier within each group.

#### Scenario: Title matches rank first
- **WHEN** one task matches the query in its title and another matches only in its description
- **THEN** the title match is listed first

#### Scenario: Stable order within a rank
- **WHEN** several tasks match in their titles
- **THEN** they are listed in ascending identifier order

### Requirement: Search output
Search SHALL report each matching task's identifier, status and title together with the matching text in context. Machine-readable output SHALL additionally identify, for each result, which field matched and the matching text. Finding no matches SHALL NOT be treated as an error.

#### Scenario: Human-readable results
- **WHEN** a search returns matches and JSON was not requested
- **THEN** each result shows the identifier, status, title and the surrounding matched text

#### Scenario: Machine-readable results
- **WHEN** a search returns matches and JSON was requested
- **THEN** each result carries the full task and a list of matches identifying the field and the matched text

#### Scenario: No matches
- **WHEN** a search matches no task
- **THEN** the command exits zero with an empty result set
