package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/weldnor/backlog/internal/task"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	st, err := Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return st
}

func add(t *testing.T, st *Store, title string) *task.Task {
	t.Helper()
	tk := task.New(title, "", nil, nil, nil, task.AuthorAgent, task.DefaultPriority, task.Source{}, time.Now())
	if err := st.Create(tk); err != nil {
		t.Fatalf("Create(%q): %v", title, err)
	}
	return tk
}

func TestDiscoverFromNestedDirectory(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "internal", "session")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	st, err := Discover(nested)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if st.Root != filepath.Join(root, DirName) {
		t.Errorf("Root = %q, want the backlog at the project root", st.Root)
	}
}

func TestDiscoverReportsNoBacklog(t *testing.T) {
	// t.TempDir sits under the system temp directory, which has no backlog in
	// it or above it.
	if _, err := Discover(t.TempDir()); err == nil {
		t.Fatal("expected an error when there is no backlog anywhere in the tree")
	} else if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestDiscoverPrefersTheNearestBacklog(t *testing.T) {
	outer := t.TempDir()
	if _, err := Init(outer); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "vendor", "nested-project")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(inner); err != nil {
		t.Fatal(err)
	}

	st, err := Discover(inner)
	if err != nil {
		t.Fatal(err)
	}
	if st.Root != filepath.Join(inner, DirName) {
		t.Errorf("Root = %q, want the nested backlog", st.Root)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	st := newStore(t)
	tk := add(t, st, "An existing task")
	before, err := os.ReadFile(tk.Path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Init(st.Project); err != nil {
		t.Fatalf("second Init: %v", err)
	}

	after, err := os.ReadFile(tk.Path)
	if err != nil {
		t.Fatalf("the task did not survive a second init: %v", err)
	}
	if string(after) != string(before) {
		t.Error("re-running init changed an existing task")
	}
}

func TestCreateAllocatesTheLowestFreeIdentifier(t *testing.T) {
	st := newStore(t)
	first := add(t, st, "First")
	if first.ID != 1 {
		t.Fatalf("first task got id %d, want 1", first.ID)
	}
	if filepath.Base(first.Path) != "001-first.md" {
		t.Errorf("file name = %q, want 001-first.md", filepath.Base(first.Path))
	}

	for i := 2; i <= 9; i++ {
		add(t, st, "Filler")
	}
	if _, err := st.Remove(4); err != nil {
		t.Fatal(err)
	}

	// The gap left by 4 is reused before 10 is reached.
	reused := add(t, st, "Reused")
	if reused.ID != 4 {
		t.Errorf("id = %d, want the gap at 4 to be reused", reused.ID)
	}
}

func TestCreateHandlesTitlesThatReduceToNothing(t *testing.T) {
	st := newStore(t)
	tk := add(t, st, "!!! ???")
	if filepath.Base(tk.Path) != "001.md" {
		t.Errorf("file name = %q, want 001.md", filepath.Base(tk.Path))
	}
	if _, err := st.Find(1); err != nil {
		t.Errorf("the task is not findable: %v", err)
	}
}

// Identifiers are claimed by an exclusive create, so parallel captures cannot
// collide however closely they are interleaved.
func TestConcurrentCreatesGetDistinctIdentifiers(t *testing.T) {
	st := newStore(t)
	const n = 12

	var wg sync.WaitGroup
	ids := make([]int, n)
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			tk := task.New("Concurrent finding", "", nil, nil, nil, task.AuthorAgent, task.DefaultPriority, task.Source{}, time.Now())
			if err := st.Create(tk); err != nil {
				errs[i] = err
				return
			}
			ids[i] = tk.ID
		}(i)
	}
	close(start)
	wg.Wait()

	seen := map[int]bool{}
	for i, id := range ids {
		if errs[i] != nil {
			t.Fatalf("Create: %v", errs[i])
		}
		if id == 0 {
			t.Fatalf("goroutine %d produced no identifier", i)
		}
		if seen[id] {
			t.Errorf("identifier %d was allocated twice", id)
		}
		seen[id] = true
	}

	// No write was lost: every task is on disk and readable.
	tasks, err := st.Tasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != n {
		t.Errorf("found %d tasks on disk, want %d", len(tasks), n)
	}
}

