package hooks

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/weldnor/backlog/internal/task"
)

func newTask(t *testing.T) *task.Task {
	t.Helper()
	tsk := task.New("a finding", "", nil, nil, nil, task.AuthorAgent, task.DefaultPriority, task.Source{}, time.Now())
	tsk.ID = 1
	tsk.Path = "/tmp/does-not-matter/001-a-finding.md"
	return tsk
}

// TestRunNoHookConfigured checks the common case - nothing under hooks/ - is
// silent: no diagnostic written, nothing run.
func TestRunNoHookConfigured(t *testing.T) {
	root := t.TempDir()
	var diag bytes.Buffer
	Run(&diag, root, root, PostAdd, newTask(t), nil)
	if diag.Len() != 0 {
		t.Fatalf("expected no diagnostics, got %q", diag.String())
	}
}

// TestRunUnixExecutableScript exercises the bare, extensionless shape: a
// script with its own shebang, made executable. Skipped on Windows, where
// that shape is deliberately unsupported (see resolve).
func TestRunUnixExecutableScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bare executable hooks are unix-only")
	}
	root := t.TempDir()
	dir := Dir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "out.txt")
	script := "#!/bin/sh\n" +
		"echo \"$BACKLOG_EVENT $BACKLOG_TASK_ID $BACKLOG_TASK_TITLE\" > \"" + out + "\"\n" +
		"cat > \"" + out + ".stdin\"\n"
	path := filepath.Join(dir, PostAdd)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	var diag bytes.Buffer
	Run(&diag, root, root, PostAdd, newTask(t), nil)
	if diag.Len() != 0 {
		t.Fatalf("expected no diagnostics from a successful hook, got %q", diag.String())
	}

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("hook did not write its output file: %v", err)
	}
	if want := "post-add 1 a finding\n"; string(got) != want {
		t.Errorf("hook saw env %q, want %q", got, want)
	}

	stdin, err := os.ReadFile(out + ".stdin")
	if err != nil {
		t.Fatalf("hook did not receive stdin: %v", err)
	}
	if !strings.Contains(string(stdin), `"title":"a finding"`) {
		t.Errorf("stdin did not carry the task JSON: %q", stdin)
	}
}

// TestRunNotExecutable checks that a hook file left non-executable is
// reported rather than silently skipped or silently run.
func TestRunNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exec-bit semantics are unix-only")
	}
	root := t.TempDir()
	dir := Dir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, PostAdd), []byte("#!/bin/sh\ntrue\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var diag bytes.Buffer
	Run(&diag, root, root, PostAdd, newTask(t), nil)
	if !strings.Contains(diag.String(), "not executable") {
		t.Errorf("expected a not-executable diagnostic, got %q", diag.String())
	}
}

// TestRunFailingHookDoesNotPanic checks that a hook exiting non-zero is
// reported as a diagnostic and nothing more - Run has no error return, by
// design, because a hook is a side effect that never speaks for the command
// that triggered it.
func TestRunFailingHookReportsDiagnostic(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shebang scripts are unix-only")
	}
	root := t.TempDir()
	dir := Dir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, PostRemove), []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var diag bytes.Buffer
	Run(&diag, root, root, PostRemove, newTask(t), nil)
	if !strings.Contains(diag.String(), "hook failed") {
		t.Errorf("expected a hook-failed diagnostic, got %q", diag.String())
	}
}

// TestRunPicksFirstMatchingShape checks the priority order: when both a
// bare script and a .sh script exist for the same event, the bare one wins.
func TestRunPicksFirstMatchingShape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bare executable hooks are unix-only")
	}
	root := t.TempDir()
	dir := Dir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, "which.txt")
	bare := "#!/bin/sh\necho bare > \"" + marker + "\"\n"
	shSuffixed := "#!/bin/sh\necho sh > \"" + marker + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, PostEdit), []byte(bare), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, PostEdit+".sh"), []byte(shSuffixed), 0o755); err != nil {
		t.Fatal(err)
	}

	var diag bytes.Buffer
	Run(&diag, root, root, PostEdit, newTask(t), nil)
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("neither hook ran: %v", err)
	}
	if string(got) != "bare\n" {
		t.Errorf("ran %q, want the bare shape to win", got)
	}
}

// TestRunNilTaskDoesNotPanic exercises the "no task" shape, in case a future
// event fires without one.
func TestRunNilTaskDoesNotPanic(t *testing.T) {
	root := t.TempDir()
	var diag bytes.Buffer
	Run(&diag, root, root, "some-event", nil, map[string]string{"BACKLOG_X": "1"})
	_ = diag // no hook configured; just must not panic
}
