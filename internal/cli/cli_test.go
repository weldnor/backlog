package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type harness struct {
	t   *testing.T
	dir string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	return &harness{t: t, dir: t.TempDir()}
}

// run invokes the CLI exactly as the process entry point does, so that the
// output convention — data on stdout, diagnostics on stderr — is what is
// actually under test.
func (h *harness) run(args ...string) (int, string, string) {
	h.t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(Env{Stdout: &stdout, Stderr: &stderr, Dir: h.dir}, args)
	return code, stdout.String(), stderr.String()
}

func (h *harness) mustRun(args ...string) string {
	h.t.Helper()
	code, stdout, stderr := h.run(args...)
	if code != 0 {
		h.t.Fatalf("backlog %v exited %d: %s", args, code, stderr)
	}
	return stdout
}

func (h *harness) initBacklog() {
	h.t.Helper()
	h.mustRun("init")
}

func (h *harness) path(parts ...string) string {
	return filepath.Join(append([]string{h.dir}, parts...)...)
}

func TestHelpListsEveryCommand(t *testing.T) {
	h := newHarness(t)
	for _, args := range [][]string{{}, {"--help"}, {"-h"}, {"help"}} {
		code, stdout, stderr := h.run(args...)
		if code != 0 {
			t.Errorf("backlog %v exited %d", args, code)
		}
		if stderr != "" {
			t.Errorf("backlog %v wrote to stderr: %q", args, stderr)
		}
		for _, name := range []string{"init", "add", "list", "search", "show", "set", "rm", "validate"} {
			if !strings.Contains(stdout, name) {
				t.Errorf("backlog %v does not list %q", args, name)
			}
		}
	}
}

func TestVersion(t *testing.T) {
	h := newHarness(t)
	for _, arg := range []string{"--version", "-v", "version"} {
		code, stdout, stderr := h.run(arg)
		if code != 0 {
			t.Errorf("backlog %s exited %d: %s", arg, code, stderr)
		}
		if strings.TrimSpace(stdout) == "" {
			t.Errorf("backlog %s printed no version", arg)
		}
	}
}

// Diagnostics never touch the data stream, so a caller can pipe stdout without
// having to filter it.
func TestUnknownCommandGoesToStderr(t *testing.T) {
	h := newHarness(t)
	code, stdout, stderr := h.run("frobnicate")
	if code == 0 {
		t.Error("an unknown command exited zero")
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want it empty", stdout)
	}
	if !strings.Contains(stderr, "frobnicate") {
		t.Errorf("stderr = %q, want it to name the command", stderr)
	}
}

func TestCommandsFailWithoutABacklog(t *testing.T) {
	h := newHarness(t)
	for _, args := range [][]string{
		{"list"}, {"add", "x"}, {"show", "1"}, {"set", "1", "done"}, {"rm", "1"}, {"search", "x"}, {"validate"},
	} {
		code, stdout, stderr := h.run(args...)
		if code == 0 {
			t.Errorf("backlog %v succeeded without a backlog", args)
		}
		if stdout != "" {
			t.Errorf("backlog %v wrote %q to stdout", args, stdout)
		}
		if !strings.Contains(stderr, "no backlog found") {
			t.Errorf("backlog %v said %q, want it to explain that no backlog was found", args, stderr)
		}
	}
}

func TestInit(t *testing.T) {
	h := newHarness(t)
	stdout := h.mustRun("init")
	if !strings.Contains(stdout, ".backlog") {
		t.Errorf("init did not report where the backlog was created: %q", stdout)
	}
	if info, err := os.Stat(h.path(filepath.FromSlash(".backlog/tasks"))); err != nil || !info.IsDir() {
		t.Errorf(".backlog/tasks was not created: %v", err)
	}
	if _, err := os.Stat(h.path(filepath.FromSlash(".backlog/archive"))); !os.IsNotExist(err) {
		t.Errorf(".backlog/archive should not be created, stat err = %v", err)
	}
}

