package cmd

import (
	"fmt"
	"os"

	"github.com/cybercapybara/envtool/internal/env"
	"github.com/spf13/cobra"
)

func newKeysCmd() *cobra.Command {
	var sortKeys bool
	cmd := &cobra.Command{
		Use:   "keys FILE",
		Short: "List the keys defined in a .env file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := env.ParseFile(args[0])
			if err != nil {
				return err
			}
			if sortKeys {
				doc = doc.SortedCopy()
			}
			for _, k := range doc.Keys() {
				fmt.Fprintln(os.Stdout, k)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&sortKeys, "sort", false, "sort keys alphabetically")
	return cmd
}
