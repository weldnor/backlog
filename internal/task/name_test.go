package task

import "testing"

func TestSlug(t *testing.T) {
	cases := []struct {
		name  string
		title string
		want  string
	}{
		{"ascii", "Race in session cache", "race-in-session-cache"},
		{"cyrillic is kept, not transliterated", "Гонка в кэше сессий", "гонка-в-кэше-сессий"},
		{"mixed scripts", "Fix кэш bug", "fix-кэш-bug"},
		{"punctuation collapses", "Fix:  the -- parser's  bug!!", "fix-the-parser-s-bug"},
		{"leading and trailing punctuation is trimmed", "...hello...", "hello"},
		{"digits are kept", "HTTP 500 on /v2/login", "http-500-on-v2-login"},
		{"empty after reduction", "!!! ??? ---", ""},
		{"empty title", "", ""},
		{"whitespace only", "   \t ", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Slug(c.title); got != c.want {
				t.Errorf("Slug(%q) = %q, want %q", c.title, got, c.want)
			}
		})
	}
}

func TestSlugIsBounded(t *testing.T) {
	long := ""
	for i := 0; i < 40; i++ {
		long += "word "
	}
	got := Slug(long)
	if len([]rune(got)) > maxSlugRunes {
		t.Errorf("slug is %d runes, want at most %d", len([]rune(got)), maxSlugRunes)
	}
	// The cut has to be deterministic, or drift detection would fire forever.
	if again := Slug(long); again != got {
		t.Errorf("slug is not deterministic: %q then %q", got, again)
	}
	if Slug(long)[len(Slug(long))-1] == '-' {
		t.Error("slug ends with a hyphen")
	}
}

func TestFileName(t *testing.T) {
	cases := []struct {
		id    int
		title string
		want  string
	}{
		{1, "Race in cache", "001-race-in-cache.md"},
		{42, "Race in cache", "042-race-in-cache.md"},
		{1234, "Race in cache", "1234-race-in-cache.md"},
		{7, "!!!", "007.md"},
	}
	for _, c := range cases {
		if got := FileName(c.id, c.title); got != c.want {
			t.Errorf("FileName(%d, %q) = %q, want %q", c.id, c.title, got, c.want)
		}
	}
}

func TestIDFromFileName(t *testing.T) {
	cases := []struct {
		name   string
		wantID int
		wantOK bool
	}{
		{"001-race.md", 1, true},
		{"042-a-b-c.md", 42, true},
		{"007.md", 7, true},
		{"1234-x.md", 1234, true},
		{"README.md", 0, false},
		{"001-race.txt", 0, false},
		{"-race.md", 0, false},
		{"000.md", 0, false},
		{"abc-race.md", 0, false},
		{".tmp-1234.md", 0, false},
	}
	for _, c := range cases {
		id, ok := IDFromFileName(c.name)
		if id != c.wantID || ok != c.wantOK {
			t.Errorf("IDFromFileName(%q) = %d, %v; want %d, %v", c.name, id, ok, c.wantID, c.wantOK)
		}
	}
}
