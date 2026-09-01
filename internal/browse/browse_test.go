package browse

import (
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/weldnor/backlog/internal/store"
)

// writerFunc adapts a func(string) into an io.Writer, one call per Write, so
// a *log.Logger's output can be observed by a test.
type writerFunc func(string)

func (w writerFunc) Write(p []byte) (int, error) {
	w(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

func testLogger(fn func(string)) *log.Logger { return log.New(writerFunc(fn), "", 0) }

func TestServeShutsDownOnInterrupt(t *testing.T) {
	st, err := store.Init(t.TempDir())
	if err != nil {
		t.Fatalf("store.Init: %v", err)
	}

	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- Serve(st, Options{Port: 0, Ready: func(url string) { ready <- url }})
	}()

	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("server never became ready")
	}

	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("sending SIGINT to self: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v after an interrupt, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return within the shutdown timeout after SIGINT")
	}
}

func TestServeFailsWhenPortAlreadyInUse(t *testing.T) {
	st, err := store.Init(t.TempDir())
	if err != nil {
		t.Fatalf("store.Init: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving a port: %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	err = Serve(st, Options{Host: "127.0.0.1", Port: port})
	if err == nil {
		t.Fatal("Serve returned nil for a port already in use, want an error")
	}
}

func TestServeLogsBrowserLaunchFailureAndKeepsServing(t *testing.T) {
	origRun := runCommand
	defer func() { runCommand = origRun }()
	runCommand = func(name string, args ...string) error { return errors.New("no browser here") }

	st, err := store.Init(t.TempDir())
	if err != nil {
		t.Fatalf("store.Init: %v", err)
	}

	var logged string
	logCh := make(chan string, 1)
	logger := testLogger(func(s string) { logCh <- s })

	ready := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		done <- Serve(st, Options{
			Port:  0,
			Open:  true,
			Log:   logger,
			Ready: func(string) { ready <- struct{}{} },
		})
	}()

	<-ready
	select {
	case logged = <-logCh:
	case <-time.After(5 * time.Second):
		t.Fatal("browser-launch failure was never logged")
	}
	if logged == "" {
		t.Error("expected a non-empty log line for the browser-launch failure")
	}

	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("sending SIGINT to self: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned %v, want nil: a browser-launch failure must not fail the command", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after SIGINT")
	}
}
