## MODIFIED Requirements

### Requirement: Browsing the task list
The web UI SHALL display the tasks in the backlog ordered by descending priority then ascending identifier, matching `backlog list`, and SHALL show tasks in every status by default. The UI SHALL provide a single filter row that narrows the shown tasks by priority (`All`, `High`, `Medium`, `Low`), by tag (one chip per tag present in the backlog), by assignee (one chip per assignee present, shown only when at least one task is assigned) and, in the list view, by status. Choosing an active chip again SHALL clear that filter, and a `Reset` control SHALL appear whenever any filter or search text is active and SHALL clear all of them. The filter row SHALL show how many tasks are visible out of the total, as `N of M tasks`. The UI SHALL also support filtering the currently loaded tasks by a free-text match against title, description and tags, evaluated locally without a further request to the server. When no task matches, the UI SHALL say so and offer a single control that clears every filter and the search text. The UI SHALL re-fetch tasks when its window regains focus, so tasks created or changed outside the UI appear without a page reload.

#### Scenario: Opening the UI
- **WHEN** the web UI is opened against a backlog containing tasks in every status
- **THEN** every task is shown, ordered by descending priority then ascending identifier, and the filter row reads `N of N tasks`

#### Scenario: Filtering by status, tag or priority
- **WHEN** a priority, tag, assignee or (in the list view) status filter is applied in the filter row
- **THEN** only tasks matching every applied filter are shown, and the count reads `<visible> of <total> tasks`

#### Scenario: Toggling a filter off
- **WHEN** an active tag chip is chosen again
- **THEN** the tag filter is cleared and the tasks it had hidden are shown again

#### Scenario: Resetting filters
- **WHEN** filters or search text are active and `Reset` is chosen
- **THEN** every filter and the search text are cleared and all tasks are shown

#### Scenario: Free-text filtering
- **WHEN** text is entered into the top bar's search field
- **THEN** only tasks whose title, description or tags contain that text, case-insensitively, remain visible, without a new request to the server

#### Scenario: Empty result
- **WHEN** the active search and filters match no task
- **THEN** a notice states that no task matches the current search and filters, and choosing its `Clear everything` control shows all tasks again

#### Scenario: Picking up outside changes
- **WHEN** a task is added with `backlog add` while the UI is open and the UI's window then regains focus
- **THEN** the new task appears without reloading the page

### Requirement: Switching between a list and a board view
The web UI SHALL offer two views of the currently filtered tasks, switchable from the top bar without a page reload: a list (one row per task) and a board (one column per status, in the fixed order `new`, `todo`, `doing`, `done`, `declined`). Both views SHALL reflect the same filters. Each board column SHALL show its status name and task count, and each card SHALL show the task's tags, title, the start of its description, its priority and its assignee (or `unassigned`). A column with no visible tasks SHALL show that status's empty note. Each column SHALL offer a menu with an action that collapses the column to a narrow rail showing only its name and count, and restores it; a collapsed column SHALL still accept dropped cards, and the collapsed state SHALL be remembered in the same browser. The column menu SHALL also offer copying the column's visible tasks to the clipboard as a markdown list. On the board view the UI SHALL allow a task's status to be changed by dragging its card into another status column; the resulting change SHALL be applied exactly as an edit of the task's status is applied — same validation, same persisted file — and SHALL leave every other field of the task unchanged. Dragging a card onto the column of its current status, or a drag that does not complete on a column, SHALL leave the task unchanged. Dragging a card onto the `declined` column SHALL first ask for a decline reason in an in-app dialog and SHALL leave the task unchanged if the dialog is cancelled or no non-empty reason is given. No field other than status SHALL be changeable by dragging, and changing a task's status from its detail SHALL remain available in both views.

#### Scenario: Switching to the board view
- **WHEN** the board view is selected
- **THEN** the currently filtered tasks are shown grouped into one column per status, in the fixed order, each column showing its task count

