package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/antonkolesov/backlog/internal/task"
)

// stringList collects a repeatable flag such as --tag.
type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(v string) error {
	*l = append(*l, v)
	return nil
}

// TaskView is the JSON shape of a task. It is the interface agents and scripts
// consume, so it is written out explicitly rather than derived from the
// internal model, and metadata.schema exists so it can be moved deliberately.
type TaskView struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Reason      string   `json:"reason"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	Metadata    MetaView `json:"metadata"`
	File        string   `json:"file"`
}

// MetaView is the JSON shape of the tool-owned metadata block.
type MetaView struct {
	Schema  int        `json:"schema"`
	Created string     `json:"created"`
	Author  string     `json:"author"`
	Source  SourceView `json:"source"`
	Refs    []string   `json:"refs"`
}

// SourceView is the JSON shape of where a finding was observed.
type SourceView struct {
	Files  []string `json:"files"`
	Branch string   `json:"branch"`
	Commit string   `json:"commit"`
}

func view(t *task.Task) TaskView {
	return TaskView{
		ID:       t.ID,
		Title:    t.Title,
		Status:   t.Status,
		Priority: t.Priority,
		// Always present, empty for a task that is not declined, so the shape
		// of the JSON does not vary with status.
		Reason:      t.Reason,
		Tags:        nonNil(t.Tags),
		Description: strings.TrimRight(t.Body, "\n"),
		Metadata: MetaView{
			Schema:  t.Meta.Schema,
			Created: t.Meta.Created,
			Author:  t.Meta.Author,
			Source: SourceView{
				Files:  nonNil(t.Meta.Source.Files),
				Branch: t.Meta.Source.Branch,
				Commit: t.Meta.Source.Commit,
			},
			Refs: nonNil(t.Meta.Refs),
		},
		File: filepath.Base(t.Path),
	}
}

func views(tasks []*task.Task) []TaskView {
	out := make([]TaskView, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, view(t))
	}
	return out
}

// nonNil keeps JSON lists as [] rather than null, so a consumer can iterate
// without a nil check.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// writeJSON emits v as indented JSON on the data stream. Diagnostics never go
// here: a command that fails in JSON mode writes to stderr and leaves stdout
// empty.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// writeTaskLines prints tasks grouped by status in lifecycle order — what is
// waiting, what is in flight, what was finished, what was declined — so that a
// listing reads in the direction a task travels.
func writeTaskLines(w io.Writer, tasks []*task.Task) {
	byStatus := map[string][]*task.Task{}
	var unknown []*task.Task
	for _, t := range tasks {
		if task.ValidStatus(t.Status) {
			byStatus[t.Status] = append(byStatus[t.Status], t)
		} else {
			unknown = append(unknown, t)
		}
	}
	order := []string{task.StatusTodo, task.StatusDoing, task.StatusDone, task.StatusDeclined}
	first := true
	for _, status := range order {
		group := byStatus[status]
		if len(group) == 0 {
			continue
		}
		if !first {
			fmt.Fprintln(w)
		}
		first = false
		fmt.Fprintf(w, "%s (%d)\n", status, len(group))
		for _, t := range group {
			fmt.Fprintln(w, "  "+taskLine(t))
		}
	}
	if len(unknown) > 0 {
		if !first {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "invalid status (%d)\n", len(unknown))
		for _, t := range unknown {
			fmt.Fprintf(w, "  %03d  %s  [status: %s]\n", t.ID, t.Title, t.Status)
		}
	}
}

func taskLine(t *task.Task) string {
	line := fmt.Sprintf("%03d  %-6s  %s", t.ID, t.Priority, t.Title)
	if len(t.Tags) > 0 {
		line += "  [" + strings.Join(t.Tags, ", ") + "]"
	}
	return line
}
