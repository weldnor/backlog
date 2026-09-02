package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// binary is the compiled CLI. The end-to-end tests drive the real executable,
// with no terminal on standard input, because that is how an agent runs it.
var binary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "backlog-e2e-")
	if err != nil {
		panic(err)
	}
	binary = filepath.Join(dir, "backlog")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		panic("building the binary failed: " + err.Error() + "\n" + string(out))
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type cliResult struct {
	code   int
	stdout string
	stderr string
}

func backlog(t *testing.T, dir string, args ...string) cliResult {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir

	// No terminal on standard input: a command that tried to prompt would
	// block or fail here rather than quietly succeeding under a test harness.
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer devNull.Close()
	cmd.Stdin = devNull

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running %v: %v", args, err)
	}
	return cliResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func mustBacklog(t *testing.T, dir string, args ...string) string {
	t.Helper()
	got := backlog(t, dir, args...)
	if got.code != 0 {
		t.Fatalf("backlog %v exited %d: %s", args, got.code, got.stderr)
	}
	return got.stdout
}

// gitProject creates a temporary git repository, or skips when git is not
// available.
func gitProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	for _, args := range [][]string{
		{"init", "-b", "trunk"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"commit", "--allow-empty", "-m", "first"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v failed: %v: %s", args, err, out)
		}
	}
	return dir
}

