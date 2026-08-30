package cli

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/antonkolesov/backlog/internal/store"
	"github.com/antonkolesov/backlog/internal/task"
)

func runAdd(env Env, args []string) error {
	fs := newFlagSet("add")
	var (
		title       = fs.String("title", "", "the task title (may also be given as the first argument)")
		description = fs.String("description", "", "the task description")
		author      = fs.String("author", task.AuthorAgent, "who recorded this task: agent or human")
		priority    = fs.String("priority", task.DefaultPriority, "how severe the finding is: high, medium or low")
		asJSON      = fs.Bool("json", false, "print the created task as JSON")
		tags        stringList
		files       stringList
		refs        stringList
	)
	fs.Var(&tags, "tag", "a tag to attach (repeatable)")
	fs.Var(&files, "file", "a source file the finding concerns (repeatable)")
	fs.Var(&refs, "ref", "a free-form reference to external work (repeatable)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	text := *title
	if text == "" {
		text = strings.Join(fs.Args(), " ")
	} else if fs.NArg() > 0 {
		return usagef("unexpected argument %q; the title was already given with --title", fs.Arg(0))
	}
	if strings.TrimSpace(text) == "" {
		return usagef("a title is required")
	}
	if *author != task.AuthorAgent && *author != task.AuthorHuman {
		return usagef("--author must be %s or %s", task.AuthorAgent, task.AuthorHuman)
	}
	if !task.ValidPriority(*priority) {
		return usagef("unknown priority %q, expected one of %s", *priority, strings.Join(task.Priorities, ", "))
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}

	t := task.New(text, *description, tags, files, refs, *author, *priority, store.Provenance(st.Project), time.Now())
	if err := st.Create(t); err != nil {
		return err
	}

	if *asJSON {
		return writeJSON(env.Stdout, view(t))
	}
	fmt.Fprintf(env.Stdout, "created %03d  %s\n", t.ID, t.Title)
	return nil
}

func runList(env Env, args []string) error {
	fs := newFlagSet("list")
	asJSON := fs.Bool("json", false, "print the tasks as JSON")
	var sc scope
	sc.register(fs)
	sc.registerPriority(fs)
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
	tasks, err := sc.apply(st)
	if err != nil {
		return err
	}
	// Severity is what a listing is read for, so it, not the identifier, is
	// the outer ordering. Search keeps its own ranking by where it matched.
	task.SortByPriorityThenID(tasks)

	if *asJSON {
		return writeJSON(env.Stdout, views(tasks))
	}
	// An empty result is an answer, not a failure.
	if len(tasks) == 0 {
		fmt.Fprintln(env.Stdout, "no tasks matched")
		return nil
	}
	writeTaskLines(env.Stdout, tasks)
	return nil
}

func runShow(env Env, args []string) error {
	fs := newFlagSet("show")
	asJSON := fs.Bool("json", false, "print the task as JSON")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := singleID(fs.Args())
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

	if *asJSON {
		return writeJSON(env.Stdout, view(t))
	}
	writeTaskDetail(env, t)
	return nil
}

func writeTaskDetail(env Env, t *task.Task) {
	w := env.Stdout
	fmt.Fprintf(w, "id       %d\n", t.ID)
	fmt.Fprintf(w, "title    %s\n", t.Title)
	fmt.Fprintf(w, "status   %s\n", t.Status)
	fmt.Fprintf(w, "priority %s\n", t.Priority)
	// Only a declined task has one, and the line would be noise on every other.
	if t.Reason != "" {
		fmt.Fprintf(w, "reason   %s\n", t.Reason)
	}
	if len(t.Tags) > 0 {
		fmt.Fprintf(w, "tags     %s\n", strings.Join(t.Tags, ", "))
	}
	fmt.Fprintf(w, "file     %s\n", filepath.Base(t.Path))
	if t.Meta.Created != "" {
		fmt.Fprintf(w, "created  %s\n", t.Meta.Created)
	}
	if t.Meta.Author != "" {
		fmt.Fprintf(w, "author   %s\n", t.Meta.Author)
	}
	if len(t.Meta.Source.Files) > 0 {
		fmt.Fprintf(w, "source   %s\n", strings.Join(t.Meta.Source.Files, ", "))
	}
	if t.Meta.Source.Branch != "" || t.Meta.Source.Commit != "" {
		fmt.Fprintf(w, "observed %s\n", strings.TrimSpace(t.Meta.Source.Branch+" "+shortCommit(t.Meta.Source.Commit)))
	}
	for _, ref := range t.Meta.Refs {
		fmt.Fprintf(w, "ref      %s\n", ref)
	}
	if body := strings.TrimRight(t.Body, "\n"); body != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, body)
	}
}

