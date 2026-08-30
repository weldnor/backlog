package task

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ErrNoFrontmatter reports a file that does not open with a YAML frontmatter
// block. Unlike the deviations collected as issues this is fatal: there is
// nothing to read.
var ErrNoFrontmatter = errors.New("file does not start with a YAML frontmatter block")

const frontmatterFence = "---"

// Parse reads a task file. name is the base file name, used to recover an
// identifier when the frontmatter has lost one.
//
// Parsing fails only when the frontmatter is absent or is not valid YAML.
// Every other deviation — a missing field, an unknown status, a stray key — is
// recorded on the returned task's Issues so that the file stays usable and
// validate is what reports it.
func Parse(name string, data []byte) (*Task, error) {
	front, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(front, &doc); err != nil {
		return nil, fmt.Errorf("frontmatter is not valid YAML: %w", err)
	}

	t := &Task{Body: string(body)}

	var root *yaml.Node
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		root = doc.Content[0]
	}
	if root == nil {
		// An empty frontmatter block: everything is missing, but the file is
		// still readable.
		root = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	} else if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("frontmatter is not a mapping")
	}
	t.front = root

	t.readID(root, name)
	t.readTitle(root)
	t.readStatus(root)
	t.readPriority(root)
	t.readReason(root)
	t.checkReasonPairing()
	t.readTags(root)
	t.readMetadata(root)
	t.notePreservedKeys(root)

	return t, nil
}

// splitFrontmatter separates the YAML block from the markdown body. The body
// is returned verbatim, so that a write which does not touch the description
// reproduces it byte for byte.
func splitFrontmatter(data []byte) (front, body []byte, err error) {
	// A byte-order mark is tolerated so that a file saved by a Windows editor
	// still reads.
	rest := bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	line, after, ok := cutLine(rest)
	if !ok || strings.TrimRight(string(line), "\r") != frontmatterFence {
		return nil, nil, ErrNoFrontmatter
	}

	var fm bytes.Buffer
	for {
		line, after, ok = cutLine(after)
		if !ok {
			return nil, nil, fmt.Errorf("frontmatter block is not closed by %q", frontmatterFence)
		}
		trimmed := strings.TrimRight(string(line), "\r")
		if trimmed == frontmatterFence || trimmed == "..." {
			return fm.Bytes(), after, nil
		}
		fm.Write(line)
		fm.WriteByte('\n')
	}
}

// cutLine splits off the first line, returning it without its terminator.
func cutLine(b []byte) (line, rest []byte, ok bool) {
	if len(b) == 0 {
		return nil, nil, false
	}
	i := bytes.IndexByte(b, '\n')
	if i < 0 {
		return b, nil, true
	}
	return b[:i], b[i+1:], true
}

func (t *Task) issue(sev Severity, repairable bool, format string, args ...any) {
	t.Issues = append(t.Issues, Issue{
		Severity:   sev,
		Repairable: repairable,
		Message:    fmt.Sprintf(format, args...),
	})
}

func (t *Task) readID(root *yaml.Node, name string) {
	n, ok := mapGet(root, "id")
	if !ok || n.Kind != yaml.ScalarNode || n.Value == "" || n.Tag == "!!null" {
		if id, ok := IDFromFileName(name); ok {
			t.ID = id
			t.issue(SeverityError, true, "id is missing; recovered %d from the file name", id)
		} else {
			t.issue(SeverityError, false, "id is missing and cannot be recovered from the file name")
		}
		return
	}
	id, err := strconv.Atoi(n.Value)
	if err != nil || id <= 0 {
		t.issue(SeverityError, false, "id must be a positive integer, found %q", n.Value)
		if recovered, ok := IDFromFileName(name); ok {
			t.ID = recovered
		}
		return
	}
	t.ID = id
}

func (t *Task) readTitle(root *yaml.Node) {
	n, ok := mapGet(root, "title")
	if !ok {
		t.issue(SeverityError, false, "title is missing")
		return
	}
	if n.Kind != yaml.ScalarNode {
		t.issue(SeverityError, false, "title must be a string")
		return
	}
	t.Title = n.Value
	if strings.TrimSpace(t.Title) == "" {
		t.issue(SeverityError, false, "title is empty")
	}
	if strings.ContainsAny(t.Title, "\n\r") {
		t.issue(SeverityError, false, "title must be a single line")
	}
}