#### Scenario: A filter applies to both views
- **WHEN** a tag filter is applied and the view is then switched from list to board
- **THEN** the board shows the same filtered set the list showed

#### Scenario: Card content
- **WHEN** a task with tags `ui` and `board`, priority `high` and assignee `ann` is shown on the board
- **THEN** its card shows a label for each tag, its title, the start of its description, `High` and `@ann`

#### Scenario: Collapsing a column
- **WHEN** `Collapse list` is chosen from the `done` column's menu
- **THEN** the column is shown as a narrow rail with its name and count, and it stays collapsed after the page is reloaded in the same browser

#### Scenario: No drag-and-drop
- **WHEN** the board view is shown
- **THEN** the only field a drag can change is a task's status; no drag changes a task's priority, tags, title or description, and the list view has no drag interaction at all

#### Scenario: Moving a task by dragging its card
- **WHEN** a task's card is dragged from its column and dropped onto the `doing` column on the board
- **THEN** the task's status is set to `doing`, the change is saved to the same task file `backlog` operates on with every other field unchanged, and the card appears in the `doing` column

#### Scenario: Dragging a card onto the declined column
- **WHEN** a task's card is dropped onto the `declined` column and a non-empty reason is entered in the reason dialog and confirmed
- **THEN** the task's status is set to `declined` with that reason recorded, and the card appears in the `declined` column

#### Scenario: Declining by drag without a reason
- **WHEN** a task's card is dropped onto the `declined` column and the reason dialog is cancelled or confirmed empty
- **THEN** no request changes the task, and the card stays in its original column

#### Scenario: Dropping a card onto its own column
- **WHEN** a task's card is dropped onto the column matching the task's current status
- **THEN** no request is made and the board is unchanged

#### Scenario: A dragged status change is consistent with the CLI
- **WHEN** a task's status is changed by dragging its card on the board and the task is then inspected with `backlog show --json`
- **THEN** the reported status matches the column the card was dropped on

### Requirement: Viewing a task's detail
Selecting a task from either view SHALL open a modal window showing that task's identifier and title, its status, priority, assignee, tags and rendered description, its links to other tasks (each with link type, target identifier, target title and target status), and — when the task is declined — its decline reason, together with its tool-owned metadata (created timestamp, author, git provenance, source files, references) displayed read-only. The window SHALL offer a `Copy command` action that copies `backlog show <id>` to the clipboard and a `Copy id` affordance on the identifier, each confirmed by a short message that disappears on its own. Choosing a link's target SHALL open that task in the same window.

#### Scenario: Opening a task's detail
- **WHEN** a task is selected
- **THEN** its identifier, title, status, priority, assignee, tags and description are shown, along with its metadata and links

#### Scenario: Metadata is read-only
- **WHEN** a task's detail is open
- **THEN** no action on the created timestamp, author, git provenance, source files or references changes any of them

#### Scenario: Decline reason is shown
- **WHEN** the detail of a declined task is opened
- **THEN** its recorded reason is displayed

#### Scenario: Copying the command
- **WHEN** `Copy command` is chosen on task 17
- **THEN** `backlog show 17` is placed on the clipboard and a `Command copied` confirmation is shown briefly

#### Scenario: Following a link
- **WHEN** a task's linked task is chosen in its detail
- **THEN** the window shows the linked task's detail

### Requirement: Creating a task from the UI
The web UI SHALL provide two ways to create a task, both recorded with author `human`: an inline draft and a full form.

