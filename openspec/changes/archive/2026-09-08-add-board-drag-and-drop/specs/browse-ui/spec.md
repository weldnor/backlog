## MODIFIED Requirements

### Requirement: Switching between a list and a board view
The web UI SHALL offer two views of the currently filtered tasks, switchable without a page reload: a list (one row per task) and a board (one column per status, in the fixed order `todo`, `doing`, `done`, `declined`). Both views SHALL reflect the same filters. On the board view the UI SHALL allow a task's status to be changed by dragging its card into another status column; the resulting change SHALL be applied exactly as an edit of the task's status is applied — same validation, same persisted file — and SHALL leave every other field of the task unchanged. Dragging a card onto the column of its current status, or a drag that does not complete on a column, SHALL leave the task unchanged. Dragging a card onto the `declined` column SHALL first prompt for a decline reason and SHALL leave the task unchanged if no non-empty reason is given. No field other than status SHALL be changeable by dragging, and changing a task's status by editing it SHALL remain available in both views.

#### Scenario: Switching to the board view
- **WHEN** the board view is selected
- **THEN** the currently filtered tasks are shown grouped into one column per status, in the fixed order, each column showing its task count

#### Scenario: A filter applies to both views
- **WHEN** a tag filter is applied and the view is then switched from list to board
- **THEN** the board shows the same filtered set the list showed

#### Scenario: No drag-and-drop
- **WHEN** the board view is shown
- **THEN** the only field a drag can change is a task's status; no drag changes a task's priority, tags, title or description, and the list view has no drag interaction at all

#### Scenario: Moving a task by dragging its card
- **WHEN** a task's card is dragged from its column and dropped onto the `doing` column on the board
- **THEN** the task's status is set to `doing`, the change is saved to the same task file `backlog` operates on with every other field unchanged, and the card appears in the `doing` column

#### Scenario: Dragging a card onto the declined column
- **WHEN** a task's card is dropped onto the `declined` column and a non-empty reason is supplied when prompted
- **THEN** the task's status is set to `declined` with that reason recorded, and the card appears in the `declined` column

#### Scenario: Declining by drag without a reason
- **WHEN** a task's card is dropped onto the `declined` column and the reason prompt is cancelled or left empty
- **THEN** no request changes the task, and the card stays in its original column

#### Scenario: Dropping a card onto its own column
- **WHEN** a task's card is dropped onto the column matching the task's current status
- **THEN** no request is made and the board is unchanged

#### Scenario: A dragged status change is consistent with the CLI
- **WHEN** a task's status is changed by dragging its card on the board and the task is then inspected with `backlog show --json`
- **THEN** the reported status matches the column the card was dropped on
