package cmd

import (
	"fmt"
	"os"

	"github.com/moveeeax/envtool/internal/env"
	"github.com/spf13/cobra"
)

// errDifferencesFound is returned by `diff --exit-code` when the two files
// differ. Its message is empty: diff has already written the differences to
// stdout, so main should set the process exit code without printing a
// redundant second "envtool: ..." line to stderr.
//
// Returning this from RunE (rather than calling os.Exit(1) directly, as a
// previous version did) keeps the command testable in-process: os.Exit tears
// down the whole test binary, not just the command under test, so that
// approach could never be exercised by cmd's own test suite.
type errDifferencesFound struct{}

func (errDifferencesFound) Error() string { return "" }

// ExitCode reports the process exit code main should use for this error.
func (errDifferencesFound) ExitCode() int { return 1 }

func newDiffCmd() *cobra.Command {
	var exitCode bool
	cmd := &cobra.Command{
		Use:   "diff FILE_A FILE_B",
		Short: "Show key differences between two .env files",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			left, err := env.ParseFile(args[0])
			if err != nil {
				return err
			}
			right, err := env.ParseFile(args[1])
			if err != nil {
				return err
			}
			changes := env.Diff(left, right)
			for _, c := range changes {
				switch c.Kind {
				case env.Added:
					fmt.Fprintf(os.Stdout, "+ %s=%s\n", c.Key, c.Right)
				case env.Removed:
					fmt.Fprintf(os.Stdout, "- %s=%s\n", c.Key, c.Left)
				case env.Changed:
					fmt.Fprintf(os.Stdout, "~ %s: %s -> %s\n", c.Key, c.Left, c.Right)
				}
			}
			if exitCode && len(changes) > 0 {
				return errDifferencesFound{}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "exit 1 when differences are found")
	return cmd
}
