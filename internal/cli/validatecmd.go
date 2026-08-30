package cli

import (
	"errors"
	"fmt"

	"github.com/weldnor/backlog/internal/validate"
)

// ErrFindings reports that validation found at least one error. It carries no
// message of its own: the findings were already written to the data stream.
var ErrFindings = errors.New("validation failed")

func runValidate(env Env, args []string) error {
	fs := newFlagSet("validate")
	var (
		fix    = fs.Bool("fix", false, "repair the findings that have a single unambiguous correction")
		strict = fs.Bool("strict", false, "treat warnings as errors")
		asJSON = fs.Bool("json", false, "print the findings as JSON")
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
	report, err := validate.Run(st, validate.Options{Fix: *fix, Strict: *strict, Version: Version})
	if err != nil {
		return err
	}

	if *asJSON {
		if err := writeJSON(env.Stdout, report); err != nil {
			return err
		}
	} else {
		writeReport(env, report)
	}

	if !report.OK() {
		// The findings are the output; the exit code is what gates a commit
		// hook or an agent's own follow-up.
		return errFindings{report.Errors}
	}
	return nil
}

type errFindings struct{ n int }

func (e errFindings) Error() string {
	if e.n == 1 {
		return "1 error found"
	}
	return fmt.Sprintf("%d errors found", e.n)
}

func (e errFindings) Unwrap() error { return ErrFindings }

func writeReport(env Env, report *validate.Report) {
	w := env.Stdout
	for _, action := range report.Repairs {
		fmt.Fprintf(w, "fixed %s\n", action)
	}
	if len(report.Repairs) > 0 && len(report.Findings) > 0 {
		fmt.Fprintln(w)
	}

	current := ""
	for _, f := range report.Findings {
		if f.File != current {
			if current != "" {
				fmt.Fprintln(w)
			}
			current = f.File
			fmt.Fprintln(w, f.File)
		}
		suffix := ""
		if f.Repairable {
			suffix = "  (fixable with --fix)"
		}
		fmt.Fprintf(w, "  %-7s %s%s\n", f.Severity, f.Message, suffix)
	}

	if len(report.Findings) > 0 {
		fmt.Fprintln(w)
	}
	if report.Errors == 0 && report.Warnings == 0 {
		fmt.Fprintln(w, "no problems found")
		return
	}
	fmt.Fprintf(w, "%s, %s\n", plural(report.Errors, "error"), plural(report.Warnings, "warning"))
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
