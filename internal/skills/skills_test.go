package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllThreeSkillsAreEmbedded(t *testing.T) {
	all := All()
	if len(all) != 3 {
		t.Fatalf("got %d skills, want the capture, sort and triage skills", len(all))
	}
	names := []string{all[0].Name, all[1].Name, all[2].Name}
	if names[0] != "backlog-capture" || names[1] != "backlog-sort" || names[2] != "backlog-triage" {
		t.Fatalf("skills = %v", names)
	}
	for _, s := range all {
		if len(s.body) < 500 {
			t.Errorf("%s looks empty (%d bytes)", s.Name, len(s.body))
		}
	}
}

// The description is what the model matches against to decide whether to load
// a skill, so the three triggers have to stay distinct.
func TestSkillsCarryDistinctTriggers(t *testing.T) {
	all := All()
	capture, sortSkill, triage := all[0].body, all[1].body, all[2].body

	for _, want := range []string{"name: backlog-capture", "description:"} {
		if !strings.Contains(capture, want) {
			t.Errorf("the capture skill is missing %q", want)
		}
	}
	if !strings.Contains(sortSkill, "name: backlog-sort") {
		t.Error("the sort skill is missing its name")
	}
	if !strings.Contains(triage, "name: backlog-triage") {
		t.Error("the triage skill is missing its name")
	}

	captureDesc := descriptionOf(t, capture)
	sortDesc := descriptionOf(t, sortSkill)
	triageDesc := descriptionOf(t, triage)
	if captureDesc == triageDesc || captureDesc == sortDesc || sortDesc == triageDesc {
		t.Fatal("two skills share a description")
	}
	if !strings.Contains(captureDesc, "Record") {
		t.Errorf("the capture description does not describe recording: %q", captureDesc)
	}
	if !strings.Contains(triageDesc, "Review") {
		t.Errorf("the triage description does not describe reviewing: %q", triageDesc)
	}
	// Each description has to say when *not* to fire, or the triggers blur.
	for name, desc := range map[string]string{"capture": captureDesc, "sort": sortDesc, "triage": triageDesc} {
		if !strings.Contains(desc, "Do not use") {
			t.Errorf("the %s description does not say when not to use it", name)
		}
	}
}

// backlog-sort only ever closes what the branch already fixed; anything else
// is left for backlog-triage, so the two must not overlap in what they decide.
func TestSortSkillNeverDecidesWhatTriageDecides(t *testing.T) {
	body := All()[1].body
	for _, want := range []string{"git log", "done", "metadata.source", "--ref"} {
		if !strings.Contains(body, want) {
			t.Errorf("the sort skill does not cover %q", want)
		}
	}
	for _, forbidden := range []string{"backlog set <id> declined", "--priority"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("the sort skill makes a triage-level call: %q", forbidden)
		}
	}
}

// Half of the capture skill is about when *not* to record. Without an explicit
// threshold an agent files every passing observation.
func TestCaptureSkillStatesTheThreshold(t *testing.T) {
	body := All()[0].body
	for _, want := range []string{
		"outside the scope",
		"not fixing it now",
		"repository, not this session",
		"Stylistic preferences",
		"Speculative refactoring",
		"already covered by work in progress",
		"backlog search",
		"--file",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the capture skill does not cover %q", want)
		}
	}
	if !strings.Contains(body, "Always search first") {
		t.Error("the capture skill does not require a search before adding")
	}
}

// All knowledge of planning systems lives in the triage skill, and even there
// it is detected at run time rather than hard-coded.
func TestTriageSkillIsPlanningSystemAgnostic(t *testing.T) {
	body := All()[2].body
	for _, want := range []string{"run time", "backlog set", "--ref", "free-form"} {
		if !strings.Contains(body, want) {
			t.Errorf("the triage skill does not cover %q", want)
		}
	}
	for _, forbidden := range []string{"openspec ", "gh issue", "jira", "linear.app"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Errorf("the triage skill hard-codes the planning system %q", forbidden)
		}
	}
}

func TestRenderStampsTheVersion(t *testing.T) {
	s := All()[0]
	out := s.Render("1.2.3")
	if !strings.Contains(out, "managed by backlog v1.2.3;") {
		t.Errorf("the version stamp is missing:\n%s", firstLines(out, 6))
	}
	// The marker sits after the frontmatter so it does not disturb the trigger.
	if idx := strings.Index(out, "<!-- managed by backlog"); idx < 0 || idx < strings.Index(out, "description:") {
		t.Error("the marker is not placed after the frontmatter")
	}
	if !strings.HasPrefix(out, "---\n") {
		t.Error("the frontmatter no longer opens the file")
	}
}

