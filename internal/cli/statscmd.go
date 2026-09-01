package cli

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/weldnor/backlog/internal/task"
)

// StatsView is the JSON shape of `backlog stats`.
type StatsView struct {
	Total      int            `json:"total"`
	ByStatus   map[string]int `json:"byStatus"`
	ByPriority map[string]int `json:"byPriority"`
	ByTag      map[string]int `json:"byTag"`
	// OpenAvgAgeDays is the average age, in days, of every todo or doing task
	// that carries a created timestamp. It is omitted (nil) rather than 0 when
	// there is nothing open to average, so JSON does not claim a false zero.
	OpenAvgAgeDays *float64 `json:"openAvgAgeDays"`
}

// runStats reports the shape of the backlog: how many tasks sit in each
// status and priority, which tags are in use, and how long what is still open
// has been waiting. It answers "what does this backlog look like" in one call
// instead of piping `list --json` through a script.
func runStats(env Env, args []string) error {
	fs := newFlagSet("stats")
	asJSON := fs.Bool("json", false, "print the summary as JSON")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return usagef("unexpected argument %q", fs.Arg(0))
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}
	tasks, err := st.Tasks()
	if err != nil {
		return err
	}

	v := buildStats(tasks, time.Now())

	if *asJSON {
		return writeJSON(env.Stdout, v)
	}
	writeStats(env.Stdout, v)
	return nil
}

func buildStats(tasks []*task.Task, now time.Time) StatsView {
	v := StatsView{
		Total:      len(tasks),
		ByStatus:   map[string]int{},
		ByPriority: map[string]int{},
		ByTag:      map[string]int{},
	}
	var ageSum time.Duration
	var ageCount int
	for _, t := range tasks {
		v.ByStatus[t.Status]++
		v.ByPriority[t.Priority]++
		for _, tag := range t.Tags {
			v.ByTag[tag]++
		}
		// Age is meaningful only for a task still waiting on someone: a
		// finished one has no "how long has this sat here" question left to
		// answer.
		if task.IsTerminal(t.Status) || t.Meta.Created == "" {
			continue
		}
		created, err := time.Parse(time.RFC3339, t.Meta.Created)
		if err != nil {
			continue
		}
		ageSum += now.Sub(created)
		ageCount++
	}
	if ageCount > 0 {
		days := ageSum.Hours() / 24 / float64(ageCount)
		v.OpenAvgAgeDays = &days
	}
	return v
}

func writeStats(w io.Writer, v StatsView) {
	fmt.Fprintf(w, "total %d\n", v.Total)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "by status")
	for _, status := range task.Statuses {
		fmt.Fprintf(w, "  %-10s %d\n", status, v.ByStatus[status])
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "by priority")
	for _, p := range task.Priorities {
		fmt.Fprintf(w, "  %-10s %d\n", p, v.ByPriority[p])
	}

	if len(v.ByTag) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "by tag")
		tags := make([]string, 0, len(v.ByTag))
		for tag := range v.ByTag {
			tags = append(tags, tag)
		}
		// Most-used first; ties broken alphabetically for a stable order.
		sort.Slice(tags, func(i, j int) bool {
			if v.ByTag[tags[i]] != v.ByTag[tags[j]] {
				return v.ByTag[tags[i]] > v.ByTag[tags[j]]
			}
			return tags[i] < tags[j]
		})
		for _, tag := range tags {
			fmt.Fprintf(w, "  %-10s %d\n", tag, v.ByTag[tag])
		}
	}

	fmt.Fprintln(w)
	if v.OpenAvgAgeDays == nil {
		fmt.Fprintln(w, "no open tasks")
	} else {
		fmt.Fprintf(w, "open tasks average age: %.1f day(s)\n", *v.OpenAvgAgeDays)
	}
}
