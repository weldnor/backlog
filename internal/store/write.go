package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/weldnor/backlog/internal/task"
)

// Create allocates an identifier for t and writes it.
//
// The identifier is the lowest positive integer not already in use. The race
// two processes would otherwise run is settled without a lock file: the
// identifier is claimed by creating the file exclusively under a name that
// depends only on the identifier, and the slug is attached by a rename
// afterwards. Exclusive create is atomic on every target filesystem, so two
// processes can never simultaneously hold that bare-numeric claim file - but
// once one of them renames it to its slugged final name, the bare name is
// free again, and a process that started its scan earlier can reclaim it. The
// post-claim existence check therefore catches that stale claim before it
// overwrites the file that already won the identifier. If a process dies
// between the claim and the rename, what it leaves behind is a complete, valid
// task file whose name validate reports as drifted and --fix corrects.
func (s *Store) Create(t *task.Task) error {
	used, err := s.UsedIDs()
	if err != nil {
		return err
	}
	dir := s.TasksPath()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	for id := 1; ; id++ {
		if used[id] {
			continue
		}
		claim := filepath.Join(dir, fmt.Sprintf("%03d.md", id))
		f, err := os.OpenFile(claim, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return err
		}

		// A concurrent process could have already finished claiming this
		// identifier: it renamed its own claim file from the bare NNN.md we
		// just recreated to its slugged final name, which frees NNN.md for us
		// to reclaim here. Renaming onto that final name ourselves would
		// silently overwrite its file, so we re-scan the directory for the
		// identifier before committing.
		if taken, err := s.idTakenByAnotherFile(id, filepath.Base(claim)); err != nil || taken {
			f.Close()
			os.Remove(claim)
			if err != nil {
				return err
			}
			continue
		}

		t.ID = id
		if err := writeAndClose(f, t.Bytes()); err != nil {
			os.Remove(claim)
			return err
		}

		final := filepath.Join(dir, t.FileName())
		if final != claim {
			if err := os.Rename(claim, final); err != nil {
				return err
			}
		}
		t.Path = final
		return nil
	}
}

func (s *Store) idTakenByAnotherFile(id int, claimName string) (bool, error) {
	names, err := taskFileNames(s.TasksPath())
	if err != nil {
		return false, err
	}
	for _, name := range names {
		if name == claimName {
			continue // the bare claim file we just created, not a collision
		}
		if got, ok := task.IDFromFileName(name); ok && got == id {
			return true, nil
		}
	}
	return false, nil
}

// Save writes t to the task directory, renaming the file in place when the
// title changed. The write is atomic, so a task file is never observed
// partially written and a failure leaves the previous content intact.
func (s *Store) Save(t *task.Task) error {
	dir := s.TasksPath()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(dir, t.FileName())
	if err := WriteFileAtomic(target, t.Bytes()); err != nil {
		return err
	}
	if t.Path != "" && t.Path != target {
		if err := os.Remove(t.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	t.Path = target
	return nil
}

// Remove deletes the task with the given identifier.
func (s *Store) Remove(id int) (*task.Task, error) {
	t, err := s.Find(id)
	if err != nil {
		return nil, err
	}
	if err := os.Remove(t.Path); err != nil {
		return nil, err
	}
	return t, nil
}

// WriteFileAtomic writes data to path via a temporary file in the same
// directory followed by a rename, so a reader sees either the whole previous
// version or the whole new one.
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	// The temporary name is a dotfile so that a crash mid-write leaves nothing
	// the scan or the stray-file check will trip over.
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if err := writeAndClose(f, data); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func writeAndClose(f *os.File, data []byte) error {
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
