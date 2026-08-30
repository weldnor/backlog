package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/antonkolesov/backlog/internal/task"
)

// prioritiesOf reports the priority of each listed task, in the order listed.
func prioritiesOf(views []TaskView) []string {
	out := make([]string, 0, len(views))
	for _, v := range views {
		out = append(out, v.Priority)
	}
	return out
}

func joinIDs(views []TaskView) string { return strings.Join(idsOf(views), ",") }

// seed builds a backlog whose priorities and identifiers deliberately disagree,
// so that ordering and filtering can be told apart from insertion order.
func seed(t *testing.T, h *harness) {
	t.Helper()
	h.mustRun("add", "One", "--priority", "low")
	h.mustRun("add", "Two", "--priority", "high", "--tag", "bug")
	h.mustRun("add", "Three", "--priority", "medium")
	h.mustRun("add", "Four", "--priority", "high")
}

func TestAddPriority(t *testing.T) {
	t.Run("defaults to medium", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()

		var got TaskView
		decode(t, h.mustRun("add", "A finding", "--json"), &got)
		if got.Priority != task.PriorityMedium {
			t.Errorf("Priority = %q, want %q", got.Priority, task.PriorityMedium)
		}
		file := readTaskFile(t, h.path(".backlog", "tasks", "001-a-finding.md"))
		if !strings.Contains(file, "priority: medium") {
			t.Errorf("the default was not written to the file:\n%s", file)
		}
	})

	t.Run("takes an explicit value", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()

		var got TaskView
		decode(t, h.mustRun("add", "A bad one", "--priority", "high", "--json"), &got)
		if got.Priority != task.PriorityHigh {
			t.Errorf("Priority = %q, want %q", got.Priority, task.PriorityHigh)
		}
	})

	t.Run("rejects an unknown value and creates nothing", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()

		code, stdout, stderr := h.run("add", "A finding", "--priority", "urgent")
		if code == 0 {
			t.Error("an unknown priority was accepted")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want it empty", stdout)
		}
		if !strings.Contains(stderr, "high, medium, low") {
			t.Errorf("stderr = %q, want it to list the permitted values", stderr)
		}
		var listed []TaskView
		decode(t, h.mustRun("list", "--json"), &listed)
		if len(listed) != 0 {
			t.Errorf("a task was created despite the failure: %+v", listed)
		}
	})
}

func TestSetPriority(t *testing.T) {
	t.Run("changes priority without touching status", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")
		h.mustRun("set", "1", "--priority", "high")

		var got TaskView
		decode(t, h.mustRun("show", "1", "--json"), &got)
		if got.Priority != task.PriorityHigh {
			t.Errorf("Priority = %q, want high", got.Priority)
		}
		if got.Status != task.StatusTodo {
			t.Errorf("Status = %q, want it unchanged", got.Status)
		}
		// Status is what decides the directory, so the file must not move.
		if _, err := os.Stat(h.path(".backlog", "tasks", "001-a-finding.md")); err != nil {
			t.Errorf("the task left the tasks directory: %v", err)
		}
	})

	t.Run("changes status and priority together", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")
		h.mustRun("set", "1", "doing", "--priority", "low")

		var got TaskView
		decode(t, h.mustRun("show", "1", "--json"), &got)
		if got.Status != task.StatusDoing || got.Priority != task.PriorityLow {
			t.Errorf("Status = %q, Priority = %q, want doing and low", got.Status, got.Priority)
		}
	})

	t.Run("rejects an unknown value and leaves the task alone", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")
		path := h.path(".backlog", "tasks", "001-a-finding.md")
		before := readTaskFile(t, path)

		code, _, stderr := h.run("set", "1", "--priority", "urgent")
		if code == 0 {
			t.Error("an unknown priority was accepted")
		}
		if !strings.Contains(stderr, "high, medium, low") {
			t.Errorf("stderr = %q, want it to list the permitted values", stderr)
		}
		if after := readTaskFile(t, path); after != before {
			t.Error("the task was modified despite the failure")
		}
	})

	t.Run("still refuses to do nothing", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")

		code, _, stderr := h.run("set", "1")
		if code == 0 {
			t.Error("set with nothing to change succeeded")
		}
		if !strings.Contains(stderr, "priority") {
			t.Errorf("stderr = %q, want it to mention the priority option", stderr)
		}
	})
}

func TestListOrdersByPriority(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	seed(t, h)

	var got []TaskView
	decode(t, h.mustRun("list", "--json"), &got)
	// Descending priority, then ascending identifier within a priority.
	if want := "2,4,3,1"; joinIDs(got) != want {
		t.Errorf("order = %s (%v), want %s", joinIDs(got), prioritiesOf(got), want)
	}
}

func TestListGroupsByStatusAndKeepsPriorityOrder(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	seed(t, h)
	h.mustRun("set", "1", "doing")

	stdout := h.mustRun("list")
	todo, doing := strings.Index(stdout, "todo ("), strings.Index(stdout, "doing (")
	if todo < 0 || doing < 0 || todo > doing {
		t.Fatalf("status is no longer the outer grouping:\n%s", stdout)
	}
	// Inside the todo group the two high ones precede the medium one.
	rest := stdout[todo:]
	two, four, three := strings.Index(rest, "Two"), strings.Index(rest, "Four"), strings.Index(rest, "Three")
	if two < 0 || four < 0 || three < 0 || two > four || four > three {
		t.Errorf("priority order was not preserved inside the status group:\n%s", stdout)
	}
	if !strings.Contains(stdout, "high") {
		t.Errorf("the listing does not show the priority:\n%s", stdout)
	}
}

