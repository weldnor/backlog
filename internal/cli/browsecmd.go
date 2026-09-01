package cli

import (
	"fmt"
	"log"
	"net"

	"github.com/weldnor/backlog/internal/browse"
)

func runBrowse(env Env, args []string) error {
	fs := newFlagSet("browse")
	var (
		host   = fs.String("host", "127.0.0.1", "the address to bind to; anything other than loopback is reachable from beyond this machine")
		port   = fs.Int("port", 0, "the port to listen on; 0 asks the OS for a free one")
		noOpen = fs.Bool("no-open", false, "do not open the URL in a browser")
		asJSON = fs.Bool("json", false, "print the URL as JSON instead of plain text")
	)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return usagef("unexpected argument %q", fs.Arg(0))
	}

	st, err := openStore(env)
	if err != nil {
		return err
	}

	if !isLoopback(*host) {
		fmt.Fprintf(env.Stderr, "backlog browse: binding to %s exposes an unauthenticated read/write API to anything that can reach this machine, not just this one\n", *host)
	}

	printed := false
	opts := browse.Options{
		Host:    *host,
		Port:    *port,
		Open:    !*noOpen,
		Version: Version,
		Log:     log.New(env.Stderr, "backlog browse: ", 0),
		Ready: func(url string) {
			printed = true
			if *asJSON {
				_ = writeJSON(env.Stdout, struct {
					URL string `json:"url"`
				}{URL: url})
				return
			}
			fmt.Fprintf(env.Stdout, "backlog browse: listening at %s\n", url)
		},
	}

	err = browse.Serve(st, opts)
	if err != nil && !printed {
		// The server never became ready — most commonly a port already in
		// use — so nothing has been printed yet and this is a plain failure.
		return err
	}
	return err
}

// isLoopback reports whether host is a loopback address or name, so that
// only a deliberate choice to bind wider triggers the exposure warning.
func isLoopback(host string) bool {
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
