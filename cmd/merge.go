package cmd

import (
	"os"

	"github.com/moveeeax/envtool/internal/env"
	"github.com/spf13/cobra"
)

func newMergeCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "merge [flags] FILE...",
		Short: "Merge multiple .env files (later files win)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := env.ParseFormat(format)
			if err != nil {
				return err
			}
			docs := make([]*env.Doc, 0, len(args))
			for _, path := range args {
				doc, err := env.ParseFile(path)
				if err != nil {
					return err
				}
				docs = append(docs, doc)
			}
			return env.Export(os.Stdout, env.Merge(docs...), f)
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "dotenv", "output format: dotenv|shell|json|yaml")
	return cmd
}
