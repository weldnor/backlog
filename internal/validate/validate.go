// Package validate checks a backlog for problems.
//
// Hand-editing task files is a supported workflow rather than a hazard, and
// this is what makes it survivable: the parser tolerates deviations, and
// validate is what tells the author about them.
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/weldnor/backlog/internal/skills"
	"github.com/weldnor/backlog/internal/store"
	"github.com/weldnor/backlog/internal/task"
)

// Finding is one problem found in a backlog.
type Finding struct {
	// File is the path the finding concerns, relative to the project root.
	// It is empty for a finding about the backlog as a whole.
	File     string `json:"file"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	// Repairable reports whether --fix can correct this unambiguously.
	Repairable bool `json:"repairable"`
}

// Report is the outcome of a validation run.
type Report struct {
	Findings []Finding `json:"findings"`
	// Repairs lists what --fix changed, in the order it changed them.
	Repairs  []string `json:"repairs"`
	Errors   int      `json:"errors"`
	Warnings int      `json:"warnings"`
}

// Options controls a validation run.
type Options struct {
	// Fix applies the repairs that have a single unambiguous correction.
	Fix bool
	// Strict treats warnings as errors.
	Strict bool
	// Version is the running binary's version, used for the skill staleness
	// check. An empty version skips that check.
	Version string
}

// OK reports whether the run should be treated as a success.
func (r *Report) OK() bool { return r.Errors == 0 }

// Run validates the backlog, repairing first when asked so that the reported
// findings are the ones that remain.
func Run(st *store.Store, opts Options) (*Report, error) {
	report := &Report{Findings: []Finding{}, Repairs: []string{}}

	if opts.Fix {
		repairs, err := repair(st)
		if err != nil {
			return nil, err
		}
		report.Repairs = repairs
	}

	findings, err := check(st, opts.Version)
	if err != nil {
		return nil, err
	}

	for _, f := range findings {
		if opts.Strict && f.Severity == string(task.SeverityWarning) {
			f.Severity = string(task.SeverityError)
		}
		if f.Severity == string(task.SeverityError) {
			report.Errors++
		} else {
			report.Warnings++
		}
		report.Findings = append(report.Findings, f)
	}
	return report, nil
}

type collector struct {
	project  string
	findings []Finding
}

func (c *collector) add(sev task.Severity, file string, repairable bool, format string, args ...any) {
	c.findings = append(c.findings, Finding{
		File:       file,
		Severity:   string(sev),
		Message:    fmt.Sprintf(format, args...),
		Repairable: repairable,
	})
}

func (c *collector) rel(path string) string {
	r, err := filepath.Rel(c.project, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
}

func check(st *store.Store, version string) ([]Finding, error) {
	c := &collector{project: st.Project}

	checkStructure(c, st)
	if err := checkFiles(c, st); err != nil {
		return nil, err
	}
	if err := checkSkills(c, st, version); err != nil {
		return nil, err
	}

	// Findings are grouped by file so that a person fixing them works through
	// one file at a time; backlog-wide findings come first.
	sort.SliceStable(c.findings, func(i, j int) bool {
		return c.findings[i].File < c.findings[j].File
	})
	return c.findings, nil
}

func checkStructure(c *collector, st *store.Store) {
	for _, d := range []struct {
		path string
		name string
	}{
		{st.TasksPath(), store.TasksDir},
		{st.ArchivePath(), store.ArchiveDir},
	} {
		info, err := os.Stat(d.path)
		if err != nil || !info.IsDir() {
			c.add(task.SeverityError, c.rel(st.Root), false, "the %s directory is missing", d.name)
		}
	}

	strays, err := st.StrayFiles()
	if err != nil {
		return
	}
	for _, p := range strays {
		c.add(task.SeverityWarning, c.rel(p), false, "not a task file; task files are named <id>-<slug>.md")
	}
}

func checkFiles(c *collector, st *store.Store) error {
	entries, err := st.Entries()
	if err != nil {
		return err
	}

	byID := map[int][]string{}
	for _, e := range entries {
		rel := c.rel(e.Path)
		if e.Err != nil {
			c.add(task.SeverityError, rel, false, "%v", e.Err)
			continue
		}
		t := e.Task
		for _, is := range t.Issues {
			c.add(is.Severity, rel, is.Repairable, "%s", is.Message)
		}

		nameID, hasNameID := task.IDFromFileName(e.Name)
		switch {
		case hasNameID && t.ID > 0 && nameID != t.ID:
			c.add(task.SeverityError, rel, false,
				"the file name says id %d but the frontmatter says %d", nameID, t.ID)
		case t.ID > 0 && e.Name != t.FileName():
			// The identifier is the identity; the slug is allowed to drift when
			// a title is edited by hand, and renaming is unambiguous.
			c.add(task.SeverityWarning, rel, true,
				"the file name no longer matches the title; expected %s", t.FileName())
		}

		if task.ValidStatus(t.Status) {
			wantArchived := task.IsTerminal(t.Status)
			if wantArchived && !e.Archived {
				c.add(task.SeverityWarning, rel, true,
					"status is %s but the file is not in %s", t.Status, store.ArchiveDir)
			}
			if !wantArchived && e.Archived {
				c.add(task.SeverityWarning, rel, true,
					"status is %s but the file is in %s", t.Status, store.ArchiveDir)
			}
		}

		if t.ID > 0 {
			byID[t.ID] = append(byID[t.ID], rel)
		}
	}

	ids := make([]int, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		files := byID[id]
		if len(files) < 2 {
			continue
		}
		// Renumbering is a judgement call — which task keeps the identifier is
		// not something the tool can decide — so this is never repairable.
		for _, f := range files {
			c.add(task.SeverityError, f, false,
				"identifier %d is used by more than one task: %s", id, joinOthers(files, f))
		}
	}
	return nil
}

func joinOthers(files []string, self string) string {
	var out []string
	for _, f := range files {
		if f != self {
			out = append(out, f)
		}
	}
	s := ""
	for i, f := range out {
		if i > 0 {
			s += ", "
		}
		s += f
	}
	return s
}

func checkSkills(c *collector, st *store.Store, version string) error {
	if version == "" {
		return nil
	}
	stale, err := skills.Stale(st.Project, version)
	if err != nil {
		return err
	}
	for _, s := range stale {
		c.add(task.SeverityWarning, c.rel(s.Path), false,
			"the %s skill was written by backlog v%s but this is v%s; refresh it with 'backlog init'",
			s.Name, string(s.Action), version)
	}
	return nil
}
