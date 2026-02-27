package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/utilities"
	"github.com/spf13/cobra"
)

func newUtilitiesCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "utilities",
		Short: "Manage utility modules",
		RunE:  runNotImplemented,
	}

	setupCommand := &cobra.Command{
		Use:   "setup",
		Short: "Create Utilities module with HttpClientUtils",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			selectedConfigPath := resolveSelectedConfigPath("", opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			projectRoot := filepath.Dir(selectedConfigPath)
			normalizedModulesPath := normalizeCLIPath(cfg.ModulesPath)

			appName := strings.TrimSpace(cfg.AppName)
			if appName == "" {
				return fmt.Errorf("app name is required in config")
			}

			if err := utilities.Setup(utilities.SetupInput{
				ProjectRoot: projectRoot,
				AppName:     appName,
				ModulesPath: normalizedModulesPath,
			}); err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Utilities setup complete")
			return err
		},
	}

	cmd.AddCommand(setupCommand)
	return cmd
}
