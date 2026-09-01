package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/weldnor/backlog/internal/validate"
)

func settingsPath(h *harness) string {
	return h.path(".claude", "settings.json")
}

func TestInitInstallsBothHooksStampedWithTheVersion(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	data, err := os.ReadFile(settingsPath(h))
	if err != nil {
		t.Fatalf("settings.json was not written: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "backlog validate --strict") {
		t.Error("the Stop hook running validate was not installed")
	}
	if !strings.Contains(text, "backlog-capture") {
		t.Error("the SessionStart hook mentioning backlog-capture was not installed")
	}
	if !strings.Contains(text, "managed by backlog v"+Version+";") {
		t.Errorf("the hooks do not carry the running version %s", Version)
	}

	// It has to actually be valid JSON a Claude Code settings file can load.
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v", err)
	}
	if _, ok := root["hooks"]; !ok {
		t.Error("settings.json has no \"hooks\" key")
	}
}

func TestInitNoHooksSkipsHookInstallation(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init", "--no-hooks")
	if _, err := os.Stat(settingsPath(h)); !os.IsNotExist(err) {
		t.Errorf("settings.json should not exist with --no-hooks, stat err = %v", err)
	}
}

func TestInitSkipsALocallyEditedHook(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	path := settingsPath(h)

	edited := strings.Replace(readTaskFile(t, path), "backlog validate --strict", "backlog validate --strict --fix", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := h.run("init")
	if code != 0 {
		t.Fatalf("init exited %d: %s", code, stderr)
	}
	if readTaskFile(t, path) != edited {
		t.Error("the local edit to a hook was overwritten without --force")
	}
	if !strings.Contains(stderr, "--force") {
		t.Errorf("the skip was not reported with a way out: %q", stderr)
	}
}

func TestInitForceReplacesALocallyEditedHook(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	path := settingsPath(h)
	edited := strings.Replace(readTaskFile(t, path), "backlog validate --strict", "backlog validate --strict --fix", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := h.run("init", "--force")
	if code != 0 {
		t.Fatalf("init --force exited %d: %s", code, stderr)
	}
	if strings.Contains(readTaskFile(t, path), "--fix") {
		t.Error("the local edit survived --force")
	}
	if !strings.Contains(stderr, "local edits") {
		t.Errorf("the overwrite was not warned about: %q", stderr)
	}
}

func TestInitPreservesHandWrittenSettings(t *testing.T) {
	h := newHarness(t)
	h.mustRun("init", "--no-hooks")
	path := settingsPath(h)
	if err := os.MkdirAll(h.path(".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"permissions": {"allow": ["Bash(go test ./...)"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	h.mustRun("init")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "go test ./...") {
		t.Error("an unrelated setting was lost when hooks were installed")
	}
	if !strings.Contains(string(data), "backlog validate --strict") {
		t.Error("the hook was not added alongside the existing setting")
	}
}

func TestValidateWarnsAboutStaleHooks(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	t.Run("current hooks produce no warning", func(t *testing.T) {
		var report validate.Report
		decode(t, h.mustRun("validate", "--json"), &report)
		for _, f := range report.Findings {
			if strings.Contains(f.Message, "hook") {
				t.Errorf("unexpected hook finding: %s", f.Message)
			}
		}
	})

	t.Run("an older hook is reported", func(t *testing.T) {
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
			if !strings.Contains(f.Message, "hook") {
				continue
			}
			found = true
			if f.Severity != "warning" {
				t.Errorf("severity = %q, want warning", f.Severity)
			}
			if !strings.Contains(f.Message, "backlog init") {
				t.Errorf("the warning does not name the refresh command: %s", f.Message)
			}
		}
		if !found {
			t.Error("no hook staleness warning was produced")
		}
	})
}
