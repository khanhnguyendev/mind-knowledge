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
	flagProject string
	flagLimit   int
	flagDB      string
)

// Version is stamped at build time with -ldflags. It defaults to "dev" so
// a plain `go build` still produces a working binary.
var Version = "dev"

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
	// There is no --plain: plain text is simply what every command emits
	// when --json is absent, so a flag asking for it would have no job.
	f.BoolVar(&flagJSON, "json", false, "emit machine-readable JSON")
	f.StringVarP(&flagProject, "project", "p", "", "scope to a project by id or name")
	f.IntVar(&flagLimit, "limit", 0, "maximum rows to return (0 means unlimited)")
	f.StringVar(&flagDB, "db", "", "database path (default $MK_DB or ~/.mind-knowledge/mk.db)")

	Root.Version = Version
	Root.SetVersionTemplate("mk {{.Version}}\n")

	Root.PersistentPreRunE = checkGlobalFlags

	// Root has no work of its own: no args prints help, and any
	// unrecognized positional arg is an invalid-input error.
	//
	// Root.Args must be set explicitly. Cobra's Command.Find special-cases
	// a nil Args on whichever command it resolves to: once that command
	// has subcommands (true for Root from the moment the first entity
	// registers one), Find raises "unknown command" itself, straight out
	// of command resolution and before persistent flags have been parsed
	// at all. Left alone, that would make e.g. `mk --json nonesuch`
	// report its error as plain text instead of a JSON envelope, since
	// flagJSON is still at its zero value when the error is produced.
	// Giving Root a non-nil Args routes the same check through the
	// normal execute() path instead, which parses flags first and only
	// then validates args.
	Root.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("%w: unknown command %q for %q", store.ErrInvalid, args[0], cmd.CommandPath())
	}
	Root.RunE = func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	}
}

// checkGlobalFlags validates the flags every command shares. It is Root's
// PersistentPreRunE; any command that needs its own must call it, because
// cobra runs only the closest PersistentPreRunE it finds up the chain.
func checkGlobalFlags(cmd *cobra.Command, args []string) error {
	// A negative --limit used to mean "unlimited", silently: the SQL
	// builders only append LIMIT when the value is positive. Nobody asks
	// for -5 rows meaning to ask for all of them.
	if flagLimit < 0 {
		return invalidf("--limit must not be negative (0 means unlimited)")
	}
	return nil
}

// rejectProjectFlag fails when the caller explicitly passed -p/--project
// to a command whose subject is cross-project by design.
//
// Silently ignoring it is the worse option: a skill threading -p through
// every call would believe it had scoped a result set that is in fact
// machine-wide, and act on another project's rows.
func rejectProjectFlag(cmd *cobra.Command, subject string) error {
	if !cmd.Flags().Changed("project") {
		return nil
	}
	return invalidf(
		"%s does not take -p/--project: %s are cross-project by design",
		cmd.CommandPath(), subject)
}

// crossProjectPreRun builds the PersistentPreRunE for a command tree whose
// entities belong to no project.
func crossProjectPreRun(subject string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := checkGlobalFlags(cmd, args); err != nil {
			return err
		}
		return rejectProjectFlag(cmd, subject)
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