func TestInitLeavesExistingTasksAlone(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "An existing finding", "--description", "with a body")

	path := h.path(".backlog", "tasks", "001-an-existing-finding.md")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	h.mustRun("init")

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the task did not survive a second init: %v", err)
	}
	if string(after) != string(before) {
		t.Error("a second init changed an existing task")
	}
}

func TestAddMinimalCapture(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	stdout := h.mustRun("add", "Race in session cache")
	if !strings.Contains(stdout, "001") {
		t.Errorf("the identifier was not reported: %q", stdout)
	}

	var got TaskView
	decode(t, h.mustRun("show", "1", "--json"), &got)
	if got.Status != "todo" {
		t.Errorf("Status = %q, want todo", got.Status)
	}
	if len(got.Tags) != 0 {
		t.Errorf("Tags = %v, want none", got.Tags)
	}
	if got.Description != "" {
		t.Errorf("Description = %q, want it empty", got.Description)
	}
}

func TestAddFullCapture(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "Race in session cache",
		"--description", "Two goroutines write the map.",
		"--tag", "bug", "--tag", "concurrency",
		"--file", "internal/session/cache.go", "--file", "internal/session/store.go")

	var got TaskView
	decode(t, h.mustRun("show", "1", "--json"), &got)
	if got.Title != "Race in session cache" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.Description != "Two goroutines write the map." {
		t.Errorf("Description = %q", got.Description)
	}
	if strings.Join(got.Tags, ",") != "bug,concurrency" {
		t.Errorf("Tags = %v", got.Tags)
	}
	if strings.Join(got.Metadata.Source.Files, ",") != "internal/session/cache.go,internal/session/store.go" {
		t.Errorf("Source.Files = %v", got.Metadata.Source.Files)
	}
}

