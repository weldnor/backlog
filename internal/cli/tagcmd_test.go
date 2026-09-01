package cli

import (
	"strings"
	"testing"
)

func TestTagRmRemovesFromEveryTask(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "One", "--tag", "bug", "--tag", "keep")
	h.mustRun("add", "Two", "--tag", "bug")
	h.mustRun("add", "Three", "--tag", "other")

	out := h.mustRun("tag", "rm", "bug")
	if !strings.Contains(out, "removed tag \"bug\" from 2 task(s)") {
		t.Errorf("output = %q", out)
	}

	var one, two, three TaskView
	decode(t, h.mustRun("show", "1", "--json"), &one)
	decode(t, h.mustRun("show", "2", "--json"), &two)
	decode(t, h.mustRun("show", "3", "--json"), &three)

	if strings.Join(one.Tags, ",") != "keep" {
		t.Errorf("task 1 tags = %v, want [keep]", one.Tags)
	}
	if len(two.Tags) != 0 {
		t.Errorf("task 2 tags = %v, want none", two.Tags)
	}
	if strings.Join(three.Tags, ",") != "other" {
		t.Errorf("task 3 tags untouched = %v", three.Tags)
	}
}

func TestTagRmIsCaseInsensitiveAndNoopWhenAbsent(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "One", "--tag", "Bug")

	out := h.mustRun("tag", "rm", "bug")
	if !strings.Contains(out, "from 1 task(s)") {
		t.Fatalf("output = %q", out)
	}

	out2 := h.mustRun("tag", "rm", "nonexistent")
	if !strings.Contains(out2, "from 0 task(s)") {
		t.Errorf("output = %q", out2)
	}
}

func TestTagRenameReplacesAcrossTasks(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "One", "--tag", "bug", "--tag", "keep")
	h.mustRun("add", "Two", "--tag", "other")

	out := h.mustRun("tag", "rename", "bug", "defect")
	if !strings.Contains(out, "renamed tag \"bug\" to \"defect\" on 1 task(s)") {
		t.Errorf("output = %q", out)
	}

	var one, two TaskView
	decode(t, h.mustRun("show", "1", "--json"), &one)
	decode(t, h.mustRun("show", "2", "--json"), &two)

	if strings.Join(one.Tags, ",") != "keep,defect" {
		t.Errorf("task 1 tags = %v, want [keep defect]", one.Tags)
	}
	if strings.Join(two.Tags, ",") != "other" {
		t.Errorf("task 2 tags untouched = %v", two.Tags)
	}
}

func TestTagRenameMergesWithExistingTarget(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "One", "--tag", "bug", "--tag", "defect")

	h.mustRun("tag", "rename", "bug", "defect")

	var one TaskView
	decode(t, h.mustRun("show", "1", "--json"), &one)
	if strings.Join(one.Tags, ",") != "defect" {
		t.Errorf("task 1 tags = %v, want [defect] (deduplicated)", one.Tags)
	}
}

func TestTagRmAndRenameRequireArguments(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	if code, _, _ := h.run("tag"); code != 2 {
		t.Errorf("backlog tag exit = %d, want 2", code)
	}
	if code, _, _ := h.run("tag", "rm"); code != 2 {
		t.Errorf("backlog tag rm exit = %d, want 2", code)
	}
	if code, _, _ := h.run("tag", "rename", "only-one"); code != 2 {
		t.Errorf("backlog tag rename exit = %d, want 2", code)
	}
	if code, _, _ := h.run("tag", "bogus"); code != 2 {
		t.Errorf("backlog tag bogus exit = %d, want 2", code)
	}
}
