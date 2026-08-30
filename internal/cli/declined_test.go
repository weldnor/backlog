package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/weldnor/backlog/internal/task"
)

// decline adds a task and declines it, which is the shape almost every case
// here starts from.
func decline(t *testing.T, h *harness, title, reason string) {
	t.Helper()
	h.mustRun("add", title)
	h.mustRun("set", "1", "declined", "--reason", reason)
}

func TestSetDeclineRules(t *testing.T) {
	t.Run("declining records the status and the reason", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")

		var got TaskView
		decode(t, h.mustRun("set", "1", "declined", "--reason", "the fix costs more than the bug", "--json"), &got)
		if got.Status != task.StatusDeclined {
			t.Errorf("Status = %q, want %q", got.Status, task.StatusDeclined)
		}
		if got.Reason != "the fix costs more than the bug" {
			t.Errorf("Reason = %q", got.Reason)
		}
		file := readTaskFile(t, h.path(".backlog", "archive", "001-a-finding.md"))
		if !strings.Contains(file, "status: declined") || !strings.Contains(file, "reason: the fix costs more than the bug") {
			t.Errorf("the decline was not written to the file:\n%s", file)
		}
	})

	t.Run("declining without a reason is a usage error", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")

		code, stdout, stderr := h.run("set", "1", "declined")
		if code == 0 {
			t.Error("a decline with no reason was accepted")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want it empty", stdout)
		}
		if !strings.Contains(stderr, "--reason") {
			t.Errorf("stderr does not name the missing flag: %q", stderr)
		}
		// The task is untouched, so nothing was half applied.
		var got TaskView
		decode(t, h.mustRun("show", "1", "--json"), &got)
		if got.Status != task.StatusTodo {
			t.Errorf("Status = %q, want the task unchanged", got.Status)
		}
	})

	t.Run("a reason with another status is a usage error", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")

		code, _, stderr := h.run("set", "1", "done", "--reason", "not worth it")
		if code == 0 {
			t.Error("a reason on a done task was accepted")
		}
		if !strings.Contains(stderr, "declined") {
			t.Errorf("stderr does not explain the rule: %q", stderr)
		}
		var got TaskView
		decode(t, h.mustRun("show", "1", "--json"), &got)
		if got.Status != task.StatusTodo || got.Reason != "" {
			t.Errorf("the task was modified: %+v", got)
		}
	})

	t.Run("a reason alone needs a task that is already declined", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")

		code, _, stderr := h.run("set", "1", "--reason", "not worth it")
		if code == 0 {
			t.Error("a reason was accepted on a todo task")
		}
		if !strings.Contains(stderr, "declined") {
			t.Errorf("stderr does not explain the rule: %q", stderr)
		}
		var got TaskView
		decode(t, h.mustRun("show", "1", "--json"), &got)
		if got.Reason != "" {
			t.Errorf("Reason = %q, want the task unchanged", got.Reason)
		}
	})

	t.Run("a reason alone revises an existing decline", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		decline(t, h, "A finding", "first attempt at saying why")

		var got TaskView
		decode(t, h.mustRun("set", "1", "--reason", "the call site is cold", "--json"), &got)
		if got.Status != task.StatusDeclined {
			t.Errorf("Status = %q, want it unchanged", got.Status)
		}
		if got.Reason != "the call site is cold" {
			t.Errorf("Reason = %q, want the revised text", got.Reason)
		}
		file := readTaskFile(t, h.path(".backlog", "archive", "001-a-finding.md"))
		if strings.Contains(file, "first attempt") {
			t.Errorf("the old reason survived:\n%s", file)
		}
	})

	t.Run("leaving declined clears the reason", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		decline(t, h, "A finding", "not worth the churn")

		var got TaskView
		decode(t, h.mustRun("set", "1", "todo", "--json"), &got)
		if got.Status != task.StatusTodo {
			t.Errorf("Status = %q, want %q", got.Status, task.StatusTodo)
		}
		// git keeps what it said; carrying it forward would describe a state
		// the task is no longer in.
		if got.Reason != "" {
			t.Errorf("Reason = %q, want it cleared", got.Reason)
		}
		file := readTaskFile(t, h.path(".backlog", "tasks", "001-a-finding.md"))
		if strings.Contains(file, "reason") {
			t.Errorf("the reason survived the reopening:\n%s", file)
		}
	})

	t.Run("nothing to do names the reason", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")

		code, _, stderr := h.run("set", "1")
		if code == 0 {
			t.Error("an empty set was accepted")
		}
		if !strings.Contains(stderr, "--reason") {
			t.Errorf("the usage error does not mention the reason: %q", stderr)
		}
	})
}

