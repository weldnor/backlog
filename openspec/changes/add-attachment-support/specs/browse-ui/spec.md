## MODIFIED Requirements

### Requirement: Viewing a task's detail
Selecting a task from either view SHALL open a dialog showing that task's title, status, priority, rendered description, and — when the task is declined — its decline reason, together with its tool-owned metadata (identifier, created timestamp, author, git provenance, attachments) displayed read-only in the metadata portion of the dialog. Attachments SHALL be listed together regardless of kind, each showing at least its type and value, in the order recorded. The dialog SHALL provide a way to switch from this read view to editing the task, per a following requirement, and, independently of read/edit mode, a way to add and remove attachments, per the "Managing attachments from the UI" requirement.

#### Scenario: Opening a task's detail
- **WHEN** a task is selected
- **THEN** its title, status, priority and description are shown, along with its metadata

#### Scenario: Attachments are listed
- **WHEN** a task with one or more attachments has its detail opened
- **THEN** every attachment is shown, in the order recorded, each identifiable as a file or a link

#### Scenario: Decline reason is shown
- **WHEN** the detail of a declined task is opened
- **THEN** its recorded reason is displayed

### Requirement: Creating a task from the UI
The web UI SHALL provide a form for creating a task, accepting the same fields `backlog add` accepts: a required title, and an optional description, tags, priority and attachments (each a file reference or a link — no file upload at creation time, matching `backlog add`). A task created through the UI SHALL be recorded with author `human`. Submitting the form without a title SHALL be rejected without creating a task.

#### Scenario: Creating a task with only a title
- **WHEN** the create form is submitted with only a title
- **THEN** a task is created with status `todo`, priority `medium`, author `human`, and appears in the list

#### Scenario: Creating a task with full context
- **WHEN** the create form is submitted with a title, description, tags, a priority and one or more attachments
- **THEN** a task is created carrying all of the supplied values

#### Scenario: Rejecting an empty title
- **WHEN** the create form is submitted with an empty or whitespace-only title
- **THEN** the UI reports the problem, no request creates a task, and the form remains open

## ADDED Requirements

### Requirement: Managing attachments from the UI

From a task's detail dialog, in either the read or the edit view, the web UI SHALL provide a way to add a link attachment, upload a file as a stored attachment, and remove any existing attachment. Each of these SHALL take effect immediately — it SHALL NOT be part of the save/cancel cycle the rest of the edit view uses — and SHALL be reachable through the JSON API as `POST /api/tasks/{id}/attachments` (a JSON body `{type, value}` for a link or a plain file reference; a multipart file upload for a stored attachment) and `DELETE /api/tasks/{id}/attachments` (identifying the attachment by its type and value), each of which SHALL return the task's current state and SHALL leave every other field of the task unchanged. Removing a stored attachment through the UI SHALL delete the underlying file, matching `backlog attach rm`.

#### Scenario: Adding a link from the UI
- **WHEN** a link value is submitted through the attachment controls in a task's detail dialog
- **THEN** the task immediately records a `link` attachment with that value and the dialog reflects it without a page reload

#### Scenario: Uploading a file from the UI
- **WHEN** a file is selected and uploaded through the attachment controls in a task's detail dialog
- **THEN** the file is stored under the task's attachment directory, the task records a `file` attachment pointing at it, and the dialog reflects it without a page reload

#### Scenario: Removing an attachment from the UI
- **WHEN** an attachment's remove control is used in a task's detail dialog
- **THEN** the attachment is removed from the task, any file it stored is deleted, and the dialog reflects the change without a page reload

#### Scenario: Attachment changes do not require Save
- **WHEN** an attachment is added or removed while the dialog is in edit mode with unsaved draft changes to other fields
- **THEN** the attachment change is persisted immediately and the unsaved draft changes to other fields remain unsaved until Save is used

#### Scenario: Consistency with the CLI
- **WHEN** an attachment is added or removed from the UI and the task is then inspected with `backlog show --json`
- **THEN** the reported attachments match exactly what was added or removed from the UI