The inline draft SHALL be opened from a `+ Add new task` slot at the end of every board column except `declined`, from a floating capture button, or by pressing `n` while no text field has focus; the button and key SHALL open the draft in the first column on the board and the full form in the list view. The draft SHALL accept one line in which `!high`, `!med` / `!medium` and `!low` set the priority, `#name` adds a tag, `@name` sets the assignee and `>status` sets the status (one of `new`, `todo`, `doing`, `done`); every other word SHALL form the title, and the parsed values SHALL be shown under the line while typing. After `#`, the draft SHALL offer up to five matching existing tags, choosable by arrow keys and `Tab` or by pointer, plus an entry to use the typed text as a new tag. `Enter` SHALL create the task in the draft's column (or the `>status` column), `Shift+Enter` SHALL open the full form prefilled with everything parsed so far, and `Esc` SHALL close the draft without creating anything. When the draft's title matches the title of an existing task, the draft SHALL show that task with a way to open it and a way to create anyway, and SHALL NOT create until one is chosen.

The full form SHALL accept the same fields `backlog add` accepts — a required title and an optional description, tags, priority, assignee, source files, references and links — plus an initial status, and SHALL offer `Create task` and `Create and add another`, the latter keeping the form open and empty after creating. `Ctrl+Enter` or `⌘+Enter` SHALL submit it.

A task created with a status other than `new` SHALL end up in that status, with the status change applied exactly as an edit of the task's status is applied. Submitting either path without a title SHALL be rejected without creating a task, with the problem shown next to the input and the input kept open. When creation fails on the server, the typed text SHALL be kept, and the UI SHALL offer to retry, to copy the typed text, and to discard it. After a successful creation, a confirmation SHALL name the new task's identifier and status for about six seconds and SHALL offer `Open`, which opens the task's detail, and `Undo`, which deletes the just-created task exactly as deleting it from its detail does.

#### Scenario: Creating a task with only a title
- **WHEN** `Add fuzzy search` is typed into the `new` column's draft and `Enter` is pressed
- **THEN** a task is created with that title, status `new`, priority `medium`, author `human`, and it appears in the `new` column while the draft stays open and empty

#### Scenario: Creating from a draft with tokens
- **WHEN** `Drop zone highlight !high #ui @ann >doing` is entered in a draft
- **THEN** a task titled `Drop zone highlight` is created with priority `high`, tag `ui`, assignee `ann` and status `doing`

#### Scenario: Creating in a column's status
- **WHEN** a task is created from the `todo` column's draft without a `>status` token
- **THEN** the task's status is `todo`

#### Scenario: Opening the full form from a draft
- **WHEN** `Fix parser !low #cli` is typed in a draft and `Shift+Enter` is pressed
- **THEN** the full form opens with title `Fix parser`, priority `low` and tag `cli` already filled in

#### Scenario: Creating a task with full context
- **WHEN** the full form is submitted with a title, description, tags, a priority, an assignee, source files, references and links
- **THEN** a task is created carrying all of the supplied values

#### Scenario: Rejecting an empty title
- **WHEN** a draft or the full form is submitted with an empty or whitespace-only title
- **THEN** the UI reports that a title is required, no request creates a task, and the draft or form remains open

#### Scenario: Warning about a likely duplicate
- **WHEN** a draft's title equals, ignoring case and surrounding whitespace, the title of existing task 11 and `Enter` is pressed
- **THEN** no task is created yet, task 11 is shown with an option to open it and an option to create anyway, and choosing `create anyway` creates the task

#### Scenario: Undoing a creation
- **WHEN** a task is created and `Undo` is chosen on the confirmation that follows
- **THEN** the created task is deleted and no longer appears in the list or board

#### Scenario: Keeping the text when creation fails
- **WHEN** creating from a draft fails on the server
- **THEN** the draft keeps the typed line, and retry, copy-text and discard actions are offered