type taskJSON struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Reason      string   `json:"reason"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	Metadata    struct {
		Schema  int    `json:"schema"`
		Created string `json:"created"`
		Author  string `json:"author"`
		Source  struct {
			Files  []string `json:"files"`
			Branch string   `json:"branch"`
			Commit string   `json:"commit"`
		} `json:"source"`
		Refs []string `json:"refs"`
	} `json:"metadata"`
	File string `json:"file"`
}

// The whole point of the tool, start to finish: an agent captures a finding
// mid-task, someone finds it again later, works it, and closes it out.
func TestCaptureThroughToDone(t *testing.T) {
	dir := gitProject(t)

	mustBacklog(t, dir, "init")

	// Capture, from a subdirectory, the way an agent would while working.
	nested := filepath.Join(dir, "internal", "session")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	mustBacklog(t, nested, "add", "Race in session cache",
		"--description", "Two goroutines write the map without a lock.",
		"--tag", "bug", "--tag", "concurrency",
		"--file", "internal/session/cache.go")

	// Provenance was recorded automatically.
	var created taskJSON
	unmarshal(t, mustBacklog(t, dir, "show", "1", "--json"), &created)
	if created.Metadata.Source.Branch != "trunk" {
		t.Errorf("branch = %q, want trunk", created.Metadata.Source.Branch)
	}
	if len(created.Metadata.Source.Commit) < 7 {
		t.Errorf("commit = %q, want a hash", created.Metadata.Source.Commit)
	}
	if created.Metadata.Author != "agent" {
		t.Errorf("author = %q", created.Metadata.Author)
	}

	// Someone later looks for it before filing something similar.
	var results []struct {
		Task    taskJSON `json:"task"`
		Matches []struct {
			Field string `json:"field"`
			Text  string `json:"text"`
		} `json:"matches"`
	}
	unmarshal(t, mustBacklog(t, dir, "search", "session cache", "--json"), &results)
	if len(results) != 1 || results[0].Task.ID != 1 {
		t.Fatalf("search returned %d results, want the captured task", len(results))
	}
	if results[0].Matches[0].Field != "title" {
		t.Errorf("matched field = %q, want title", results[0].Matches[0].Field)
	}

	// Work starts.
	mustBacklog(t, dir, "set", "1", "doing")
	if _, err := os.Stat(filepath.Join(dir, ".backlog", "tasks", "001-race-in-session-cache.md")); err != nil {
		t.Errorf("a task in progress left the tasks directory: %v", err)
	}

	// Work finishes, with a link back to wherever it was done.
	mustBacklog(t, dir, "set", "1", "done", "--ref", "change:fix-session-cache")
	if _, err := os.Stat(filepath.Join(dir, ".backlog", "tasks", "001-race-in-session-cache.md")); err != nil {
		t.Fatalf("a done task left the tasks directory: %v", err)
	}

	// It is still in the default listing — every status is shown — and readable
	// by identifier.
	if out := mustBacklog(t, dir, "list"); !strings.Contains(out, "Race in session cache") {
		t.Errorf("a done task is missing from the default listing:\n%s", out)
	}
	if out := mustBacklog(t, dir, "list", "todo"); strings.Contains(out, "Race in session cache") {
		t.Errorf("a done task appears under `list todo`:\n%s", out)
	}
	var done taskJSON
	unmarshal(t, mustBacklog(t, dir, "show", "1", "--json"), &done)
	if done.Status != "done" {
		t.Errorf("status = %q", done.Status)
	}
	if strings.Join(done.Metadata.Refs, ",") != "change:fix-session-cache" {
		t.Errorf("refs = %v", done.Metadata.Refs)
	}
	if done.Description != "Two goroutines write the map without a lock." {
		t.Errorf("the description did not survive the round trip: %q", done.Description)
	}

	// And the whole thing is still consistent.
	if got := backlog(t, dir, "validate"); got.code != 0 {
		t.Errorf("validate exited %d:\n%s%s", got.code, got.stdout, got.stderr)
	}
}

func TestFreshBacklogValidatesCleanAndCorruptionIsCaught(t *testing.T) {
	dir := t.TempDir()
	mustBacklog(t, dir, "init")
	mustBacklog(t, dir, "add", "A well formed finding", "--tag", "bug")
	mustBacklog(t, dir, "add", "Another one")
	mustBacklog(t, dir, "set", "2", "done")

	t.Run("clean", func(t *testing.T) {
		got := backlog(t, dir, "validate")
		if got.code != 0 {
			t.Fatalf("a freshly created backlog does not validate: exit %d\n%s%s", got.code, got.stdout, got.stderr)
		}
		if !strings.Contains(got.stdout, "no problems found") {
			t.Errorf("stdout = %q", got.stdout)
		}
	})

	t.Run("corrupted", func(t *testing.T) {
		tasks := filepath.Join(dir, ".backlog", "tasks")

		// One file the parser cannot read at all.
		write(t, filepath.Join(tasks, "003-unreadable.md"), "---\nid: 3\ntitle: [unclosed\n---\n")
		// One with a typo in a tool-owned key and a status nobody recognises.
		write(t, filepath.Join(tasks, "004-mangled.md"),
			"---\nid: 4\ntitle: Mangled\nstatus: blocked\ntags: []\nmetadata:\n  schema: 1\n  creted: 2026-08-30T20:59:51Z\n---\n")
		// One whose identifier collides with an existing task.
		write(t, filepath.Join(tasks, "001-a-copy.md"),
			"---\nid: 1\ntitle: A copy\nstatus: todo\ntags: []\nmetadata:\n  schema: 1\n  created: 2026-08-30T20:59:51Z\n  refs: []\n---\n")
		// And a file that has no business being there.
		write(t, filepath.Join(tasks, "scratch.txt"), "notes\n")

		got := backlog(t, dir, "validate", "--json")
		if got.code == 0 {
			t.Fatalf("a corrupted backlog validated clean:\n%s", got.stdout)
		}
		if got.stderr == "" {
			t.Error("nothing was written to stderr")
		}

		var report struct {
			Findings []struct {
				File       string `json:"file"`
				Severity   string `json:"severity"`
				Message    string `json:"message"`
				Repairable bool   `json:"repairable"`
			} `json:"findings"`
			Errors   int `json:"errors"`
			Warnings int `json:"warnings"`
		}
		unmarshal(t, got.stdout, &report)

		want := []string{
			"not valid YAML",             // 003, unreadable
			"expected one of new",        // 004, unknown status
			"did you mean created?",      // 004, misspelled tool-owned key
			"used by more than one task", // 001, duplicate identifier
			"not a task file",            // scratch.txt
		}
		for _, w := range want {
			if !anyMessage(report.Findings, w) {
				t.Errorf("no finding mentions %q; got %+v", w, report.Findings)
			}
		}
		if report.Errors < 4 {
			t.Errorf("errors = %d, want at least 4", report.Errors)
		}
		if report.Warnings < 1 {
			t.Errorf("warnings = %d, want at least 1", report.Warnings)
		}

		// Repair must not touch the findings that need a judgement.
		before := read(t, filepath.Join(tasks, "003-unreadable.md"))
		fixed := backlog(t, dir, "validate", "--fix")
		if fixed.code == 0 {
			t.Error("--fix silently resolved findings it cannot repair")
		}
		if read(t, filepath.Join(tasks, "003-unreadable.md")) != before {
			t.Error("--fix rewrote a file it could not parse")
		}
		if _, err := os.Stat(filepath.Join(tasks, "001-a-copy.md")); err != nil {
			t.Error("--fix moved a task with a duplicate identifier")
		}
	})
}

// The identifier race the exclusive-create claim exists to settle, run across
// real processes rather than goroutines.
func TestParallelProcessesGetDistinctIdentifiers(t *testing.T) {
	dir := t.TempDir()
	mustBacklog(t, dir, "init")

	const n = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			cmd := exec.Command(binary, "add", "A concurrent finding")
			cmd.Dir = dir
			_ = cmd.Run()
		}()
	}
	close(start)
	wg.Wait()

	var tasks []taskJSON
	unmarshal(t, mustBacklog(t, dir, "list", "--json"), &tasks)
	if len(tasks) != n {
		t.Fatalf("got %d tasks, want %d — a write was lost", len(tasks), n)
	}
	seen := map[int]bool{}
	for _, task := range tasks {
		if seen[task.ID] {
			t.Errorf("identifier %d was allocated twice", task.ID)
		}
		seen[task.ID] = true
	}
	if got := backlog(t, dir, "validate"); got.code != 0 {
		t.Errorf("the backlog does not validate after parallel captures:\n%s", got.stdout)
	}
}

func anyMessage(findings []struct {
	File       string `json:"file"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Repairable bool   `json:"repairable"`
}, substr string) bool {
	for _, f := range findings {
		if strings.Contains(f.Message, substr) {
			return true
		}
	}
	return false
}

