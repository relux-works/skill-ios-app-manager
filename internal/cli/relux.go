package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/relux"
	"github.com/spf13/cobra"
)

func newReluxCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relux",
		Short: "Manage Relux integration",
		RunE:  runNotImplemented,
	}

	setupCommand := &cobra.Command{
		Use:   "setup",
		Short: "Set up Relux in the project (requires ioc setup first)",
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

			if err := relux.Setup(relux.SetupInput{
				ProjectRoot: projectRoot,
				AppName:     appName,
				ModulesPath: normalizedModulesPath,
			}); err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Relux setup complete")
			return err
		},
	}

	cmd.AddCommand(setupCommand)
	return cmd
}
