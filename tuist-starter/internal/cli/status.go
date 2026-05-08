package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/components"
	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/spf13/cobra"
)

func newStatusCommand(opts *RootOptions, appManager components.AppManager) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show project status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("status does not accept positional arguments")
			}

			configPath := selectedStatusConfigPath(opts)
			if err := appManager.Init(cmd.Context(), configPath); err != nil {
				return fmt.Errorf("load project config: %w", err)
			}

			status, err := appManager.Status(cmd.Context())
			if err != nil {
				return fmt.Errorf("load project status: %w", err)
			}

			return writeProjectStatus(cmd.OutOrStdout(), status)
		},
	}
}

func selectedStatusConfigPath(opts *RootOptions) string {
	if opts != nil && strings.TrimSpace(opts.ConfigPath) != "" {
		return strings.TrimSpace(opts.ConfigPath)
	}

	return config.DefaultConfigPath
}

func writeProjectStatus(w io.Writer, status *components.ProjectStatus) error {
	if status == nil {
		return fmt.Errorf("project status is nil")
	}

	cfg := status.Config
	productName := strings.TrimSpace(cfg.ProductName)
	if productName == "" {
		productName = strings.TrimSpace(cfg.AppName)
	}

	if _, err := fmt.Fprintf(
		w,
		"project:\n  config: %s\n  app: %s\n  product: %s\n  bundle: %s\n  team: %s\n  min target: %s\n  swift: %s\n",
		status.ConfigPath,
		cfg.AppName,
		productName,
		cfg.BundleID,
		cfg.TeamID,
		cfg.MinTarget,
		cfg.SwiftVersion,
	); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		w,
		"modules:\n  path: %s\n  count: %d\n",
		status.ModulesPath,
		len(status.Modules),
	); err != nil {
		return err
	}

	for _, module := range status.Modules {
		if _, err := fmt.Fprintf(w, "  - %s\n", module); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, "dependency graph: %s\n", status.DependencyGraphHealth)
	return err
}
