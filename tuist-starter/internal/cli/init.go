package cli

import (
	"fmt"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	templaterenderer "github.com/relux-works/ios-app-manager/internal/template"
	"github.com/spf13/cobra"
)

func newInitCommand(opts *RootOptions) *cobra.Command {
	configPath := config.DefaultConfigPath
	if opts != nil && strings.TrimSpace(opts.ConfigPath) != "" {
		configPath = strings.TrimSpace(opts.ConfigPath)
	}

	var outputDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			selectedConfigPath := strings.TrimSpace(configPath)
			if (selectedConfigPath == "" || selectedConfigPath == config.DefaultConfigPath) &&
				opts != nil &&
				strings.TrimSpace(opts.ConfigPath) != "" {
				selectedConfigPath = strings.TrimSpace(opts.ConfigPath)
			}
			if selectedConfigPath == "" {
				selectedConfigPath = config.DefaultConfigPath
			}

			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			targetOutputDir := strings.TrimSpace(outputDir)
			if targetOutputDir == "" {
				targetOutputDir = "."
			}

			renderer := templaterenderer.NewRenderer(templaterenderer.WithRootDir(targetOutputDir))
			scaffolder := scaffold.New(renderer)

			written, err := scaffolder.Scaffold(cfg, targetOutputDir, force)
			if err != nil {
				return fmt.Errorf("scaffold project: %w", err)
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"scaffolded %d files in %s\n",
				len(written),
				targetOutputDir,
			)
			return err
		},
	}

	cmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		configPath,
		"Path to project config JSON file",
	)
	cmd.PersistentFlags().StringVar(
		&outputDir,
		"output",
		".",
		"Target directory where the project scaffold is created",
	)
	cmd.PersistentFlags().BoolVar(
		&force,
		"force",
		false,
		"Overwrite existing scaffold files in the output directory",
	)

	return cmd
}
