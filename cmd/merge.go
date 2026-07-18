package cmd

import (
	"os"

	"github.com/cybercapybara/envtool/internal/env"
	"github.com/spf13/cobra"
)

func newMergeCmd() *cobra.Command {
	var (
		format   string
		sortKeys bool
	)
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
			merged := env.Merge(docs...)
			if sortKeys {
				merged = merged.SortedCopy()
			}
			return env.Export(os.Stdout, merged, f)
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "dotenv", "output format: dotenv|shell|json|yaml")
	cmd.Flags().BoolVar(&sortKeys, "sort", false, "sort keys alphabetically")
	return cmd
}
