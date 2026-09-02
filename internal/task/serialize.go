package task

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// New builds a task ready to be written. The identifier is assigned by the
// store when the file is created.
func New(title, body string, tags, files, refs []string, author, priority string, src Source, now time.Time) *Task {
	if refs == nil {
		refs = []string{}
	}
	if priority == "" {
		priority = DefaultPriority
	}
	src.Files = files
	return &Task{
		Title:    strings.TrimSpace(title),
		Status:   StatusNew,
		Priority: priority,
		Tags:     NormalizeTags(tags),
		Body:     normalizeBody(body),
		Meta: Metadata{
			Schema:  SchemaVersion,
			Created: now.UTC().Format(time.RFC3339),
			Author:  author,
			Source:  src,
			Refs:    refs,
		},
		hasMeta: true,
	}
}

// normalizeBody gives a freshly supplied description the shape a markdown file
// wants: no leading blank space, and exactly one trailing newline when there
// is anything to end.
func normalizeBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	return body + "\n"
}

// SetBody replaces the description, normalising it the way New does.
func (t *Task) SetBody(body string) { t.Body = normalizeBody(body) }

// AddRef appends a free-form reference. The string is stored verbatim: the CLI
// makes no attempt to interpret or resolve it.
func (t *Task) AddRef(ref string) {
	if t.Meta.Refs == nil {
		t.Meta.Refs = []string{}
	}
	for _, existing := range t.Meta.Refs {
		if existing == ref {
			return
		}
	}
	t.Meta.Refs = append(t.Meta.Refs, ref)
	t.hasMeta = true
}

// metadataPresent reports whether the file should carry a metadata block. A
// hand-written file that has none is left without one rather than gaining one
// as a side effect of an unrelated status change.
func (t *Task) metadataPresent() bool {
	return t.hasMeta ||
		t.Meta.Schema != 0 ||
		t.Meta.Created != "" ||
		t.Meta.Author != "" ||
		!t.Meta.Source.Empty() ||
		t.Meta.Refs != nil
}

// FileName returns the canonical file name for this task.
func (t *Task) FileName() string { return FileName(t.ID, t.Title) }

// Bytes renders the task as a complete markdown file.
//
// The frontmatter is emitted by hand rather than through a YAML encoder so
// that the layout is fixed and predictable: these files are read in diffs and
// edited by hand, and an encoder that reflows quoting or indentation between
// versions would make every unrelated write noisy.
func (t *Task) Bytes() []byte {
	var b bytes.Buffer
	b.WriteString(frontmatterFence)
	b.WriteByte('\n')

	written := map[string]bool{}
	emit := func(key string) {
		if written[key] {
			return
		}
		written[key] = true
		switch key {
		case "id":
			fmt.Fprintf(&b, "id: %d\n", t.ID)
		case "title":
			fmt.Fprintf(&b, "title: %s\n", scalar(t.Title))
		case "status":
			fmt.Fprintf(&b, "status: %s\n", scalar(t.Status))
		case "priority":
			fmt.Fprintf(&b, "priority: %s\n", scalar(t.Priority))
		case "reason":
			// Written only when there is one, so a task that was never declined
			// carries no empty key.
			if t.Reason != "" {
				fmt.Fprintf(&b, "reason: %s\n", scalar(t.Reason))
			}
		case "tags":
			writeList(&b, 0, "tags", t.Tags)
		case "metadata":
			t.writeMetadata(&b)
		}
	}

	// A known key the author's file does not carry is emitted in its canonical
	// slot rather than appended at the end, so that a file which gains a field
	// ends up looking like one written from scratch.
	emitMissingBefore := func(key string) {
		for _, k := range TopLevelKeys {
			if k == key {
				return
			}
			if !written[k] && !hasKey(t.front, k) {
				emit(k)
			}
		}
	}

	// Keys the author added are preserved where they were written, so that a
	// write by the CLI does not reshuffle a file someone arranged by hand.
	if t.front != nil {
		for i := 0; i+1 < len(t.front.Content); i += 2 {
			key := t.front.Content[i].Value
			if known(key, TopLevelKeys) {
				emitMissingBefore(key)
				emit(key)
				continue
			}
			if written[key] {
				continue
			}
			written[key] = true
			b.WriteString(encodePreserved(t.front.Content[i], t.front.Content[i+1]))
		}
	}
	for _, key := range TopLevelKeys {
		emit(key)
	}

	b.WriteString(frontmatterFence)
	b.WriteByte('\n')
	b.WriteString(t.Body)
	return b.Bytes()
}

