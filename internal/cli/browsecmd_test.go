package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// syncBuffer is a bytes.Buffer usable from the goroutine running Run and the
// test goroutine polling its output at the same time.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// waitFor polls get until it returns a non-empty string or the timeout
// elapses.
func waitFor(t *testing.T, get func() string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s := get(); s != "" {
			return s
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting for output")
	return ""
}

func interruptSelf(t *testing.T) {
	t.Helper()
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("sending SIGINT to self: %v", err)
	}
}

func TestBrowseStartsPrintsURLAndShutsDownCleanly(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	var stdout, stderr syncBuffer
	env := Env{Stdout: &stdout, Stderr: &stderr, Dir: h.dir}
	done := make(chan int, 1)
	go func() { done <- Run(env, []string{"browse", "--port", "0", "--no-open", "--json"}) }()

	out := waitFor(t, stdout.String)
	var v struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("stdout %q is not the expected {\"url\": ...} JSON: %v", out, err)
	}
	if !regexp.MustCompile(`^http://127\.0\.0\.1:\d+$`).MatchString(v.URL) {
		t.Errorf("url = %q, want http://127.0.0.1:<port>", v.URL)
	}

	interruptSelf(t)
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit code = %d after a clean interrupt, want 0; stderr: %s", code, stderr.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("backlog browse did not exit after SIGINT")
	}
}

func TestBrowseFailsWithNoBacklog(t *testing.T) {
	h := newHarness(t) // no initBacklog: no .backlog anywhere under h.dir
	code, _, stderr := h.run("browse", "--port", "0", "--no-open")
	if code == 0 {
		t.Fatal("exit code = 0 with no backlog present, want non-zero")
	}
	if !strings.Contains(stderr, "no backlog found") {
		t.Errorf("stderr = %q, want a message about no backlog found", stderr)
	}
}

func TestBrowseWarnsOnlyForNonLoopbackHost(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	// Default host: no warning.
	var stdout, stderr syncBuffer
	done := make(chan int, 1)
	go func() {
		done <- Run(Env{Stdout: &stdout, Stderr: &stderr, Dir: h.dir}, []string{"browse", "--port", "0", "--no-open", "--json"})
	}()
	waitFor(t, stdout.String)
	interruptSelf(t)
	<-done
	if strings.Contains(stderr.String(), "exposes") {
		t.Errorf("stderr warned about exposure on the default loopback host: %q", stderr.String())
	}

	// A non-loopback host: warned, before the server even starts.
	var stdout2, stderr2 syncBuffer
	done2 := make(chan int, 1)
	go func() {
		done2 <- Run(Env{Stdout: &stdout2, Stderr: &stderr2, Dir: h.dir}, []string{"browse", "--host", "0.0.0.0", "--port", "0", "--no-open", "--json"})
	}()
	waitFor(t, stdout2.String)
	warned := waitFor(t, stderr2.String)
	if !strings.Contains(warned, "0.0.0.0") || !strings.Contains(warned, "exposes") {
		t.Errorf("stderr = %q, want a warning naming the exposure for a non-loopback host", warned)
	}
	interruptSelf(t)
	<-done2
}

func TestBrowseLogsBrowserLaunchFailureWithoutFailingTheCommand(t *testing.T) {
	// --no-open omitted deliberately: this exercises the real opener, which
	// has nothing to launch in the sandbox this test runs in, so it fails
	// and should only be logged, not fail the command.
	h := newHarness(t)
	h.initBacklog()

	var stdout, stderr syncBuffer
	done := make(chan int, 1)
	go func() {
		done <- Run(Env{Stdout: &stdout, Stderr: &stderr, Dir: h.dir}, []string{"browse", "--port", "0", "--json"})
	}()
	waitFor(t, stdout.String)
	logged := waitFor(t, stderr.String)
	if !strings.Contains(logged, "could not open a browser") {
		t.Errorf("stderr = %q, want the browser-launch failure logged", logged)
	}

	interruptSelf(t)
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit code = %d, want 0: a browser-launch failure must not fail the command; stderr: %s", code, stderr.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("backlog browse did not exit after SIGINT")
	}
}
