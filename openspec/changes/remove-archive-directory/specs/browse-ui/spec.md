## MODIFIED Requirements

### Requirement: Browsing the task list
The web UI SHALL display the tasks in the backlog ordered by descending priority then ascending identifier, matching `backlog list`, and SHALL support narrowing the list by status, by tag and by priority. The UI SHALL also support filtering the currently loaded list by a free-text match against title, description and tags, evaluated locally without a further request to the server.

#### Scenario: Opening the UI
- **WHEN** the web UI is opened against a backlog containing tasks in every status
- **THEN** every task in status `todo` or `doing` is shown by default, ordered by descending priority then ascending identifier

#### Scenario: Filtering by status, tag or priority
- **WHEN** a status, tag or priority filter is applied in the UI
- **THEN** only tasks matching every applied filter are shown

#### Scenario: Free-text filtering
- **WHEN** text is entered into the UI's filter box
- **THEN** only tasks whose title, description or tags contain that text, case-insensitively, remain visible, without a new page load