func TestAddRejectsAnEmptyTitle(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	for _, args := range [][]string{{"add"}, {"add", ""}, {"add", "--title", "   "}} {
		code, stdout, stderr := h.run(args...)
		if code == 0 {
			t.Errorf("backlog %v succeeded", args)
		}
		if stdout != "" {
			t.Errorf("backlog %v wrote %q to stdout", args, stdout)
		}
		if !strings.Contains(stderr, "title") {
			t.Errorf("backlog %v said %q, want it to name the problem", args, stderr)
		}
	}
	// Nothing was created along the way.
	entries, err := os.ReadDir(h.path(".backlog", "tasks"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("a failed add created %d files", len(entries))
	}
}

// Capture has to cost one command; a prompt in the middle of it would strand an
// agent with no terminal.
func TestAddNeverReadsStandardInput(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	saved := os.Stdin
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = devNull
	defer func() {
		os.Stdin = saved
		devNull.Close()
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.mustRun("add", "A finding captured with no terminal")
	}()
	<-done

	if _, err := os.Stat(h.path(".backlog", "tasks", "001-a-finding-captured-with-no-terminal.md")); err != nil {
		t.Errorf("the task was not created: %v", err)
	}
}

func TestList(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "First finding", "--tag", "bug")
	h.mustRun("add", "Second finding", "--tag", "flake")
	h.mustRun("add", "Third finding", "--tag", "bug")
	h.mustRun("set", "2", "doing")
	h.mustRun("set", "3", "done")

	t.Run("default scope covers every status", func(t *testing.T) {
		var got []TaskView
		decode(t, h.mustRun("list", "--json"), &got)
		if ids := idsOf(got); strings.Join(ids, ",") != "1,2,3" {
			t.Errorf("ids = %v, want every task", ids)
		}
	})

	t.Run("a status subcommand narrows to one status", func(t *testing.T) {
		var got []TaskView
		decode(t, h.mustRun("list", "done", "--json"), &got)
		if len(got) != 1 || got[0].ID != 3 {
			t.Errorf("got %v, want only task 3", idsOf(got))
		}
	})

	t.Run("a subcommand and a tag filter combine", func(t *testing.T) {
		var got []TaskView
		decode(t, h.mustRun("list", "done", "--tag", "bug", "--json"), &got)
		if len(got) != 1 || got[0].ID != 3 {
			t.Errorf("got %v, want only task 3", idsOf(got))
		}
	})

	t.Run("an unknown subcommand is rejected", func(t *testing.T) {
		code, _, stderr := h.run("list", "blocked")
		if code == 0 {
			t.Error("an unknown subcommand was accepted")
		}
		if !strings.Contains(stderr, "todo, doing, done") {
			t.Errorf("stderr = %q, want the permitted values", stderr)
		}
	})

	t.Run("--status is gone", func(t *testing.T) {
		if code, _, _ := h.run("list", "--status", "done"); code == 0 {
			t.Error("list still accepts --status")
		}
		if code, _, _ := h.run("list", "--all"); code == 0 {
			t.Error("list still accepts --all")
		}
	})

	t.Run("an empty result is not a failure", func(t *testing.T) {
		code, stdout, _ := h.run("list", "--tag", "nonexistent")
		if code != 0 {
			t.Errorf("exit = %d, want 0", code)
		}
		if !strings.Contains(stdout, "no tasks matched") {
			t.Errorf("stdout = %q, want it to say nothing matched", stdout)
		}
	})

	t.Run("human output groups by status", func(t *testing.T) {
		stdout := h.mustRun("list")
		todo := strings.Index(stdout, "todo")
		doing := strings.Index(stdout, "doing")
		done := strings.Index(stdout, "done")
		if todo < 0 || doing < 0 || done < 0 {
			t.Fatalf("output is not grouped by status:\n%s", stdout)
		}
		if !(todo < doing && doing < done) {
			t.Errorf("groups are out of lifecycle order:\n%s", stdout)
		}
	})

	t.Run("listing is deterministic", func(t *testing.T) {
		first := h.mustRun("list", "--json")
		for i := 0; i < 3; i++ {
			if got := h.mustRun("list", "--json"); got != first {
				t.Fatalf("run %d differed:\n%s\n%s", i, got, first)
			}
		}
	})
}

func TestShow(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "An active task", "--description", "the body")
	h.mustRun("add", "An archived task")
	h.mustRun("set", "2", "done")

	t.Run("an active task", func(t *testing.T) {
		stdout := h.mustRun("show", "1")
		for _, want := range []string{"An active task", "todo", "the body"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("output does not contain %q:\n%s", want, stdout)
			}
		}
	})

	t.Run("an archived task", func(t *testing.T) {
		stdout := h.mustRun("show", "2")
		if !strings.Contains(stdout, "An archived task") {
			t.Errorf("an archived task was not found:\n%s", stdout)
		}
	})

	t.Run("an unknown identifier", func(t *testing.T) {
		code, stdout, stderr := h.run("show", "99")
		if code == 0 {
			t.Error("an unknown identifier exited zero")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want it empty", stdout)
		}
		if !strings.Contains(stderr, "no such task") {
			t.Errorf("stderr = %q", stderr)
		}
	})
}

