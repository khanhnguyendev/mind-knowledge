// Package cli defines mk's cobra command tree. Commands call the store and
// hand results to render; they contain no workflow logic.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/render"
	"github.com/khanhnguyendev/mind-knowledge/internal/store"
)

var (
	flagJSON    bool
	flagPlain   bool
	flagProject string
	flagLimit   int
	flagDB      string
)

// Root is the top-level command. Each entity's file attaches its
// subcommands to it from an init function.
var Root = &cobra.Command{
	Use:           "mk",
	Short:         "mind-knowledge: work items and a wiki for every project on this machine",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	f := Root.PersistentFlags()
	f.BoolVar(&flagJSON, "json", false, "emit machine-readable JSON")
	f.BoolVar(&flagPlain, "plain", false, "emit plain text without decoration")
	f.StringVarP(&flagProject, "project", "p", "", "scope to a project by id or name")
	f.IntVar(&flagLimit, "limit", 0, "maximum rows to return (0 means unlimited)")
	f.StringVar(&flagDB, "db", "", "database path (default $MK_DB or ~/.mind-knowledge/mk.db)")

	// Root has no work of its own: no args prints help, and any
	// unrecognized positional arg is an invalid-input error. Cobra only
	// generates its own "unknown command" error once subcommands exist
	// (see legacyArgs in cobra's args.go), so with no subcommands
	// registered yet this is what turns "mk nonesuch" into exit code 2
	// instead of a silent, successful help print.
	Root.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return fmt.Errorf("%w: unknown command %q for %q", store.ErrInvalid, args[0], cmd.CommandPath())
	}
}

// JSONMode reports whether the caller asked for JSON output.
func JSONMode() bool { return flagJSON }

// ProjectFlag returns the global --project value.
func ProjectFlag() string { return flagProject }

// LimitFlag returns the global --limit value; 0 means unlimited.
func LimitFlag() int { return flagLimit }

// OpenStore opens the database the flags select.
func OpenStore() (*store.Store, error) {
	path := flagDB
	if path == "" {
		p, err := store.DefaultPath()
		if err != nil {
			return nil, err
		}
		path = p
	}
	return store.Open(path)
}

// ExitCode maps an error to mk's process exit code.
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, store.ErrNotFound):
		return 1
	case errors.Is(err, store.ErrInvalid):
		return 2
	case errors.Is(err, store.ErrDB):
		return 3
	default:
		// Unclassified failures are treated as bad input, which is the
		// class a caller can most usefully retry after correcting.
		return 2
	}
}

// Execute runs the command tree and reports any error in the requested
// output shape.
func Execute() error {
	err := Root.Execute()
	if err != nil {
		render.ErrorOut(os.Stdout, flagJSON, ExitCode(err), errMessage(err))
	}
	return err
}

// errMessage strips the sentinel prefix so output carries only the detail.
func errMessage(err error) string {
	msg := err.Error()
	for _, sentinel := range []error{store.ErrNotFound, store.ErrInvalid, store.ErrDB} {
		prefix := sentinel.Error() + ": "
		if len(msg) > len(prefix) && msg[:len(prefix)] == prefix {
			return msg[len(prefix):]
		}
		if msg == sentinel.Error() {
			return msg
		}
	}
	return msg
}

// invalidf builds an ErrInvalid-wrapped error with a formatted detail.
func invalidf(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{store.ErrInvalid}, args...)...)
}
