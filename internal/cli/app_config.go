package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/appconfig"
	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/spf13/cobra"
)

func newAppConfigCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app-config",
		Short: "Manage AppConfig environment switching and API configuration",
		RunE:  runNotImplemented,
	}

	setupCommand := &cobra.Command{
		Use:   "setup",
		Short: "Scaffold AppConfig manager with environment switching and ApiConfigurator",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			selectedConfigPath := resolveSelectedConfigPath("", opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			projectRoot := filepath.Dir(selectedConfigPath)

			appName := strings.TrimSpace(cfg.AppName)
			if appName == "" {
				return fmt.Errorf("app name is required in config")
			}

			if err := appconfig.Setup(appconfig.SetupInput{
				ProjectRoot: projectRoot,
				AppName:     appName,
			}); err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "AppConfig setup complete")
			return err
		},
	}

	cmd.AddCommand(setupCommand)
	return cmd
}
