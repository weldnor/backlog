package cli

import (
	"testing"

	"github.com/antonkolesov/backlog/internal/task"
)

// selectedStatuses is what list and search agree, or deliberately disagree, on.
func selectedStatuses(t *testing.T, sc *scope) map[string]bool {
	t.Helper()
	got, err := sc.selected()
	if err != nil {
		t.Fatalf("selected: %v", err)
	}
	return got
}

func TestScopeSelectedStatuses(t *testing.T) {
	cases := []struct {
		name string
		sc   scope
		want []string
	}{
		{
			name: "list by default leaves the whole archive out",
			sc:   scope{},
			want: []string{task.StatusTodo, task.StatusDoing},
		},
		{
			name: "search by default still sees declined tasks",
			sc:   scope{alwaysDeclined: true},
			want: []string{task.StatusTodo, task.StatusDoing, task.StatusDeclined},
		},
		{
			name: "--all on list takes in the whole archive",
			sc:   scope{all: true},
			want: []string{task.StatusTodo, task.StatusDoing, task.StatusDone, task.StatusDeclined},
		},
		{
			name: "--all on search adds only done, which it did not already have",
			sc:   scope{all: true, alwaysDeclined: true},
			want: []string{task.StatusTodo, task.StatusDoing, task.StatusDone, task.StatusDeclined},
		},
		{
			name: "an explicit status is taken at face value on a search",
			sc:   scope{alwaysDeclined: true, statuses: stringList{task.StatusTodo}},
			want: []string{task.StatusTodo},
		},
		{
			name: "an explicit status may name declined",
			sc:   scope{statuses: stringList{task.StatusDeclined}},
			want: []string{task.StatusDeclined},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sc := c.sc
			got := selectedStatuses(t, &sc)
			if len(got) != len(c.want) {
				t.Fatalf("selected %v, want %v", got, c.want)
			}
			for _, s := range c.want {
				if !got[s] {
					t.Errorf("%s is not in scope: %v", s, got)
				}
			}
		})
	}
}

func TestScopeRejectsAnUnknownStatus(t *testing.T) {
	sc := scope{statuses: stringList{"wontfix"}}
	if _, err := sc.selected(); err == nil {
		t.Error("an unknown status was accepted")
	}
}
