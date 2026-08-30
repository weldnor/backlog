package store

import (
	"os/exec"
	"strings"

	"github.com/weldnor/backlog/internal/task"
)

// Provenance records the branch and commit a finding was observed on.
//
// It degrades silently: a project that is not a git repository, or a machine
// without git, simply yields nothing. Capture is a hot path, and failing a
// capture because provenance was unavailable would be the wrong trade.
func Provenance(dir string) task.Source {
	var src task.Source
	if branch, ok := git(dir, "rev-parse", "--abbrev-ref", "HEAD"); ok && branch != "HEAD" {
		src.Branch = branch
	}
	if commit, ok := git(dir, "rev-parse", "HEAD"); ok {
		src.Commit = commit
	}
	return src
}

func git(dir string, args ...string) (string, bool) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", false
	}
	return s, true
}
