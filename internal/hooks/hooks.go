// Package hooks installs and maintains the Claude Code hooks that make the
// backlog enforce itself instead of relying on an agent remembering to run it:
// validating the backlog once a turn ends, and reminding an agent at the start
// of a session that backlog-capture exists.
//
// Unlike the skill files, a hook lives inside a project's shared
// `.claude/settings.json`, a file the CLI does not otherwise own. Install
// therefore merges its two managed entries into whatever is already there
// rather than writing the file wholesale, the same way `git config` edits one
// key without disturbing the rest of the file.
package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SettingsPath is where a project's Claude Code hook configuration lives.
const SettingsPath = ".claude/settings.json"

// spec describes one hook backlog manages.
type spec struct {
	// ID is the stable identifier embedded in the command's marker, so a hook
	// can be found again regardless of where it sits in the array.
	ID string
	// Event is the Claude Code hook event this entry belongs under.
	Event string
}

var specs = []spec{
	{ID: "validate", Event: "Stop"},
	{ID: "session-context", Event: "SessionStart"},
}

const sessionContextMessage = "This project keeps a backlog in .backlog/. If, while working, you notice a defect, flaky test or other concrete problem outside the scope of what you were asked to do, use the backlog-capture skill: search the backlog first, then record it, then return to work."

// command renders the exact command string a spec is installed with at a
// given version. Rendering it from the version rather than hand-writing two
// near-identical strings is what lets find tell "stale" (same template, older
// version) apart from "modified" (something other than the version differs).
func command(id, version string) string {
	var cmd string
	switch id {
	case "validate":
		cmd = "backlog validate --strict"
	case "session-context":
		cmd = "echo " + strconv.Quote(sessionContextMessage)
	default:
		panic("hooks: unknown id " + id)
	}
	return cmd + " " + marker(id, version)
}

func marker(id, version string) string {
	return fmt.Sprintf("# managed by backlog v%s; hook: %s", version, id)
}

var markerRE = regexp.MustCompile(`# managed by backlog v([^;]+); hook: ([a-z-]+)$`)

type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hookBlock struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []hookCommand `json:"hooks"`
}

// Action records what Install did to one hook.
type Action string

const (
	// Written means the hook entry was added.
	Written Action = "written"
	// Refreshed means an unmodified entry was rewritten to the current version.
	Refreshed Action = "refreshed"
	// Unchanged means the entry already carried exactly this content.
	Unchanged Action = "unchanged"
	// Skipped means the entry was edited locally and was left alone.
	Skipped Action = "skipped"
	// Overwritten means a locally edited entry was replaced under --force.
	Overwritten Action = "overwritten"
)

// Result is what Install (or Stale) did to, or found about, one hook.
type Result struct {
	ID     string
	Event  string
	Action Action
}

// Install merges backlog's managed hooks into projectDir's
// .claude/settings.json, creating the file if it does not exist. Every other
// key in the file, and every other hook entry under Stop or SessionStart, is
// left untouched.
//
// An entry that has been edited locally is never silently replaced: it is
// skipped, and only an explicit force overwrites it — the same rule Install in
// the skills package applies to a skill file.
func Install(projectDir, version string, force bool) ([]Result, error) {
	path := filepath.Join(projectDir, filepath.FromSlash(SettingsPath))
	root, err := loadSettings(path)
	if err != nil {
		return nil, err
	}

	hooksRaw, err := hooksObject(root)
	if err != nil {
		return nil, err
	}

	var out []Result
	changed := false
	for _, s := range specs {
		blocks, err := blocksFor(hooksRaw, s.Event)
		if err != nil {
			return nil, err
		}

		bi, ci, foundVersion, modified := find(blocks, s.ID)
		want := command(s.ID, version)

		var action Action
		switch {
		case bi < 0:
			blocks = append(blocks, hookBlock{Hooks: []hookCommand{{Type: "command", Command: want}}})
			action = Written
		case modified:
			if !force {
				out = append(out, Result{s.ID, s.Event, Skipped})
				continue
			}
			blocks[bi].Hooks[ci].Command = want
			action = Overwritten
		case foundVersion != version:
			blocks[bi].Hooks[ci].Command = want
			action = Refreshed
		default:
			out = append(out, Result{s.ID, s.Event, Unchanged})
			continue
		}

		raw, err := json.Marshal(blocks)
		if err != nil {
			return nil, err
		}
		hooksRaw[s.Event] = raw
		out = append(out, Result{s.ID, s.Event, action})
		changed = true
	}

	if !changed {
		return out, nil
	}
	raw, err := json.Marshal(hooksRaw)
	if err != nil {
		return nil, err
	}
	root["hooks"] = raw
	return out, writeSettings(path, root)
}

// Stale reports the installed hooks that were produced by a version older
// than the one running, so that validate can tell the author to refresh them.
// A hook that has been edited locally is not reported: it is no longer
// something init can safely bring forward on its own.
func Stale(projectDir, version string) ([]Result, error) {
	path := filepath.Join(projectDir, filepath.FromSlash(SettingsPath))
	root, err := loadSettings(path)
	if err != nil {
		return nil, err
	}
	hooksRaw, err := hooksObject(root)
	if err != nil {
		return nil, err
	}

	var out []Result
	for _, s := range specs {
		blocks, err := blocksFor(hooksRaw, s.Event)
		if err != nil {
			return nil, err
		}
		_, _, foundVersion, modified := find(blocks, s.ID)
		if foundVersion == "" || modified {
			continue
		}
		if olderThan(foundVersion, version) {
			out = append(out, Result{s.ID, s.Event, Action(foundVersion)})
		}
	}
	return out, nil
}

// find locates the managed entry for id within blocks and reports whether its
// command still matches the template for the version its marker names —
// "modified" means someone hand-edited a line backlog wrote.
func find(blocks []hookBlock, id string) (blockIdx, cmdIdx int, version string, modified bool) {
	for bi, b := range blocks {
		for ci, h := range b.Hooks {
			m := markerRE.FindStringSubmatch(h.Command)
			if m == nil || m[2] != id {
				continue
			}
			return bi, ci, m[1], h.Command != command(id, m[1])
		}
	}
	return -1, -1, "", false
}

func loadSettings(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]json.RawMessage{}, nil
	}
	if err != nil {
		return nil, err
	}
	root := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("hooks: %s is not valid JSON: %w", path, err)
	}
	return root, nil
}

func hooksObject(root map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	hooksRaw := map[string]json.RawMessage{}
	raw, ok := root["hooks"]
	if !ok {
		return hooksRaw, nil
	}
	if err := json.Unmarshal(raw, &hooksRaw); err != nil {
		return nil, fmt.Errorf("hooks: %s: \"hooks\" is not an object: %w", SettingsPath, err)
	}
	return hooksRaw, nil
}

func blocksFor(hooksRaw map[string]json.RawMessage, event string) ([]hookBlock, error) {
	raw, ok := hooksRaw[event]
	if !ok {
		return nil, nil
	}
	var blocks []hookBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("hooks: %s: %q hooks are not a list: %w", SettingsPath, event, err)
	}
	return blocks, nil
}

func writeSettings(path string, root map[string]json.RawMessage) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// olderThan compares two version strings numerically component by component,
// falling back to reporting no staleness when a component is not a number, so
// that a custom build string never produces a spurious warning. Duplicated
// from the skills package rather than shared: the two are small, independent,
// and each free to diverge if a future version scheme needs it.
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