func TestListFiltersByPriority(t *testing.T) {
	t.Run("one value", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		seed(t, h)

		var got []TaskView
		decode(t, h.mustRun("list", "--priority", "high", "--json"), &got)
		if want := "2,4"; joinIDs(got) != want {
			t.Errorf("ids = %s, want %s", joinIDs(got), want)
		}
	})

	t.Run("several values are a disjunction", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		seed(t, h)

		var got []TaskView
		decode(t, h.mustRun("list", "--priority", "high", "--priority", "medium", "--json"), &got)
		if want := "2,4,3"; joinIDs(got) != want {
			t.Errorf("ids = %s, want %s", joinIDs(got), want)
		}
		for _, v := range got {
			if v.Priority == task.PriorityLow {
				t.Errorf("a low priority task passed the filter: %+v", v)
			}
		}
	})

	t.Run("combines with a tag filter", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		seed(t, h)

		var got []TaskView
		decode(t, h.mustRun("list", "--priority", "high", "--tag", "bug", "--json"), &got)
		if want := "2"; joinIDs(got) != want {
			t.Errorf("ids = %s, want %s", joinIDs(got), want)
		}
	})

	t.Run("rejects an unknown value", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		seed(t, h)

		code, stdout, stderr := h.run("list", "--priority", "urgent")
		if code == 0 {
			t.Error("an unknown priority filter was accepted")
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want it empty", stdout)
		}
		if !strings.Contains(stderr, "high, medium, low") {
			t.Errorf("stderr = %q, want it to list the permitted values", stderr)
		}
	})
}

func TestShowDisplaysPriority(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "A finding", "--priority", "high")

	if stdout := h.mustRun("show", "1"); !strings.Contains(stdout, "priority high") {
		t.Errorf("show did not display the priority:\n%s", stdout)
	}
}

func TestJSONCarriesPriorityEverywhere(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "A finding", "--priority", "high")

	t.Run("add", func(t *testing.T) {
		var got TaskView
		decode(t, h.mustRun("add", "Another", "--priority", "low", "--json"), &got)
		if got.Priority != task.PriorityLow {
			t.Errorf("Priority = %q", got.Priority)
		}
	})
	t.Run("show", func(t *testing.T) {
		var got TaskView
		decode(t, h.mustRun("show", "1", "--json"), &got)
		if got.Priority != task.PriorityHigh {
			t.Errorf("Priority = %q", got.Priority)
		}
	})
	t.Run("list", func(t *testing.T) {
		var got []TaskView
		decode(t, h.mustRun("list", "--json"), &got)
		if len(got) == 0 {
			t.Fatal("nothing was listed")
		}
		for _, v := range got {
			if v.Priority == "" {
				t.Errorf("a listed task carried no priority: %+v", v)
			}
		}
	})
	t.Run("search", func(t *testing.T) {
		var got []SearchResultView
		decode(t, h.mustRun("search", "finding", "--json"), &got)
		if len(got) == 0 {
			t.Fatal("the search found nothing to check")
		}
		for _, r := range got {
			if r.Task.Priority == "" {
				t.Errorf("a search result carried no priority: %+v", r.Task)
			}
		}
	})
	t.Run("set", func(t *testing.T) {
		var got TaskView
		decode(t, h.mustRun("set", "1", "--priority", "medium", "--json"), &got)
		if got.Priority != task.PriorityMedium {
			t.Errorf("Priority = %q", got.Priority)
		}
	})
	t.Run("rm", func(t *testing.T) {
		var got TaskView
		decode(t, h.mustRun("rm", "2", "--json"), &got)
		if got.Priority != task.PriorityLow {
			t.Errorf("Priority = %q", got.Priority)
		}
	})
}

// Search deliberately takes no priority filter: it answers whether a finding
// has already been recorded, and a severity filter would let a duplicate hide
// behind it.
func TestSearchHasNoPriorityFilter(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	seed(t, h)

	code, stdout, stderr := h.run("search", "One", "--priority", "low")
	if code == 0 {
		t.Error("search accepted a priority filter")
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want it empty", stdout)
	}
	if !strings.Contains(stderr, "priority") {
		t.Errorf("stderr = %q, want it to name the rejected flag", stderr)
	}
}

// Search ranks by where the query matched, and priority must not disturb it.
func TestSearchOrderIgnoresPriority(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "cache", "--priority", "low")
	h.mustRun("add", "cache", "--priority", "high")
	h.mustRun("add", "unrelated", "--description", "cache", "--priority", "high")

	var got []SearchResultView
	decode(t, h.mustRun("search", "cache", "--json"), &got)
	ids := make([]string, 0, len(got))
	for _, r := range got {
		ids = append(ids, itoa(r.Task.ID))
	}
	// Title matches first in ascending identifier order, then the body match,
	// unchanged by the fact that 2 and 3 are the high ones.
	if want := "1,2,3"; strings.Join(ids, ",") != want {
		t.Errorf("order = %s, want %s", strings.Join(ids, ","), want)
	}
}
