// Package task defines the on-disk representation of a single backlog task:
// its model, the frontmatter schema, and the rules for reading and writing it.
//
// Both a person and an agent edit these files, and the agent will sometimes
// edit them without going through the CLI. Reading is therefore deliberately
// tolerant: deviations are reported as issues rather than refused, so that
// `backlog validate` — not a crash in an unrelated command — is what tells the
// author something is wrong.
package task

import (
	"sort"

	"gopkg.in/yaml.v3"
)

// SchemaVersion is the format version stamped into metadata.schema. It exists
// to give a future format migration a foothold.
const SchemaVersion = 1

// The four permitted statuses. StatusDone and StatusDeclined are terminal;
// the other two are not.
const (
	StatusTodo  = "todo"
	StatusDoing = "doing"
	StatusDone  = "done"
	// StatusDeclined records a finding that was recorded correctly and that a
	// reviewer decided not to act on. It is terminal like StatusDone, and it
	// is the disposition that keeps the decision findable instead of deleting
	// it along with its reasoning.
	StatusDeclined = "declined"
)

// Statuses lists the permitted status values in lifecycle order.
var Statuses = []string{StatusTodo, StatusDoing, StatusDone, StatusDeclined}

// ValidStatus reports whether s is one of the permitted status values.
func ValidStatus(s string) bool {
	for _, v := range Statuses {
		if v == s {
			return true
		}
	}
	return false
}

// IsTerminal reports whether s is a status a task has finished in — either
// acted on or declined. It classifies status for grouping a listing and for
// the decline-reason rules.
func IsTerminal(s string) bool {
	return s == StatusDone || s == StatusDeclined
}

// The three permitted priorities. Priority records how bad a finding is — the
// consequence of leaving it unfixed — and not when it will be worked on, which
// is a decision for whatever planning system the project already uses.
const (
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// Priorities lists the permitted priority values from most to least severe.
var Priorities = []string{PriorityHigh, PriorityMedium, PriorityLow}

// DefaultPriority is the priority a task is given when none was supplied, and
// the one a file that declares no priority is read as.
const DefaultPriority = PriorityMedium

// ValidPriority reports whether p is one of the permitted priority values.
func ValidPriority(p string) bool {
	for _, v := range Priorities {
		if v == p {
			return true
		}
	}
	return false
}

// PriorityRank orders priorities from most to least severe, for sorting. A
// value outside the permitted set ranks after low: it is reported as an error
// when the file is read, and a listing still needs a total order.
func PriorityRank(p string) int {
	for i, v := range Priorities {
		if v == p {
			return i
		}
	}
	return len(Priorities)
}

// Authors recorded in metadata.author.
const (
	AuthorAgent = "agent"
	AuthorHuman = "human"
)

// Source records where a finding was observed. Every field is optional: a
// finding not tied to a code location has no files, and a project that is not
// a git repository has no branch or commit.
type Source struct {
	Files  []string
	Branch string
	Commit string
}

// Empty reports whether the source carries nothing worth writing.
func (s Source) Empty() bool {
	return len(s.Files) == 0 && s.Branch == "" && s.Commit == ""
}

// Metadata holds the tool-owned frontmatter block. Its key set is closed: an
// unrecognised key under `metadata` is a typo, not an experiment, so it is
// reported as an error.
type Metadata struct {
	Schema  int
	Created string // RFC 3339, written once and never updated
	Author  string
	Source  Source
	Refs    []string
}

// MetadataKeys is the closed set of keys permitted under `metadata`.
var MetadataKeys = []string{"schema", "created", "author", "source", "refs"}

// SourceKeys is the closed set of keys permitted under `metadata.source`.
var SourceKeys = []string{"files", "branch", "commit"}

// TopLevelKeys are the frontmatter keys the CLI understands. Unlike the
// metadata block the top level is author-owned, so a key outside this set is
// preserved and reported only as a warning.
var TopLevelKeys = []string{"id", "title", "status", "priority", "reason", "tags", "metadata"}

// Task is one backlog entry.
type Task struct {
	ID       int
	Title    string
	Status   string
	Priority string
	// Reason is the prose explanation of a decline. It is present exactly
	// when Status is StatusDeclined: a decline nobody can audit is the state
	// the field exists to prevent, and a reason on a live task describes a
	// state the task is no longer in.
	Reason string
	Tags   []string
	Meta   Metadata

	// Body is the markdown description. It may be empty.
	Body string

	// Path is the absolute path the task was read from, empty for a task that
	// has not been written yet.
	Path string

	// Issues collects the deviations found while reading the file. Commands
	// that only need the task ignore them; validate is what reports them.
	Issues []Issue

	// hasMeta records that the file carried a metadata block, so that a write
	// neither adds one to a file without it nor drops an empty one.
	hasMeta bool

	// front is the parsed frontmatter mapping, retained so that top-level
	// fields the CLI does not understand survive a write unchanged.
	front *yaml.Node
}

// Severity classifies an issue found while reading or checking a task.
type Severity string

const (
	// SeverityError marks a backlog that cannot be read or operated on
	// reliably.
	SeverityError Severity = "error"
	// SeverityWarning marks a backlog that is readable but violates a
	// convention.
	SeverityWarning Severity = "warning"
)

// Issue is a single deviation found while reading a task file. Parsing
// collects issues instead of failing so that a hand-edited file stays usable.
type Issue struct {
	Severity Severity
	Message  string
	// Repairable reports whether `validate --fix` can correct this
	// unambiguously.
	Repairable bool
}

// NormalizeTags removes empty entries and duplicates while preserving the
// order the author wrote.
func NormalizeTags(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// HasTag reports whether the task carries tag, compared case-insensitively so
// that filters behave the way a person expects.
func (t *Task) HasTag(tag string) bool {
	for _, v := range t.Tags {
		if equalFold(v, tag) {
			return true
		}
	}
	return false
}

// SortByID orders tasks by ascending identifier, which is the deterministic
// order every listing uses.
func SortByID(tasks []*Task) {
	sort.SliceStable(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
}

// SortByPriorityThenID orders tasks from most to least severe, breaking ties by
// ascending identifier. This is the order `list` reports. Search is left out of
// it deliberately: it keeps its own ranking by where the query matched.
func SortByPriorityThenID(tasks []*Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		ri, rj := PriorityRank(tasks[i].Priority), PriorityRank(tasks[j].Priority)
		if ri != rj {
			return ri < rj
		}
		return tasks[i].ID < tasks[j].ID
	})
}
