package cli

import (
	"strings"
	"testing"
)

func searchFixture(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "Race in session Cache", "--tag", "bug")
	h.mustRun("add", "Unrelated title", "--description", "but the cache is mentioned here")
	h.mustRun("add", "An archived finding about the cache")
	h.mustRun("set", "3", "done")
	return h
}

func TestSearchScopeMatchesList(t *testing.T) {
	h := searchFixture(t)

	var got []SearchResultView
	decode(t, h.mustRun("search", "cache", "--json"), &got)
	for _, r := range got {
		if r.Task.ID == 3 {
			t.Error("an archived task was returned without --all")
		}
	}
	if len(got) != 2 {
		t.Errorf("got %d results, want 2", len(got))
	}

	decode(t, h.mustRun("search", "cache", "--all", "--json"), &got)
	if len(got) != 3 {
		t.Errorf("with --all, got %d results, want 3", len(got))
	}

	// The same status and tag filters as list apply.
	decode(t, h.mustRun("search", "cache", "--tag", "bug", "--json"), &got)
	if len(got) != 1 || got[0].Task.ID != 1 {
		t.Errorf("with a tag filter, got %v", got)
	}
}

func TestSearchHumanOutputShowsMatchedTextInContext(t *testing.T) {
	h := searchFixture(t)
	stdout := h.mustRun("search", "cache")

	if !strings.Contains(stdout, "001") || !strings.Contains(stdout, "todo") {
		t.Errorf("a result is missing its identifier or status:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Race in session Cache") {
		t.Errorf("a result is missing its title:\n%s", stdout)
	}
	if !strings.Contains(stdout, "description:") {
		t.Errorf("the matching field is not named:\n%s", stdout)
	}
	if !strings.Contains(stdout, "but the cache is mentioned here") {
		t.Errorf("the matched text is not shown in context:\n%s", stdout)
	}
}

func TestSearchJSONIdentifiesTheMatchingField(t *testing.T) {
	h := searchFixture(t)
	var got []SearchResultView
	decode(t, h.mustRun("search", "cache", "--json"), &got)

	if len(got) == 0 {
		t.Fatal("no results")
	}
	// Title matches rank first.
	if got[0].Task.ID != 1 {
		t.Errorf("first result is %d, want the title match", got[0].Task.ID)
	}
	if len(got[0].Matches) == 0 {
		t.Fatal("a result carries no matches")
	}
	m := got[0].Matches[0]
	if m.Field != "title" || m.Text == "" {
		t.Errorf("match = %+v, want the field and the matched text", m)
	}
	// The full task travels with the result, so a caller need not fetch it.
	if got[0].Task.Title == "" || got[0].Task.Metadata.Created == "" {
		t.Errorf("the result does not carry the full task: %+v", got[0].Task)
	}
}

func TestSearchWithNoMatchesSucceeds(t *testing.T) {
	h := searchFixture(t)

	code, stdout, _ := h.run("search", "nothing at all like this")
	if code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "no tasks matched") {
		t.Errorf("stdout = %q", stdout)
	}

	var got []SearchResultView
	decode(t, h.mustRun("search", "nothing at all like this", "--json"), &got)
	if len(got) != 0 {
		t.Errorf("got %d results, want an empty set", len(got))
	}
}

func TestSearchRegex(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "HTTP 500 on login")
	h.mustRun("add", "A quiet task")

	var got []SearchResultView
	decode(t, h.mustRun("search", `HTTP [45]0\d`, "--regex", "--json"), &got)
	if len(got) != 1 || got[0].Task.ID != 1 {
		t.Errorf("got %v, want the one matching task", got)
	}

	code, stdout, stderr := h.run("search", "HTTP [45", "--regex")
	if code == 0 {
		t.Error("an invalid regular expression exited zero")
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want it empty", stdout)
	}
	if !strings.Contains(stderr, "invalid regular expression") {
		t.Errorf("stderr = %q, want the syntax error reported", stderr)
	}
}

func TestSearchRequiresAQuery(t *testing.T) {
	h := searchFixture(t)
	if code, _, _ := h.run("search"); code == 0 {
		t.Error("a search with no query exited zero")
	}
}

// Search exists to answer whether a finding has already been recorded, and a
// decline is the most consequential form of having recorded one — so it is in
// scope without the caller having to know to ask.
func TestSearchAlwaysSeesDeclinedTasks(t *testing.T) {
	h := searchFixture(t)
	h.mustRun("add", "A declined finding about the cache")
	h.mustRun("set", "4", "declined", "--reason", "the call site is cold")

	var got []SearchResultView
	decode(t, h.mustRun("search", "cache", "--json"), &got)

	var sawDeclined, sawDone bool
	for _, r := range got {
		switch r.Task.ID {
		case 4:
			sawDeclined = true
			// The caller has to be able to tell a decline from a live task.
			if r.Task.Status != "declined" {
				t.Errorf("the declined result reports status %q", r.Task.Status)
			}
			if r.Task.Reason != "the call site is cold" {
				t.Errorf("the declined result reports reason %q", r.Task.Reason)
			}
		case 3:
			sawDone = true
		}
	}
	if !sawDeclined {
		t.Errorf("the declined task was not found without --all: %v", got)
	}
	// done is genuinely different: a fixed problem that reappears is a
	// regression, and new information.
	if sawDone {
		t.Error("a done task was returned without --all")
	}
}

func TestSearchStatusFilterIsTakenAtFaceValue(t *testing.T) {
	h := searchFixture(t)
	h.mustRun("add", "A declined finding about the cache")
	h.mustRun("set", "4", "declined", "--reason", "the call site is cold")

	var got []SearchResultView
	decode(t, h.mustRun("search", "cache", "--status", "todo", "--json"), &got)
	for _, r := range got {
		if r.Task.ID == 4 {
			t.Error("--status todo returned a declined task")
		}
	}

	decode(t, h.mustRun("search", "cache", "--status", "declined", "--json"), &got)
	if len(got) != 1 || got[0].Task.ID != 4 {
		t.Errorf("--status declined returned %d results, want only the declined one", len(got))
	}
}

// The reason describes the disposition, not the finding, so it is not content
// search matches on.
func TestSearchDoesNotMatchTheReason(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "A finding")
	h.mustRun("set", "1", "declined", "--reason", "superseded by the impending rewrite")

	var got []SearchResultView
	decode(t, h.mustRun("search", "impending", "--json"), &got)
	if len(got) != 0 {
		t.Errorf("the reason was searched: %v", got)
	}
}
