package cli

import (
	"fmt"
	"strings"

	"github.com/weldnor/backlog/internal/hooks"
)

// runAssign changes who a task is assigned to. It is deliberately its own
// command rather than a flag on `set` or `edit`: assignment is neither
// workflow state (status, priority) nor prose content (title, description,
// tags) — it is who is doing the work — and it fires its own pre/post hooks
// so a project can wire up notifications without also catching every status
// or content change.
func runAssign(env Env, args []string) error {
	fs := newFlagSet("assign")
	clear := fs.Bool("clear", false, "unassign the task")
	asJSON := fs.Bool("json", false, "print the updated task as JSON")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return usagef("usage: backlog assign <id> <assignee> | backlog assign <id> --clear")
	}
	id, err := parseID(rest[0])
	if err != nil {
		return err
	}
	rest = rest[1:]

	var assignee string
	if *clear {
		if len(rest) > 0 {
			return usagef("unexpected argument %q; --clear takes no assignee", rest[0])
		}
	} else {
		if len(rest) == 0 {
			return usagef("an assignee is required, or pass --clear to unassign")
		}
		if len(rest) > 1 {
			return usagef("unexpected argument %q", rest[1])
		}
		assignee = strings.TrimSpace(rest[0])
		if assignee == "" {
			return usagef("an assignee may not be empty")
		}
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}
	t, err := st.Find(id)
	if err != nil {
		return err
	}

	prevAssignee := t.Assignee
	if err := hooks.RunPre(env.Stderr, st.Root, st.Project, hooks.PreAssign, t, map[string]string{
		"BACKLOG_NEW_ASSIGNEE": assignee,
	}); err != nil {
		return err
	}
	t.Assignee = assignee
	if err := st.Save(t); err != nil {
		return err
	}
	hooks.Run(env.Stderr, st.Root, st.Project, hooks.PostAssign, t, map[string]string{
		"BACKLOG_PREVIOUS_ASSIGNEE": prevAssignee,
	})

	if *asJSON {
		return writeJSON(env.Stdout, view(t))
	}
	if t.Assignee == "" {
		fmt.Fprintf(env.Stdout, "unassigned %03d  %s\n", t.ID, t.Title)
	} else {
		fmt.Fprintf(env.Stdout, "assigned %03d  %s  to %s\n", t.ID, t.Title, t.Assignee)
	}
	return nil
}
