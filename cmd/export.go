package cmd

import (
	"os"

	"github.com/moveeeax/envtool/internal/env"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "export [flags] FILE",
		Short: "Convert a .env file to another format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := env.ParseFormat(format)
			if err != nil {
				return err
			}
			doc, err := env.ParseFile(args[0])
			if err != nil {
				return err
			}
			return env.Export(os.Stdout, doc, f)
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "json", "output format: dotenv|shell|json|yaml")
	return cmd
}