func unmarshal(t *testing.T, data string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), v); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, data)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Priority travels with a task from the moment it is captured: through the
// listing, its filter and its order, through a revision that touches nothing
// else, and out the other side in both output forms.
func TestPriorityThroughTheLifecycle(t *testing.T) {
	dir := t.TempDir()
	mustBacklog(t, dir, "init")

	// Captured without a judgement, and captured with one.
	var defaulted taskJSON
	unmarshal(t, mustBacklog(t, dir, "add", "Unjudged finding", "--json"), &defaulted)
	if defaulted.Priority != "medium" {
		t.Errorf("a task added without --priority has priority %q, want medium", defaulted.Priority)
	}
	var urgent taskJSON
	unmarshal(t, mustBacklog(t, dir, "add", "Data loss on retry", "--priority", "high", "--json"), &urgent)
	if urgent.Priority != "high" {
		t.Errorf("priority = %q, want high", urgent.Priority)
	}
	mustBacklog(t, dir, "add", "Cosmetic wording", "--priority", "low")

	// It is on disk as an author-owned field, next to the status.
	onDisk := read(t, filepath.Join(dir, ".backlog", "tasks", "002-data-loss-on-retry.md"))
	if !strings.Contains(onDisk, "\nstatus: new\npriority: high\n") {
		t.Errorf("the file does not declare the priority after the status:\n%s", onDisk)
	}

	// The listing is one sequence, most severe first, identifier breaking ties.
	var listed []taskJSON
	unmarshal(t, mustBacklog(t, dir, "list", "--json"), &listed)
	var order []string
	for _, task := range listed {
		order = append(order, task.Priority)
	}
	if strings.Join(order, ",") != "high,medium,low" {
		t.Errorf("list order = %v, want descending priority", order)
	}
	if listed[0].ID != 2 {
		t.Errorf("the high-priority task is not first: %+v", listed[0])
	}

	// Human output carries it too, on the task line.
	if out := mustBacklog(t, dir, "list"); !strings.Contains(out, "high") || !strings.Contains(out, "low") {
		t.Errorf("the listing does not show priorities:\n%s", out)
	}

	// The filter selects by severity, and several values are a disjunction.
	var filtered []taskJSON
	unmarshal(t, mustBacklog(t, dir, "list", "--priority", "high", "--json"), &filtered)
	if len(filtered) != 1 || filtered[0].ID != 2 {
		t.Fatalf("--priority high returned %+v", filtered)
	}
	unmarshal(t, mustBacklog(t, dir, "list", "--priority", "high", "--priority", "low", "--json"), &filtered)
	if len(filtered) != 2 {
		t.Errorf("two --priority values returned %d tasks, want a disjunction of 2", len(filtered))
	}

	// A revision that changes only the severity leaves the status alone.
	var revised taskJSON
	unmarshal(t, mustBacklog(t, dir, "set", "1", "--priority", "high", "--json"), &revised)
	if revised.Priority != "high" || revised.Status != "new" {
		t.Errorf("set --priority gave %+v, want a high new task", revised)
	}
	if _, err := os.Stat(filepath.Join(dir, ".backlog", "tasks", "001-unjudged-finding.md")); err != nil {
		t.Errorf("changing only the priority moved the file: %v", err)
	}

	// And show reports it in both forms.
	var shown taskJSON
	unmarshal(t, mustBacklog(t, dir, "show", "1", "--json"), &shown)
	if shown.Priority != "high" {
		t.Errorf("show --json priority = %q, want high", shown.Priority)
	}
	if out := mustBacklog(t, dir, "show", "1"); !strings.Contains(out, "priority high") {
		t.Errorf("show does not print the priority:\n%s", out)
	}

	if got := backlog(t, dir, "validate", "--strict"); got.code != 0 {
		t.Errorf("validate --strict exited %d:\n%s%s", got.code, got.stdout, got.stderr)
	}
}