### Requirement: Editing a task from the UI
From a task's detail window, the web UI SHALL let each of the task's title, description, tags, assignee, priority, status and links be edited in place, saving each change to the same task file `backlog` operates on without changing any field that was not edited. Title, tags and assignee SHALL look like plain text until activated, then become inputs where `Enter` saves and `Esc` cancels. Status and priority SHALL be chosen from inline selects that save on choice. The description SHALL be edited in a markdown editor with `Write` and `Preview` modes and `Save` / `Cancel`, where `Ctrl+Enter` or `⌘+Enter` saves and `Esc` cancels. Links SHALL be removable individually and addable through a picker that searches other tasks by identifier or title after a link type is chosen. Choosing `declined` as the status SHALL first ask for a reason in an in-app dialog and SHALL save nothing if it is cancelled or no non-empty reason is given. Validation SHALL match `backlog add` and `backlog set`: the title SHALL NOT be empty, the status and priority SHALL each be one of their permitted values, and a reason SHALL be present exactly when the resulting status is `declined`. Setting a declined task to any other status SHALL clear its reason, matching `backlog set`. A rejected change SHALL be reported in the detail window and SHALL leave the displayed value at what is saved.

#### Scenario: Editing a task's description
- **WHEN** a task's description is changed in the `Write` mode and saved
- **THEN** the task file on disk reflects the new description and every other field is unchanged

#### Scenario: Editing a single field in place
- **WHEN** a task's assignee is activated, changed to `weldnor` and `Enter` is pressed
- **THEN** the task's assignee is saved as `weldnor` and every other field is unchanged

#### Scenario: Cancelling an in-place edit
- **WHEN** a task's title is activated, changed, and `Esc` is pressed
- **THEN** no request changes the task and the original title is shown

#### Scenario: Changing status to declined
- **WHEN** `declined` is chosen from a task's status select and a reason is entered and confirmed
- **THEN** the task's status and reason are updated and the task moves to the `declined` column on the board

#### Scenario: Declining without a reason
- **WHEN** `declined` is chosen from a task's status select and the reason dialog is cancelled or confirmed empty
- **THEN** no request changes the task, and its status stays as it was

#### Scenario: Reopening a declined task
- **WHEN** a task in status `declined` is set to `todo` from its status select
- **THEN** the status is updated, the reason is cleared, and the task moves out of the `declined` column

#### Scenario: Rejecting an empty title on edit
- **WHEN** a task's title is cleared to empty and `Enter` is pressed
- **THEN** the change is rejected, the problem is reported, and the task's title is unchanged

#### Scenario: Returning to the read view
- **WHEN** an in-place edit is saved successfully
- **THEN** the edited field returns to its at-rest display showing the saved value, and the rest of the detail window, including its metadata, reflects the task as saved

#### Scenario: Adding and removing a link
- **WHEN** a `blocks` link to task 8 is added from the picker and a `related` link is removed
- **THEN** the task's saved links contain the `blocks` link to task 8 and no longer contain the removed link

#### Scenario: Consistency with the CLI
- **WHEN** a task is edited from the UI and then inspected with `backlog show --json`
- **THEN** the reported fields match exactly what was saved from the UI

### Requirement: Deleting a task from the UI
The web UI SHALL provide a way to permanently delete a task from its detail window. Triggering it SHALL first open an in-app confirmation dialog, layered over the detail window, that names the task by identifier and title; the task SHALL be deleted only if the confirmation is accepted, and cancelling SHALL leave the task and every other task unchanged and the detail window open. Deleting a task SHALL remove the same task file `backlog rm` removes, leaving no other file changed. After a confirmed delete the detail window SHALL close and the list and board SHALL refresh so the deleted task no longer appears. Deletion SHALL also be reachable through the JSON API as `DELETE /api/tasks/{id}`, which SHALL remove the identified task and report the removed task, and SHALL fail with a not-found response when no task has that identifier.

#### Scenario: Deleting a task after confirming
- **WHEN** `Delete` is chosen in a task's detail window and the confirmation dialog's `Delete` is chosen
- **THEN** the task's file is removed, the detail window closes, and the task no longer appears in the list or the board

#### Scenario: Cancelling the confirmation
- **WHEN** `Delete` is chosen and the confirmation dialog is cancelled or closed with `Escape`
- **THEN** no request deletes the task, the task's file is unchanged, and the detail window stays open

