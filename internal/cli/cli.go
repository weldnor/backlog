// Package cli implements subcommand dispatch and the shared conventions every
// backlog command follows: data on stdout, diagnostics on stderr, a non-zero
// exit code on failure.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Version is the build-time version string. Override with
// -ldflags "-X github.com/weldnor/backlog/internal/cli.Version=..."
var Version = "0.4.0"

// Env carries everything a command needs from the outside world, so that tests
// can run commands in a temporary directory with captured output.
type Env struct {
	Stdout io.Writer
	Stderr io.Writer
	Dir    string
}

// OSEnv returns the Env for a real process run.
func OSEnv() Env {
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	return Env{Stdout: os.Stdout, Stderr: os.Stderr, Dir: dir}
}

// UsageError marks a failure caused by how the command was invoked rather than
// by the state of the backlog. It is reported with the command's usage text.
type UsageError struct{ err error }

func usagef(format string, args ...any) error {
	return &UsageError{fmt.Errorf(format, args...)}
}

func (e *UsageError) Error() string { return e.err.Error() }
func (e *UsageError) Unwrap() error { return e.err }

type command struct {
	name    string
	summary string
	run     func(Env, []string) error
}

func commands() []command {
	return []command{
		{"init", "create a backlog in the current directory and install the agent skills", runInit},
		{"add", "record a new task", runAdd},
		{"list", "list tasks (add new|todo|doing|done|declined to show one status)", runList},
		{"search", "find tasks by title, description or tag", runSearch},
		{"show", "show a single task", runShow},
		{"set", "change a task's status or attach a reference", runSet},
		{"edit", "change a task's title, description or tags", runEdit},
		{"tag", "rename or remove a tag across every task (rm|rename)", runTag},
		{"rm", "delete a task", runRm},
		{"stats", "summarize the backlog by status, priority and tag", runStats},
		{"validate", "check the backlog for problems", runValidate},
		{"browse", "start a local web UI for the backlog", runBrowse},
	}
}

func lookup(name string) (command, bool) {
	for _, c := range commands() {
		if c.name == name {
			return c, true
		}
	}
	return command{}, false
}

// Run dispatches args (excluding the program name) and returns the process exit
// code. Nothing is written to stdout unless the command produced data.
func Run(env Env, args []string) int {
	if len(args) == 0 {
		writeHelp(env.Stdout)
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		writeHelp(env.Stdout)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintln(env.Stdout, Version)
		return 0
	}

	if strings.HasPrefix(args[0], "-") {
		fmt.Fprintf(env.Stderr, "backlog: unknown option %q\n", args[0])
		writeHelp(env.Stderr)
		return 2
	}

	cmd, ok := lookup(args[0])
	if !ok {
		fmt.Fprintf(env.Stderr, "backlog: unknown command %q\n", args[0])
		writeHelp(env.Stderr)
		return 2
	}

	if err := cmd.run(env, args[1:]); err != nil {
		var usage *UsageError
		if errors.As(err, &usage) {
			fmt.Fprintf(env.Stderr, "backlog %s: %v\n", cmd.name, err)
			return 2
		}
		fmt.Fprintf(env.Stderr, "backlog %s: %v\n", cmd.name, err)
		return 1
	}
	return 0
}

func writeHelp(w io.Writer) {
	fmt.Fprintln(w, "backlog — a per-project capture inbox for findings worth deferring.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  backlog <command> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	cs := commands()
	sort.Slice(cs, func(i, j int) bool { return cs[i].name < cs[j].name })
	width := 0
	for _, c := range cs {
		if len(c.name) > width {
			width = len(c.name)
		}
	}
	for _, c := range cs {
		fmt.Fprintf(w, "  %-*s  %s\n", width, c.name, c.summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --help     show this help")
	fmt.Fprintln(w, "  --version  print the version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'backlog <command> --help' for the options of a single command.")
}

// newFlagSet returns a flag set that reports errors through the returned error
// rather than writing to stderr itself, keeping the output convention in one
// place.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return fs
}

// parseFlags parses args and converts flag errors into UsageError so they exit
// with the usage status.
//
// Arguments are permuted first so that options may follow operands. The hot
// path is `backlog add "a title" --tag bug --file x.go`, and the standard flag
// package would otherwise stop parsing at the title and silently swallow every
// option after it.
func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(permute(fs, args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return &UsageError{errors.New(usageText(fs))}
		}
		return &UsageError{err}
	}
	return nil
}

func usageText(fs *flag.FlagSet) string {
	var sb strings.Builder
	sb.WriteString("options:")
	fs.VisitAll(func(f *flag.Flag) {
		sb.WriteString("\n  --" + f.Name)
		if f.DefValue != "false" && f.DefValue != "" {
			sb.WriteString(" (default " + f.DefValue + ")")
		}
		if f.Usage != "" {
			sb.WriteString("\n      " + f.Usage)
		}
	})
	return sb.String()
}

// permute moves options ahead of operands, preserving the order within each
// group. It has to know which options take a value, which it asks the flag set
// rather than guessing.
func permute(fs *flag.FlagSet, args []string) []string {
	var options, operands []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			operands = append(operands, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			operands = append(operands, a)
			continue
		}
		options = append(options, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") {
			continue // --flag=value carries its own value
		}
		f := fs.Lookup(name)
		if f == nil {
			continue // unknown: let the flag package report it
		}
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			continue
		}
		if i+1 < len(args) {
			i++
			options = append(options, args[i])
		}
	}
	return append(options, operands...)
}
