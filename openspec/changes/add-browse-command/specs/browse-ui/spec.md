## ADDED Requirements

### Requirement: Starting the browse server
The CLI SHALL provide a `browse` command that starts a local HTTP server serving a web UI backed by the current backlog, and SHALL fail the way every other command does when no backlog is found. The server SHALL bind to `127.0.0.1` unless a different host is given, and SHALL listen on an OS-assigned free port unless a specific port is given. The command SHALL print the URL the UI is reachable at, SHALL open that URL in the system's default browser unless opening is suppressed, and SHALL run until interrupted, at which point it SHALL shut down and exit successfully.

#### Scenario: Starting with defaults
- **WHEN** `backlog browse` is run in a project with a backlog
- **THEN** a server bound to `127.0.0.1` on an OS-assigned port is started, its URL is printed, and the default browser is opened to that URL

#### Scenario: No backlog found
- **WHEN** `backlog browse` is run in a directory with no backlog in it or any ancestor
- **THEN** the command exits non-zero and reports that no backlog was found, and no server is started

#### Scenario: Choosing a port
- **WHEN** `backlog browse` is run with a specific port requested
- **THEN** the server listens on that port, or the command exits non-zero if the port is already in use

#### Scenario: Suppressing the browser
- **WHEN** `backlog browse` is run with browser-opening suppressed
- **THEN** the server starts and its URL is printed, and no browser is launched

#### Scenario: Stopping the server
- **WHEN** the running `backlog browse` process receives an interrupt
- **THEN** the server shuts down and the command exits zero

#### Scenario: Binding beyond loopback
- **WHEN** `backlog browse` is run with a non-loopback host requested
- **THEN** the server binds to that host and a warning is printed that the UI's write API is now reachable without authentication from beyond the local machine

### Requirement: Browsing the task list
The web UI SHALL display the tasks in the backlog ordered by descending priority then ascending identifier, matching `backlog list`, and SHALL support narrowing the list by status, by tag and by priority, matching the selection rules `backlog list --all` already applies. The UI SHALL also support filtering the currently loaded list by a free-text match against title, description and tags, evaluated locally without a further request to the server.

#### Scenario: Opening the UI
- **WHEN** the web UI is opened against a backlog containing tasks in every status
- **THEN** every task in status `todo` or `doing` is shown by default, ordered by descending priority then ascending identifier

#### Scenario: Filtering by status, tag or priority
- **WHEN** a status, tag or priority filter is applied in the UI
- **THEN** only tasks matching every applied filter are shown

#### Scenario: Free-text filtering
- **WHEN** text is entered into the UI's filter box
- **THEN** only tasks whose title, description or tags contain that text, case-insensitively, remain visible, without a new page load

### Requirement: Switching between a list and a board view
The web UI SHALL offer two views of the currently filtered tasks, switchable without a page reload: a list (one row per task) and a board (one column per status, in the fixed order `todo`, `doing`, `done`, `declined`). Both views SHALL reflect the same filters. The board view SHALL NOT support changing a task's status, or any other field, by dragging it; changing a task's status SHALL be done the same way in both views — by editing the task.

#### Scenario: Switching to the board view
- **WHEN** the board view is selected
- **THEN** the currently filtered tasks are shown grouped into one column per status, in the fixed order, each column showing its task count

#### Scenario: A filter applies to both views
- **WHEN** a tag filter is applied and the view is then switched from list to board
- **THEN** the board shows the same filtered set the list showed

#### Scenario: No drag-and-drop
- **WHEN** the board view is shown
- **THEN** no action in the UI allows a task to be moved between columns by dragging it

### Requirement: Viewing a task's detail
Selecting a task from either view SHALL open a dialog showing that task's title, status, priority, rendered description, and — when the task is declined — its decline reason, together with its tool-owned metadata (identifier, created timestamp, author, git provenance, source files, references) displayed read-only. The dialog SHALL provide a way to switch from this read view to editing the task, per the following requirement.