func TestInstallWritesBothSkillsIntoTheProject(t *testing.T) {
	dir := t.TempDir()
	results, err := Install(dir, "1.0.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	for _, r := range results {
		if r.Action != Written {
			t.Errorf("%s: action = %q, want written", r.Name, r.Action)
		}
		// The files go into the project so that they are committed with it.
		want := filepath.Join(dir, ".claude", "skills", r.Name, "SKILL.md")
		if r.Path != want {
			t.Errorf("path = %q, want %q", r.Path, want)
		}
		data, err := os.ReadFile(r.Path)
		if err != nil {
			t.Fatalf("%s was not written: %v", r.Name, err)
		}
		if !strings.Contains(string(data), "managed by backlog v1.0.0;") {
			t.Errorf("%s does not carry the version", r.Name)
		}
	}
}

func TestInstallRefreshesAnUnmodifiedSkill(t *testing.T) {
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
			t.Errorf("%s: action = %q, want refreshed", r.Name, r.Action)
		}
		data, _ := os.ReadFile(r.Path)
		if !strings.Contains(string(data), "managed by backlog v1.1.0;") {
			t.Errorf("%s was not brought up to the new version", r.Name)
		}
	}
}

func TestInstallSkipsAModifiedSkill(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	path := All()[0].Path(dir)
	edited := mustRead(t, path) + "\n## A section someone added by hand\n"
	write(t, path, edited)

	results, err := Install(dir, "1.1.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Action != Skipped {
		t.Errorf("action = %q, want skipped", results[0].Action)
	}
	if got := mustRead(t, path); got != edited {
		t.Error("the locally edited skill was changed")
	}
	// The other skill is untouched by its neighbour's edit.
	if results[1].Action != Refreshed {
		t.Errorf("the second skill: action = %q, want refreshed", results[1].Action)
	}
}

func TestInstallForceOverwritesAModifiedSkill(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	path := All()[0].Path(dir)
	write(t, path, mustRead(t, path)+"\nlocal edit\n")

	results, err := Install(dir, "1.1.0", true)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Action != Overwritten {
		t.Errorf("action = %q, want overwritten", results[0].Action)
	}
	if strings.Contains(mustRead(t, path), "local edit") {
		t.Error("the local edit survived a forced overwrite")
	}
}

func TestInspectDetectsModification(t *testing.T) {
	dir := t.TempDir()
	if _, err := Install(dir, "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	path := All()[0].Path(dir)

	if state, version, err := Inspect(path); err != nil || state != Current || version != "1.0.0" {
		t.Fatalf("Inspect = %v, %q, %v; want Current, 1.0.0", state, version, err)
	}
	write(t, path, strings.Replace(mustRead(t, path), "The threshold", "The bar", 1))
	if state, _, err := Inspect(path); err != nil || state != Modified {
		t.Fatalf("Inspect after an edit = %v, %v; want Modified", state, err)
	}
	if state, _, err := Inspect(filepath.Join(dir, "nowhere", "SKILL.md")); err != nil || state != Absent {
		t.Fatalf("Inspect on a missing file = %v, %v; want Absent", state, err)
	}
}

func TestStaleReportsOnlyOlderSkills(t *testing.T) {
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
	if len(stale) != 3 {
		t.Fatalf("Stale at a newer version returned %d, want 3", len(stale))
	}
	if string(stale[0].Action) != "1.0.0" {
		t.Errorf("the installed version was not reported: %q", stale[0].Action)
	}
	// A skill ahead of the binary is not stale.
	if stale, err := Stale(dir, "0.9.0"); err != nil || len(stale) != 0 {
		t.Errorf("Stale against an older binary = %v, %v; want none", stale, err)
	}
}

func TestOlderThan(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.1.0", true},
		{"0.9.0", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.1.0", "1.0.0", false},
		{"1.0", "1.0.1", true},
		{"1.0.0-dev", "1.0.0", false},
		{"custom", "1.0.0", false}, // an unparseable build never warns
	}
	for _, c := range cases {
		if got := olderThan(c.a, c.b); got != c.want {
			t.Errorf("olderThan(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func descriptionOf(t *testing.T, body string) string {
	t.Helper()
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	t.Fatal("no description in the frontmatter")
	return ""
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
