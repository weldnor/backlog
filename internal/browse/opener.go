package browse

import (
	"fmt"
	"os/exec"
	"runtime"
)

// goos is runtime.GOOS, held in a variable so it reads the same as the
// browserCommand parameter it feeds.
var goos = runtime.GOOS

// runCommand launches name with args and does not wait for it to finish. It
// is a variable so tests can stub it without actually launching a browser.
var runCommand = func(name string, args ...string) error {
	return exec.Command(name, args...).Start()
}

// browserCommand returns the command that opens url in the default browser
// on the given GOOS value. It is a pure function of goos so it can be tested
// for every platform regardless of which one the test runs on.
func browserCommand(goos, url string) (name string, args []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		// xdg-open covers Linux and the other Unix-likes go supports.
		return "xdg-open", []string{url}
	}
}

// openBrowser launches the system's default browser at url. It returns an
// error rather than panicking when no opener is found, so the caller can log
// and keep serving.
func openBrowser(url string) error {
	name, args := browserCommand(goos, url)
	if err := runCommand(name, args...); err != nil {
		return fmt.Errorf("launching %s: %w", name, err)
	}
	return nil
}
