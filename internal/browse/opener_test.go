package browse

import (
	"errors"
	"testing"
)

var errNoOpener = errors.New("exec: no such executable")

func TestBrowserCommandPerPlatform(t *testing.T) {
	cases := []struct {
		goos     string
		wantName string
	}{
		{"darwin", "open"},
		{"windows", "rundll32"},
		{"linux", "xdg-open"},
		{"freebsd", "xdg-open"}, // anything not darwin/windows falls back to xdg-open
	}
	for _, c := range cases {
		t.Run(c.goos, func(t *testing.T) {
			name, args := browserCommand(c.goos, "http://127.0.0.1:8080")
			if name != c.wantName {
				t.Errorf("browserCommand(%q, ...) name = %q, want %q", c.goos, name, c.wantName)
			}
			if len(args) == 0 {
				t.Fatalf("browserCommand(%q, ...) args = %v, want the URL somewhere in it", c.goos, args)
			}
			if args[len(args)-1] != "http://127.0.0.1:8080" {
				t.Errorf("browserCommand(%q, ...) args = %v, want the URL as the last argument", c.goos, args)
			}
		})
	}
}

func TestOpenBrowserReportsAnErrorRatherThanPanicking(t *testing.T) {
	orig := runCommand
	defer func() { runCommand = orig }()

	called := false
	runCommand = func(name string, args ...string) error {
		called = true
		return errNoOpener
	}

	err := openBrowser("http://127.0.0.1:1")
	if err == nil {
		t.Fatal("openBrowser returned nil, want an error when the opener cannot be launched")
	}
	if !called {
		t.Fatal("runCommand was never invoked")
	}
}
