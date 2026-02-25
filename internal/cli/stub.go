package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const notImplementedMessage = "not implemented"

func runNotImplemented(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), notImplementedMessage)
	return err
}
