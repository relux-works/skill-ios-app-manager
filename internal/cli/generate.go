package cli

import (
	"errors"
	"fmt"
	"os"
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

	makefileCommand := &cobra.Command{
		Use:   "makefile",
		Short: "Generate or regenerate Makefile from project config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("generate makefile does not accept positional arguments")
			}

			selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			makefilePath := filepath.Clean(filepath.Join(filepath.Dir(selectedConfigPath), "Makefile"))
			existingContent, err := os.ReadFile(makefilePath)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("read existing makefile %q: %w", makefilePath, err)
			}

			existing := ""
			hadExisting := err == nil
			if hadExisting {
				existing = string(existingContent)
			}

			regenerated := scaffold.GenerateMakefilePreservingCustom(cfg, existing)
			if err := os.WriteFile(makefilePath, []byte(regenerated), 0o644); err != nil {
				return fmt.Errorf("write makefile %q: %w", makefilePath, err)
			}

			if !hadExisting {
				_, err = fmt.Fprintf(
					cmd.OutOrStdout(),
					"created %s (generated section + default custom section)\n",
					makefilePath,
				)
				return err
			}
			if regenerated == existing {
				_, err = fmt.Fprintf(
					cmd.OutOrStdout(),
					"makefile already up to date at %s (generated section verified, custom section preserved)\n",
					makefilePath,
				)
				return err
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"regenerated %s (updated generated section, preserved custom section)\n",
				makefilePath,
			)
			return err
		},
	}

	swiftlintCommand := &cobra.Command{
		Use:   "swiftlint",
		Short: "Generate or regenerate .swiftlint.yml from project config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("generate swiftlint does not accept positional arguments")
			}

			selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			swiftlintPath := filepath.Clean(filepath.Join(filepath.Dir(selectedConfigPath), ".swiftlint.yml"))
			existingContent, err := os.ReadFile(swiftlintPath)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("read existing swiftlint config %q: %w", swiftlintPath, err)
			}

			existing := ""
			hadExisting := err == nil
			if hadExisting {
				existing = string(existingContent)
			}

			regenerated := scaffold.GenerateSwiftLintConfig(cfg)
			if err := os.WriteFile(swiftlintPath, []byte(regenerated), 0o644); err != nil {
				return fmt.Errorf("write swiftlint config %q: %w", swiftlintPath, err)
			}

			if !hadExisting {
				_, err = fmt.Fprintf(
					cmd.OutOrStdout(),
					"created %s (generated from project config)\n",
					swiftlintPath,
				)
				return err
			}
			if regenerated == existing {
				_, err = fmt.Fprintf(
					cmd.OutOrStdout(),
					"swiftlint config already up to date at %s\n",
					swiftlintPath,
				)
				return err
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"regenerated %s (updated from project config)\n",
				swiftlintPath,
			)
			return err
		},
	}

	cmd.AddCommand(makefileCommand, swiftlintCommand)

	return cmd
}