#### Scenario: Delete is offered in both views
- **WHEN** a task's detail window is opened from the list view, and again from the board view
- **THEN** a `Delete` action is present in each

#### Scenario: Confirmation names the task
- **WHEN** `Delete` is chosen on task 17 titled `Card covers on the board view`
- **THEN** the confirmation dialog shows `017` and `Card covers on the board view`

#### Scenario: Deleting through the API
- **WHEN** `DELETE /api/tasks/{id}` is requested for an existing task
- **THEN** the task's file is removed and the response reports the removed task

#### Scenario: Deleting a task that does not exist
- **WHEN** `DELETE /api/tasks/{id}` is requested with an identifier no task has
- **THEN** the response is a not-found error and no file is removed

#### Scenario: Consistency with the CLI
- **WHEN** a task is deleted from the UI and the backlog is then listed with `backlog list --all`
- **THEN** the deleted task is absent, exactly as if it had been removed with `backlog rm`

### Requirement: Keyboard and focus handling for the task dialog
While the task detail window, the full create form, or a confirmation dialog is open, the web UI SHALL keep keyboard focus within the topmost of them, SHALL close only the topmost one when `Escape` is pressed, and SHALL return focus to the element that opened it once it closes. When an in-place field edit or the description editor is active inside the detail window, `Escape` SHALL cancel that edit first, without saving it and without closing the window. Closing by `Escape` SHALL behave the same as closing by the window's close control and SHALL NOT save an in-progress edit.

#### Scenario: Closing the dialog with Escape
- **WHEN** a task's detail window is open with no field being edited and the `Escape` key is pressed
- **THEN** the window closes, and focus returns to the task that opened it

#### Scenario: Escape cancels an in-place edit first
- **WHEN** a task's title is being edited in place and `Escape` is pressed
- **THEN** the edit is cancelled without saving and the detail window stays open

#### Scenario: Escape closes only the confirmation
- **WHEN** the delete confirmation is open over a task's detail window and `Escape` is pressed
- **THEN** the confirmation closes, the detail window stays open, and no task is deleted

#### Scenario: Focus stays in the dialog
- **WHEN** a detail window is open and focus is moved forward from its last focusable control
- **THEN** focus moves to a control inside the window rather than to the page behind it

### Requirement: Offline, self-contained UI
The web UI SHALL be a compiled front-end bundle whose build output — HTML,
scripts, styles and its typeface (Poppins, in the light, regular and medium
weights) — is committed to the repository and embedded in the `backlog` binary.
Building the Go binary SHALL NOT require any JavaScript toolchain: the embedded
assets are consumed as-is. The served UI SHALL NOT request any font, script,
stylesheet or other resource from a network location, and SHALL be fully
usable with no network access available.

#### Scenario: Using the UI with no network access
- **WHEN** the web UI is opened on a machine with no network access
- **THEN** the page loads fully styled, in its typeface, and functional, with no failed requests to an external host

#### Scenario: Building the binary without a JavaScript toolchain
- **WHEN** the Go binary is built or installed on a machine with no Node or npm available
- **THEN** the build succeeds and the resulting binary serves the same UI as a build made where the front-end was rebuilt from source

## ADDED Requirements

### Requirement: Light and dark theme
The web UI SHALL offer a light and a dark theme. On first use it SHALL follow the operating system's color-scheme preference; a toggle in the top bar SHALL switch between the two themes, and the chosen theme SHALL be remembered in the same browser across reloads. Both themes SHALL present the same content and controls.

#### Scenario: Following the system preference
- **WHEN** the UI is opened for the first time in a browser whose system preference is dark
- **THEN** the dark theme is shown

#### Scenario: Switching and remembering the theme
- **WHEN** the theme toggle is used to switch to the light theme and the page is reloaded
- **THEN** the light theme is still shown