#### Scenario: Opening a task's detail
- **WHEN** a task is selected
- **THEN** its title, status, priority and description are shown, along with its metadata

#### Scenario: Metadata is read-only
- **WHEN** a task's detail is open and not in edit mode
- **THEN** no action in the metadata portion of the dialog changes any of it

#### Scenario: Decline reason is shown
- **WHEN** the detail of a declined task is opened
- **THEN** its recorded reason is displayed

### Requirement: Creating a task from the UI
The web UI SHALL provide a form for creating a task, accepting the same fields `backlog add` accepts: a required title, and an optional description, tags, priority, source files and references. A task created through the UI SHALL be recorded with author `human`. Submitting the form without a title SHALL be rejected without creating a task.

#### Scenario: Creating a task with only a title
- **WHEN** the create form is submitted with only a title
- **THEN** a task is created with status `todo`, priority `medium`, author `human`, and appears in the list

#### Scenario: Creating a task with full context
- **WHEN** the create form is submitted with a title, description, tags, a priority and file references
- **THEN** a task is created carrying all of the supplied values

#### Scenario: Rejecting an empty title
- **WHEN** the create form is submitted with an empty or whitespace-only title
- **THEN** the UI reports the problem, no request creates a task, and the form remains open

### Requirement: Editing a task from the UI
From a task's detail dialog, the web UI SHALL provide a way to switch to editing that task's title, description, tags, priority, status and, when the resulting status is `declined`, its reason, saving the change to the same task file `backlog` operates on. The reason field SHALL be shown only while the selected status is `declined`. Validation SHALL match `backlog add` and `backlog set`: the title SHALL NOT be empty, the status and priority SHALL each be one of their permitted values, and a reason SHALL be present exactly when the resulting status is `declined`. Setting a declined task to any other status SHALL clear its reason, matching `backlog set`.

#### Scenario: Editing a task's description
- **WHEN** a task's description is changed and saved from its edit view
- **THEN** the task file on disk reflects the new description and every other field is unchanged

#### Scenario: Changing status to declined
- **WHEN** a task is set to `declined` from its edit view with a reason supplied
- **THEN** the task's status and reason are updated and the task moves to the `declined` group in the board view

#### Scenario: Declining without a reason
- **WHEN** a task is set to `declined` from its edit view with no reason supplied
- **THEN** the change is rejected, the task is unchanged, and the problem is reported in the dialog

#### Scenario: Reopening a declined task
- **WHEN** a task in status `declined` is set to `todo` from its edit view
- **THEN** the status is updated, the reason is cleared, and the task moves out of the `declined` group

#### Scenario: Rejecting an empty title on edit
- **WHEN** a task's title is cleared to empty and the edit is saved
- **THEN** the change is rejected and the task's title is unchanged

#### Scenario: Returning to the read view
- **WHEN** an edit is saved successfully
- **THEN** the dialog shows the task's updated read view, including its refreshed metadata

#### Scenario: Consistency with the CLI
- **WHEN** a task is edited from the UI and then inspected with `backlog show --json`
- **THEN** the reported fields match exactly what was saved from the UI

### Requirement: No deletion from the UI
The web UI SHALL NOT provide a way to permanently delete a task. Removing a task SHALL remain available only through `backlog rm`.

#### Scenario: No delete action present
- **WHEN** a task's detail dialog is open, whether reading or editing
- **THEN** no action in the UI permanently removes the task

### Requirement: Offline, self-contained UI
The web UI's assets, including its typeface, SHALL be embedded in the `backlog` binary and SHALL be served without requesting any font, script, stylesheet or other resource from a network location. The UI SHALL be fully usable with no network access available.

#### Scenario: Using the UI with no network access
- **WHEN** the web UI is opened on a machine with no network access
- **THEN** the page loads fully styled, in its typeface, and functional, with no failed requests to an external host