func (t *Task) readStatus(root *yaml.Node) {
	n, ok := mapGet(root, "status")
	if !ok {
		t.issue(SeverityError, false, "status is missing")
		return
	}
	if n.Kind != yaml.ScalarNode {
		t.issue(SeverityError, false, "status must be a string")
		return
	}
	t.Status = n.Value
	if !ValidStatus(t.Status) {
		t.issue(SeverityError, false, "status is %q, expected one of %s", t.Status, strings.Join(Statuses, ", "))
	}
}

func (t *Task) readPriority(root *yaml.Node) {
	n, ok := mapGet(root, "priority")
	if !ok {
		// A file written before the field existed, or by hand without it, is
		// still perfectly readable: the default is what it meant. Recording it
		// is a convention, and one --fix can settle without a judgement.
		t.Priority = DefaultPriority
		t.issue(SeverityWarning, true, "priority is missing; read as %s", DefaultPriority)
		return
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		t.Priority = DefaultPriority
		t.issue(SeverityWarning, true, "priority is empty; read as %s", DefaultPriority)
		return
	}
	if n.Kind != yaml.ScalarNode {
		t.issue(SeverityError, false, "priority must be a string")
		return
	}
	// The raw value is kept even when it is not one of the permitted ones, the
	// way an unrecognised status is: replacing what someone deliberately typed
	// is a judgement, so it is reported rather than corrected.
	t.Priority = n.Value
	if !ValidPriority(t.Priority) {
		t.issue(SeverityError, false, "priority is %q, expected one of %s", t.Priority, strings.Join(Priorities, ", "))
	}
}

func (t *Task) readReason(root *yaml.Node) {
	n, ok := mapGet(root, "reason")
	if !ok {
		return
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		// An empty value is the same situation as no value at all; whether
		// that is a problem depends on the status, which the pairing check
		// below decides.
		return
	}
	if n.Kind != yaml.ScalarNode {
		t.issue(SeverityError, false, "reason must be a string")
		return
	}
	// Stored verbatim: it is prose a person wrote, and nothing about it is the
	// tool's to normalise.
	t.Reason = n.Value
}

// checkReasonPairing enforces that a reason is present exactly when the status
// is declined. Neither direction is repairable: one needs prose written and
// the other needs prose deleted, and both are judgements.
func (t *Task) checkReasonPairing() {
	declined := t.Status == StatusDeclined
	empty := strings.TrimSpace(t.Reason) == ""
	switch {
	case declined && empty:
		t.issue(SeverityError, false,
			"status is %s but no reason is recorded; a decline has to say why", StatusDeclined)
	case !declined && !empty:
		t.issue(SeverityError, false,
			"reason is recorded but the status is %q; a reason applies only to a %s task", t.Status, StatusDeclined)
	}
}

func (t *Task) readTags(root *yaml.Node) {
	n, ok := mapGet(root, "tags")
	if !ok {
		return
	}
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return
	}
	if n.Kind != yaml.SequenceNode {
		t.issue(SeverityError, false, "tags must be a list of strings")
		return
	}
	seen := map[string]bool{}
	for _, item := range n.Content {
		if item.Kind != yaml.ScalarNode {
			t.issue(SeverityError, false, "tags must contain only strings")
			continue
		}
		if item.Value == "" {
			t.issue(SeverityError, false, "tags contains an empty entry")
			continue
		}
		if seen[item.Value] {
			t.issue(SeverityWarning, true, "tags lists %q more than once", item.Value)
			continue
		}
		seen[item.Value] = true
		t.Tags = append(t.Tags, item.Value)
	}
}

func (t *Task) readMetadata(root *yaml.Node) {
	n, ok := mapGet(root, "metadata")
	if !ok {
		t.issue(SeverityWarning, false, "metadata is missing")
		return
	}
	if n.Kind != yaml.MappingNode {
		t.issue(SeverityError, false, "metadata must be a mapping")
		return
	}
	t.hasMeta = true

	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i].Value
		val := n.Content[i+1]
		switch key {
		case "schema":
			v, err := strconv.Atoi(val.Value)
			if err != nil || v <= 0 {
				t.issue(SeverityError, false, "metadata.schema must be a positive integer, found %q", val.Value)
				continue
			}
			t.Meta.Schema = v
		case "created":
			t.Meta.Created = val.Value
			t.checkTimestamp(val.Value)
		case "author":
			t.Meta.Author = val.Value
			if val.Value != AuthorAgent && val.Value != AuthorHuman {
				t.issue(SeverityError, false, "metadata.author is %q, expected %s or %s", val.Value, AuthorAgent, AuthorHuman)
			}
		case "source":
			t.readSource(val)
		case "refs":
			t.readRefs(val)
		default:
			msg := fmt.Sprintf("metadata.%s is not a permitted key", key)
			if s, ok := nearest(key, MetadataKeys); ok {
				msg += fmt.Sprintf("; did you mean %s?", s)
			}
			t.issue(SeverityError, false, "%s", msg)
		}
	}

	if !hasKey(n, "schema") {
		t.issue(SeverityWarning, true, "metadata.schema is missing")
	}
	if !hasKey(n, "created") {
		t.issue(SeverityWarning, false, "metadata.created is missing")
	}
}

