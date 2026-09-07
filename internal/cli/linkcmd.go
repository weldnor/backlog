package cli

import (
	"fmt"
	"strings"

	"github.com/weldnor/backlog/internal/hooks"
	"github.com/weldnor/backlog/internal/store"
	"github.com/weldnor/backlog/internal/task"
)

// runLink dispatches the two link-maintenance subcommands. Unlike `tag`,
// which acts across the whole backlog, a link is scoped to the one task it is
// recorded on, so `add` and `rm` each touch exactly one file — the same
// operation `edit --link` performs, but incrementally: neither needs the
// caller to already know and restate the task's existing links, which
// matters more here than it does for tags, since two agents adding different
// links to the same task around the same time would otherwise race on a
// full-replacement write.
func runLink(env Env, args []string) error {
	usage := "usage: backlog link add <id> <type> <target-id> | backlog link rm <id> <type> <target-id>"
	if len(args) == 0 {
		return usagef("%s", usage)
	}
	switch args[0] {
	case "-h", "--help":
		return usagef("%s", usage)
	case "add":
		return runLinkAdd(env, args[1:])
	case "rm":
		return runLinkRm(env, args[1:])
	default:
		return usagef("unknown subcommand %q, expected add or rm", args[0])
	}
}

func runLinkAdd(env Env, args []string) error {
	fs := newFlagSet("link add")
	asJSON := fs.Bool("json", false, "print the updated task as JSON")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, typ, target, err := linkArgs(fs.Args())
	if err != nil {
		return err
	}
	if id == target {
		return usagef("task %d cannot link to itself", id)
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}
	t, err := st.Find(id)
	if err != nil {
		return err
	}
	newLinks := task.NormalizeLinks(append(append([]task.Link{}, t.Links...), task.Link{Type: typ, ID: target}))
	if err := saveLinks(env, st, t, newLinks); err != nil {
		return err
	}

	if *asJSON {
		return writeJSON(env.Stdout, view(t))
	}
	fmt.Fprintf(env.Stdout, "linked %03d  %s:%d\n", t.ID, typ, target)
	return nil
}

func runLinkRm(env Env, args []string) error {
	fs := newFlagSet("link rm")
	asJSON := fs.Bool("json", false, "print the updated task as JSON")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, typ, target, err := linkArgs(fs.Args())
	if err != nil {
		return err
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}
	t, err := st.Find(id)
	if err != nil {
		return err
	}
	newLinks := make([]task.Link, 0, len(t.Links))
	for _, l := range t.Links {
		if l.Type == typ && l.ID == target {
			continue
		}
		newLinks = append(newLinks, l)
	}
	if err := saveLinks(env, st, t, newLinks); err != nil {
		return err
	}

	if *asJSON {
		return writeJSON(env.Stdout, view(t))
	}
	fmt.Fprintf(env.Stdout, "unlinked %03d  %s:%d\n", t.ID, typ, target)
	return nil
}

// linkArgs parses the positional `<id> <type> <target-id>` shared by add and
// rm.
func linkArgs(args []string) (id int, typ string, target int, err error) {
	if len(args) < 3 {
		return 0, "", 0, usagef("usage: <id> <type> <target-id>, e.g. 5 blocks 12")
	}
	if len(args) > 3 {
		return 0, "", 0, usagef("unexpected argument %q", args[3])
	}
	id, err = parseID(args[0])
	if err != nil {
		return 0, "", 0, err
	}
	typ = args[1]
	if !task.ValidLinkType(typ) {
		return 0, "", 0, usagef("unknown link type %q, expected one of %s", typ, strings.Join(task.LinkTypes, ", "))
	}
	target, err = parseID(args[2])
	if err != nil {
		return 0, "", 0, err
	}
	return id, typ, target, nil
}

// saveLinks fires the same pre/post edit hooks `edit --link` does — a link
// change is content, not workflow state — and saves.
func saveLinks(env Env, st *store.Store, t *task.Task, newLinks []task.Link) error {
	if err := hooks.RunPre(env.Stderr, st.Root, st.Project, hooks.PreEdit, t, map[string]string{
		"BACKLOG_NEW_TITLE":       "",
		"BACKLOG_NEW_DESCRIPTION": "",
		"BACKLOG_NEW_LINKS":       strings.Join(linkStrings(newLinks), ","),
	}); err != nil {
		return err
	}
	t.Links = newLinks
	if err := st.Save(t); err != nil {
		return err
	}
	hooks.Run(env.Stderr, st.Root, st.Project, hooks.PostEdit, t, nil)
	return nil
}
