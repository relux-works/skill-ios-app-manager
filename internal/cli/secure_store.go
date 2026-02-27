package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/securestore"
	"github.com/spf13/cobra"
)

func newSecureStoreCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secure-store",
		Short: "Manage SecureStore module",
		RunE:  runNotImplemented,
	}

	var accessGroup string

	setupCommand := &cobra.Command{
		Use:   "setup",
		Short: "Create SecureStore kit module with Keychain wrapper",
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

			// Validate --access-group flag against config app_groups.
			accessGroup = strings.TrimSpace(accessGroup)
			if err := validateAccessGroup(accessGroup, cfg.AppGroups); err != nil {
				return err
			}

			if err := securestore.Setup(securestore.SetupInput{
				ProjectRoot: projectRoot,
				AppName:     appName,
				ModulesPath: normalizedModulesPath,
				AccessGroup: accessGroup,
			}); err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "SecureStore setup complete")
			return err
		},
	}

	setupCommand.Flags().StringVar(&accessGroup, "access-group", "", "app group for shared keychain access (must exist in config app_groups)")

	cmd.AddCommand(setupCommand)
	return cmd
}

func validateAccessGroup(group string, configGroups []string) error {
	if group == "" {
		if len(configGroups) == 0 {
			return fmt.Errorf("--access-group is required but no app_groups defined in config\nadd groups via \"app_groups\" field in ios-app-manager.json, e.g.:\n  \"app_groups\": [\"group.com.example.app\"]")
		}
		return fmt.Errorf("--access-group is required\navailable groups in config: %v", configGroups)
	}

	for _, g := range configGroups {
		if g == group {
			return nil
		}
	}

	return fmt.Errorf("access group %q not found in config\navailable groups: %v", group, configGroups)
}
