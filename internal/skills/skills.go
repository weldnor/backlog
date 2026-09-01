// Package skills installs and maintains the Claude Code skills that teach an
// agent how to use the backlog.
//
// The guidance lives in these files rather than behind a `backlog instructions`
// command: fetching it at run time would keep it perfectly in step with the
// binary, but at the cost of an extra tool call on the hot path and one more
// thing that can fail. A version stamp plus a staleness warning from validate
// solves the same problem for free.
package skills

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

//go:embed files/*.md
var content embed.FS

// SkillsDir is where a project's Claude Code skills live. The files are
// written into the project rather than into the user's home directory so that
// they are committed alongside the backlog and travel with a clone.
const SkillsDir = ".claude/skills"

// SkillFile is the conventional name of a skill's body.
const SkillFile = "SKILL.md"

// Skill is one installable skill.
type Skill struct {
	Name string
	body string
}

// All returns the skills the CLI installs, in a stable order.
//
// Capture, sort and triage are separate files because a skill's description is
// what the model matches against to decide whether to load it. "Record a
// finding you just hit", "close what the branch already fixed" and "review
// what has accumulated" are three different situations; merged into one
// description the trigger blurs and fires at the wrong times.
func All() []Skill {
	names := []string{"backlog-capture", "backlog-sort", "backlog-triage"}
	out := make([]Skill, 0, len(names))
	for _, name := range names {
		body, err := content.ReadFile("files/" + name + ".md")
		if err != nil {
			panic("skills: embedded file missing: " + name)
		}
		out = append(out, Skill{Name: name, body: string(body)})
	}
	return out
}

// Path is where a skill is installed inside a project.
func (s Skill) Path(projectDir string) string {
	return filepath.Join(projectDir, filepath.FromSlash(SkillsDir), s.Name, SkillFile)
}

const checksumPlaceholder = "pending"

var markerRE = regexp.MustCompile(`(?m)^<!-- managed by backlog v([^;]+); checksum: ([0-9a-f]{64}|` + checksumPlaceholder + `) -->$`)

// Render returns the file content for a skill, carrying the version that
// produced it and a checksum of everything else. The checksum is what lets
// init tell a file it wrote itself from one someone has since edited.
func (s Skill) Render(version string) string {
	body := insertMarker(s.body, version, checksumPlaceholder)
	sum := sha256.Sum256([]byte(body))
	return insertMarker(s.body, version, hex.EncodeToString(sum[:]))
}

// insertMarker places the marker immediately after the YAML frontmatter, where
// it is out of the way of the guidance itself.
func insertMarker(body, version, checksum string) string {
	marker := fmt.Sprintf("<!-- managed by backlog v%s; checksum: %s -->", version, checksum)
	lines := strings.SplitN(body, "\n", 2)
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		return marker + "\n\n" + body
	}
	rest := lines[1]
	if i := strings.Index(rest, "\n---\n"); i >= 0 {
		head := rest[:i+len("\n---\n")]
		return lines[0] + "\n" + head + marker + "\n" + rest[i+len("\n---\n"):]
	}
	return marker + "\n\n" + body
}

// State describes an installed skill file.
type State int

const (
	// Absent means the file is not installed.
	Absent State = iota
	// Current means the file is exactly as some version of the CLI wrote it.
	Current
	// Modified means the file has been edited since it was written, or was not
	// written by the CLI at all.
	Modified
)

// Inspect reports the state of an installed skill and the version it records.
func Inspect(path string) (State, string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Absent, "", nil
	}
	if err != nil {
		return Absent, "", err
	}
	text := string(data)
	m := markerRE.FindStringSubmatch(text)
	if m == nil {
		return Modified, "", nil
	}
	version, sum := m[1], m[2]

	// Recompute over the content with the checksum blanked back out, which is
	// exactly what Render hashed.
	blanked := strings.Replace(text, m[0],
		fmt.Sprintf("<!-- managed by backlog v%s; checksum: %s -->", version, checksumPlaceholder), 1)
	want := sha256.Sum256([]byte(blanked))
	if hex.EncodeToString(want[:]) != sum {
		return Modified, version, nil
	}
	return Current, version, nil
}

// Action records what installing did to one skill file.
type Action string

const (
	// Written means the file was created.
	Written Action = "written"
	// Refreshed means an unmodified file was rewritten to the current version.
	Refreshed Action = "refreshed"
	// Unchanged means the file already carried exactly this content.
	Unchanged Action = "unchanged"
	// Skipped means the file was edited locally and was left alone.
	Skipped Action = "skipped"
	// Overwritten means a locally edited file was replaced under --force.
	Overwritten Action = "overwritten"
)

// Result is what Install did to one skill.
type Result struct {
	Name   string
	Path   string
	Action Action
}

// Install writes the skills into projectDir.
//
// A file that has been edited locally is never silently replaced: it is
// skipped, and only an explicit force overwrites it.
func Install(projectDir, version string, force bool) ([]Result, error) {
	var out []Result
	for _, s := range All() {
		path := s.Path(projectDir)
		state, _, err := Inspect(path)
		if err != nil {
			return nil, err
		}

		action := Written
		switch state {
		case Current:
			action = Refreshed
		case Modified:
			if !force {
				out = append(out, Result{Name: s.Name, Path: path, Action: Skipped})
				continue
			}
			action = Overwritten
		}

		want := s.Render(version)
		if state == Current {
			if existing, err := os.ReadFile(path); err == nil && string(existing) == want {
				out = append(out, Result{Name: s.Name, Path: path, Action: Unchanged})
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			return nil, err
		}
		out = append(out, Result{Name: s.Name, Path: path, Action: action})
	}
	return out, nil
}

// Stale reports the installed skills that were produced by a version older
// than the one running, so that validate can tell the author to refresh them.
func Stale(projectDir, version string) ([]Result, error) {
	var out []Result
	for _, s := range All() {
		path := s.Path(projectDir)
		state, installed, err := Inspect(path)
		if err != nil {
			return nil, err
		}
		if state == Absent || installed == "" {
			continue
		}
		if olderThan(installed, version) {
			out = append(out, Result{Name: s.Name, Path: path, Action: Action(installed)})
		}
	}
	return out, nil
}

// olderThan compares two version strings numerically component by component,
// falling back to reporting no staleness when a component is not a number, so
// that a custom build string never produces a spurious warning.
func olderThan(a, b string) bool {
	as := strings.Split(strings.TrimPrefix(a, "v"), ".")
	bs := strings.Split(strings.TrimPrefix(b, "v"), ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		ai, aok := component(as, i)
		bi, bok := component(bs, i)
		if !aok || !bok {
			return false
		}
		if ai != bi {
			return ai < bi
		}
	}
	return false
}

func component(parts []string, i int) (int, bool) {
	if i >= len(parts) {
		return 0, true
	}
	// Trim any pre-release or build suffix before comparing.
	p := parts[i]
	if j := strings.IndexAny(p, "-+"); j >= 0 {
		p = p[:j]
	}
	v, err := strconv.Atoi(p)
	if err != nil {
		return 0, false
	}
	return v, true
}