func shortCommit(c string) string {
	if len(c) > 12 {
		return c[:12]
	}
	return c
}

func runSet(env Env, args []string) error {
	fs := newFlagSet("set")
	var (
		status   = fs.String("status", "", "the new status: todo, doing, done or declined")
		priority = fs.String("priority", "", "the new priority: high, medium or low")
		reason   = fs.String("reason", "", "why the task is being declined; only for status declined")
		asJSON   = fs.Bool("json", false, "print the updated task as JSON")
		refs     stringList
	)
	fs.Var(&refs, "ref", "a free-form reference to attach (repeatable)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return usagef("a task identifier is required")
	}
	id, err := parseID(rest[0])
	if err != nil {
		return err
	}
	rest = rest[1:]
	if len(rest) > 0 {
		if *status != "" {
			return usagef("unexpected argument %q; the status was already given with --status", rest[0])
		}
		*status = rest[0]
		rest = rest[1:]
	}
	if len(rest) > 0 {
		return usagef("unexpected argument %q", rest[0])
	}
	if *status == "" && *priority == "" && *reason == "" && len(refs) == 0 {
		return usagef("nothing to do: supply a status, a --priority, a --reason, a --ref, or any combination")
	}
	if *status != "" && !task.ValidStatus(*status) {
		return usagef("unknown status %q, expected one of %s", *status, strings.Join(task.Statuses, ", "))
	}
	if *priority != "" && !task.ValidPriority(*priority) {
		return usagef("unknown priority %q, expected one of %s", *priority, strings.Join(task.Priorities, ", "))
	}
	// A decline nobody can audit is the state the status exists to eliminate,
	// so the reason is required rather than merely encouraged.
	if *status == task.StatusDeclined && strings.TrimSpace(*reason) == "" {
		return usagef("declining a task requires --reason, so that the decision can be read later")
	}
	if *status != "" && *status != task.StatusDeclined && *reason != "" {
		return usagef("--reason applies only to a %s task", task.StatusDeclined)
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}
	t, err := st.Find(id)
	if err != nil {
		return err
	}

	// A reason on its own revises the prose of a decline that already stands;
	// it is not a way to record one without saying so.
	if *status == "" && *reason != "" && t.Status != task.StatusDeclined {
		return usagef("task %d is %s, not %s; --reason applies only to a %s task",
			t.ID, t.Status, task.StatusDeclined, task.StatusDeclined)
	}
	if *status != "" {
		// Leaving declined drops the reason: it describes a state the task is
		// no longer in, and git keeps what it said.
		if t.Status == task.StatusDeclined && *status != task.StatusDeclined {
			t.Reason = ""
		}
		t.Status = *status
	}
	if *reason != "" {
		t.Reason = *reason
	}
	if *priority != "" {
		t.Priority = *priority
	}
	for _, ref := range refs {
		if strings.TrimSpace(ref) == "" {
			return usagef("a reference may not be empty")
		}
		t.AddRef(ref)
	}
	if err := st.Save(t); err != nil {
		return err
	}

	if *asJSON {
		return writeJSON(env.Stdout, view(t))
	}
	fmt.Fprintf(env.Stdout, "updated %03d  %s  (%s)\n", t.ID, t.Title, t.Status)
	return nil
}

func runRm(env Env, args []string) error {
	fs := newFlagSet("rm")
	asJSON := fs.Bool("json", false, "print the removed task as JSON")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	id, err := singleID(fs.Args())
	if err != nil {
		return err
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}
	t, err := st.Remove(id)
	if err != nil {
		return err
	}

	if *asJSON {
		return writeJSON(env.Stdout, view(t))
	}
	fmt.Fprintf(env.Stdout, "removed %03d  %s\n", t.ID, t.Title)
	return nil
}

func singleID(args []string) (int, error) {
	if len(args) == 0 {
		return 0, usagef("a task identifier is required")
	}
	if len(args) > 1 {
		return 0, usagef("unexpected argument %q", args[1])
	}
	return parseID(args[0])
}

func parseID(s string) (int, error) {
	id, err := strconv.Atoi(strings.TrimLeft(s, "#"))
	if err != nil || id <= 0 {
		return 0, usagef("%q is not a task identifier", s)
	}
	return id, nil
}
