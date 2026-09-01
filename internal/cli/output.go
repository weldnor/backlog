package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/weldnor/backlog/internal/task"
	"github.com/weldnor/backlog/internal/taskview"
)

// stringList collects a repeatable flag such as --tag.
type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(v string) error {
	*l = append(*l, v)
	return nil
}

// TaskView is the JSON shape of a task, shared with internal/browse so the
// two never disagree about what one looks like. See internal/taskview.
type TaskView = taskview.TaskView

// MetaView is the JSON shape of the tool-owned metadata block.
type MetaView = taskview.MetaView

// SourceView is the JSON shape of where a finding was observed.
type SourceView = taskview.SourceView

func view(t *task.Task) TaskView { return taskview.View(t) }

func views(tasks []*task.Task) []TaskView { return taskview.Views(tasks) }

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