func TestDeclineMovesTheFile(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	decline(t, h, "A finding", "not worth the churn")

	if _, err := os.Stat(h.path(".backlog", "archive", "001-a-finding.md")); err != nil {
		t.Errorf("the declined task is not in the archive: %v", err)
	}
	if _, err := os.Stat(h.path(".backlog", "tasks", "001-a-finding.md")); !os.IsNotExist(err) {
		t.Error("the declined task was left among the active tasks")
	}

	// Reopening brings it back, without the reason.
	h.mustRun("set", "1", "todo")
	if _, err := os.Stat(h.path(".backlog", "tasks", "001-a-finding.md")); err != nil {
		t.Errorf("the reopened task is not among the active tasks: %v", err)
	}
	if _, err := os.Stat(h.path(".backlog", "archive", "001-a-finding.md")); !os.IsNotExist(err) {
		t.Error("the reopened task was left in the archive")
	}
	if file := readTaskFile(t, h.path(".backlog", "tasks", "001-a-finding.md")); strings.Contains(file, "reason") {
		t.Errorf("the reopened file still declares a reason:\n%s", file)
	}
}

// Both terminal statuses live in the archive, so moving between them is a
// change of status and nothing else.
func TestDoneToDeclinedStaysInTheArchive(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "A finding")
	h.mustRun("set", "1", "done")

	var got TaskView
	decode(t, h.mustRun("set", "1", "declined", "--reason", "reverted; it was never really fixed", "--json"), &got)
	if got.Status != task.StatusDeclined {
		t.Errorf("Status = %q, want %q", got.Status, task.StatusDeclined)
	}
	if _, err := os.Stat(h.path(".backlog", "archive", "001-a-finding.md")); err != nil {
		t.Errorf("the task left the archive: %v", err)
	}
	if _, err := os.Stat(h.path(".backlog", "tasks", "001-a-finding.md")); !os.IsNotExist(err) {
		t.Error("the task was moved out of the archive")
	}
}

func TestShowDisplaysTheReason(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	decline(t, h, "A finding", "the call site is cold")

	stdout := h.mustRun("show", "1")
	if !strings.Contains(stdout, "reason") || !strings.Contains(stdout, "the call site is cold") {
		t.Errorf("show did not print the reason:\n%s", stdout)
	}

	h.mustRun("add", "A live finding")
	if stdout := h.mustRun("show", "2"); strings.Contains(stdout, "reason") {
		t.Errorf("show printed a reason line for a todo task:\n%s", stdout)
	}
}

// The shape of the JSON must not vary with status, so a consumer can read the
// field without first asking whether it is there.
func TestReasonIsAlwaysPresentInJSON(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	var added TaskView
	decode(t, h.mustRun("add", "A finding", "--json"), &added)
	if added.Reason != "" {
		t.Errorf("add reported Reason = %q, want empty", added.Reason)
	}
	for _, out := range []string{
		h.mustRun("show", "1", "--json"),
		h.mustRun("set", "1", "--priority", "low", "--json"),
	} {
		if !strings.Contains(out, `"reason": ""`) {
			t.Errorf("the field is absent rather than empty:\n%s", out)
		}
	}
	var listed []TaskView
	decode(t, h.mustRun("list", "--json"), &listed)
	if len(listed) != 1 || listed[0].Reason != "" {
		t.Errorf("list did not carry an empty reason: %+v", listed)
	}

	h.mustRun("set", "1", "declined", "--reason", "not worth the churn")
	var shown TaskView
	decode(t, h.mustRun("show", "1", "--json"), &shown)
	if shown.Reason != "not worth the churn" {
		t.Errorf("show reported Reason = %q", shown.Reason)
	}
	var found []SearchResultView
	decode(t, h.mustRun("search", "finding", "--json"), &found)
	if len(found) != 1 || found[0].Task.Reason != "not worth the churn" {
		t.Errorf("search did not carry the reason: %+v", found)
	}
}

func TestListTreatsDeclinedAsTerminal(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "A declined finding")
	h.mustRun("add", "A live finding")
	h.mustRun("add", "A finished finding")
	h.mustRun("add", "A finding in flight")
	h.mustRun("set", "1", "declined", "--reason", "not worth the churn")
	h.mustRun("set", "3", "done")
	h.mustRun("set", "4", "doing")

	t.Run("excluded by default", func(t *testing.T) {
		var got []TaskView
		decode(t, h.mustRun("list", "--json"), &got)
		if joinIDs(got) != "2,4" {
			t.Errorf("default listing = %s, want only the live tasks", joinIDs(got))
		}
	})

	t.Run("included by --all, in the last group", func(t *testing.T) {
		stdout := h.mustRun("list", "--all")
		todo := strings.Index(stdout, "todo (")
		doing := strings.Index(stdout, "doing (")
		done := strings.Index(stdout, "done (")
		declined := strings.Index(stdout, "declined (")
		if todo < 0 || doing < 0 || done < 0 || declined < 0 {
			t.Fatalf("a status group is missing:\n%s", stdout)
		}
		if !(todo < doing && doing < done && done < declined) {
			t.Errorf("groups are not in the order todo, doing, done, declined:\n%s", stdout)
		}
	})

	t.Run("selectable on its own", func(t *testing.T) {
		var got []TaskView
		decode(t, h.mustRun("list", "--status", "declined", "--json"), &got)
		if len(got) != 1 || got[0].ID != 1 {
			t.Errorf("--status declined returned %v", idsOf(got))
		}
	})
}
