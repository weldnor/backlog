package task

import (
	"fmt"
	"strings"
	"testing"
)

const canonical = `---
id: 1
title: Race in session cache
status: todo
priority: high
tags:
  - bug
  - concurrency
metadata:
  schema: 1
  created: 2026-08-30T20:59:51Z
  author: agent
  source:
    files:
      - internal/session/cache.go
    branch: main
    commit: 0badc0ffee
  refs:
    - openspec:add-auth
---
Two goroutines write the map.

A second paragraph, kept verbatim.
`

func parseOK(t *testing.T, name, content string) *Task {
	t.Helper()
	task, err := Parse(name, []byte(content))
	if err != nil {
		t.Fatalf("Parse(%q) failed: %v", name, err)
	}
	return task
}

func TestParseReadsEveryField(t *testing.T) {
	got := parseOK(t, "001-race-in-session-cache.md", canonical)

	if got.ID != 1 {
		t.Errorf("ID = %d, want 1", got.ID)
	}
	if got.Title != "Race in session cache" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.Status != StatusTodo {
		t.Errorf("Status = %q", got.Status)
	}
	if got.Priority != PriorityHigh {
		t.Errorf("Priority = %q", got.Priority)
	}
	if strings.Join(got.Tags, ",") != "bug,concurrency" {
		t.Errorf("Tags = %v", got.Tags)
	}
	if got.Meta.Schema != 1 || got.Meta.Created != "2026-08-30T20:59:51Z" || got.Meta.Author != AuthorAgent {
		t.Errorf("Meta = %+v", got.Meta)
	}
	if strings.Join(got.Meta.Source.Files, ",") != "internal/session/cache.go" {
		t.Errorf("Source.Files = %v", got.Meta.Source.Files)
	}
	if got.Meta.Source.Branch != "main" || got.Meta.Source.Commit != "0badc0ffee" {
		t.Errorf("Source = %+v", got.Meta.Source)
	}
	if strings.Join(got.Meta.Refs, ",") != "openspec:add-auth" {
		t.Errorf("Refs = %v", got.Meta.Refs)
	}
	if !strings.Contains(got.Body, "A second paragraph, kept verbatim.") {
		t.Errorf("body lost a paragraph: %q", got.Body)
	}
	if len(got.Issues) != 0 {
		t.Errorf("clean file produced issues: %v", got.Issues)
	}
}

// A file the CLI wrote has to survive a read and a write unchanged, or every
// unrelated status change would show up as a reformatting diff.
func TestRoundTripIsByteIdentical(t *testing.T) {
	files := map[string]string{
		"canonical": canonical,
		"minimal": `---
id: 3
title: A bare task
status: doing
priority: medium
tags: []
metadata:
  schema: 1
  created: 2026-08-30T20:59:51Z
  author: human
  refs: []
---
`,
		"empty body": `---
id: 4
title: No description at all
status: done
priority: low
tags: []
metadata:
  schema: 1
  created: 2026-08-30T20:59:51Z
  author: agent
  refs: []
---
`,
		"quoted title": `---
id: 5
title: 'Кириллический заголовок: проверка'
status: todo
priority: medium
tags: []
metadata:
  schema: 1
  created: 2026-08-30T20:59:51Z
  author: agent
  refs: []
---
`,
	}
	for name, content := range files {
		t.Run(name, func(t *testing.T) {
			got := parseOK(t, "001-x.md", content)
			if out := string(got.Bytes()); out != content {
				t.Errorf("round trip changed the file.\n--- got ---\n%s\n--- want ---\n%s", out, content)
			}
		})
	}
}

