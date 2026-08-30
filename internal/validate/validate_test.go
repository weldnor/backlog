package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weldnor/backlog/internal/store"
)

// clean is a task file with nothing wrong with it.
const clean = `---
id: 1
title: A clean task
status: todo
priority: high
tags:
  - bug
metadata:
  schema: 1
  created: 2026-08-30T20:59:51Z
  author: agent
  refs: []
---
A description.
`

func newBacklog(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func writeTask(t *testing.T, st *store.Store, archived bool, name, content string) string {
	t.Helper()
	dir := st.TasksPath()
	if archived {
		dir = st.ArchivePath()
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func run(t *testing.T, st *store.Store, opts Options) *Report {
	t.Helper()
	report, err := Run(st, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return report
}

func find(report *Report, substr string) (Finding, bool) {
	for _, f := range report.Findings {
		if strings.Contains(f.Message, substr) {
			return f, true
		}
	}
	return Finding{}, false
}

func mustFind(t *testing.T, report *Report, substr string) Finding {
	t.Helper()
	f, ok := find(report, substr)
	if !ok {
		t.Fatalf("no finding containing %q; got %v", substr, messages(report))
	}
	return f
}

func messages(report *Report) []string {
	var out []string
	for _, f := range report.Findings {
		out = append(out, f.Severity+" "+f.File+": "+f.Message)
	}
	return out
}

func TestCleanBacklog(t *testing.T) {
	st := newBacklog(t)
	writeTask(t, st, false, "001-a-clean-task.md", clean)

	report := run(t, st, Options{})
	if !report.OK() || report.Errors != 0 || report.Warnings != 0 {
		t.Errorf("a clean backlog produced %v", messages(report))
	}
}

func TestWarningsDoNotFailUnlessStrict(t *testing.T) {
	st := newBacklog(t)
	// An unknown top-level field is author-owned territory: a warning, not an
	// error.
	writeTask(t, st, false, "001-a-clean-task.md", strings.Replace(clean, "status: todo", "status: todo\nowner: someone", 1))

	report := run(t, st, Options{})
	if !report.OK() {
		t.Errorf("warnings alone must not fail: %v", messages(report))
	}
	if report.Warnings == 0 {
		t.Error("the warning was not reported")
	}
	f := mustFind(t, report, "owner")
	if f.Severity != "warning" {
		t.Errorf("severity = %q, want warning", f.Severity)
	}

	strict := run(t, st, Options{Strict: true})
	if strict.OK() {
		t.Error("strict mode must treat warnings as errors")
	}
	if got := mustFind(t, strict, "owner"); got.Severity != "error" {
		t.Errorf("under strict, severity = %q, want error", got.Severity)
	}
}

func TestErrorAndWarningReportedTogether(t *testing.T) {
	st := newBacklog(t)
	writeTask(t, st, false, "001-not-valid.md", "not a task file at all\n")
	writeTask(t, st, false, "002-drifted-name.md", strings.Replace(
		strings.Replace(clean, "id: 1", "id: 2", 1), "title: A clean task", "title: A different title", 1))

	report := run(t, st, Options{})
	if report.Errors != 1 {
		t.Errorf("errors = %d, want 1: %v", report.Errors, messages(report))
	}
	if report.Warnings != 1 {
		t.Errorf("warnings = %d, want 1: %v", report.Warnings, messages(report))
	}
	if report.OK() {
		t.Error("an error must fail the run")
	}
}

func TestStructuralChecks(t *testing.T) {
	t.Run("missing archive directory", func(t *testing.T) {
		st := newBacklog(t)
		if err := os.Remove(st.ArchivePath()); err != nil {
			t.Fatal(err)
		}
		report := run(t, st, Options{})
		f := mustFind(t, report, "archive directory is missing")
		if f.Severity != "error" {
			t.Errorf("severity = %q, want error", f.Severity)
		}
	})

	t.Run("stray file among the tasks", func(t *testing.T) {
		st := newBacklog(t)
		writeTask(t, st, false, "notes.txt", "scratch\n")
		report := run(t, st, Options{})
		f := mustFind(t, report, "not a task file")
		if !strings.HasSuffix(f.File, "notes.txt") {
			t.Errorf("File = %q, want it to name the stray file", f.File)
		}
	})
}

func TestPerFileChecks(t *testing.T) {
	cases := []struct {
		name     string
		file     string
		content  string
		severity string
		message  string
	}{
		{
			name:     "unparseable frontmatter",
			file:     "001-broken.md",
			content:  "---\nid: 1\ntitle: [unclosed\n---\n",
			severity: "error",
			message:  "not valid YAML",
		},
		{
			name:     "no frontmatter at all",
			file:     "001-broken.md",
			content:  "just markdown\n",
			severity: "error",
			message:  "frontmatter",
		},
		{
			name:     "missing title",
			file:     "001-x.md",
			content:  "---\nid: 1\nstatus: todo\ntags: []\n---\n",
			severity: "error",
			message:  "title is missing",
		},
		{
			name:     "status outside the permitted set",
			file:     "001-x.md",
			content:  "---\nid: 1\ntitle: x\nstatus: blocked\ntags: []\n---\n",
			severity: "error",
			message:  "expected one of todo, doing, done",
		},
		{
			name:     "tags is not a list",
			file:     "001-x.md",
			content:  "---\nid: 1\ntitle: x\nstatus: todo\ntags: bug\n---\n",
			severity: "error",
			message:  "tags must be a list",
		},
		{
			name:     "empty tag",
			file:     "001-x.md",
			content:  "---\nid: 1\ntitle: x\nstatus: todo\ntags:\n  - \"\"\n---\n",
			severity: "error",
			message:  "tags contains an empty entry",
		},
		{
			name:     "timestamp is not RFC 3339",
			file:     "001-x.md",
			content:  "---\nid: 1\ntitle: x\nstatus: todo\ntags: []\nmetadata:\n  schema: 1\n  created: whenever\n---\n",
			severity: "error",
			message:  "not a valid RFC 3339",
		},
		{
			name:     "misspelled metadata key",
			file:     "001-x.md",
			content:  "---\nid: 1\ntitle: x\nstatus: todo\ntags: []\nmetadata:\n  schema: 1\n  creted: 2026-08-30T20:59:51Z\n---\n",
			severity: "error",
			message:  "did you mean created?",
		},
		{
			name:     "unrecognised top-level field",
			file:     "001-x.md",
			content:  strings.Replace(clean, "status: todo", "status: todo\nowner: someone", 1),
			severity: "warning",
			message:  "owner is not a field the CLI understands",
		},
		{
			name:     "identifier disagrees with the file name",
			file:     "007-x.md",
			content:  "---\nid: 3\ntitle: x\nstatus: todo\ntags: []\nmetadata:\n  schema: 1\n  created: 2026-08-30T20:59:51Z\n  refs: []\n---\n",
			severity: "error",
			message:  "the file name says id 7 but the frontmatter says 3",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := newBacklog(t)
			writeTask(t, st, false, c.file, c.content)
			report := run(t, st, Options{})
			f := mustFind(t, report, c.message)
			if f.Severity != c.severity {
				t.Errorf("severity = %q, want %q (%v)", f.Severity, c.severity, messages(report))
			}
			if !strings.HasSuffix(f.File, c.file) {
				t.Errorf("File = %q, want it to name %q", f.File, c.file)
			}
		})
	}
}

func TestCrossFileChecks(t *testing.T) {
	t.Run("duplicate identifier", func(t *testing.T) {
		st := newBacklog(t)
		writeTask(t, st, false, "001-a-clean-task.md", clean)
		writeTask(t, st, false, "001-a-copy.md", strings.Replace(clean, "title: A clean task", "title: A copy", 1))

		report := run(t, st, Options{})
		var named int
		for _, f := range report.Findings {
			if strings.Contains(f.Message, "identifier 1 is used by more than one task") {
				named++
				if f.Repairable {
					t.Error("renumbering is a judgement call and must not be repairable")
				}
			}
		}
		// Both files are named, so either one can be opened from the report.
		if named != 2 {
			t.Errorf("the duplicate was reported on %d files, want 2: %v", named, messages(report))
		}
	})

	t.Run("slug drifted from the title", func(t *testing.T) {
		st := newBacklog(t)
		writeTask(t, st, false, "001-the-old-title.md", clean)
		report := run(t, st, Options{})
		f := mustFind(t, report, "no longer matches the title")
		if f.Severity != "warning" || !f.Repairable {
			t.Errorf("finding = %+v, want a repairable warning", f)
		}
	})

	t.Run("done task left among the active tasks", func(t *testing.T) {
		st := newBacklog(t)
		writeTask(t, st, false, "001-a-clean-task.md", strings.Replace(clean, "status: todo", "status: done", 1))
		report := run(t, st, Options{})
		f := mustFind(t, report, "status is done but the file is not in archive")
		if f.Severity != "warning" || !f.Repairable {
			t.Errorf("finding = %+v, want a repairable warning", f)
		}
	})

	t.Run("active task sitting in the archive", func(t *testing.T) {
		st := newBacklog(t)
		writeTask(t, st, true, "001-a-clean-task.md", clean)
		report := run(t, st, Options{})
		f := mustFind(t, report, "status is todo but the file is in archive")
		if f.Severity != "warning" || !f.Repairable {
			t.Errorf("finding = %+v, want a repairable warning", f)
		}
	})
}

func TestReferencesAreCheckedButNotResolved(t *testing.T) {
	st := newBacklog(t)
	writeTask(t, st, false, "001-a-clean-task.md", strings.Replace(clean,
		"  refs: []", "  refs:\n    - openspec:add-auth\n    - https://example.invalid/issues/9\n    - anything at all", 1))

	report := run(t, st, Options{})
	if !report.OK() || report.Errors != 0 {
		t.Errorf("arbitrary references must be accepted: %v", messages(report))
	}

	st2 := newBacklog(t)
	writeTask(t, st2, false, "001-a-clean-task.md", strings.Replace(clean,
		"  refs: []", "  refs:\n    - \"\"", 1))
	report2 := run(t, st2, Options{})
	f := mustFind(t, report2, "empty reference")
	if f.Severity != "error" {
		t.Errorf("severity = %q, want error", f.Severity)
	}
}

func TestFixIsOptIn(t *testing.T) {
	st := newBacklog(t)
	path := writeTask(t, st, false, "001-the-old-title.md", clean)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	report := run(t, st, Options{})
	if len(report.Repairs) != 0 {
		t.Errorf("validation without --fix reported repairs: %v", report.Repairs)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("the file was moved or renamed without --fix")
	}
	if string(after) != string(before) {
		t.Error("the file was rewritten without --fix")
	}
}

func TestFixRepairsTheUnambiguousFindings(t *testing.T) {
	t.Run("renames a drifted file name", func(t *testing.T) {
		st := newBacklog(t)
		writeTask(t, st, false, "001-the-old-title.md", clean)

		report := run(t, st, Options{Fix: true})
		if _, err := os.Stat(filepath.Join(st.TasksPath(), "001-a-clean-task.md")); err != nil {
			t.Fatalf("the file was not renamed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(st.TasksPath(), "001-the-old-title.md")); !os.IsNotExist(err) {
			t.Error("the old name was left behind")
		}
		if len(report.Repairs) == 0 || !strings.Contains(strings.Join(report.Repairs, " "), "renamed") {
			t.Errorf("the rename was not reported: %v", report.Repairs)
		}
		if !report.OK() || report.Warnings != 0 {
			t.Errorf("the finding survived the repair: %v", messages(report))
		}
	})

	t.Run("moves a task to the directory its status requires", func(t *testing.T) {
		st := newBacklog(t)
		writeTask(t, st, false, "001-a-clean-task.md", strings.Replace(clean, "status: todo", "status: done", 1))

		run(t, st, Options{Fix: true})
		if _, err := os.Stat(filepath.Join(st.ArchivePath(), "001-a-clean-task.md")); err != nil {
			t.Fatalf("the task was not moved to the archive: %v", err)
		}
	})

	t.Run("adds a missing format version", func(t *testing.T) {
		st := newBacklog(t)
		path := writeTask(t, st, false, "001-a-clean-task.md", strings.Replace(clean, "  schema: 1\n", "", 1))

		run(t, st, Options{Fix: true})
		if got := readFile(t, path); !strings.Contains(got, "schema: 1") {
			t.Errorf("the format version was not added:\n%s", got)
		}
	})

	t.Run("normalises timestamp formatting", func(t *testing.T) {
		st := newBacklog(t)
		path := writeTask(t, st, false, "001-a-clean-task.md",
			strings.Replace(clean, "created: 2026-08-30T20:59:51Z", "created: 2026-08-30 20:59:51", 1))

		run(t, st, Options{Fix: true})
		if got := readFile(t, path); !strings.Contains(got, "created: 2026-08-30T20:59:51Z") {
			t.Errorf("the timestamp was not normalised:\n%s", got)
		}
	})

	t.Run("de-duplicates tags", func(t *testing.T) {
		st := newBacklog(t)
		path := writeTask(t, st, false, "001-a-clean-task.md",
			strings.Replace(clean, "  - bug\n", "  - bug\n  - bug\n", 1))

		run(t, st, Options{Fix: true})
		got := readFile(t, path)
		if strings.Count(got, "- bug") != 1 {
			t.Errorf("the duplicate tag survived:\n%s", got)
		}
	})
}

func TestFixLeavesAmbiguousFindingsAlone(t *testing.T) {
	t.Run("duplicate identifiers", func(t *testing.T) {
		st := newBacklog(t)
		// Both also have drifted names, so a repair would be visible if one
		// were attempted.
		a := writeTask(t, st, false, "001-old-name-one.md", clean)
		b := writeTask(t, st, false, "001-old-name-two.md", strings.Replace(clean, "title: A clean task", "title: A copy", 1))
		beforeA, beforeB := readFile(t, a), readFile(t, b)

		report := run(t, st, Options{Fix: true})
		if readFile(t, a) != beforeA || readFile(t, b) != beforeB {
			t.Error("a task with a duplicate identifier was modified")
		}
		if _, ok := find(report, "identifier 1 is used by more than one task"); !ok {
			t.Errorf("the duplicate is no longer reported: %v", messages(report))
		}
	})

	t.Run("unparseable frontmatter", func(t *testing.T) {
		st := newBacklog(t)
		path := writeTask(t, st, false, "001-broken.md", "---\nid: 1\ntitle: [unclosed\n---\n")
		before := readFile(t, path)

		report := run(t, st, Options{Fix: true})
		if readFile(t, path) != before {
			t.Error("an unparseable file was rewritten")
		}
		if report.OK() {
			t.Error("the error is no longer reported")
		}
	})

	t.Run("an invalid status blocks repair of the same file", func(t *testing.T) {
		st := newBacklog(t)
		path := writeTask(t, st, false, "001-old-name.md", strings.Replace(clean, "status: todo", "status: blocked", 1))
		before := readFile(t, path)

		run(t, st, Options{Fix: true})
		if readFile(t, path) != before {
			t.Error("a file with an unrecognised status was rewritten")
		}
	})
}

func TestFindingsAreGroupedByFile(t *testing.T) {
	st := newBacklog(t)
	writeTask(t, st, false, "002-b.md", "---\nid: 2\ntitle: b\nstatus: nope\ntags: []\n---\n")
	writeTask(t, st, false, "001-a.md", "---\nid: 1\ntitle: a\nstatus: nope\ntags: []\n---\n")

	report := run(t, st, Options{})
	seen := map[string]bool{}
	last := ""
	for _, f := range report.Findings {
		if f.File != last {
			if seen[f.File] {
				t.Fatalf("findings for %q are not contiguous: %v", f.File, messages(report))
			}
			seen[f.File] = true
			last = f.File
		}
	}
	if len(seen) != 2 {
		t.Errorf("got findings for %d files, want 2", len(seen))
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPriorityChecks(t *testing.T) {
	cases := []struct {
		name       string
		content    string
		severity   string
		message    string
		repairable bool
	}{
		{
			name:       "outside the permitted set",
			content:    strings.Replace(clean, "priority: high", "priority: urgent", 1),
			severity:   "error",
			message:    "expected one of high, medium, low",
			repairable: false,
		},
		{
			name:       "missing",
			content:    strings.Replace(clean, "priority: high\n", "", 1),
			severity:   "warning",
			message:    "priority is missing",
			repairable: true,
		},
		{
			name:       "empty",
			content:    strings.Replace(clean, "priority: high", "priority:", 1),
			severity:   "warning",
			message:    "priority is empty",
			repairable: true,
		},
		{
			name:       "not a string",
			content:    strings.Replace(clean, "priority: high", "priority:\n  - high", 1),
			severity:   "error",
			message:    "priority must be a string",
			repairable: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := newBacklog(t)
			writeTask(t, st, false, "001-a-clean-task.md", c.content)

			report := run(t, st, Options{})
			f := mustFind(t, report, c.message)
			if f.Severity != c.severity {
				t.Errorf("severity = %q, want %q (%v)", f.Severity, c.severity, messages(report))
			}
			if f.Repairable != c.repairable {
				t.Errorf("Repairable = %v, want %v", f.Repairable, c.repairable)
			}
			// A missing priority is a convention violation, not a reason to
			// fail a commit hook.
			if c.severity == "warning" && !report.OK() {
				t.Errorf("a warning made validation fail: %v", messages(report))
			}
			if c.severity == "error" && report.OK() {
				t.Errorf("an error did not fail validation: %v", messages(report))
			}
		})
	}
}

func TestFixAddsAMissingPriority(t *testing.T) {
	st := newBacklog(t)
	path := writeTask(t, st, false, "001-a-clean-task.md", strings.Replace(clean, "priority: high\n", "", 1))

	report := run(t, st, Options{Fix: true})
	got := readFile(t, path)
	if !strings.Contains(got, "priority: medium") {
		t.Errorf("the default priority was not written:\n%s", got)
	}
	if !strings.Contains(got, "status: todo\npriority: medium") {
		t.Errorf("the priority was not written in its canonical place:\n%s", got)
	}
	if len(report.Repairs) == 0 {
		t.Error("the repair was not reported")
	}
	// Nothing should be left to report once the file has been brought up to
	// the convention.
	if !report.OK() || report.Warnings != 0 {
		t.Errorf("the finding survived the repair: %v", messages(report))
	}
}

func TestFixLeavesAnUnrecognisedPriorityAlone(t *testing.T) {
	st := newBacklog(t)
	content := strings.Replace(clean, "priority: high", "priority: urgent", 1)
	path := writeTask(t, st, false, "001-a-clean-task.md", content)

	report := run(t, st, Options{Fix: true})
	got := readFile(t, path)
	if got != content {
		t.Errorf("a value someone typed was rewritten:\n--- got ---\n%s\n--- want ---\n%s", got, content)
	}
	if strings.Contains(got, "priority: medium") {
		t.Error("the default was substituted for a deliberate value")
	}
	// Still reported, so the author is the one who decides what it should be.
	f := mustFind(t, report, "expected one of high, medium, low")
	if f.Severity != "error" {
		t.Errorf("severity = %q, want error", f.Severity)
	}
}

func TestReasonChecks(t *testing.T) {
	declined := strings.Replace(clean, "status: todo", "status: declined", 1)
	cases := []struct {
		name    string
		content string
		message string
	}{
		{
			name:    "declined with no reason",
			content: declined,
			message: "no reason is recorded",
		},
		{
			name:    "declined with an empty reason",
			content: strings.Replace(declined, "priority: high", "priority: high\nreason:", 1),
			message: "no reason is recorded",
		},
		{
			name:    "reason on a task that is not declined",
			content: strings.Replace(clean, "priority: high", "priority: high\nreason: not worth it", 1),
			message: "a reason applies only to a declined task",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := newBacklog(t)
			writeTask(t, st, true, "001-a-clean-task.md", c.content)

			report := run(t, st, Options{})
			f := mustFind(t, report, c.message)
			if f.Severity != "error" {
				t.Errorf("severity = %q, want error (%v)", f.Severity, messages(report))
			}
			// One needs prose written and the other needs prose deleted; neither
			// is a correction a tool can make.
			if f.Repairable {
				t.Error("the pairing finding was marked repairable")
			}
			if report.OK() {
				t.Errorf("an error did not fail validation: %v", messages(report))
			}
		})
	}

	t.Run("a declined task with a reason is clean", func(t *testing.T) {
		st := newBacklog(t)
		writeTask(t, st, true, "001-a-clean-task.md",
			strings.Replace(declined, "priority: high", "priority: high\nreason: the cost outweighs the benefit", 1))

		report := run(t, st, Options{})
		if !report.OK() || report.Errors != 0 {
			t.Errorf("a well-formed decline was reported: %v", messages(report))
		}
	})
}

func TestDeclinedTaskLeftAmongTheActiveTasks(t *testing.T) {
	st := newBacklog(t)
	writeTask(t, st, false, "001-a-clean-task.md", strings.Replace(
		strings.Replace(clean, "status: todo", "status: declined", 1),
		"priority: high", "priority: high\nreason: not worth the churn", 1))

	report := run(t, st, Options{})
	f := mustFind(t, report, "status is declined but the file is not in archive")
	if f.Severity != "warning" || !f.Repairable {
		t.Errorf("finding = %+v, want a repairable warning", f)
	}
}

func TestFixMovesADeclinedTaskToTheArchive(t *testing.T) {
	st := newBacklog(t)
	writeTask(t, st, false, "001-a-clean-task.md", strings.Replace(
		strings.Replace(clean, "status: todo", "status: declined", 1),
		"priority: high", "priority: high\nreason: not worth the churn", 1))

	report := run(t, st, Options{Fix: true})
	if _, err := os.Stat(filepath.Join(st.ArchivePath(), "001-a-clean-task.md")); err != nil {
		t.Errorf("the declined task was not moved to the archive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(st.TasksPath(), "001-a-clean-task.md")); !os.IsNotExist(err) {
		t.Error("the declined task was left behind among the active tasks")
	}
	if len(report.Repairs) == 0 {
		t.Error("the move was not reported")
	}
	if !report.OK() || report.Warnings != 0 {
		t.Errorf("the finding survived the repair: %v", messages(report))
	}
}

func TestFixLeavesTheReasonPairingAlone(t *testing.T) {
	t.Run("does not invent a reason", func(t *testing.T) {
		st := newBacklog(t)
		content := strings.Replace(clean, "status: todo", "status: declined", 1)
		path := writeTask(t, st, true, "001-a-clean-task.md", content)

		report := run(t, st, Options{Fix: true})
		if got := readFile(t, path); got != content {
			t.Errorf("the file was rewritten:\n--- got ---\n%s\n--- want ---\n%s", got, content)
		}
		if f := mustFind(t, report, "no reason is recorded"); f.Severity != "error" {
			t.Errorf("severity = %q, want error", f.Severity)
		}
	})

	t.Run("does not delete a misplaced reason", func(t *testing.T) {
		st := newBacklog(t)
		content := strings.Replace(clean, "priority: high", "priority: high\nreason: not worth it", 1)
		path := writeTask(t, st, false, "001-a-clean-task.md", content)

		report := run(t, st, Options{Fix: true})
		got := readFile(t, path)
		if got != content {
			t.Errorf("the file was rewritten:\n--- got ---\n%s\n--- want ---\n%s", got, content)
		}
		if !strings.Contains(got, "reason: not worth it") {
			t.Error("prose someone wrote was deleted by --fix")
		}
		if f := mustFind(t, report, "a reason applies only to a declined task"); f.Severity != "error" {
			t.Errorf("severity = %q, want error", f.Severity)
		}
	})
}