func TestSaveMovesTaskBetweenDirectories(t *testing.T) {
	st := newStore(t)
	tk := add(t, st, "Travelling task")

	tk.Status = task.StatusDone
	if err := st.Save(tk); err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(tk.Path) != st.ArchivePath() {
		t.Errorf("a done task is at %q, want the archive", tk.Path)
	}
	if _, err := os.Stat(filepath.Join(st.TasksPath(), "001-travelling-task.md")); !os.IsNotExist(err) {
		t.Error("the task was left behind in the active directory")
	}
	if filepath.Base(tk.Path) != "001-travelling-task.md" {
		t.Errorf("the file name changed on the move: %q", filepath.Base(tk.Path))
	}

	tk.Status = task.StatusTodo
	if err := st.Save(tk); err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(tk.Path) != st.TasksPath() {
		t.Errorf("a todo task is at %q, want the active directory", tk.Path)
	}
	if _, err := os.Stat(filepath.Join(st.ArchivePath(), "001-travelling-task.md")); !os.IsNotExist(err) {
		t.Error("the task was left behind in the archive")
	}
}

func TestSaveMovesDeclinedTaskToTheArchive(t *testing.T) {
	st := newStore(t)
	tk := add(t, st, "Declined task")

	tk.Status = task.StatusDeclined
	tk.Reason = "not worth the churn"
	if err := st.Save(tk); err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(tk.Path) != st.ArchivePath() {
		t.Errorf("a declined task is at %q, want the archive", tk.Path)
	}
	// It has to be discoverable there, not merely written there.
	found, err := st.Find(tk.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found.Status != task.StatusDeclined || found.Reason != "not worth the churn" {
		t.Errorf("read back status %q reason %q", found.Status, found.Reason)
	}
	if _, err := os.Stat(filepath.Join(st.TasksPath(), "001-declined-task.md")); !os.IsNotExist(err) {
		t.Error("the task was left behind in the active directory")
	}
}

func TestSaveRenamesWhenTheTitleChanges(t *testing.T) {
	st := newStore(t)
	tk := add(t, st, "Old title")
	tk.Title = "New title"
	if err := st.Save(tk); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(tk.Path) != "001-new-title.md" {
		t.Errorf("file name = %q, want 001-new-title.md", filepath.Base(tk.Path))
	}
	names, err := taskFileNames(st.TasksPath())
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 {
		t.Errorf("the old file was left behind: %v", names)
	}
}

func TestWriteFileAtomicLeavesNoPartialFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "task.md")
	if err := WriteFileAtomic(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("second")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "second" {
		t.Errorf("content = %q, want the complete new version", got)
	}
	// The temporary file must not be left lying around for the scan to find.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("directory holds %d entries, want only the task file", len(entries))
	}
}

func TestFindAcrossBothDirectories(t *testing.T) {
	st := newStore(t)
	active := add(t, st, "Active")
	archived := add(t, st, "Archived")
	archived.Status = task.StatusDone
	if err := st.Save(archived); err != nil {
		t.Fatal(err)
	}

	if got, err := st.Find(active.ID); err != nil || got.Title != "Active" {
		t.Errorf("Find(active) = %v, %v", got, err)
	}
	if got, err := st.Find(archived.ID); err != nil || got.Title != "Archived" {
		t.Errorf("Find(archived) = %v, %v", got, err)
	}
	if _, err := st.Find(99); err == nil {
		t.Error("expected an error for an unknown identifier")
	}
}

func TestProvenanceInAGitRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-b", "trunk"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"commit", "--allow-empty", "-m", "first"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v failed, skipping: %v: %s", args, err, out)
		}
	}

	src := Provenance(dir)
	if src.Branch != "trunk" {
		t.Errorf("Branch = %q, want trunk", src.Branch)
	}
	if len(src.Commit) < 7 {
		t.Errorf("Commit = %q, want a commit hash", src.Commit)
	}
}

// Capture must never fail because provenance was unavailable.
func TestProvenanceOutsideAGitRepositoryIsSilent(t *testing.T) {
	src := Provenance(t.TempDir())
	if !src.Empty() {
		t.Errorf("Provenance outside a repository returned %+v, want nothing", src)
	}
}

func TestStrayFiles(t *testing.T) {
	st := newStore(t)
	add(t, st, "A real task")
	if err := os.WriteFile(filepath.Join(st.TasksPath(), "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(st.TasksPath(), ".keep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	strays, err := st.StrayFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(strays) != 1 || filepath.Base(strays[0]) != "notes.txt" {
		t.Errorf("StrayFiles = %v, want only notes.txt", strays)
	}
}