func TestParseRecoversMissingIDFromFileName(t *testing.T) {
	content := `---
title: The identifier went missing
status: todo
tags: []
metadata:
  schema: 1
  created: 2026-08-30T20:59:51Z
  author: agent
  refs: []
---
`
	got := parseOK(t, "017-the-identifier-went-missing.md", content)
	if got.ID != 17 {
		t.Errorf("ID = %d, want 17 recovered from the file name", got.ID)
	}
	if !hasIssue(got, SeverityError, "recovered 17") {
		t.Errorf("expected an issue reporting the recovery, got %v", got.Issues)
	}
	if !strings.Contains(string(got.Bytes()), "id: 17") {
		t.Error("the recovered identifier was not written back")
	}
}

func TestParseReportsInvalidStatusWithoutFailing(t *testing.T) {
	content := `---
id: 1
title: Task with a bad status
status: in-progress
tags: []
metadata:
  schema: 1
  created: 2026-08-30T20:59:51Z
  author: agent
  refs: []
---
`
	got := parseOK(t, "001-x.md", content)
	if got.Status != "in-progress" {
		t.Errorf("the raw status was not kept: %q", got.Status)
	}
	if !hasIssue(got, SeverityError, "expected one of") {
		t.Errorf("expected an error issue naming the permitted values, got %v", got.Issues)
	}
}

func TestParseFailsOnlyWhenThereIsNothingToRead(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"no frontmatter", "Just a markdown file.\n"},
		{"unterminated frontmatter", "---\nid: 1\ntitle: x\n"},
		{"not valid YAML", "---\nid: 1\n title: [unclosed\n---\n"},
		{"frontmatter is not a mapping", "---\n- a\n- b\n---\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Parse("001-x.md", []byte(c.content)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestParseToleranceCases(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		severity Severity
		contains string
	}{
		{
			name:     "misspelled metadata key",
			content:  meta("  creted: 2026-08-30T20:59:51Z\n"),
			severity: SeverityError,
			contains: "did you mean created?",
		},
		{
			name:     "unknown metadata key with no near miss",
			content:  meta("  banana: 3\n"),
			severity: SeverityError,
			contains: "metadata.banana is not a permitted key",
		},
		{
			name:     "empty reference",
			content:  meta("  refs:\n    - \"\"\n"),
			severity: SeverityError,
			contains: "empty reference",
		},
		{
			name:     "arbitrary reference is accepted",
			content:  meta("  refs:\n    - anything at all #42\n"),
			severity: "",
		},
		{
			name:     "loose timestamp is repairable",
			content:  meta("  created: 2026-08-30 20:59:51\n"),
			severity: SeverityWarning,
			contains: "not RFC 3339",
		},
		{
			name:     "unusable timestamp is an error",
			content:  meta("  created: last tuesday\n"),
			severity: SeverityError,
			contains: "not a valid RFC 3339",
		},
		{
			name:     "duplicate tag",
			content:  "---\nid: 1\ntitle: x\nstatus: todo\ntags:\n  - bug\n  - bug\n---\n",
			severity: SeverityWarning,
			contains: "more than once",
		},
		{
			name:     "unknown author",
			content:  meta("  author: robot\n"),
			severity: SeverityError,
			contains: "metadata.author",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseOK(t, "001-x.md", c.content)
			if c.severity == "" {
				if countSeverity(got, SeverityError) != 0 {
					t.Errorf("expected no errors, got %v", got.Issues)
				}
				return
			}
			if !hasIssue(got, c.severity, c.contains) {
				t.Errorf("expected a %s containing %q, got %v", c.severity, c.contains, got.Issues)
			}
		})
	}
}

func TestParseDeduplicatesTagsOnRead(t *testing.T) {
	got := parseOK(t, "001-x.md", "---\nid: 1\ntitle: x\nstatus: todo\ntags:\n  - bug\n  - bug\n  - flake\n---\n")
	if strings.Join(got.Tags, ",") != "bug,flake" {
		t.Errorf("Tags = %v, want the duplicate dropped", got.Tags)
	}
}

// meta wraps a metadata fragment in an otherwise clean task file.
func meta(fragment string) string {
	return "---\nid: 1\ntitle: x\nstatus: todo\ntags: []\nmetadata:\n  schema: 1\n" + fragment + "---\n"
}

