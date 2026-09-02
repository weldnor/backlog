package cli

import (
	"flag"
	"strings"

	"github.com/weldnor/backlog/internal/store"
	"github.com/weldnor/backlog/internal/task"
)

// scope holds the selection list and search share, so that the two commands
// answer questions about the same set of tasks in the same way.
type scope struct {
	// status is a single status a list subcommand narrowed to, or "" for
	// every status. Search never sets it: a finding that has already been
	// recorded must be found whatever status its task is in.
	status string
	tags   stringList
	// priorities is bound by list alone. Search answers whether a finding has
	// already been recorded, which is a question about content: letting a
	// severity filter narrow it would let a duplicate hide behind the filter.
	priorities stringList
}

func (s *scope) register(fs *flag.FlagSet) {
	fs.Var(&s.tags, "tag", "only tasks carrying this tag (repeatable; all must match)")
}

// registerPriority binds the priority filter. It is separate from register
// because only list offers it.
func (s *scope) registerPriority(fs *flag.FlagSet) {
	fs.Var(&s.priorities, "priority", "only tasks with this priority (repeatable; any may match)")
}

// selectedPriorities returns the priorities in scope, or nil when the filter
// was not used and every priority is in scope.
func (s *scope) selectedPriorities() (map[string]bool, error) {
	if len(s.priorities) == 0 {
		return nil, nil
	}
	out := map[string]bool{}
	for _, v := range s.priorities {
		v = strings.ToLower(strings.TrimSpace(v))
		if !task.ValidPriority(v) {
			return nil, usagef("unknown priority %q, expected one of %s", v, strings.Join(task.Priorities, ", "))
		}
		out[v] = true
	}
	return out, nil
}

// selected returns the statuses in scope: the one a list subcommand narrowed
// to, or all five.
func (s *scope) selected() map[string]bool {
	out := map[string]bool{}
	if s.status != "" {
		out[s.status] = true
		return out
	}
	for _, v := range task.Statuses {
		out[v] = true
	}
	return out
}

// apply reads the store and returns the matching tasks in ascending identifier
// order, which is the deterministic order every listing uses.
func (s *scope) apply(st *store.Store) ([]*task.Task, error) {
	statuses := s.selected()
	priorities, err := s.selectedPriorities()
	if err != nil {
		return nil, err
	}
	all, err := st.Tasks()
	if err != nil {
		return nil, err
	}
	var out []*task.Task
	for _, t := range all {
		if !statuses[t.Status] {
			continue
		}
		if priorities != nil && !priorities[t.Priority] {
			continue
		}
		if !hasAllTags(t, s.tags) {
			continue
		}
		out = append(out, t)
	}
	task.SortByID(out)
	return out, nil
}

func hasAllTags(t *task.Task, tags []string) bool {
	for _, tag := range tags {
		if !t.HasTag(tag) {
			return false
		}
	}
	return true
}

func openStore(env Env) (*store.Store, error) {
	return store.Discover(env.Dir)
}
