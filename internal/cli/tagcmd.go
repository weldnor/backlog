package cli

import (
	"fmt"
	"strings"

	"github.com/weldnor/backlog/internal/store"
	"github.com/weldnor/backlog/internal/task"
)

// runTag dispatches the two tag-maintenance subcommands. A tag lives only as
// text repeated on every task that carries it, so a typo made once at capture
// time is otherwise fixed one task at a time by hand; `tag rm` and
// `tag rename` do it across the whole backlog in one call.
func runTag(env Env, args []string) error {
	if len(args) == 0 {
		return usagef("usage: backlog tag rm <name> | backlog tag rename <old> <new>")
	}
	switch args[0] {
	case "-h", "--help":
		return usagef("usage: backlog tag rm <name> | backlog tag rename <old> <new>")
	case "rm":
		return runTagRm(env, args[1:])
	case "rename":
		return runTagRename(env, args[1:])
	default:
		return usagef("unknown subcommand %q, expected rm or rename", args[0])
	}
}

func runTagRm(env Env, args []string) error {
	fs := newFlagSet("tag rm")
	asJSON := fs.Bool("json", false, "print the changed tasks as JSON")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return usagef("a tag name is required")
	}
	if len(rest) > 1 {
		return usagef("unexpected argument %q", rest[1])
	}
	name := rest[0]
	if strings.TrimSpace(name) == "" {
		return usagef("a tag name is required")
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}
	changed, err := updateTags(st, func(t *task.Task) bool {
		if !t.HasTag(name) {
			return false
		}
		t.Tags = withoutTag(t.Tags, name)
		return true
	})
	if err != nil {
		return err
	}

	if *asJSON {
		return writeJSON(env.Stdout, views(changed))
	}
	fmt.Fprintf(env.Stdout, "removed tag %q from %d task(s)\n", name, len(changed))
	for _, t := range changed {
		fmt.Fprintln(env.Stdout, "  "+taskLine(t))
	}
	return nil
}

func runTagRename(env Env, args []string) error {
	fs := newFlagSet("tag rename")
	asJSON := fs.Bool("json", false, "print the changed tasks as JSON")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 2 {
		return usagef("an old and a new tag name are required")
	}
	if len(rest) > 2 {
		return usagef("unexpected argument %q", rest[2])
	}
	oldName, newName := rest[0], rest[1]
	if strings.TrimSpace(oldName) == "" || strings.TrimSpace(newName) == "" {
		return usagef("a tag name may not be empty")
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}
	changed, err := updateTags(st, func(t *task.Task) bool {
		if !t.HasTag(oldName) {
			return false
		}
		// NormalizeTags drops the duplicate this creates when a task already
		// carries newName under a different case, or both spellings at once.
		t.Tags = task.NormalizeTags(append(withoutTag(t.Tags, oldName), newName))
		return true
	})
	if err != nil {
		return err
	}

	if *asJSON {
		return writeJSON(env.Stdout, views(changed))
	}
	fmt.Fprintf(env.Stdout, "renamed tag %q to %q on %d task(s)\n", oldName, newName, len(changed))
	for _, t := range changed {
		fmt.Fprintln(env.Stdout, "  "+taskLine(t))
	}
	return nil
}

// updateTags applies mutate to every task in the store, saving and collecting
// those it reports changing. Tasks are visited in ascending identifier order,
// the same order every other listing uses.
func updateTags(st *store.Store, mutate func(*task.Task) bool) ([]*task.Task, error) {
	tasks, err := st.Tasks()
	if err != nil {
		return nil, err
	}
	task.SortByID(tasks)
	var changed []*task.Task
	for _, t := range tasks {
		if !mutate(t) {
			continue
		}
		if err := st.Save(t); err != nil {
			return nil, err
		}
		changed = append(changed, t)
	}
	return changed, nil
}

// withoutTag returns tags with every entry equal to name, case-insensitively,
// removed.
func withoutTag(tags []string, name string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if strings.EqualFold(t, name) {
			continue
		}
		out = append(out, t)
	}
	return out
}
