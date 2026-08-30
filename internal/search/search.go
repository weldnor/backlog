// Package search finds tasks by content rather than by identifier.
//
// Matching is deliberately mechanical — case-insensitive substring, or a
// regular expression when asked. There is no fuzzy matching: the caller is
// usually an agent deciding whether a finding has already been recorded, and
// that decision needs results it can reproduce and reason about. Judging
// whether two descriptions mean the same thing is the model's job, not the
// binary's.
package search

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/antonkolesov/backlog/internal/task"
)

// The fields a query is matched against. Status, timestamps and everything
// under metadata are deliberately excluded: searching them would make a query
// like "todo" return the whole backlog, and a commit hash is not something
// anyone searches for by hand.
const (
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldTag         = "tag"
)

// Match is one place a query was found.
type Match struct {
	Field string `json:"field"`
	// Text is the matched text exactly as it appears in the task.
	Text string `json:"text"`
	// Context is the matched text with enough of its surroundings to be
	// readable in a terminal.
	Context string `json:"context"`
}

// Result is a matching task together with every place the query was found.
type Result struct {
	Task    *task.Task
	Matches []Match
}

// TitleMatch reports whether the query was found in the title, which is what
// puts a result in the first rank.
func (r Result) TitleMatch() bool {
	for _, m := range r.Matches {
		if m.Field == FieldTitle {
			return true
		}
	}
	return false
}

// maxMatchesPerField bounds how much context a single long description can
// contribute, so that one task cannot flood the output.
const maxMatchesPerField = 5

// Search returns the matching tasks, ordered with title matches first and by
// ascending identifier within each rank. The order is a pure function of the
// query and the tasks, so repeated runs agree.
func Search(tasks []*task.Task, query string, useRegex bool) ([]Result, error) {
	m, err := newMatcher(query, useRegex)
	if err != nil {
		return nil, err
	}

	var results []Result
	for _, t := range tasks {
		var matches []Match
		matches = append(matches, m.find(FieldTitle, t.Title)...)
		matches = append(matches, m.find(FieldDescription, t.Body)...)
		for _, tag := range t.Tags {
			matches = append(matches, m.find(FieldTag, tag)...)
		}
		if len(matches) > 0 {
			results = append(results, Result{Task: t, Matches: matches})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		ti, tj := results[i].TitleMatch(), results[j].TitleMatch()
		if ti != tj {
			return ti
		}
		return results[i].Task.ID < results[j].Task.ID
	})
	return results, nil
}

type matcher struct {
	re *regexp.Regexp
}

// newMatcher compiles the query. Substring queries are turned into a quoted
// regular expression so that both modes share one code path and one definition
// of "where did it match".
func newMatcher(query string, useRegex bool) (*matcher, error) {
	if query == "" {
		return nil, fmt.Errorf("a query is required")
	}
	pattern := regexp.QuoteMeta(query)
	if useRegex {
		pattern = query
	}
	// Search is case-insensitive in both modes; a regular expression that needs
	// case sensitivity can turn it back off with (?-i).
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		if useRegex {
			return nil, fmt.Errorf("invalid regular expression %q: %w", query, err)
		}
		return nil, err
	}
	return &matcher{re: re}, nil
}

func (m *matcher) find(field, text string) []Match {
	if text == "" {
		return nil
	}
	locs := m.re.FindAllStringIndex(text, maxMatchesPerField)
	if len(locs) == 0 {
		return nil
	}
	out := make([]Match, 0, len(locs))
	for _, loc := range locs {
		out = append(out, Match{
			Field:   field,
			Text:    text[loc[0]:loc[1]],
			Context: contextAround(text, loc[0], loc[1]),
		})
	}
	return out
}

// contextWindow is how much text is shown on either side of a match.
const contextWindow = 40

// contextAround returns the match with its surroundings, collapsed onto a
// single line so that a multi-paragraph description stays readable in a list.
func contextAround(text string, start, end int) string {
	from := start - contextWindow
	if from < 0 {
		from = 0
	}
	to := end + contextWindow
	if to > len(text) {
		to = len(text)
	}
	// Do not cut a rune in half.
	for from > 0 && !isRuneStart(text[from]) {
		from--
	}
	for to < len(text) && !isRuneStart(text[to]) {
		to++
	}

	snippet := collapse(text[from:to])
	if from > 0 {
		snippet = "…" + snippet
	}
	if to < len(text) {
		snippet += "…"
	}
	return snippet
}

func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }

func collapse(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\n", " ")), " ")
}
