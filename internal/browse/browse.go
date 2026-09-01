// Package browse implements `backlog browse`: a local HTTP server serving a
// single-page web UI backed by a *store.Store, and the JSON API that UI
// talks to. See openspec/changes/add-browse-command for the design this
// package implements.
package browse

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/weldnor/backlog/internal/store"
)

// shutdownTimeout bounds how long Serve waits for in-flight requests to
// finish once an interrupt is received.
const shutdownTimeout = 5 * time.Second

// Options configures Serve.
type Options struct {
	// Host is the address to bind to. Empty defaults to 127.0.0.1.
	Host string
	// Port is the port to listen on. Zero asks the OS for a free port.
	Port int
	// Open, when true, launches the system's default browser at the server's
	// URL once the listener is up.
	Open bool
	// Ready, when non-nil, is called exactly once with the URL the server is
	// reachable at, before Serve blocks. The caller uses this to print it.
	Ready func(url string)
	// Log receives diagnostics that should not fail the command, such as a
	// browser-launch failure. A nil Log discards them.
	Log *log.Logger
	// Version is shown in the UI's top bar. Empty shows nothing.
	Version string
}

func (o Options) logf(format string, args ...any) {
	if o.Log != nil {
		o.Log.Printf(format, args...)
	}
}

// Serve starts the HTTP server backed by st and blocks until the process
// receives an interrupt (SIGINT or SIGTERM), at which point it shuts down
// gracefully and returns nil. It returns an error if the server cannot be
// started, or if it stops for any reason other than a clean interrupt.
func Serve(st *store.Store, opts Options) error {
	host := opts.Host
	if host == "" {
		host = "127.0.0.1"
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", opts.Port)))
	if err != nil {
		return err
	}

	mux, err := newMux(st, opts)
	if err != nil {
		ln.Close()
		return err
	}
	srv := &http.Server{Handler: mux}

	url := "http://" + ln.Addr().String()
	if opts.Ready != nil {
		opts.Ready(url)
	}
	if opts.Open {
		if err := openBrowser(url); err != nil {
			opts.logf("could not open a browser: %v", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		// Shutdown waits for Serve to return, but draining the channel keeps
		// the goroutine from leaking if it hasn't reported yet.
		<-serveErr
		return nil
	}
}
