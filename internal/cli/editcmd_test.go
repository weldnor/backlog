package cli

import (
	"strings"
	"testing"
)

func TestEditChangesTitleDescriptionAndTags(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "Original title", "--description", "Original body.", "--tag", "old")

	var got TaskView
	decode(t, h.mustRun("edit", "1",
		"--title", "New title",
		"--description", "New body.",
		"--tag", "fresh",
		"--json",
	), &got)

	if got.Title != "New title" {
		t.Errorf("Title = %q, want %q", got.Title, "New title")
	}
	if got.Description != "New body." {
		t.Errorf("Description = %q, want %q", got.Description, "New body.")
	}
	if strings.Join(got.Tags, ",") != "fresh" {
		t.Errorf("Tags = %v, want [fresh]", got.Tags)
	}

	// The file was renamed to match the new title, the same as `set` does not
	// do but a title change on disk always must.
	if h.path(".backlog", "tasks", "001-new-title.md") == "" {
		t.Fatal("expected renamed file")
	}
	file := readTaskFile(t, h.path(".backlog", "tasks", "001-new-title.md"))
	if !strings.Contains(file, "New body.") {
		t.Errorf("file does not contain the new body:\n%s", file)
	}
}

func TestEditPartialChangeLeavesOtherFieldsAlone(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "Keep me", "--description", "Keep this.", "--tag", "keep")

	var got TaskView
	decode(t, h.mustRun("edit", "1", "--description", "Changed only this.", "--json"), &got)

	if got.Title != "Keep me" {
		t.Errorf("Title changed unexpectedly: %q", got.Title)
	}
	if strings.Join(got.Tags, ",") != "keep" {
		t.Errorf("Tags changed unexpectedly: %v", got.Tags)
	}
	if got.Description != "Changed only this." {
		t.Errorf("Description = %q", got.Description)
	}
}

func TestEditRequiresAtLeastOneField(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "A task")

	code, _, stderr := h.run("edit", "1")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (usage error); stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "nothing to edit") {
		t.Errorf("stderr = %q, want mention of 'nothing to edit'", stderr)
	}
}

func TestEditRejectsBlankTitle(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "A task")

	code, _, stderr := h.run("edit", "1", "--title", "   ")
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "title must not be empty") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestEditUnknownTask(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	code, _, stderr := h.run("edit", "9", "--title", "x")
	if code == 0 {
		t.Fatal("expected failure for an unknown task")
	}
	if !strings.Contains(stderr, "no such task") {
		t.Errorf("stderr = %q", stderr)
	}
}
