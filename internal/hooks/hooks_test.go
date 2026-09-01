package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readSettings(t *testing.T, dir string) map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(SettingsPath)))
	if err != nil {
		t.Fatal(err)
	}
	root := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatal(err)
	}
	return root
}

func commandsFor(t *testing.T, dir, event string) []string {
	t.Helper()
	root := readSettings(t, dir)
	hooksRaw := map[string]json.RawMessage{}
	if err := json.Unmarshal(root["hooks"], &hooksRaw); err != nil {
		t.Fatal(err)
	}
	var blocks []hookBlock
	raw, ok := hooksRaw[event]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, b := range blocks {
		for _, h := range b.Hooks {
			out = append(out, h.Command)
		}
	}
	return out
}

func TestInstallWritesBothHooksIntoANewSettingsFile(t *testing.T) {
	dir := t.TempDir()
	results, err := Install(dir, "1.0.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for _, r := range results {
		if r.Action != Written {
			t.Errorf("%s: action = %q, want written", r.ID, r.Action)
		}
	}

	stop := commandsFor(t, dir, "Stop")
	if len(stop) != 1 || !strings.HasPrefix(stop[0], "backlog validate --strict") {
		t.Errorf("Stop hooks = %v", stop)
	}
	start := commandsFor(t, dir, "SessionStart")
	if len(start) != 1 || !strings.Contains(start[0], "backlog-capture") {
		t.Errorf("SessionStart hooks = %v", start)
	}
}

// A hook lives in a file the CLI does not otherwise own, so installing it must
// not disturb whatever else a project has already put there.
func TestInstallPreservesUnrelatedSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := `{
  "permissions": {"allow": ["Bash(npm test)"]},
  "hooks": {
    "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "echo pre"}]}],
    "Stop": [{"hooks": [{"type": "command", "command": "echo someone-elses-hook"}]}]
  }
}`
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}

	root := readSettings(t, dir)
	if !strings.Contains(string(root["permissions"]), "npm test") {
		t.Error("an unrelated top-level key was lost")
	}
	stop := commandsFor(t, dir, "Stop")
	found := map[string]bool{}
	for _, c := range stop {
		found[c] = true
	}
	if !found["echo someone-elses-hook"] {
		t.Error("an unrelated Stop hook was dropped")
	}
	var hasValidate bool
	for c := range found {
		if strings.HasPrefix(c, "backlog validate --strict") {
			hasValidate = true
		}
	}
	if !hasValidate {
		t.Error("backlog's Stop hook was not added alongside the existing one")
	}

	hooksRaw := map[string]json.RawMessage{}
	if err := json.Unmarshal(root["hooks"], &hooksRaw); err != nil {
		t.Fatal(err)
	}
	if _, ok := hooksRaw["PreToolUse"]; !ok {
		t.Error("an unrelated hook event was dropped")
	}
}

func TestInstallRefreshesAnUnmodifiedHook(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	results, err := Install(dir, "1.1.0", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Action != Refreshed {
			t.Errorf("%s: action = %q, want refreshed", r.ID, r.Action)
		}
	}
	stop := commandsFor(t, dir, "Stop")
	if len(stop) != 1 || !strings.Contains(stop[0], "v1.1.0") {
		t.Errorf("Stop hooks = %v, want the v1.1.0 marker", stop)
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	results, err := Install(dir, "1.0.0", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Action != Unchanged {
			t.Errorf("%s: action = %q, want unchanged", r.ID, r.Action)
		}
	}
}

func TestInstallSkipsAModifiedHook(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".claude", "settings.json")
	data, _ := os.ReadFile(path)
	edited := strings.Replace(string(data), "backlog validate --strict", "backlog validate --strict --fix", 1)
	if edited == string(data) {
		t.Fatal("test setup did not actually change the file")
	}
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Install(dir, "1.1.0", false)
	if err != nil {
		t.Fatal(err)
	}
	var sawSkipped, sawRefreshed bool
	for _, r := range results {
		switch r.ID {
		case "validate":
			if r.Action != Skipped {
				t.Errorf("validate: action = %q, want skipped", r.Action)
			}
			sawSkipped = true
		case "session-context":
			if r.Action != Refreshed {
				t.Errorf("session-context: action = %q, want refreshed", r.Action)
			}
			sawRefreshed = true
		}
	}
	if !sawSkipped || !sawRefreshed {
		t.Fatalf("results = %+v", results)
	}
	stop := commandsFor(t, dir, "Stop")
	if len(stop) != 1 || !strings.Contains(stop[0], "--fix") {
		t.Errorf("the local edit was not preserved: %v", stop)
	}
}

func TestInstallForceOverwritesAModifiedHook(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".claude", "settings.json")
	data, _ := os.ReadFile(path)
	edited := strings.Replace(string(data), "backlog validate --strict", "backlog validate --strict --fix", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Install(dir, "1.1.0", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.ID == "validate" && r.Action != Overwritten {
			t.Errorf("validate: action = %q, want overwritten", r.Action)
		}
	}
	stop := commandsFor(t, dir, "Stop")
	if len(stop) != 1 || strings.Contains(stop[0], "--fix") {
		t.Errorf("the local edit survived a forced overwrite: %v", stop)
	}
}

func TestStaleReportsOnlyOlderHooks(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	if stale, err := Stale(dir, "1.0.0"); err != nil || len(stale) != 0 {
		t.Errorf("Stale at the same version = %v, %v; want none", stale, err)
	}
	stale, err := Stale(dir, "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 2 {
		t.Fatalf("Stale at a newer version returned %d, want 2", len(stale))
	}
	if stale, err := Stale(dir, "0.9.0"); err != nil || len(stale) != 0 {
		t.Errorf("Stale against an older binary = %v, %v; want none", stale, err)
	}
}

func TestStaleIgnoresAModifiedHook(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".claude", "settings.json")
	data, _ := os.ReadFile(path)
	edited := strings.Replace(string(data), "backlog validate --strict", "backlog validate --strict --fix", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err := Stale(dir, "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range stale {
		if s.ID == "validate" {
			t.Error("a hand-edited hook was reported as merely stale")
		}
	}
}

func TestInstallOnAbsentSettingsCreatesTheDirectory(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "settings.json")); err != nil {
		t.Fatal(err)
	}
}

func TestOlderThan(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.1.0", "1.0.0", false},
		{"custom", "1.0.0", false},
	}
	for _, c := range cases {
		if got := olderThan(c.a, c.b); got != c.want {
			t.Errorf("olderThan(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
