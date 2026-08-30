package task

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// maxSlugRunes bounds the slug so that a long title cannot produce a file name
// that trips a path-length limit. The cut is deterministic, so a slug derived
// twice from the same title is always the same and drift detection stays
// meaningful.
const maxSlugRunes = 60

// Slug reduces a title to the kebab-case fragment used in a task's file name.
//
// Non-ASCII letters are kept rather than transliterated or stripped: titles in
// this project are often Cyrillic, transliteration tables are a lasting
// maintenance burden with no single right answer, and stripping would reduce
// such a name to a bare number.
func Slug(title string) string {
	var b strings.Builder
	lastHyphen := true // suppresses a leading hyphen
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastHyphen = false
		default:
			// Whitespace and punctuation alike collapse to a single hyphen.
			if !lastHyphen {
				b.WriteRune('-')
				lastHyphen = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")

	runes := []rune(s)
	if len(runes) > maxSlugRunes {
		s = string(runes[:maxSlugRunes])
		if i := strings.LastIndex(s, "-"); i > 0 {
			s = s[:i]
		}
		s = strings.Trim(s, "-")
	}
	return s
}

// FileName returns the canonical file name for a task. A title that reduces to
// nothing leaves the identifier standing alone, which is also the name a task
// carries for the moment between its identifier being claimed and its slug
// being attached.
func FileName(id int, title string) string {
	slug := Slug(title)
	if slug == "" {
		return fmt.Sprintf("%03d.md", id)
	}
	return fmt.Sprintf("%03d-%s.md", id, slug)
}

// IDFromFileName recovers the identifier a file name carries. It accepts both
// `NNN-slug.md` and the bare `NNN.md`, and reports false for any other shape.
func IDFromFileName(name string) (int, bool) {
	if !strings.HasSuffix(name, ".md") {
		return 0, false
	}
	stem := strings.TrimSuffix(name, ".md")
	digits := stem
	if i := strings.Index(stem, "-"); i >= 0 {
		digits = stem[:i]
	}
	if digits == "" {
		return 0, false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	id, err := strconv.Atoi(digits)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// SlugFromFileName returns the slug part of a task file name, empty when the
// name carries only an identifier.
func SlugFromFileName(name string) string {
	stem := strings.TrimSuffix(name, ".md")
	if i := strings.Index(stem, "-"); i >= 0 {
		return stem[i+1:]
	}
	return ""
}

func equalFold(a, b string) bool { return strings.EqualFold(a, b) }
