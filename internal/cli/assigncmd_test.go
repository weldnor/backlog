package cli

import (
	"strings"
	"testing"
)

func TestAssignSetsAssignee(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "One")

	out := h.mustRun("assign", "1", "alice")
	if !strings.Contains(out, "assigned 001") || !strings.Contains(out, "alice") {
		t.Errorf("output = %q", out)
	}

	var view TaskView
	decode(t, h.mustRun("show", "1", "--json"), &view)
	if view.Assignee != "alice" {
		t.Errorf("assignee = %q, want alice", view.Assignee)
	}
}

func TestAssignClearUnassigns(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "One")
	h.mustRun("assign", "1", "alice")

	out := h.mustRun("assign", "1", "--clear")
	if !strings.Contains(out, "unassigned 001") {
		t.Errorf("output = %q", out)
	}

	var view TaskView
	decode(t, h.mustRun("show", "1", "--json"), &view)
	if view.Assignee != "" {
		t.Errorf("assignee = %q, want empty", view.Assignee)
	}
}

func TestAssignRequiresAssigneeOrClear(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "One")

	code, _, errOut := h.run("assign", "1")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %q", code, errOut)
	}
}

func TestAssignRejectsEmptyAssignee(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "One")

	code, _, errOut := h.run("assign", "1", "  ")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %q", code, errOut)
	}
}

func TestListFiltersByAssignee(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "One")
	h.mustRun("add", "Two")
	h.mustRun("assign", "1", "alice")
	h.mustRun("assign", "2", "bob")

	var views []TaskView
	decode(t, h.mustRun("list", "--assignee", "alice", "--json"), &views)
	if len(views) != 1 || views[0].ID != 1 {
		t.Errorf("views = %+v, want just task 1", views)
	}

	// Case-insensitive, matching the tag and priority filters.
	decode(t, h.mustRun("list", "--assignee", "ALICE", "--json"), &views)
	if len(views) != 1 || views[0].ID != 1 {
		t.Errorf("views = %+v, want just task 1", views)
	}
}
