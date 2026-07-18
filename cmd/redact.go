package cmd

import (
	"os"
	"strings"

	"github.com/cybercapybara/envtool/internal/env"
	"github.com/spf13/cobra"
)

func newRedactCmd() *cobra.Command {
	var (
		format  string
		keep    int
		matchStr string
	)
	cmd := &cobra.Command{
		Use:   "redact [flags] FILE",
		Short: "Mask values of sensitive keys",
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
			var matchers []string
			if strings.TrimSpace(matchStr) != "" {
				matchers = strings.Split(matchStr, ",")
			}
			return env.Export(os.Stdout, env.Redact(doc, matchers, keep), f)
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "dotenv", "output format: dotenv|shell|json|yaml")
	cmd.Flags().IntVar(&keep, "keep", 0, "keep this many leading characters visible")
	cmd.Flags().StringVar(&matchStr, "match", "", "comma-separated key substrings to treat as secret")
	return cmd
}