func TestSet(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "A travelling task")

	t.Run("todo to doing keeps the file in place", func(t *testing.T) {
		h.mustRun("set", "1", "doing")
		if _, err := os.Stat(h.path(".backlog", "tasks", "001-a-travelling-task.md")); err != nil {
			t.Errorf("the task left the tasks directory: %v", err)
		}
	})

	t.Run("setting the status it already has succeeds", func(t *testing.T) {
		code, _, stderr := h.run("set", "1", "doing")
		if code != 0 {
			t.Errorf("exit = %d: %s", code, stderr)
		}
		var got TaskView
		decode(t, h.mustRun("show", "1", "--json"), &got)
		if got.Status != "doing" {
			t.Errorf("Status = %q", got.Status)
		}
	})

	t.Run("done keeps the file in tasks and records the reference", func(t *testing.T) {
		h.mustRun("set", "1", "done", "--ref", "openspec:add-auth")
		if _, err := os.Stat(h.path(".backlog", "tasks", "001-a-travelling-task.md")); err != nil {
			t.Errorf("a done task left the tasks directory: %v", err)
		}
		var got TaskView
		decode(t, h.mustRun("show", "1", "--json"), &got)
		if strings.Join(got.Metadata.Refs, ",") != "openspec:add-auth" {
			t.Errorf("Refs = %v", got.Metadata.Refs)
		}
	})

	t.Run("back out of done leaves the file in place", func(t *testing.T) {
		h.mustRun("set", "1", "todo")
		if _, err := os.Stat(h.path(".backlog", "tasks", "001-a-travelling-task.md")); err != nil {
			t.Errorf("the task file moved on a status change: %v", err)
		}
	})

	t.Run("an invalid status changes nothing", func(t *testing.T) {
		before := readTaskFile(t, h.path(".backlog", "tasks", "001-a-travelling-task.md"))
		code, stdout, stderr := h.run("set", "1", "blocked")
		if code == 0 {
			t.Error("an invalid status was accepted")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want it empty", stdout)
		}
		if !strings.Contains(stderr, "todo, doing, done") {
			t.Errorf("stderr = %q, want the permitted values listed", stderr)
		}
		if after := readTaskFile(t, h.path(".backlog", "tasks", "001-a-travelling-task.md")); after != before {
			t.Error("the task was modified by a rejected status")
		}
	})

	t.Run("an unknown task", func(t *testing.T) {
		if code, _, _ := h.run("set", "99", "done"); code == 0 {
			t.Error("setting an unknown task exited zero")
		}
	})
}

func TestRm(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "Doomed")
	h.mustRun("add", "Also doomed")
	h.mustRun("set", "2", "done")

	stdout := h.mustRun("rm", "1")
	if !strings.Contains(stdout, "Doomed") {
		t.Errorf("the removal was not reported: %q", stdout)
	}
	if _, err := os.Stat(h.path(".backlog", "tasks", "001-doomed.md")); !os.IsNotExist(err) {
		t.Error("the task file is still there")
	}

	// Removal reaches into the archive too.
	h.mustRun("rm", "2")
	if _, err := os.Stat(h.path(".backlog", "archive", "002-also-doomed.md")); !os.IsNotExist(err) {
		t.Error("the archived task is still there")
	}

	h.mustRun("add", "A survivor")
	code, stdout, stderr := h.run("rm", "99")
	if code == 0 {
		t.Error("removing an unknown task exited zero")
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want it empty", stdout)
	}
	if !strings.Contains(stderr, "no such task") {
		t.Errorf("stderr = %q", stderr)
	}
	if _, err := os.Stat(h.path(".backlog", "tasks", "001-a-survivor.md")); err != nil {
		t.Error("a failed removal deleted something")
	}
}

func TestJSONOutputIsCleanOnFailure(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	for _, args := range [][]string{
		{"show", "99", "--json"},
		{"list", "blocked", "--json"},
		{"search", "[unclosed", "--regex", "--json"},
		{"rm", "99", "--json"},
	} {
		code, stdout, stderr := h.run(args...)
		if code == 0 {
			t.Errorf("backlog %v exited zero", args)
		}
		if stdout != "" {
			t.Errorf("backlog %v wrote %q to the data stream", args, stdout)
		}
		if stderr == "" {
			t.Errorf("backlog %v reported nothing on stderr", args)
		}
	}
}

func decode(t *testing.T, data string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), v); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, data)
	}
}

func idsOf(views []TaskView) []string {
	out := make([]string, 0, len(views))
	for _, v := range views {
		out = append(out, itoa(v.ID))
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func readTaskFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
