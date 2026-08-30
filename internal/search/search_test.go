package search

import (
	"reflect"
	"testing"

	"github.com/antonkolesov/backlog/internal/task"
)

func mk(id int, title, body, status string, tags ...string) *task.Task {
	return &task.Task{
		ID:     id,
		Title:  title,
		Body:   body,
		Status: status,
		Tags:   tags,
		Meta: task.Metadata{
			Created: "2026-08-30T20:59:51Z",
			Author:  task.AuthorAgent,
			Source: task.Source{
				Files:  []string{"internal/session/cache.go"},
				Branch: "feature/widget",
				Commit: "0badc0ffee1234567890",
			},
			Refs: []string{"openspec:add-widget"},
		},
	}
}

func ids(results []Result) []int {
	out := make([]int, 0, len(results))
	for _, r := range results {
		out = append(out, r.Task.ID)
	}
	return out
}

func search(t *testing.T, tasks []*task.Task, query string, useRegex bool) []Result {
	t.Helper()
	got, err := Search(tasks, query, useRegex)
	if err != nil {
		t.Fatalf("Search(%q): %v", query, err)
	}
	return got
}

func TestMatchesTitleDescriptionAndTags(t *testing.T) {
	tasks := []*task.Task{
		mk(1, "Race in session Cache", "", task.StatusTodo),
		mk(2, "Something else", "the cache is not locked\n", task.StatusTodo),
		mk(3, "Third", "", task.StatusTodo, "cache"),
		mk(4, "Unrelated", "nothing here\n", task.StatusTodo, "flake"),
	}
	// Substring matching is case-insensitive, so Cache matches cache.
	got := search(t, tasks, "cache", false)
	if want := []int{1, 2, 3}; !reflect.DeepEqual(ids(got), want) {
		t.Errorf("ids = %v, want %v", ids(got), want)
	}

	fields := map[int]string{}
	for _, r := range got {
		fields[r.Task.ID] = r.Matches[0].Field
	}
	want := map[int]string{1: FieldTitle, 2: FieldDescription, 3: FieldTag}
	if !reflect.DeepEqual(fields, want) {
		t.Errorf("matched fields = %v, want %v", fields, want)
	}
}

// Searching structural fields would make a query like "todo" return the whole
// backlog, and nobody searches for a commit hash by hand.
func TestStatusAndMetadataAreNotSearched(t *testing.T) {
	tasks := []*task.Task{
		mk(1, "A task", "", task.StatusTodo),
		mk(2, "Says todo in the title", "", task.StatusDoing),
	}

	got := search(t, tasks, "todo", false)
	if want := []int{2}; !reflect.DeepEqual(ids(got), want) {
		t.Errorf("querying a status value returned %v, want only the title match %v", ids(got), want)
	}

	for _, query := range []string{"0badc0ffee", "internal/session/cache.go", "feature/widget", "openspec:add-widget", "2026-08-30"} {
		if got := search(t, tasks, query, false); len(got) != 0 {
			t.Errorf("query %q matched metadata: %v", query, ids(got))
		}
	}
}

func TestRegexMatching(t *testing.T) {
	tasks := []*task.Task{
		mk(1, "HTTP 500 on login", "", task.StatusTodo),
		mk(2, "HTTP 404 on logout", "", task.StatusTodo),
		mk(3, "No status code here", "", task.StatusTodo),
	}
	got := search(t, tasks, `HTTP [45]0\d`, true)
	if want := []int{1, 2}; !reflect.DeepEqual(ids(got), want) {
		t.Errorf("ids = %v, want %v", ids(got), want)
	}

	// The same string without --regex is matched literally.
	if got := search(t, tasks, `HTTP [45]0\d`, false); len(got) != 0 {
		t.Errorf("a regular expression matched as a substring: %v", ids(got))
	}
}

func TestInvalidRegexIsReported(t *testing.T) {
	_, err := Search(nil, "HTTP [45", true)
	if err == nil {
		t.Fatal("expected an error for an invalid regular expression")
	}
	if got := err.Error(); got == "" || !contains(got, "invalid regular expression") {
		t.Errorf("error = %q, want it to name the syntax problem", got)
	}
}

func TestRankingPutsTitleMatchesFirst(t *testing.T) {
	tasks := []*task.Task{
		mk(1, "Nothing relevant", "the cache is cold\n", task.StatusTodo),
		mk(2, "Cache eviction is wrong", "", task.StatusTodo),
		mk(3, "Also about the cache", "", task.StatusTodo),
		mk(4, "Another one", "", task.StatusTodo, "cache"),
	}
	got := search(t, tasks, "cache", false)
	// Title matches first, then the rest, ascending by identifier within each.
	if want := []int{2, 3, 1, 4}; !reflect.DeepEqual(ids(got), want) {
		t.Errorf("ids = %v, want %v", ids(got), want)
	}
}

func TestRepeatedSearchesAgree(t *testing.T) {
	tasks := []*task.Task{
		mk(3, "Cache one", "", task.StatusTodo),
		mk(1, "Cache two", "", task.StatusTodo),
		mk(2, "Cache three", "", task.StatusTodo),
	}
	first := ids(search(t, tasks, "cache", false))
	for i := 0; i < 5; i++ {
		if got := ids(search(t, tasks, "cache", false)); !reflect.DeepEqual(got, first) {
			t.Fatalf("run %d returned %v, want %v", i, got, first)
		}
	}
	if want := []int{1, 2, 3}; !reflect.DeepEqual(first, want) {
		t.Errorf("ids = %v, want ascending identifiers %v", first, want)
	}
}

func TestMatchCarriesTextAndContext(t *testing.T) {
	long := "A long description that goes on for a while before it finally mentions the cache and then keeps going for a while afterwards too.\n"
	got := search(t, []*task.Task{mk(1, "Unrelated title", long, task.StatusTodo)}, "cache", false)
	if len(got) != 1 || len(got[0].Matches) != 1 {
		t.Fatalf("got %d results", len(got))
	}
	m := got[0].Matches[0]
	if m.Text != "cache" {
		t.Errorf("Text = %q, want the matched text", m.Text)
	}
	if !contains(m.Context, "cache") {
		t.Errorf("Context = %q, want it to contain the match", m.Context)
	}
	if len(m.Context) >= len(long) {
		t.Errorf("Context = %q, want only the surrounding text", m.Context)
	}
}

func TestNoMatchesIsNotAnError(t *testing.T) {
	got, err := Search([]*task.Task{mk(1, "Something", "", task.StatusTodo)}, "nothing like this", false)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d results, want none", len(got))
	}
}

func TestNoFuzzyMatching(t *testing.T) {
	tasks := []*task.Task{mk(1, "Race in session cache", "", task.StatusTodo)}
	if got := search(t, tasks, "cahce", false); len(got) != 0 {
		t.Errorf("a near miss matched: %v", ids(got))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
