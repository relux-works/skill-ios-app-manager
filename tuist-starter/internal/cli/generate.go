package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	"github.com/spf13/cobra"
)

func newGenerateCommand(opts *RootOptions) *cobra.Command {
	configPath := config.DefaultConfigPath
	if opts != nil && strings.TrimSpace(opts.ConfigPath) != "" {
		configPath = strings.TrimSpace(opts.ConfigPath)
	}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate project artifacts",
	}

	cmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		configPath,
		"Path to project config JSON file",
	)
	for _, plugin := range scaffold.AllGenerators() {
		plugin := plugin
		subcommand := &cobra.Command{
			Use:   plugin.Name,
			Short: plugin.Short,
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) != 0 {
					return fmt.Errorf("generate %s does not accept positional arguments", plugin.Name)
				}

				selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
				cfg, err := config.LoadConfig(selectedConfigPath)
				if err != nil {
					return fmt.Errorf("load config: %w", err)
				}

				result, err := plugin.Run(scaffold.GenerateInput{
					ConfigPath:  selectedConfigPath,
					ProjectRoot: filepath.Dir(selectedConfigPath),
					Config:      cfg,
				})
				if err != nil {
					return fmt.Errorf("run generate plugin %s: %w", plugin.Name, err)
				}

				if result.Message == "" {
					return nil
				}

				_, err = fmt.Fprint(cmd.OutOrStdout(), result.Message)
				return err
			},
		}

		cmd.AddCommand(subcommand)
	}

	return cmd
}