func hasIssue(t *Task, sev Severity, substr string) bool {
	for _, is := range t.Issues {
		if is.Severity == sev && strings.Contains(is.Message, substr) {
			return true
		}
	}
	return false
}

func countSeverity(t *Task, sev Severity) int {
	n := 0
	for _, is := range t.Issues {
		if is.Severity == sev {
			n++
		}
	}
	return n
}

func TestParsePreservesUnknownTopLevelField(t *testing.T) {
	content := `---
id: 1
title: Task with an experiment
status: todo
priority: high
severity: blocker
tags: []
metadata:
  schema: 1
  created: 2026-08-30T20:59:51Z
  author: agent
  refs: []
---
`
	got := parseOK(t, "001-x.md", content)
	if got.Title != "Task with an experiment" {
		t.Fatalf("the task did not survive the unknown field: %+v", got)
	}
	out := string(got.Bytes())
	if !strings.Contains(out, "severity: blocker") {
		t.Errorf("the unknown field was dropped on write:\n%s", out)
	}
	// The top level is author-owned, so an unknown key is room to experiment.
	if !hasIssue(got, SeverityWarning, "severity") {
		t.Errorf("expected a warning about the unknown field, got %v", got.Issues)
	}
	if countSeverity(got, SeverityError) != 0 {
		t.Errorf("an unknown top-level field must not be an error: %v", got.Issues)
	}
}

// priority is a field the CLI understands, so it must never be reported the way
// an unrecognised key is.
func TestParseTreatsPriorityAsAKnownField(t *testing.T) {
	got := parseOK(t, "001-x.md", "---\nid: 1\ntitle: x\nstatus: todo\npriority: high\ntags: []\n---\n")
	if got.Priority != PriorityHigh {
		t.Errorf("Priority = %q, want %q", got.Priority, PriorityHigh)
	}
	if hasIssue(got, SeverityWarning, "not a field the CLI understands") {
		t.Errorf("priority was reported as an unrecognised field: %v", got.Issues)
	}
	if countSeverity(got, SeverityError) != 0 {
		t.Errorf("a valid priority produced errors: %v", got.Issues)
	}
}

func TestParsePriorityCases(t *testing.T) {
	const shell = "---\nid: 1\ntitle: x\nstatus: todo\n%stags: []\n---\n"
	cases := []struct {
		name     string
		fragment string
		want     string
		severity Severity
		contains string
	}{
		{
			name:     "permitted value",
			fragment: "priority: low\n",
			want:     PriorityLow,
			severity: "",
		},
		{
			name:     "absent is the default, and repairable",
			fragment: "",
			want:     DefaultPriority,
			severity: SeverityWarning,
			contains: "priority is missing",
		},
		{
			name:     "empty is the default, and repairable",
			fragment: "priority:\n",
			want:     DefaultPriority,
			severity: SeverityWarning,
			contains: "priority is empty",
		},
		{
			name:     "outside the permitted set is kept verbatim",
			fragment: "priority: urgent\n",
			want:     "urgent",
			severity: SeverityError,
			contains: "expected one of",
		},
		{
			name:     "not a string",
			fragment: "priority:\n  - high\n",
			want:     "",
			severity: SeverityError,
			contains: "priority must be a string",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseOK(t, "001-x.md", fmt.Sprintf(shell, c.fragment))
			if got.Priority != c.want {
				t.Errorf("Priority = %q, want %q", got.Priority, c.want)
			}
			if c.severity == "" {
				// The shell carries no metadata block, which warns on its own;
				// what matters here is that priority contributed nothing.
				if hasIssue(got, SeverityWarning, "priority") || hasIssue(got, SeverityError, "priority") {
					t.Errorf("a valid priority produced issues: %v", got.Issues)
				}
				return
			}
			if !hasIssue(got, c.severity, c.contains) {
				t.Errorf("expected a %s containing %q, got %v", c.severity, c.contains, got.Issues)
			}
			// Only the missing and empty cases may be repaired; a value someone
			// typed is a judgement and must block repair instead.
			wantRepairable := c.severity == SeverityWarning
			if got.HasRepairableIssue() != wantRepairable {
				t.Errorf("HasRepairableIssue = %v, want %v", got.HasRepairableIssue(), wantRepairable)
			}
			if got.HasBlockingIssue() != (c.severity == SeverityError) {
				t.Errorf("HasBlockingIssue = %v, want %v", got.HasBlockingIssue(), c.severity == SeverityError)
			}
		})
	}
}

