// Package hooks lets a backlog run an external script when a task is added or
// changed — the mechanism a planning system, a chat notification or a custom
// sync can hang off, without the backlog binary knowing anything exists on
// the other end. It is deliberately best-effort: a hook can observe a change,
// never block or undo one, so a missing interpreter or a failing script never
// turns a successful `backlog` command into a failed one.
package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/weldnor/backlog/internal/task"
	"github.com/weldnor/backlog/internal/taskview"
)

// DirName is the directory hook scripts live in, relative to the backlog
// root (the same .backlog a Store opens).
const DirName = "hooks"

// Event names a point in a task's lifecycle a hook can be written for. They
// name commands, not internal states, so a script author can predict which
// one fires from the command they ran.
const (
	PostAdd    = "post-add"  // after `backlog add` creates a task
	PostSet    = "post-set"  // after `backlog set` changes status, priority, reason or refs
	PostEdit   = "post-edit" // after `backlog edit` or `backlog tag` changes title, description or tags
	PostRemove = "post-rm"   // after `backlog rm` deletes a task
)

// Dir returns the hooks directory under a backlog root (Store.Root).
func Dir(root string) string { return filepath.Join(root, DirName) }

// Run looks up the hook for event under root and, if one is configured, runs
// it against t. Diagnostics — a hook that failed, or one that exists but
// could not be run on this platform — are written to diag rather than
// returned, since a hook is a side effect: it never changes whether the
// command that triggered it succeeded. A nil diag discards them. extra adds
// further BACKLOG_-style environment variables (e.g. what changed) on top of
// the ones every event gets.
func Run(diag io.Writer, root, project, event string, t *task.Task, extra map[string]string) {
	if diag == nil {
		diag = io.Discard
	}
	cmd, note, err := resolve(Dir(root), event)
	if err != nil {
		fmt.Fprintf(diag, "backlog: %s hook: %v\n", event, err)
		return
	}
	if cmd == nil {
		return // no hook configured for this event; the common case
	}
	if note != "" {
		fmt.Fprintf(diag, "backlog: %s hook: %s\n", event, note)
	}

	cmd.Dir = project
	cmd.Env = append(os.Environ(), envFor(root, project, event, t, extra)...)
	cmd.Stdout = diag
	cmd.Stderr = diag
	if t != nil {
		if data, err := json.Marshal(taskview.View(t)); err == nil {
			cmd.Stdin = bytes.NewReader(data)
		}
	}

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(diag, "backlog: %s hook failed: %v\n", event, err)
	}
}

// envFor builds the environment variables passed to every hook. The task's
// full JSON view also arrives on stdin, so these cover what a one-line shell
// script needs without parsing JSON; a script that wants more reads stdin.
func envFor(root, project, event string, t *task.Task, extra map[string]string) []string {
	env := []string{
		"BACKLOG_EVENT=" + event,
		"BACKLOG_ROOT=" + root,
		"BACKLOG_PROJECT=" + project,
	}
	if t != nil {
		env = append(env,
			"BACKLOG_TASK_ID="+strconv.Itoa(t.ID),
			"BACKLOG_TASK_TITLE="+t.Title,
			"BACKLOG_TASK_STATUS="+t.Status,
			"BACKLOG_TASK_PRIORITY="+t.Priority,
			"BACKLOG_TASK_TAGS="+strings.Join(t.Tags, ","),
			"BACKLOG_TASK_FILE="+t.Path,
		)
	}
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}

// candidate is one file shape a hook script may take, and how to turn it into
// a runnable command. Shapes are tried in order and the first whose file
// exists wins the event — one script per event, the same rule git hooks use.
type candidate struct {
	// suffix appended to the event name to form the file name to look for.
	suffix string
	// build turns a resolved path into a command, or reports that this shape
	// cannot be run here (e.g. no interpreter on PATH). unsupported
	// distinguishes "skip to the next shape" from a real failure.
	build func(path string) (cmd *exec.Cmd, unsupported string)
}

// candidates lists the supported hook shapes so that the same backlog checkout
// works whether the hook was written as a Unix shell script, a PowerShell
// script (PowerShell itself is cross-platform), or a Windows batch file.
func candidates() []candidate {
	return []candidate{
		// A bare, executable file: relies on the OS to run it, which only
		// Unix does directly (via its shebang line).
		{"", func(path string) (*exec.Cmd, string) {
			if runtime.GOOS == "windows" {
				return nil, "a hook with no extension cannot be run on Windows; add .ps1, .cmd or .bat"
			}
			info, err := os.Stat(path)
			if err == nil && info.Mode()&0o111 == 0 {
				return nil, "found but not executable; chmod +x it"
			}
			return exec.Command(path), ""
		}},
		// PowerShell: prefer pwsh (PowerShell 7+, itself cross-platform),
		// fall back to the powershell.exe every Windows install carries.
		{".ps1", func(path string) (*exec.Cmd, string) {
			if bin, err := exec.LookPath("pwsh"); err == nil {
				return exec.Command(bin, "-NoProfile", "-NonInteractive", "-File", path), ""
			}
			if bin, err := exec.LookPath("powershell"); err == nil {
				return exec.Command(bin, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path), ""
			}
			return nil, "found but neither pwsh nor powershell is on PATH"
		}},
		// A shell script run explicitly, for a Windows machine that has sh
		// on PATH (Git for Windows, WSL) but no exec bit to rely on.
		{".sh", func(path string) (*exec.Cmd, string) {
			bin, err := exec.LookPath("sh")
			if err != nil {
				return nil, "found but sh is not on PATH"
			}
			return exec.Command(bin, path), ""
		}},
		{".cmd", cmdHost}, {".bat", cmdHost},
		// A compiled or otherwise self-contained executable.
		{".exe", func(path string) (*exec.Cmd, string) { return exec.Command(path), "" }},
	}
}

func cmdHost(path string) (*exec.Cmd, string) {
	bin, err := exec.LookPath("cmd")
	if err != nil {
		return nil, "found but cmd is not on PATH"
	}
	return exec.Command(bin, "/C", path), ""
}

// resolve finds the hook file for event under dir and builds the command to
// run it. It returns a nil command with no error when no hook is configured,
// which is the common case and not worth reporting. When one or more hook
// files exist but none could be turned into a runnable command (e.g. a .ps1
// hook with no PowerShell installed), it returns an error describing why, so
// the caller can tell the user rather than running nothing silently.
func resolve(dir, event string) (*exec.Cmd, string, error) {
	var problems []string
	for _, c := range candidates() {
		path := filepath.Join(dir, event+c.suffix)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		cmd, unsupported := c.build(path)
		if unsupported != "" {
			problems = append(problems, filepath.Base(path)+": "+unsupported)
			continue
		}
		return cmd, "", nil
	}
	if len(problems) > 0 {
		return nil, "", fmt.Errorf("found a hook but could not run it (%s)", strings.Join(problems, "; "))
	}
	return nil, "", nil
}