func (t *Task) writeMetadata(b *bytes.Buffer) {
	if !t.metadataPresent() {
		return
	}
	b.WriteString("metadata:\n")
	if t.Meta.Schema != 0 {
		fmt.Fprintf(b, "  schema: %d\n", t.Meta.Schema)
	}
	if t.Meta.Created != "" {
		fmt.Fprintf(b, "  created: %s\n", scalar(t.Meta.Created))
	}
	if t.Meta.Author != "" {
		fmt.Fprintf(b, "  author: %s\n", scalar(t.Meta.Author))
	}
	if !t.Meta.Source.Empty() {
		b.WriteString("  source:\n")
		if len(t.Meta.Source.Files) > 0 {
			writeList(b, 4, "files", t.Meta.Source.Files)
		}
		if t.Meta.Source.Branch != "" {
			fmt.Fprintf(b, "    branch: %s\n", scalar(t.Meta.Source.Branch))
		}
		if t.Meta.Source.Commit != "" {
			fmt.Fprintf(b, "    commit: %s\n", scalar(t.Meta.Source.Commit))
		}
	}
	if t.Meta.Refs != nil {
		writeList(b, 2, "refs", t.Meta.Refs)
	}
}

func writeList(b *bytes.Buffer, indent int, key string, items []string) {
	pad := strings.Repeat(" ", indent)
	if len(items) == 0 {
		fmt.Fprintf(b, "%s%s: []\n", pad, key)
		return
	}
	fmt.Fprintf(b, "%s%s:\n", pad, key)
	for _, item := range items {
		fmt.Fprintf(b, "%s  - %s\n", pad, scalar(item))
	}
}

// encodePreserved renders a top-level key the CLI does not understand, using
// the YAML encoder since its shape is arbitrary.
func encodePreserved(key, val *yaml.Node) string {
	doc := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{key, val}}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return ""
	}
	_ = enc.Close()
	return buf.String()
}

// scalar renders s as a YAML scalar, quoting only when plain style would not
// round-trip. The check is syntactic: every field is read back as raw scalar
// text, so a value that merely looks like a number or a date needs no quoting.
func scalar(s string) string {
	if plainSafe(s) {
		return s
	}
	if strings.ContainsAny(s, "\n\r\t") {
		return doubleQuote(s)
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func plainSafe(s string) bool {
	if s == "" {
		return false
	}
	if s != strings.TrimSpace(s) {
		return false
	}
	if strings.ContainsAny(s, "\n\r\t") {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	// Indicators that change meaning at the start of a plain scalar.
	switch s[0] {
	case '-', '?', ':', ',', '[', ']', '{', '}', '#', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`':
		return false
	}
	// Indicators that change meaning anywhere inside one.
	if strings.Contains(s, ": ") || strings.Contains(s, " #") {
		return false
	}
	if strings.ContainsAny(s, "[]{},") {
		return false
	}
	return !strings.HasSuffix(s, ":")
}

func doubleQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// NormalizeCreated rewrites a timestamp the parser accepted loosely into
// canonical RFC 3339 form, reporting false when the value is already canonical
// or cannot be understood at all.
func NormalizeCreated(v string) (string, bool) {
	if v == "" {
		return "", false
	}
	if _, err := time.Parse(time.RFC3339, v); err == nil {
		return v, false
	}
	ts, ok := parseLooseTime(v)
	if !ok {
		return v, false
	}
	return ts.UTC().Format(time.RFC3339), true
}

// HasBlockingIssue reports whether the task carries an error the repair mode
// must not try to work around, such as an unrecognised status or a duplicate
// key. Repair only touches files whose remaining problems have a single
// unambiguous correction.
func (t *Task) HasBlockingIssue() bool {
	for _, is := range t.Issues {
		if is.Severity == SeverityError && !is.Repairable {
			return true
		}
	}
	return false
}

// HasRepairableIssue reports whether any issue found while reading the file can
// be corrected by rewriting it.
func (t *Task) HasRepairableIssue() bool {
	for _, is := range t.Issues {
		if is.Repairable {
			return true
		}
	}
	return false
}
