package cmd

import (
	"fmt"
	"os"

	"github.com/moveeeax/envtool/internal/env"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get KEY FILE",
		Short: "Print the value of a single key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := env.ParseFile(args[1])
			if err != nil {
				return err
			}
			value, ok := doc.Get(args[0])
			if !ok {
				return fmt.Errorf("key %q not found", args[0])
			}
			fmt.Fprintln(os.Stdout, value)
			return nil
		},
	}
	return cmd
}
