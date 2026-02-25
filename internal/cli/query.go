package cli

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/dsl"
	"github.com/spf13/cobra"
)

type queryCommandOptions struct {
	format string
}

func newQueryCommand(_ *RootOptions) *cobra.Command {
	executor := dsl.NewStubQueryExecutor()
	opts := &queryCommandOptions{format: outputFormatPretty}

	cmd := &cobra.Command{
		Use:   "q <expression>",
		Short: "Run query DSL",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("q expects exactly one DSL expression")
			}

			expression, err := dsl.ParseExpression(args[0])
			if err != nil {
				return fmt.Errorf("parse query expression: %w", err)
			}

			result, err := executor.Execute(expression)
			if err != nil {
				return err
			}

			return writeDSLResult(cmd.OutOrStdout(), opts.format, result)
		},
	}

	cmd.PersistentFlags().StringVar(
		&opts.format,
		"format",
		outputFormatPretty,
		"Output format: pretty or compact",
	)

	return cmd
}
