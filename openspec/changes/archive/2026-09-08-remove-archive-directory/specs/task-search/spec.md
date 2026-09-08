## REMOVED Requirements

### Requirement: Search scope and filters
**Reason**: The scope rules — `todo` and `doing` by default, `done` only when "all tasks" is requested, `declined` always in scope — and the `--all` flag existed to keep the archive directory from hiding a previously recorded finding. With one directory and no `--all` flag, search simply covers every status, as redefined below.
**Migration**: Drop `--all` from any `backlog search` invocation; search now covers every status unconditionally. The `--tag` filter is unchanged. There is no longer a status filter on `search`.

## ADDED Requirements

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
