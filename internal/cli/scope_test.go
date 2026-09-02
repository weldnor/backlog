package cli

import (
	"testing"

	"github.com/weldnor/backlog/internal/task"
)

func TestScopeSelectedStatuses(t *testing.T) {
	cases := []struct {
		name string
		sc   scope
		want []string
	}{
		{
			name: "no subcommand covers every status",
			sc:   scope{},
			want: []string{task.StatusNew, task.StatusTodo, task.StatusDoing, task.StatusDone, task.StatusDeclined},
		},
		{
			name: "a status subcommand narrows to that one status",
			sc:   scope{status: task.StatusDone},
			want: []string{task.StatusDone},
		},
		{
			name: "the subcommand may be declined",
			sc:   scope{status: task.StatusDeclined},
			want: []string{task.StatusDeclined},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.sc.selected()
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