// A backlog written before priority existed keeps working untouched, and one
// --fix run brings the whole thing up to the new convention.
func TestBacklogWithoutPriorityMigratesWithFix(t *testing.T) {
	dir := t.TempDir()
	mustBacklog(t, dir, "init")

	tasks := filepath.Join(dir, ".backlog", "tasks")
	for _, f := range []struct{ name, title string }{
		{"001-an-older-finding.md", "An older finding"},
		{"002-another-older-finding.md", "Another older finding"},
	} {
		write(t, filepath.Join(tasks, f.name),
			"---\nid: "+f.name[2:3]+"\ntitle: "+f.title+"\nstatus: todo\ntags: []\n"+
				"metadata:\n  schema: 1\n  created: 2026-08-30T20:59:51Z\n  author: agent\n  refs: []\n---\nBody.\n")
	}

	// Readable as it stands: every task is treated as medium.
	var listed []taskJSON
	unmarshal(t, mustBacklog(t, dir, "list", "--json"), &listed)
	if len(listed) != 2 {
		t.Fatalf("got %d tasks, want 2", len(listed))
	}
	for _, task := range listed {
		if task.Priority != "medium" {
			t.Errorf("task %d reads as %q, want medium", task.ID, task.Priority)
		}
	}

	// Reported, but only as a repairable warning, so a commit hook still passes.
	got := backlog(t, dir, "validate", "--json")
	if got.code != 0 {
		t.Fatalf("a priority-less backlog failed validation: exit %d\n%s", got.code, got.stdout)
	}
	var report struct {
		Findings []struct {
			File       string `json:"file"`
			Severity   string `json:"severity"`
			Message    string `json:"message"`
			Repairable bool   `json:"repairable"`
		} `json:"findings"`
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
	}
	unmarshal(t, got.stdout, &report)
	var warned int
	for _, f := range report.Findings {
		if !strings.Contains(f.Message, "priority") {
			continue
		}
		warned++
		if f.Severity != "warning" {
			t.Errorf("%s: severity = %q, want warning", f.File, f.Severity)
		}
		if !f.Repairable {
			t.Errorf("%s: the missing priority is not marked repairable", f.File)
		}
	}
	if warned != 2 {
		t.Errorf("%d files were reported for a missing priority, want 2", warned)
	}
	if report.Errors != 0 {
		t.Errorf("errors = %d, want none", report.Errors)
	}

	// One run brings them up to the convention.
	if fixed := backlog(t, dir, "validate", "--fix"); fixed.code != 0 {
		t.Fatalf("--fix exited %d:\n%s%s", fixed.code, fixed.stdout, fixed.stderr)
	}
	entries, err := os.ReadDir(tasks)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		content := read(t, filepath.Join(tasks, e.Name()))
		if !strings.Contains(content, "\npriority: medium\n") {
			t.Errorf("%s still declares no priority:\n%s", e.Name(), content)
		}
	}

	if strict := backlog(t, dir, "validate", "--strict"); strict.code != 0 {
		t.Errorf("validate --strict exited %d after the repair:\n%s%s", strict.code, strict.stdout, strict.stderr)
	}
}

