package cli

import (
	"flag"
	"strings"

	"github.com/antonkolesov/backlog/internal/store"
	"github.com/antonkolesov/backlog/internal/task"
)

// scope holds the selection flags list and search share, so that the two
// commands answer questions about the same set of tasks in the same way.
type scope struct {
	all      bool
	statuses stringList
	tags     stringList
	// priorities is bound by list alone. Search answers whether a finding has
	// already been recorded, which is a question about content: letting a
	// severity filter narrow it would let a duplicate hide behind the filter.
	priorities stringList
	// alwaysDeclined is set by search and not by list. A decline is the most
	// consequential form of having recorded a finding, so letting the archive
	// scope hide it would let a duplicate hide behind the scope in exactly the
	// way a severity filter would. done is not treated this way: a fixed
	// problem that reappears is a regression, and genuinely new information.
	alwaysDeclined bool
}

func (s *scope) register(fs *flag.FlagSet) {
	fs.BoolVar(&s.all, "all", false, "include tasks in the archive")
	fs.Var(&s.statuses, "status", "only tasks with this status (repeatable)")
	fs.Var(&s.tags, "tag", "only tasks carrying this tag (repeatable; all must match)")
}

// registerPriority binds the priority filter. It is separate from register
// because only list offers it.
func (s *scope) registerPriority(fs *flag.FlagSet) {
	fs.Var(&s.priorities, "priority", "only tasks with this priority (repeatable; any may match)")
}

// searchScope marks the scope as search's, which always sees declined tasks.
// It is a separate call for the same reason registerPriority is: the two
// commands share this type, so an asymmetry between them has to be asked for.
func (s *scope) searchScope() { s.alwaysDeclined = true }

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

// selected returns the statuses in scope. An explicit --status is taken at
// face value, including when it names done or declined; otherwise the archive
// is left out unless --all was given, except that search always sees declined
// tasks.
func (s *scope) selected() (map[string]bool, error) {
	out := map[string]bool{}
	if len(s.statuses) > 0 {
		for _, v := range s.statuses {
			v = strings.ToLower(strings.TrimSpace(v))
			if !task.ValidStatus(v) {
				return nil, usagef("unknown status %q, expected one of %s", v, strings.Join(task.Statuses, ", "))
			}
			out[v] = true
		}
		return out, nil
	}
	out[task.StatusTodo] = true
	out[task.StatusDoing] = true
	if s.all {
		out[task.StatusDone] = true
	}
	if s.alwaysDeclined || s.all {
		out[task.StatusDeclined] = true
	}
	return out, nil
}

// apply reads the store and returns the matching tasks in ascending identifier
// order, which is the deterministic order every listing uses.
func (s *scope) apply(st *store.Store) ([]*task.Task, error) {
	statuses, err := s.selected()
	if err != nil {
		return nil, err
	}
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
