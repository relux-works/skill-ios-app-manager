package cli

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/entitlements"
	"github.com/spf13/cobra"
)

func newEntitlementsCommand(opts *RootOptions) *cobra.Command {
	configPath := config.DefaultConfigPath
	if opts != nil && strings.TrimSpace(opts.ConfigPath) != "" {
		configPath = strings.TrimSpace(opts.ConfigPath)
	}

	var entitlementsPath string
	cmd := &cobra.Command{
		Use:   "entitlements",
		Short: "List entitlements from plist file",
		RunE:  runNotImplemented,
	}

	cmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		configPath,
		"Path to project config JSON file",
	)
	cmd.PersistentFlags().StringVarP(
		&entitlementsPath,
		"path",
		"p",
		"",
		"Path to entitlements plist file",
	)

	listCommand := &cobra.Command{
		Use:   "list",
		Short: "List entitlements",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("entitlements list does not accept positional arguments")
			}

			resolvedPath, err := resolveEntitlementsPath(entitlementsPath, configPath, opts)
			if err != nil {
				return fmt.Errorf("resolve entitlements path: %w", err)
			}

			entries, err := entitlements.List(resolvedPath)
			if err != nil {
				return fmt.Errorf("list entitlements: %w", err)
			}

			if len(entries) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "no entitlements configured")
				return err
			}

			for _, entry := range entries {
				if _, err := fmt.Fprintf(
					cmd.OutOrStdout(),
					"%s = %s\n",
					entry.Key,
					formatEntitlementValue(entry.Value),
				); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.AddCommand(listCommand)

	return cmd
}

func resolveEntitlementsPath(entitlementsPath string, configPath string, opts *RootOptions) (string, error) {
	explicitPath := strings.TrimSpace(entitlementsPath)
	if explicitPath != "" {
		return filepath.Clean(explicitPath), nil
	}

	selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
	cfg, err := config.LoadConfig(selectedConfigPath)
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	appName := strings.TrimSpace(cfg.AppName)
	if appName == "" {
		return "", fmt.Errorf("config %q contains empty app_name", selectedConfigPath)
	}

	return filepath.Clean(filepath.Join(filepath.Dir(selectedConfigPath), appName+".entitlements")), nil
}

func resolveSelectedConfigPath(configPath string, opts *RootOptions) string {
	selectedConfigPath := strings.TrimSpace(configPath)
	if (selectedConfigPath == "" || selectedConfigPath == config.DefaultConfigPath) &&
		opts != nil &&
		strings.TrimSpace(opts.ConfigPath) != "" {
		selectedConfigPath = strings.TrimSpace(opts.ConfigPath)
	}

	if selectedConfigPath == "" {
		selectedConfigPath = config.DefaultConfigPath
	}

	return selectedConfigPath
}

func formatEntitlementValue(value entitlements.Value) string {
	switch value.Kind {
	case entitlements.ValueKindString:
		return value.StringValue
	case entitlements.ValueKindBool:
		return strconv.FormatBool(value.BoolValue)
	case entitlements.ValueKindStringArray:
		return "[" + strings.Join(value.ArrayValue, ", ") + "]"
	default:
		return "<unknown>"
	}
}
