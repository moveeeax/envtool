package cmd

import (
	"fmt"
	"os"

	"github.com/moveeeax/envtool/internal/env"
	"github.com/spf13/cobra"
)

func newSchemaCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "schema [flags] FILE",
		Short: "Document an annotated .env.example as a Markdown table or a JSON Schema",
		Long: `Print the contract declared in an annotated example file (see check) as a
Markdown table (name, type, required, default, description) or as a JSON
Schema draft 2020-12 document. Descriptions come from the comment text around
each key. Defaults of keys that look secret are left out.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			schema, err := env.ParseSchemaFile(args[0])
			if err != nil {
				return fmt.Errorf("example %s: %w", args[0], err)
			}
			return env.WriteSchema(os.Stdout, schema, format)
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "markdown", "output format: markdown|jsonschema")
	return cmd
}
