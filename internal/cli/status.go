package cli

import (
	"github.com/relux-works/ios-app-manager/internal/components"
	"github.com/spf13/cobra"
)

func newStatusCommand(_ *RootOptions, _ components.AppManager) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show project status",
		RunE:  runNotImplemented,
	}
}
