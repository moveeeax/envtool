package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/moveeeax/envtool/internal/env"
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
			rules := env.RulesFromKeys(strings.Split(required, ","))
			if len(rules) == 0 {
				// Without this guard a missing or mistyped --required silently
				// succeeds, so a CI gate would pass while checking nothing.
				return fmt.Errorf("--required must name at least one key")
			}
			violations := env.Validate(doc, rules)
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
