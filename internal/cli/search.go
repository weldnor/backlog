package cli

import (
	"fmt"
	"strings"

	"github.com/weldnor/backlog/internal/search"
)

// SearchResultView is the JSON shape of one search result: the full task, plus
// where the query matched so that a caller can show the reason without
// re-running the match itself.
type SearchResultView struct {
	Task    TaskView       `json:"task"`
	Matches []search.Match `json:"matches"`
}

func runSearch(env Env, args []string) error {
	fs := newFlagSet("search")
	var (
		useRegex = fs.Bool("regex", false, "interpret the query as a regular expression")
		asJSON   = fs.Bool("json", false, "print the results as JSON")
	)
	var sc scope
	sc.register(fs)
	sc.searchScope()
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	query := strings.Join(fs.Args(), " ")
	if strings.TrimSpace(query) == "" {
		return usagef("a query is required")
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}
	tasks, err := sc.apply(st)
	if err != nil {
		return err
	}
	results, err := search.Search(tasks, query, *useRegex)
	if err != nil {
		return err
	}

	if *asJSON {
		out := make([]SearchResultView, 0, len(results))
		for _, r := range results {
			out = append(out, SearchResultView{Task: view(r.Task), Matches: r.Matches})
		}
		return writeJSON(env.Stdout, out)
	}

	// Finding nothing is an answer, not a failure: it is how an agent learns
	// that a finding has not been recorded yet.
	if len(results) == 0 {
		fmt.Fprintln(env.Stdout, "no tasks matched")
		return nil
	}
	for _, r := range results {
		fmt.Fprintf(env.Stdout, "%03d  %-8s  %s\n", r.Task.ID, r.Task.Status, r.Task.Title)
		for _, m := range r.Matches {
			fmt.Fprintf(env.Stdout, "       %s: %s\n", m.Field, m.Context)
		}
	}
	return nil
}
