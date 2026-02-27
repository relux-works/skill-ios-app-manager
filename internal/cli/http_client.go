package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/httpclient"
	"github.com/spf13/cobra"
)

func newHttpClientCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "http-client",
		Short: "Manage HttpClient IoC registration",
		RunE:  runNotImplemented,
	}

	setupCommand := &cobra.Command{
		Use:   "setup",
		Short: "Add HttpClient IoC registration with swift-httpclient and configuration",
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

			if err := httpclient.Setup(httpclient.SetupInput{
				ProjectRoot: projectRoot,
				AppName:     appName,
				ModulesPath: normalizedModulesPath,
			}); err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "HttpClient setup complete")
			return err
		},
	}

	cmd.AddCommand(setupCommand)
	return cmd
}
