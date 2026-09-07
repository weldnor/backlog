package cli

import (
	"strings"
	"testing"
)

func TestLinkAddAndRm(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "First task")
	h.mustRun("add", "Second task")

	var got TaskView
	decode(t, h.mustRun("link", "add", "1", "blocks", "2", "--json"), &got)
	if len(got.Links) != 1 || got.Links[0].Type != "blocks" || got.Links[0].ID != 2 {
		t.Fatalf("Links = %+v, want [{blocks 2}]", got.Links)
	}

	// A second link is added, not substituted for the first — add is
	// incremental, unlike edit --link.
	decode(t, h.mustRun("link", "add", "1", "related", "2", "--json"), &got)
	if len(got.Links) != 2 {
		t.Fatalf("Links = %+v, want two entries", got.Links)
	}

	decode(t, h.mustRun("link", "rm", "1", "blocks", "2", "--json"), &got)
	if len(got.Links) != 1 || got.Links[0].Type != "related" {
		t.Fatalf("Links = %+v, want only the related link left", got.Links)
	}
}

func TestLinkAddRejectsSelfLink(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "Solo task")
	code, _, stderr := h.run("link", "add", "1", "related", "1")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage error): %s", code, stderr)
	}
}

func TestLinkAddRejectsUnknownType(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "First task")
	h.mustRun("add", "Second task")
	code, _, stderr := h.run("link", "add", "1", "blockd-by", "2")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage error): %s", code, stderr)
	}
	if !strings.Contains(stderr, "unknown link type") {
		t.Errorf("stderr = %q, want it to name the bad type", stderr)
	}
}

func TestLinkAddIsIdempotentUnderDeduplication(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "First task")
	h.mustRun("add", "Second task")
	h.mustRun("link", "add", "1", "blocks", "2")

	var got TaskView
	decode(t, h.mustRun("link", "add", "1", "blocks", "2", "--json"), &got)
	if len(got.Links) != 1 {
		t.Errorf("Links = %+v, want the duplicate collapsed", got.Links)
	}
}

func TestAddAndEditAcceptLinkFlag(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "Target task")

	var created TaskView
	decode(t, h.mustRun("add", "Source task", "--link", "blocks:1", "--json"), &created)
	if len(created.Links) != 1 || created.Links[0].Type != "blocks" || created.Links[0].ID != 1 {
		t.Fatalf("Links = %+v, want [{blocks 1}]", created.Links)
	}

	var edited TaskView
	decode(t, h.mustRun("edit", "2", "--link", "related:1", "--json"), &edited)
	if len(edited.Links) != 1 || edited.Links[0].Type != "related" {
		t.Fatalf("Links = %+v, want the link list replaced with [{related 1}]", edited.Links)
	}
}