// The loop a decline exists to close: a finding is recorded, weighed, declined
// with its reasoning, and is still found by the search an agent runs before
// recording the same thing again.
func TestDeclineThroughTheLifecycle(t *testing.T) {
	dir := t.TempDir()
	mustBacklog(t, dir, "init")
	mustBacklog(t, dir, "add", "Race in session cache",
		"--description", "Two goroutines write the map without a lock.")

	// Triage weighs it and decides against it. The reason is not optional.
	if got := backlog(t, dir, "set", "1", "declined"); got.code == 0 {
		t.Error("a decline with no reason was accepted")
	}
	var declined taskJSON
	unmarshal(t, mustBacklog(t, dir, "set", "1", "declined",
		"--reason", "the cache has a single writer; the race cannot happen", "--json"), &declined)
	if declined.Status != "declined" {
		t.Errorf("status = %q, want declined", declined.Status)
	}

	// A status change never moves the file: the declined task stays in
	// .backlog/tasks/ and carries the reasoning with it.
	taskFile := filepath.Join(dir, ".backlog", "tasks", "001-race-in-session-cache.md")
	onDisk := read(t, taskFile)
	if !strings.Contains(onDisk, "status: declined") {
		t.Errorf("the file does not declare the status:\n%s", onDisk)
	}
	if !strings.Contains(onDisk, "reason: the cache has a single writer; the race cannot happen") {
		t.Errorf("the file does not carry the reason:\n%s", onDisk)
	}

	// The next agent to walk into the same code searches before recording, and
	// finds the decision: search covers every status.
	var results []struct {
		Task taskJSON `json:"task"`
	}
	unmarshal(t, mustBacklog(t, dir, "search", "session cache", "--json"), &results)
	if len(results) != 1 || results[0].Task.ID != 1 {
		t.Fatalf("the plain search did not find the declined task: %v", results)
	}
	if results[0].Task.Status != "declined" {
		t.Errorf("the result reports status %q, want declined", results[0].Task.Status)
	}
	if results[0].Task.Reason == "" {
		t.Error("the result carries no reason, so the caller cannot say why it was declined")
	}

	// A person reading the backlog sees the declined task, grouped last.
	var listed []taskJSON
	unmarshal(t, mustBacklog(t, dir, "list", "--json"), &listed)
	if len(listed) != 1 || listed[0].ID != 1 {
		t.Errorf("list did not show the declined task by default: %v", listed)
	}
	if !strings.Contains(mustBacklog(t, dir, "list"), "declined (1)") {
		t.Error("the human listing did not show the declined group")
	}
	if out := mustBacklog(t, dir, "list", "todo"); strings.Contains(out, "Race in session cache") {
		t.Error("`list todo` showed a declined task")
	}

	// Reopening drops the reason — it now describes a state the task is no
	// longer in, and git keeps what it said — without moving the file.
	var reopened taskJSON
	unmarshal(t, mustBacklog(t, dir, "set", "1", "todo", "--json"), &reopened)
	if reopened.Status != "todo" || reopened.Reason != "" {
		t.Errorf("reopened task = %+v, want todo with no reason", reopened)
	}
	active := read(t, taskFile)
	if strings.Contains(active, "reason") {
		t.Errorf("the reopened file still declares a reason:\n%s", active)
	}

	if got := backlog(t, dir, "validate"); got.code != 0 {
		t.Errorf("validate exited %d after the round trip: %s", got.code, got.stdout+got.stderr)
	}
}

// browseServer starts `backlog browse` against dir as a real subprocess and
// returns its base URL, ready to receive requests. It is shut down (SIGINT,
// the same signal Ctrl+C sends) and reaped when the test ends.
func browseServer(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command(binary, "browse", "--port", "0", "--no-open", "--json")
	cmd.Dir = dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting backlog browse: %v", err)
	}

	var v struct {
		URL string `json:"url"`
	}
	dec := json.NewDecoder(stdout)
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("reading the printed URL: %v", err)
	}

	t.Cleanup(func() {
		_ = cmd.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("backlog browse did not exit cleanly: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("backlog browse did not exit within 5s of an interrupt")
			cmd.Process.Kill()
		}
	})
	return v.URL
}

func browseAPI(t *testing.T, method, url string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer res.Body.Close()
	got, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, got
}

