// Command backlog is a per-project capture inbox: one command to record a
// finding you are not going to act on now, and a way to review what has
// accumulated later.
package main

import (
	"os"

	"github.com/weldnor/backlog/internal/cli"
)

func main() {
	os.Exit(cli.Run(cli.OSEnv(), os.Args[1:]))
}
