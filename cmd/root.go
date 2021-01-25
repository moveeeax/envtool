package cmd

import "github.com/spf13/cobra"

// version is overridden at build time via -ldflags "-X ...".
var version = "dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "envtool",
		Short:         "envtool merges, diffs, validates and redacts .env files",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newMergeCmd(),
		newDiffCmd(),
		newValidateCmd(),
		newRedactCmd(),
		newExportCmd(),
	)
	return root
}

// Execute runs the root command against os.Args.
func Execute() error {
	return newRootCmd().Execute()
}