// TestBrowseAPIMatchesCLI drives `backlog browse`'s HTTP API directly —
// create, list, and an edit through every status transition, including
// title/description/tags, which only the UI can reach — and asserts the
// resulting task files match what the equivalent `add`/`set` CLI invocations
// produce against a separate, otherwise identical backlog.
func TestBrowseAPIMatchesCLI(t *testing.T) {
	uiDir := gitProject(t)
	mustBacklog(t, uiDir, "init")
	cliDir := gitProject(t)
	mustBacklog(t, cliDir, "init")

	base := browseServer(t, uiDir)

	// Create, the way `add` would.
	createBody := map[string]any{
		"title": "Race in session cache", "description": "Two goroutines write the map without a lock.",
		"tags": []string{"bug", "concurrency"}, "priority": "high",
	}
	status, respBody := browseAPI(t, http.MethodPost, base+"/api/tasks", createBody)
	if status != http.StatusCreated {
		t.Fatalf("POST /api/tasks status = %d: %s", status, respBody)
	}
	var created taskJSON
	unmarshal(t, string(respBody), &created)

	mustBacklog(t, cliDir, "add", "Race in session cache",
		"--description", "Two goroutines write the map without a lock.",
		"--tag", "bug", "--tag", "concurrency", "--priority", "high", "--author", "human")

	// List: the default view should show exactly the one todo task.
	status, respBody = browseAPI(t, http.MethodGet, base+"/api/tasks", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /api/tasks status = %d: %s", status, respBody)
	}
	var listed []taskJSON
	unmarshal(t, string(respBody), &listed)
	if len(listed) != 1 || listed[0].Title != "Race in session cache" {
		t.Fatalf("GET /api/tasks = %+v, want just the one task just created", listed)
	}

	// Edit title, description and tags together — a UI-only capability, per
	// design.md's "Editing is a full-record PATCH" decision.
	patchBody := map[string]any{
		"title":       "Session cache is not safe for concurrent readers",
		"description": "Two goroutines write the map without a lock. The reader path is\nfine, but refresh mutates the same map from a background ticker.",
		"tags":        []string{"bug", "concurrency", "race"},
	}
	taskURL := fmt.Sprintf("%s/api/tasks/%d", base, created.ID)
	status, respBody = browseAPI(t, http.MethodPatch, taskURL, patchBody)
	if status != http.StatusOK {
		t.Fatalf("PATCH title/description/tags status = %d: %s", status, respBody)
	}

	// Every status transition: todo -> doing -> done -> declined -> todo.
	for _, step := range []struct {
		status, reason string
	}{
		{"doing", ""},
		{"done", ""},
		{"declined", "superseded by a different fix"},
		{"todo", ""},
	} {
		body := map[string]any{"status": step.status, "reason": step.reason}
		status, respBody = browseAPI(t, http.MethodPatch, taskURL, body)
		if status != http.StatusOK {
			t.Fatalf("PATCH status=%s failed: %d: %s", step.status, status, respBody)
		}
	}

	mustBacklog(t, cliDir, "set", "1", "doing")
	mustBacklog(t, cliDir, "set", "1", "done")
	mustBacklog(t, cliDir, "set", "1", "declined", "--reason", "superseded by a different fix")
	mustBacklog(t, cliDir, "set", "1", "todo")

	// The browse-only fields have no CLI equivalent to compare against
	// directly, so apply the same edit through the file the CLI wrote, using
	// the same on-disk format both sides produce.
	cliFile := filepath.Join(cliDir, ".backlog", "tasks", "001-race-in-session-cache.md")
	renamed := filepath.Join(cliDir, ".backlog", "tasks", "001-session-cache-is-not-safe-for-concurrent-readers.md")
	if err := os.Rename(cliFile, renamed); err != nil {
		t.Fatal(err)
	}
	src := read(t, renamed)
	src = strings.Replace(src, "title: Race in session cache", "title: Session cache is not safe for concurrent readers", 1)
	src = strings.Replace(src,
		"tags:\n  - bug\n  - concurrency\n",
		"tags:\n  - bug\n  - concurrency\n  - race\n", 1)
	src = strings.Replace(src,
		"Two goroutines write the map without a lock.\n",
		"Two goroutines write the map without a lock. The reader path is\nfine, but refresh mutates the same map from a background ticker.\n", 1)
	if err := os.WriteFile(renamed, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	uiFinal := read(t, filepath.Join(uiDir, ".backlog", "tasks", "001-session-cache-is-not-safe-for-concurrent-readers.md"))
	cliFinal := read(t, renamed)

	// created/schema timestamps are identical only if the two runs land in
	// the same second; compare everything else, which is deterministic.
	strip := func(s string) string {
		lines := strings.Split(s, "\n")
		out := lines[:0]
		for _, l := range lines {
			if strings.Contains(l, "created:") || strings.Contains(l, "commit:") {
				continue
			}
			out = append(out, l)
		}
		return strings.Join(out, "\n")
	}
	if strip(uiFinal) != strip(cliFinal) {
		t.Errorf("browse-produced file does not match the CLI-equivalent sequence:\n--- browse ---\n%s\n--- cli ---\n%s", uiFinal, cliFinal)
	}

	// GET reflects the same state PATCH just wrote.
	status, respBody = browseAPI(t, http.MethodGet, taskURL, nil)
	if status != http.StatusOK {
		t.Fatalf("GET %s status = %d: %s", taskURL, status, respBody)
	}
	var final taskJSON
	unmarshal(t, string(respBody), &final)
	if final.Status != "todo" || final.Reason != "" || len(final.Tags) != 3 {
		t.Errorf("final task = %+v, want todo, no reason, 3 tags", final)
	}
	if final.Metadata.Author != "human" {
		t.Errorf("author = %q, want human: everything created through the UI is", final.Metadata.Author)
	}

	if got := backlog(t, uiDir, "validate"); got.code != 0 {
		t.Errorf("validate exited %d after the browse-driven round trip: %s", got.code, got.stdout+got.stderr)
	}
}

// TestHooksFireOnLifecycleEvents drives the real binary against a project
// with a post-add and a post-set hook installed, and checks that each fires
// with the environment and stdin the hook README promises. Hook scripts are
// shell here because that is what a Unix CI runner has; the interpreter
// selection itself (.ps1, .cmd, bare-with-shebang) is covered directly by
// internal/hooks, which is where the cross-platform logic lives.
func TestHooksFireOnLifecycleEvents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test's hook scripts are shell; internal/hooks covers Windows shapes directly")
	}
	dir := t.TempDir()
	mustBacklog(t, dir, "init")

	hooksDir := filepath.Join(dir, ".backlog", "hooks")
	addLog := filepath.Join(dir, "post-add.log")
	setLog := filepath.Join(dir, "post-set.log")

	writeHook := func(name, body string) {
		if err := os.WriteFile(filepath.Join(hooksDir, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeHook("post-add", "#!/bin/sh\n"+
		"printf '%s %s %s\\n' \"$BACKLOG_EVENT\" \"$BACKLOG_TASK_ID\" \"$BACKLOG_TASK_TITLE\" > \""+addLog+"\"\n"+
		"cat > \""+addLog+".stdin\"\n")
	writeHook("post-set", "#!/bin/sh\n"+
		"printf '%s %s %s->%s\\n' \"$BACKLOG_EVENT\" \"$BACKLOG_TASK_ID\" \"$BACKLOG_PREVIOUS_STATUS\" \"$BACKLOG_TASK_STATUS\" > \""+setLog+"\"\n")

	out := mustBacklog(t, dir, "add", "Something to fix")
	if !strings.Contains(out, "created 001") {
		t.Fatalf("add did not report success: %s", out)
	}

	got, err := os.ReadFile(addLog)
	if err != nil {
		t.Fatalf("post-add hook did not run: %v", err)
	}
	if want := "post-add 1 Something to fix\n"; string(got) != want {
		t.Errorf("post-add saw %q, want %q", got, want)
	}
	stdin, err := os.ReadFile(addLog + ".stdin")
	if err != nil {
		t.Fatalf("post-add hook did not receive stdin: %v", err)
	}
	var fromHook taskJSON
	unmarshal(t, string(stdin), &fromHook)
	if fromHook.Title != "Something to fix" {
		t.Errorf("stdin task title = %q", fromHook.Title)
	}

	mustBacklog(t, dir, "set", "1", "todo")
	got, err = os.ReadFile(setLog)
	if err != nil {
		t.Fatalf("post-set hook did not run: %v", err)
	}
	if want := "post-set 1 new->todo\n"; string(got) != want {
		t.Errorf("post-set saw %q, want %q", got, want)
	}

	// A hook is a side effect: a failing one is reported, but the command it
	// rides on still succeeds.
	writeHook("post-rm", "#!/bin/sh\nexit 7\n")
	res := backlog(t, dir, "rm", "1")
	if res.code != 0 {
		t.Fatalf("rm failed because its hook failed: code=%d stderr=%s", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "post-rm hook failed") {
		t.Errorf("stderr did not report the failing hook: %s", res.stderr)
	}
}
