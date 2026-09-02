package cli

import (
	"fmt"
	"path/filepath"

	"github.com/weldnor/backlog/internal/skills"
	"github.com/weldnor/backlog/internal/store"
)

func runInit(env Env, args []string) error {
	fs := newFlagSet("init")
	var (
		force      = fs.Bool("force", false, "overwrite skill entries that have been edited locally")
		skipSkills = fs.Bool("no-skills", false, "create the backlog without installing the agent skills")
		asJSON     = fs.Bool("json", false, "print the result as JSON")
	)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return usagef("unexpected argument %q", fs.Arg(0))
	}

	// Init never destroys anything: the directories are created only if
	// missing, so re-running it over a backlog that holds tasks is safe.
	st, err := store.Init(env.Dir)
	if err != nil {
		return err
	}

	var installed []skills.Result
	if !*skipSkills {
		installed, err = skills.Install(st.Project, Version, *force)
		if err != nil {
			return err
		}
	}

	if *asJSON {
		type skillView struct {
			Name   string `json:"name"`
			Path   string `json:"path"`
			Action string `json:"action"`
		}
		out := struct {
			Backlog string      `json:"backlog"`
			Version string      `json:"version"`
			Skills  []skillView `json:"skills"`
		}{Backlog: filepath.ToSlash(st.Root), Version: Version, Skills: []skillView{}}
		for _, r := range installed {
			out.Skills = append(out.Skills, skillView{r.Name, filepath.ToSlash(rel(st.Project, r.Path)), string(r.Action)})
		}
		return writeJSON(env.Stdout, out)
	}

	fmt.Fprintf(env.Stdout, "backlog ready at %s\n", st.Root)
	for _, r := range installed {
		switch r.Action {
		case skills.Skipped:
			fmt.Fprintf(env.Stderr, "backlog init: %s has local edits and was left alone; use --force to replace it\n", rel(st.Project, r.Path))
		case skills.Overwritten:
			fmt.Fprintf(env.Stderr, "backlog init: %s was overwritten and its local edits are gone\n", rel(st.Project, r.Path))
		default:
			fmt.Fprintf(env.Stdout, "  skill %s %s\n", r.Name, r.Action)
		}
	}
	return nil
}

func rel(base, path string) string {
	r, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}
