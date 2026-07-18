package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cybercapybara/envtool/internal/env"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var required string
	cmd := &cobra.Command{
		Use:   "validate [flags] FILE",
		Short: "Check that required keys are present and non-empty",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := env.ParseFile(args[0])
			if err != nil {
				return err
			}
			keys := strings.Split(required, ",")
			violations := env.Validate(doc, env.RulesFromKeys(keys))
			if len(violations) == 0 {
				return nil
			}
			for _, v := range violations {
				fmt.Fprintln(os.Stderr, v)
			}
			return fmt.Errorf("%d validation error(s)", len(violations))
		},
	}
	cmd.Flags().StringVar(&required, "required", "", "comma-separated required keys")
	return cmd
}
