package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/weldnor/backlog/internal/validate"
)

func TestValidateExitStatus(t *testing.T) {
	t.Run("a clean backlog exits zero", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A perfectly good finding")

		code, stdout, _ := h.run("validate")
		if code != 0 {
			t.Errorf("exit = %d, want 0", code)
		}
		if !strings.Contains(stdout, "no problems found") {
			t.Errorf("stdout = %q", stdout)
		}
	})

	t.Run("warnings alone exit zero but are still reported", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")
		editTask(t, h, "001-a-finding.md", func(s string) string {
			return strings.Replace(s, "status: new", "status: new\nowner: someone", 1)
		})

		code, stdout, _ := h.run("validate")
		if code != 0 {
			t.Errorf("exit = %d, want 0", code)
		}
		if !strings.Contains(stdout, "warning") {
			t.Errorf("the warning was not reported:\n%s", stdout)
		}
	})

	t.Run("strict promotes warnings and fails", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		h.mustRun("add", "A finding")
		editTask(t, h, "001-a-finding.md", func(s string) string {
			return strings.Replace(s, "status: new", "status: new\nowner: someone", 1)
		})

		if code, _, _ := h.run("validate", "--strict"); code == 0 {
			t.Error("strict mode exited zero with warnings present")
		}
	})

	t.Run("errors exit non-zero", func(t *testing.T) {
		h := newHarness(t)
		h.initBacklog()
		writeRaw(t, h, "001-broken.md", "---\nid: 1\ntitle: [unclosed\n---\n")

		code, _, stderr := h.run("validate")
		if code == 0 {
			t.Error("exit = 0, want non-zero")
		}
		if !strings.Contains(stderr, "error") {
			t.Errorf("stderr = %q, want a summary of the failure", stderr)
		}
	})
}

func TestValidateJSONOutput(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	writeRaw(t, h, "001-broken.md", "---\nid: 1\ntitle: [unclosed\n---\n")
	h.mustRun("add", "A finding")
	editTask(t, h, "002-a-finding.md", func(s string) string {
		return strings.Replace(s, "title: A finding", "title: A renamed finding", 1)
	})

	_, stdout, _ := h.run("validate", "--json")
	var report validate.Report
	decode(t, stdout, &report)

	if len(report.Findings) < 2 {
		t.Fatalf("got %d findings, want at least 2: %+v", len(report.Findings), report)
	}
	if report.Errors == 0 || report.Warnings == 0 {
		t.Errorf("counts = %d errors, %d warnings", report.Errors, report.Warnings)
	}
	for _, f := range report.Findings {
		if f.File == "" || f.Severity == "" || f.Message == "" {
			t.Errorf("finding is missing a field: %+v", f)
		}
		if f.Severity != "error" && f.Severity != "warning" {
			t.Errorf("severity = %q", f.Severity)
		}
	}

	var repairable bool
	for _, f := range report.Findings {
		if f.Repairable {
			repairable = true
		}
	}
	if !repairable {
		t.Error("no finding was marked repairable, though the slug drifted")
	}
}

func TestValidateHumanOutputGroupsByFileAndFlagsRepairability(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "A finding")
	editTask(t, h, "001-a-finding.md", func(s string) string {
		return strings.Replace(s, "title: A finding", "title: A renamed finding", 1)
	})

	stdout := h.mustRun("validate")
	if !strings.Contains(stdout, ".backlog/tasks/001-a-finding.md") {
		t.Errorf("the finding is not grouped under its file:\n%s", stdout)
	}
	if !strings.Contains(stdout, "--fix") {
		t.Errorf("repairability is not indicated:\n%s", stdout)
	}
	if !strings.Contains(stdout, "1 warning") {
		t.Errorf("the summary count is missing:\n%s", stdout)
	}
}

func TestValidateFixReportsWhatItChanged(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "A finding")
	editTask(t, h, "001-a-finding.md", func(s string) string {
		return strings.Replace(s, "title: A finding", "title: A renamed finding", 1)
	})

	code, stdout, _ := h.run("validate", "--fix")
	if code != 0 {
		t.Errorf("exit = %d after a successful repair", code)
	}
	if !strings.Contains(stdout, "renamed") {
		t.Errorf("the repair was not reported:\n%s", stdout)
	}
	if _, err := os.Stat(h.path(".backlog", "tasks", "001-a-renamed-finding.md")); err != nil {
		t.Errorf("the file was not renamed: %v", err)
	}
	if !strings.Contains(h.mustRun("validate"), "no problems found") {
		t.Error("the backlog does not validate clean after the repair")
	}
}

func editTask(t *testing.T, h *harness, name string, edit func(string) string) {
	t.Helper()
	path := h.path(".backlog", "tasks", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(edit(string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeRaw(t *testing.T, h *harness, name, content string) {
	t.Helper()
	if err := os.WriteFile(h.path(".backlog", "tasks", name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
