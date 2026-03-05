package cli

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/dsl"
	"github.com/spf13/cobra"
)

type mutationCommandOptions struct {
	format string
}

func newMutationCommand(_ *RootOptions) *cobra.Command {
	executor := dsl.NewStubMutationExecutor()
	opts := &mutationCommandOptions{format: outputFormatPretty}

	cmd := &cobra.Command{
		Use:   "m <expression>",
		Short: "Run mutation DSL",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("m expects exactly one DSL expression")
			}

			expression, err := dsl.ParseExpression(args[0])
			if err != nil {
				return fmt.Errorf("parse mutation expression: %w", err)
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