func TestWriteAddsTheDefaultPriorityToAFileWithout(t *testing.T) {
	got := parseOK(t, "001-x.md", "---\nid: 1\ntitle: x\nstatus: todo\ntags: []\n---\n")
	out := string(got.Bytes())
	if !strings.Contains(out, "priority: medium\n") {
		t.Errorf("the default was not written down:\n%s", out)
	}
	// It belongs with the other author-owned judgements, right after status.
	if !strings.Contains(out, "status: todo\npriority: medium\n") {
		t.Errorf("priority was not written after status:\n%s", out)
	}
}

func TestUnrelatedWritePreservesPriority(t *testing.T) {
	got := parseOK(t, "001-x.md", "---\nid: 1\ntitle: x\nstatus: todo\npriority: high\ntags: []\n---\n")
	got.Status = StatusDoing
	out := string(got.Bytes())
	if !strings.Contains(out, "priority: high\n") {
		t.Errorf("a status change lost the priority:\n%s", out)
	}
	if !strings.Contains(out, "status: doing\n") {
		t.Errorf("the status change was not written:\n%s", out)
	}
}

// reason is a field the CLI understands, so it must never be reported the way
// an unrecognised key is.
func TestParseTreatsReasonAsAKnownField(t *testing.T) {
	got := parseOK(t, "001-x.md", "---\nid: 1\ntitle: x\nstatus: declined\npriority: high\nreason: superseded by the rewrite\ntags: []\n---\n")
	if got.Reason != "superseded by the rewrite" {
		t.Errorf("Reason = %q", got.Reason)
	}
	if hasIssue(got, SeverityWarning, "not a field the CLI understands") {
		t.Errorf("reason was reported as an unrecognised field: %v", got.Issues)
	}
	if countSeverity(got, SeverityError) != 0 {
		t.Errorf("a declined task with a reason produced errors: %v", got.Issues)
	}
}

func TestParseReasonValueCases(t *testing.T) {
	const shell = "---\nid: 1\ntitle: x\nstatus: declined\n%stags: []\n---\n"
	cases := []struct {
		name     string
		fragment string
		want     string
		contains string
	}{
		{
			name:     "a string is stored verbatim",
			fragment: "reason: the fix costs more than the bug\n",
			want:     "the fix costs more than the bug",
		},
		{
			name:     "a list is not a reason",
			fragment: "reason:\n  - too costly\n",
			contains: "reason must be a string",
		},
		{
			name:     "a mapping is not a reason",
			fragment: "reason:\n  why: too costly\n",
			contains: "reason must be a string",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseOK(t, "001-x.md", fmt.Sprintf(shell, c.fragment))
			if got.Reason != c.want {
				t.Errorf("Reason = %q, want %q", got.Reason, c.want)
			}
			if c.contains == "" {
				if countSeverity(got, SeverityError) != 0 {
					t.Errorf("a valid reason produced errors: %v", got.Issues)
				}
				return
			}
			if !hasIssue(got, SeverityError, c.contains) {
				t.Errorf("expected an error containing %q, got %v", c.contains, got.Issues)
			}
			// Neither a reason to write nor one to delete is something --fix can
			// decide, so every reason error has to block repair.
			if !got.HasBlockingIssue() {
				t.Errorf("a malformed reason did not block repair: %v", got.Issues)
			}
		})
	}
}