func (t *Task) checkTimestamp(v string) {
	if v == "" {
		t.issue(SeverityError, false, "metadata.created is empty")
		return
	}
	if _, err := time.Parse(time.RFC3339, v); err == nil {
		return
	}
	if _, ok := parseLooseTime(v); ok {
		t.issue(SeverityWarning, true, "metadata.created is %q, which is not RFC 3339 formatting", v)
		return
	}
	t.issue(SeverityError, false, "metadata.created is %q, which is not a valid RFC 3339 timestamp", v)
}

// parseLooseTime accepts the timestamp spellings a YAML author or another tool
// might reasonably produce, so that repair can normalise them rather than
// making the author retype the value.
func parseLooseTime(v string) (time.Time, bool) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04Z07:00",
		"2006-01-02",
	}
	for _, l := range layouts {
		if ts, err := time.Parse(l, v); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

func (t *Task) readSource(n *yaml.Node) {
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return
	}
	if n.Kind != yaml.MappingNode {
		t.issue(SeverityError, false, "metadata.source must be a mapping")
		return
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i].Value
		val := n.Content[i+1]
		switch key {
		case "files":
			if val.Kind == yaml.ScalarNode && val.Tag == "!!null" {
				continue
			}
			if val.Kind != yaml.SequenceNode {
				t.issue(SeverityError, false, "metadata.source.files must be a list of strings")
				continue
			}
			for _, item := range val.Content {
				if item.Kind != yaml.ScalarNode || item.Value == "" {
					t.issue(SeverityError, false, "metadata.source.files contains an empty entry")
					continue
				}
				t.Meta.Source.Files = append(t.Meta.Source.Files, item.Value)
			}
		case "branch":
			t.Meta.Source.Branch = val.Value
		case "commit":
			t.Meta.Source.Commit = val.Value
		default:
			msg := fmt.Sprintf("metadata.source.%s is not a permitted key", key)
			if s, ok := nearest(key, SourceKeys); ok {
				msg += fmt.Sprintf("; did you mean %s?", s)
			}
			t.issue(SeverityError, false, "%s", msg)
		}
	}
}

func (t *Task) readRefs(n *yaml.Node) {
	if n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return
	}
	if n.Kind != yaml.SequenceNode {
		t.issue(SeverityError, false, "metadata.refs must be a list of strings")
		return
	}
	t.Meta.Refs = []string{}
	for _, item := range n.Content {
		if item.Kind != yaml.ScalarNode {
			t.issue(SeverityError, false, "metadata.refs must contain only strings")
			continue
		}
		// References are stored verbatim and never resolved; the only thing
		// worth checking is that there is something there.
		if strings.TrimSpace(item.Value) == "" {
			t.issue(SeverityError, false, "metadata.refs contains an empty reference")
			continue
		}
		t.Meta.Refs = append(t.Meta.Refs, item.Value)
	}
}

// notePreservedKeys records the top-level keys the CLI does not understand.
// The top level is author-owned, so an unknown key is room to experiment, not
// an error: it is preserved on write and reported only as a warning.
func (t *Task) notePreservedKeys(root *yaml.Node) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		if known(key, TopLevelKeys) {
			continue
		}
		t.issue(SeverityWarning, false, "%s is not a field the CLI understands; it is preserved unchanged", key)
	}
}

func mapGet(m *yaml.Node, key string) (*yaml.Node, bool) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1], true
		}
	}
	return nil, false
}

func hasKey(m *yaml.Node, key string) bool {
	_, ok := mapGet(m, key)
	return ok
}

func known(key string, set []string) bool {
	for _, k := range set {
		if k == key {
			return true
		}
	}
	return false
}

// nearest returns the permitted key closest to key, when one is close enough
// that the difference is plausibly a typo.
func nearest(key string, set []string) (string, bool) {
	best, bestDist := "", 1<<30
	for _, k := range set {
		d := editDistance(strings.ToLower(key), k)
		if d < bestDist {
			best, bestDist = k, d
		}
	}
	limit := 2
	if len([]rune(key)) <= 4 {
		limit = 1
	}
	if bestDist == 0 || bestDist > limit {
		return "", false
	}
	return best, true
}

func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	cur := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		cur[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(br)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
