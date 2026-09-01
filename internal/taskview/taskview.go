// Package taskview defines the JSON shape of a task shared by every consumer
// that reports one: the CLI's `--json` output and the browse package's HTTP
// API. Both live in this one place so that they cannot quietly drift apart —
// a field added here reaches every consumer at once.
package taskview

import (
	"path/filepath"
	"strings"

	"github.com/weldnor/backlog/internal/task"
)

// TaskView is the JSON shape of a task. It is the interface agents, scripts
// and the web UI consume, so it is written out explicitly rather than derived
// from the internal model, and metadata.schema exists so it can be moved
// deliberately.
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

// View converts a task to its JSON shape.
func View(t *task.Task) TaskView {
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

// Views converts a list of tasks to their JSON shape, in the order given.
func Views(tasks []*task.Task) []TaskView {
	out := make([]TaskView, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, View(t))
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
