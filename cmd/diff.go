package cmd

import (
	"fmt"
	"os"

	"github.com/cybercapybara/envtool/internal/env"
	"github.com/spf13/cobra"
)

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
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "exit 1 when differences are found")
	return cmd
}
