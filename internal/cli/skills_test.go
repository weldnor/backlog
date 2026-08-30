package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/weldnor/backlog/internal/validate"
)

func skillPath(h *harness, name string) string {
	return h.path(".claude", "skills", name, "SKILL.md")
}

func TestInitInstallsBothSkillsStampedWithTheVersion(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	for _, name := range []string{"backlog-capture", "backlog-triage"} {
		data, err := os.ReadFile(skillPath(h, name))
		if err != nil {
			t.Fatalf("%s was not installed: %v", name, err)
		}
		if !strings.Contains(string(data), "managed by backlog v"+Version+";") {
			t.Errorf("%s does not carry the running version %s", name, Version)
		}
		if !strings.HasPrefix(string(data), "---\n") {
			t.Errorf("%s does not open with its frontmatter", name)
		}
	}
}

// The skills live inside the project so that a clone has them without any
// further installation step.
func TestSkillsAreInstalledIntoTheProject(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	rel, err := filepath.Rel(h.dir, skillPath(h, "backlog-capture"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.ToSlash(rel) != ".claude/skills/backlog-capture/SKILL.md" {
		t.Errorf("the skill was installed at %q", rel)
	}
}

func TestInitSkipsALocallyEditedSkill(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	path := skillPath(h, "backlog-capture")

	edited := readTaskFile(t, path) + "\n## A section added by hand\n"
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := h.run("init")
	if code != 0 {
		t.Fatalf("init exited %d: %s", code, stderr)
	}
	if readTaskFile(t, path) != edited {
		t.Error("the local edit was overwritten without --force")
	}
	if !strings.Contains(stderr, "--force") {
		t.Errorf("the skip was not reported with a way out: %q", stderr)
	}
}

func TestInitForceReplacesALocallyEditedSkill(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	path := skillPath(h, "backlog-capture")
	if err := os.WriteFile(path, []byte(readTaskFile(t, path)+"\nlocal edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := h.run("init", "--force")
	if code != 0 {
		t.Fatalf("init --force exited %d: %s", code, stderr)
	}
	if strings.Contains(readTaskFile(t, path), "local edit") {
		t.Error("the local edit survived --force")
	}
	if !strings.Contains(stderr, "local edits") {
		t.Errorf("the overwrite was not warned about: %q", stderr)
	}
}

func TestInitRefreshesAnUnmodifiedSkill(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	path := skillPath(h, "backlog-capture")

	saved := Version
	Version = "99.0.0"
	defer func() { Version = saved }()

	code, _, stderr := h.run("init")
	if code != 0 {
		t.Fatalf("init exited %d: %s", code, stderr)
	}
	if !strings.Contains(readTaskFile(t, path), "managed by backlog v99.0.0;") {
		t.Error("an unmodified skill was not brought up to the new version")
	}
	if strings.Contains(stderr, "left alone") {
		t.Errorf("an unmodified refresh warned: %q", stderr)
	}
}

func TestValidateWarnsAboutStaleSkills(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	t.Run("current skills produce no warning", func(t *testing.T) {
		var report validate.Report
		decode(t, h.mustRun("validate", "--json"), &report)
		for _, f := range report.Findings {
			if strings.Contains(f.Message, "skill") {
				t.Errorf("unexpected skill finding: %s", f.Message)
			}
		}
	})

	t.Run("an older skill is reported", func(t *testing.T) {
		saved := Version
		Version = "99.0.0"
		defer func() { Version = saved }()

		code, stdout, _ := h.run("validate", "--json")
		if code != 0 {
			t.Errorf("a staleness warning must not fail the run, exit = %d", code)
		}
		var report validate.Report
		decode(t, stdout, &report)

		var found bool
		for _, f := range report.Findings {
			if !strings.Contains(f.Message, "backlog-capture skill") {
				continue
			}
			found = true
			if f.Severity != "warning" {
				t.Errorf("severity = %q, want warning", f.Severity)
			}
			if !strings.Contains(f.Message, "backlog init") {
				t.Errorf("the warning does not name the refresh command: %s", f.Message)
			}
			if !strings.Contains(f.Message, saved) {
				t.Errorf("the warning does not name the installed version: %s", f.Message)
			}
		}
		if !found {
			t.Error("no staleness warning was produced")
		}
	})
}

// The version is bumped whenever the embedded skill text changes, so that a
// project still holding the previous release's skills is told to refresh.
func TestSkillsInstalledByThePreviousVersionAreStale(t *testing.T) {
	const previous = "0.2.0"
	if Version == previous {
		t.Fatalf("Version is still %s although the skill text changed", previous)
	}

	h := newHarness(t)
	func() {
		saved := Version
		Version = previous
		defer func() { Version = saved }()
		h.initBacklog()
	}()

	code, stdout, _ := h.run("validate", "--json")
	if code != 0 {
		t.Errorf("a staleness warning must not fail the run, exit = %d", code)
	}
	var report validate.Report
	decode(t, stdout, &report)

	seen := map[string]bool{}
	for _, f := range report.Findings {
		for _, name := range []string{"backlog-capture", "backlog-triage"} {
			if strings.Contains(f.Message, name+" skill") && strings.Contains(f.Message, previous) {
				seen[name] = true
				if f.Severity != "warning" {
					t.Errorf("%s severity = %q, want warning", name, f.Severity)
				}
				if !strings.Contains(f.Message, "backlog init") {
					t.Errorf("the %s warning does not name the refresh command: %s", name, f.Message)
				}
			}
		}
	}
	for _, name := range []string{"backlog-capture", "backlog-triage"} {
		if !seen[name] {
			t.Errorf("the %s skill from v%s was not reported as stale", name, previous)
		}
	}
}
