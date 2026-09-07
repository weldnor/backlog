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
	info, err := os.Stat(st.TasksPath())
	if err != nil || !info.IsDir() {
		c.add(task.SeverityError, c.rel(st.Root), false, "the %s directory is missing", store.TasksDir)
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

	checkLinkTargets(c, entries, byID)
	checkLinkReciprocity(c, entries)
	return nil
}

// checkLinkTargets flags a link whose target id names no task in this
// backlog. It runs only after every file has been read, since whether a
// target exists is a cross-file question a single file's parse cannot
// answer; a self-link and a malformed entry are already caught there.
// Never repairable: whether the id was mistyped or the target was removed on
// purpose is a judgement, not something --fix can guess.
func checkLinkTargets(c *collector, entries []store.Entry, byID map[int][]string) {
	for _, e := range entries {
		if e.Err != nil || e.Task == nil {
			continue
		}
		rel := c.rel(e.Path)
		for _, l := range e.Task.Links {
			if _, ok := byID[l.ID]; !ok {
				c.add(task.SeverityWarning, rel, false,
					"links to task %d (%s) which does not exist", l.ID, l.Type)
			}
		}
	}
}

// reciprocalType returns the link type expected on the other side of a
// paired relationship — blocks/blocked-by and duplicates/duplicated-by each
// name the same fact from opposite ends — and false for a type with no
// counterpart to check, which is only related, since it is symmetric with
// itself and a related link pointing back is exactly what it already looks
// like on the other file.
func reciprocalType(t string) (string, bool) {
	switch t {
	case task.LinkBlocks:
		return task.LinkBlockedBy, true
	case task.LinkBlockedBy:
		return task.LinkBlocks, true
	case task.LinkDuplicates:
		return task.LinkDuplicatedBy, true
	case task.LinkDuplicatedBy:
		return task.LinkDuplicates, true
	case task.LinkRelated:
		return task.LinkRelated, true
	}
	return "", false
}

// checkLinkReciprocity flags a link whose target exists but does not link
// back with the expected type — a link is written on one file only (see
// task.Link), so nothing keeps the two sides in agreement automatically.
// This is what notices a link added on one side of a decision and never
// carried to the other, or a stale one left behind after the other file's
// link was removed or retyped. It is a warning, and never repairable: fixing
// it means writing to the *other* task's file, which is a second author's
// decision to make, not something a single-file operation can decide for
// them. A target that does not exist at all is reported once already, by
// checkLinkTargets, so this skips it rather than reporting it again.
func checkLinkReciprocity(c *collector, entries []store.Entry) {
	byID := map[int]*task.Task{}
	for _, e := range entries {
		if e.Err == nil && e.Task != nil && e.Task.ID > 0 {
			byID[e.Task.ID] = e.Task
		}
	}
	for _, e := range entries {
		if e.Err != nil || e.Task == nil {
			continue
		}
		rel := c.rel(e.Path)
		for _, l := range e.Task.Links {
			target, ok := byID[l.ID]
			if !ok {
				continue
			}
			want, ok := reciprocalType(l.Type)
			if !ok || hasLink(target.Links, want, e.Task.ID) {
				continue
			}
			c.add(task.SeverityWarning, rel, false,
				"task %d does not link back with %s; this task records %s:%d",
				l.ID, want, l.Type, l.ID)
		}
	}
}

func hasLink(links []task.Link, typ string, id int) bool {
	for _, l := range links {
		if l.Type == typ && l.ID == id {
			return true
		}
	}
	return false
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
