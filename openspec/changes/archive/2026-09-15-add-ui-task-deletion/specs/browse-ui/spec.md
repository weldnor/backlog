## REMOVED Requirements

### Requirement: No deletion from the UI
**Reason**: The UI now supports deleting a task, guarded by a confirmation prompt. See the added "Deleting a task from the UI" requirement.
**Migration**: None. `backlog rm` remains available and unchanged; the UI delete removes the same task file it does.

## ADDED Requirements

### Requirement: Deleting a task from the UI
The web UI SHALL provide a way to permanently delete a task, available from both the read view and the edit view of a task's detail dialog. Triggering it SHALL first ask for confirmation in a prompt that names the task; the task SHALL be deleted only if the confirmation is accepted, and cancelling SHALL leave the task and every other task unchanged. Deleting a task SHALL remove the same task file `backlog rm` removes, leaving no other file changed. After a confirmed delete the dialog SHALL close and the list and board SHALL refresh so the deleted task no longer appears. Deletion SHALL also be reachable through the JSON API as `DELETE /api/tasks/{id}`, which SHALL remove the identified task and report the removed task, and SHALL fail with a not-found response when no task has that identifier.

#### Scenario: Deleting a task after confirming
- **WHEN** the Delete action is triggered from a task's detail dialog and the confirmation prompt is accepted
- **THEN** the task's file is removed, the dialog closes, and the task no longer appears in the list or the board

#### Scenario: Cancelling the confirmation
- **WHEN** the Delete action is triggered and the confirmation prompt is dismissed or declined
- **THEN** no request deletes the task, the task's file is unchanged, and the dialog stays open

#### Scenario: Delete is offered in both views
- **WHEN** a task's detail dialog is open in the read view, and again in the edit view
- **THEN** a Delete action is present in each

#### Scenario: Deleting through the API
- **WHEN** `DELETE /api/tasks/{id}` is requested for an existing task
- **THEN** the task's file is removed and the response reports the removed task

#### Scenario: Deleting a task that does not exist
- **WHEN** `DELETE /api/tasks/{id}` is requested with an identifier no task has
- **THEN** the response is a not-found error and no file is removed

#### Scenario: Consistency with the CLI
- **WHEN** a task is deleted from the UI and the backlog is then listed with `backlog list --all`
- **THEN** the deleted task is absent, exactly as if it had been removed with `backlog rm`
