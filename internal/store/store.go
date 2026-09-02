// Package store implements the on-disk backlog: finding it, reading it, and
// modifying it without ever leaving a task file half written.
package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/weldnor/backlog/internal/hooks"
	"github.com/weldnor/backlog/internal/task"
)

// Directory names. They are fixed: there is no configuration file, so a
// backlog looks the same in every project.
const (
	DirName  = ".backlog"
	TasksDir = "tasks"
)

// ErrNotFound reports that no backlog exists in the working directory or any
// of its ancestors.
var ErrNotFound = errors.New("no backlog found in this directory or any parent; run 'backlog init' to create one")

// ErrNoSuchTask reports a task identifier that is not present in the task
// directory.
var ErrNoSuchTask = errors.New("no such task")

// Store is one project's backlog.
type Store struct {
	// Root is the absolute path of the .backlog directory.
	Root string
	// Project is the directory containing .backlog, used as the working
	// directory for git provenance.
	Project string
}

// Discover walks up from start to the nearest ancestor containing a .backlog
// directory, so that a command works from anywhere inside a project the way
// git does. The nearest backlog wins, which lets a nested project keep its own.
func Discover(start string) (*Store, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		candidate := filepath.Join(dir, DirName)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return &Store{Root: candidate, Project: dir}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNotFound
		}
		dir = parent
	}
}

// hooksReadme is dropped into a freshly created hooks directory so that a
// backlog explains the mechanism where anyone browsing the directory will
// find it, rather than only in the project README. It is written once, at
// Init, and never overwritten - a project is free to replace it, and re-init
// must not clobber that.
const hooksReadme = `Drop a script here named for the event it should run on, and backlog will
run it, with no configuration beyond the file existing. Two kinds:

    post-add    after ` + "`backlog add`" + ` creates a task
    post-set    after ` + "`backlog set`" + ` changes status, priority, reason or refs
    post-edit   after ` + "`backlog edit`" + ` or ` + "`backlog tag`" + ` changes title, description or tags
    post-rm     after ` + "`backlog rm`" + ` deletes a task

    pre-add     before add; a non-zero exit stops the task from being created
    pre-set     before set; a non-zero exit stops the change
    pre-edit    before edit or tag; a non-zero exit stops the change
    pre-rm      before rm; a non-zero exit stops the task from being deleted

A post- hook is best-effort: it can observe a change, never block or undo
one, so a failing or missing post- hook never fails the backlog command that
triggered it. A pre- hook is a gate: it runs before anything is written, and
a non-zero exit - or a hook that exists but could not be run at all, such as
a .ps1 with no PowerShell installed - aborts the command with nothing
changed and reports why.

The task is passed as JSON on stdin and as BACKLOG_TASK_* environment
variables (ID, TITLE, STATUS, PRIORITY, TAGS, FILE), alongside BACKLOG_EVENT,
BACKLOG_ROOT and BACKLOG_PROJECT. For pre-add, the id and file are not yet
assigned, since add claims them atomically as it writes the task; both
variables are empty there. pre-set and pre-edit also carry BACKLOG_NEW_*
variables describing the change being proposed (e.g. BACKLOG_NEW_STATUS),
alongside the task's current, unmodified state.

To work on both Linux and Windows without maintaining two scripts, name the
file for how it should run - the first one found for an event wins:

    post-add        an executable script with its own shebang (Unix only)
    post-add.ps1     PowerShell - pwsh if present, else Windows PowerShell
    post-add.sh      run explicitly with sh (needs sh on PATH)
    post-add.cmd     run with cmd /C (needs cmd on PATH)
    post-add.bat     same as .cmd
    post-add.exe     run directly

A .ps1 hook is usually the one script that works everywhere: Windows carries
PowerShell out of the box, and pwsh (PowerShell 7+) runs on Linux and macOS
too.
`

// Init creates the backlog directory structure under dir. It is idempotent:
// re-running it over a backlog that already holds tasks leaves every task
// untouched.
func Init(dir string) (*Store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(abs, DirName)
	if err := os.MkdirAll(filepath.Join(root, TasksDir), 0o755); err != nil {
		return nil, err
	}
	hooksDir := hooks.Dir(root)
	if _, err := os.Stat(hooksDir); errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(hooksDir, 0o755); err != nil {
			return nil, err
		}
		readme := filepath.Join(hooksDir, "README")
		if _, err := os.Stat(readme); errors.Is(err, fs.ErrNotExist) {
			if err := os.WriteFile(readme, []byte(hooksReadme), 0o644); err != nil {
				return nil, err
			}
		}
	} else if err != nil {
		return nil, err
	}
	return &Store{Root: root, Project: abs}, nil
}

// TasksPath is the one directory a backlog's tasks live in, regardless of
// status.
func (s *Store) TasksPath() string { return filepath.Join(s.Root, TasksDir) }

// Entry is one file found in the task directory, together with whatever could
// be made of it. A file that cannot be parsed is reported rather than skipped,
// so that validate can name it.
type Entry struct {
	Path string
	Name string
	Task *task.Task
	Err  error
}

// Entries reads every task file in the task directory, in ascending identifier
// order.
func (s *Store) Entries() ([]Entry, error) {
	var out []Entry
	names, err := taskFileNames(s.TasksPath())
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		path := filepath.Join(s.TasksPath(), name)
		e := Entry{Path: path, Name: name}
		data, err := os.ReadFile(path)
		if err != nil {
			e.Err = err
		} else if t, err := task.Parse(name, data); err != nil {
			e.Err = err
		} else {
			t.Path = path
			e.Task = t
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		ii, jj := entryID(out[i]), entryID(out[j])
		if ii != jj {
			return ii < jj
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func entryID(e Entry) int {
	if e.Task != nil && e.Task.ID > 0 {
		return e.Task.ID
	}
	if id, ok := task.IDFromFileName(e.Name); ok {
		return id
	}
	return 1 << 30
}

// Tasks returns the readable tasks, discarding files that could not be parsed.
// Commands that operate on tasks use this; validate uses Entries so that it
// can report the failures.
func (s *Store) Tasks() ([]*task.Task, error) {
	entries, err := s.Entries()
	if err != nil {
		return nil, err
	}
	out := make([]*task.Task, 0, len(entries))
	for _, e := range entries {
		if e.Task != nil {
			out = append(out, e.Task)
		}
	}
	return out, nil
}

// Find returns the task with the given identifier.
func (s *Store) Find(id int) (*task.Task, error) {
	tasks, err := s.Tasks()
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, fmt.Errorf("%w: %d", ErrNoSuchTask, id)
}

// StrayFiles lists entries in the task directory that are not task files, so
// that validate can report them.
func (s *Store) StrayFiles() ([]string, error) {
	var out []string
	dir := s.TasksPath()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue // dotfiles are the tool's own scratch space
		}
		if e.IsDir() {
			out = append(out, filepath.Join(dir, name))
			continue
		}
		if _, ok := task.IDFromFileName(name); !ok {
			out = append(out, filepath.Join(dir, name))
		}
	}
	sort.Strings(out)
	return out, nil
}

func taskFileNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if _, ok := task.IDFromFileName(e.Name()); ok {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// UsedIDs returns the identifiers currently taken in the task directory, as
// read from file names. File names are used rather than frontmatter because
// allocation has to be cheap and the two agree in any backlog that validates.
func (s *Store) UsedIDs() (map[int]bool, error) {
	used := map[int]bool{}
	names, err := taskFileNames(s.TasksPath())
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if id, ok := task.IDFromFileName(name); ok {
			used[id] = true
		}
	}
	return used, nil
}