func TestParseReasonAndStatusMustAgree(t *testing.T) {
	const shell = "---\nid: 1\ntitle: x\nstatus: %s\n%stags: []\n---\n"
	cases := []struct {
		name     string
		status   string
		fragment string
		contains string
	}{
		{
			name:     "declined with a reason is the pairing that is asked for",
			status:   StatusDeclined,
			fragment: "reason: not worth the churn\n",
		},
		{
			name:     "declined without a reason",
			status:   StatusDeclined,
			contains: "no reason is recorded",
		},
		{
			name:     "declined with an empty reason",
			status:   StatusDeclined,
			fragment: "reason:\n",
			contains: "no reason is recorded",
		},
		{
			name:     "declined with a whitespace-only reason",
			status:   StatusDeclined,
			fragment: "reason: '   '\n",
			contains: "no reason is recorded",
		},
		{
			name:   "done without a reason",
			status: StatusDone,
		},
		{
			name:     "done with a reason",
			status:   StatusDone,
			fragment: "reason: not worth the churn\n",
			contains: "a reason applies only to a declined task",
		},
		{
			name:   "todo without a reason",
			status: StatusTodo,
		},
		{
			name:     "todo with a reason",
			status:   StatusTodo,
			fragment: "reason: not worth the churn\n",
			contains: "a reason applies only to a declined task",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseOK(t, "001-x.md", fmt.Sprintf(shell, c.status, c.fragment))
			if c.contains == "" {
				if countSeverity(got, SeverityError) != 0 {
					t.Errorf("a well-paired file produced errors: %v", got.Issues)
				}
				return
			}
			if !hasIssue(got, SeverityError, c.contains) {
				t.Errorf("expected an error containing %q, got %v", c.contains, got.Issues)
			}
			if !got.HasBlockingIssue() {
				t.Errorf("the pairing error did not block repair: %v", got.Issues)
			}
		})
	}
}

func TestWriteEmitsReasonOnlyWhenThereIsOne(t *testing.T) {
	declined := parseOK(t, "001-x.md", "---\nid: 1\ntitle: x\nstatus: declined\npriority: high\nreason: 'the cost outweighs the benefit: the call site is cold'\ntags: []\n---\n")
	out := string(declined.Bytes())
	// A colon inside the prose has to survive as prose rather than turning the
	// value into a mapping.
	if !strings.Contains(out, "reason: 'the cost outweighs the benefit: the call site is cold'\n") {
		t.Errorf("the reason did not round-trip:\n%s", out)
	}
	// It belongs with the other author-owned judgements, right after priority.
	if !strings.Contains(out, "priority: high\nreason: ") {
		t.Errorf("reason was not written after priority:\n%s", out)
	}
	again := parseOK(t, "001-x.md", out)
	if again.Reason != declined.Reason {
		t.Errorf("reason changed on the second read: %q", again.Reason)
	}

	live := parseOK(t, "001-x.md", "---\nid: 1\ntitle: x\nstatus: todo\npriority: high\ntags: []\n---\n")
	if out := string(live.Bytes()); strings.Contains(out, "reason") {
		t.Errorf("a todo task gained a reason key:\n%s", out)
	}
}

func TestUnrelatedWritePreservesReason(t *testing.T) {
	got := parseOK(t, "001-x.md", "---\nid: 1\ntitle: x\nstatus: declined\npriority: low\nreason: superseded by the rewrite\ntags: []\n---\n")
	got.Priority = PriorityHigh
	out := string(got.Bytes())
	if !strings.Contains(out, "reason: superseded by the rewrite\n") {
		t.Errorf("a priority change lost the reason:\n%s", out)
	}
	if !strings.Contains(out, "priority: high\n") {
		t.Errorf("the priority change was not written:\n%s", out)
	}
}
