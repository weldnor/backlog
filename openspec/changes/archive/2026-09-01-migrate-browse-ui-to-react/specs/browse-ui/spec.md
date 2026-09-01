## MODIFIED Requirements

### Requirement: Offline, self-contained UI
The web UI SHALL be a compiled front-end bundle whose build output — HTML,
scripts, styles and its typeface — is committed to the repository and embedded
in the `backlog` binary. Building the Go binary SHALL NOT require any
JavaScript toolchain: the embedded assets are consumed as-is. The served UI
SHALL NOT request any font, script, stylesheet or other resource from a network
location, and SHALL be fully usable with no network access available.

#### Scenario: Using the UI with no network access
- **WHEN** the web UI is opened on a machine with no network access
- **THEN** the page loads fully styled, in its typeface, and functional, with no failed requests to an external host

#### Scenario: Building the binary without a JavaScript toolchain
- **WHEN** the Go binary is built or installed on a machine with no Node or npm available
- **THEN** the build succeeds and the resulting binary serves the same UI as a build made where the front-end was rebuilt from source

## ADDED Requirements

### Requirement: Keyboard and focus handling for the task dialog
While the task detail, edit or create dialog is open, the web UI SHALL close it
when the `Escape` key is pressed, SHALL keep keyboard focus within the dialog
while it is open, and SHALL return focus to the element that opened the dialog
once it closes. Closing by `Escape` SHALL behave the same as closing by the
dialog's close control and SHALL NOT save an in-progress edit.

#### Scenario: Closing the dialog with Escape
- **WHEN** a task dialog is open and the `Escape` key is pressed
- **THEN** the dialog closes without saving any in-progress edit, and focus returns to the task that opened it

#### Scenario: Focus stays in the dialog
- **WHEN** a task dialog is open and focus is moved forward from its last focusable control
- **THEN** focus moves to a control inside the dialog rather than to the page behind it
