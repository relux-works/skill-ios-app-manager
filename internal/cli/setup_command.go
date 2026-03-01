package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/registry"
	"github.com/spf13/cobra"
)

// NewSetupCommand creates a cobra command tree for a registry module.
// It creates a parent command (mod.CLIUse) with a "setup" subcommand
// that runs the two-phase Plan → Setup flow.
func NewSetupCommand(mod *registry.Module, opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   mod.CLIUse,
		Short: mod.CLIShort,
		RunE:  runNotImplemented,
	}

	var (
		yes    bool
		dryRun bool
	)

	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: mod.SetupShort,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Load config.
			selectedConfigPath := resolveSelectedConfigPath("", opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			projectRoot := filepath.Dir(selectedConfigPath)
			modulesPath := normalizeCLIPath(cfg.ModulesPath)

			appName := strings.TrimSpace(cfg.AppName)
			if appName == "" {
				return fmt.Errorf("app name is required in config")
			}

			// Collect ExtraFlags into ExtraArgs.
			extraArgs := make(map[string]string)
			for _, f := range mod.ExtraFlags {
				val, _ := cmd.Flags().GetString(f.Name)
				val = strings.TrimSpace(val)
				if f.Required && val == "" {
					return fmt.Errorf("--%s is required", f.Name)
				}
				if val != "" {
					extraArgs[f.ArgKey] = val
				}
			}

			input := registry.SetupInput{
				ProjectRoot: projectRoot,
				AppName:     appName,
				ModulesPath: modulesPath,
				ExtraArgs:   extraArgs,
			}

			// Verify module dependencies before creating the plan.
			if len(mod.Dependencies) > 0 {
				registryPath := filepath.Join(projectRoot, "Targets", appName, "Sources", "App", appName+".Registry.swift")
				content, err := os.ReadFile(registryPath)
				if err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("Registry.swift not found — run 'ioc setup' first")
					}
					return fmt.Errorf("read Registry.swift: %w", err)
				}
				if err := registry.CheckDependencies(mod.ID, string(content)); err != nil {
					return err
				}
			}

			// Phase 1: Plan.
			plan, err := mod.Plan(input)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), plan)

			if mod.UsageGuide != "" {
				fmt.Fprintln(cmd.OutOrStdout(), mod.UsageGuide)
			}

			// Dry-run: stop after plan.
			if dryRun {
				return nil
			}

			// Phase 2: Confirm (unless --yes).
			if !yes {
				fmt.Fprint(cmd.OutOrStdout(), "Proceed? [y/N] ")
				reader := bufio.NewReader(cmd.InOrStdin())
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "y" && answer != "yes" {
					return nil
				}
			}

			// Phase 3: Setup.
			if err := mod.Setup(input); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s setup complete\n", mod.Name)
			return nil
		},
	}

	setupCmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	setupCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print plan only, do not execute setup")

	for _, f := range mod.ExtraFlags {
		setupCmd.Flags().String(f.Name, "", f.Usage)
		if f.Required {
			_ = setupCmd.MarkFlagRequired(f.Name)
		}
	}

	cmd.AddCommand(setupCmd)
	return cmd
}
