package cli

import (
	"fmt"
	"strings"

	"github.com/weldnor/backlog/internal/hooks"
	"github.com/weldnor/backlog/internal/task"
)

// editTags collects the --tag flag for `edit` and, unlike stringList alone,
// records whether the flag was given at all. Tags are edited by full
// replacement — the same semantics `browse`'s PATCH uses — so "not given" and
// "given with a value that normalises away" have to be told apart.
type editTags struct {
	stringList
	touched bool
}

func (e *editTags) Set(v string) error {
	e.touched = true
	return e.stringList.Set(v)
}

// runEdit changes a task's title, description or tags — the fields `set`
// deliberately does not reach, because they are prose and content rather than
// workflow state. Before this command they could only be edited through
// `browse`, which needs a terminal that can open a browser; `edit` is the
// same operation for a headless session.
func runEdit(env Env, args []string) error {
	fs := newFlagSet("edit")
	var (
		title       = fs.String("title", "", "the new title")
		description = fs.String("description", "", "the new description (markdown body)")
		asJSON      = fs.Bool("json", false, "print the updated task as JSON")
		tags        editTags
	)
	fs.Var(&tags, "tag", "a tag to attach; repeatable, replaces the entire existing tag list")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := singleID(fs.Args())
	if err != nil {
		return err
	}
	if *title == "" && *description == "" && !tags.touched {
		return usagef("nothing to edit: supply --title, --description, --tag, or any combination")
	}
	if *title != "" && strings.TrimSpace(*title) == "" {
		return usagef("title must not be empty")
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}
	t, err := st.Find(id)
	if err != nil {
		return err
	}

	newTags := t.Tags
	if tags.touched {
		newTags = task.NormalizeTags(tags.stringList)
	}
	// Runs before anything on t changes, so a decline leaves it untouched.
	if err := hooks.RunPre(env.Stderr, st.Root, st.Project, hooks.PreEdit, t, map[string]string{
		"BACKLOG_NEW_TITLE":       *title,
		"BACKLOG_NEW_DESCRIPTION": *description,
		"BACKLOG_NEW_TAGS":        strings.Join(newTags, ","),
	}); err != nil {
		return err
	}

	if *title != "" {
		t.Title = strings.TrimSpace(*title)
	}
	if *description != "" {
		t.SetBody(*description)
	}
	if tags.touched {
		t.Tags = newTags
	}
	if err := st.Save(t); err != nil {
		return err
	}
	hooks.Run(env.Stderr, st.Root, st.Project, hooks.PostEdit, t, nil)

	if *asJSON {
		return writeJSON(env.Stdout, view(t))
	}
	fmt.Fprintf(env.Stdout, "edited %03d  %s\n", t.ID, t.Title)
	return nil
}
