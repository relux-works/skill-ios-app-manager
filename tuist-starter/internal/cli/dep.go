package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/deps"
	"github.com/spf13/cobra"
)

func newDependencyCommand(opts *RootOptions) *cobra.Command {
	configPath := config.DefaultConfigPath
	if opts != nil && strings.TrimSpace(opts.ConfigPath) != "" {
		configPath = strings.TrimSpace(opts.ConfigPath)
	}

	cmd := &cobra.Command{
		Use:   "dep",
		Short: "Manage dependencies",
	}

	cmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		configPath,
		"Path to project config JSON file",
	)

	var addDependsOn string
	addCommand := &cobra.Command{
		Use:   "add <module>",
		Short: "Add an internal module dependency",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dependency := strings.TrimSpace(addDependsOn)
			if dependency == "" {
				return fmt.Errorf("--depends-on is required")
			}

			modulesRoot, err := resolveDependencyModulesRoot(configPath, opts)
			if err != nil {
				return err
			}

			moduleName := strings.TrimSpace(args[0])
			if err := deps.AddInternalDep(moduleName, dependency, modulesRoot); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "added dependency %q -> %q\n", moduleName, dependency)
			return err
		},
	}
	addCommand.Flags().StringVar(
		&addDependsOn,
		"depends-on",
		"",
		"Dependency module interface name",
	)

	var removeDependsOn string
	removeCommand := &cobra.Command{
		Use:   "remove <module>",
		Short: "Remove an internal module dependency",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dependency := strings.TrimSpace(removeDependsOn)
			if dependency == "" {
				return fmt.Errorf("--depends-on is required")
			}

			modulesRoot, err := resolveDependencyModulesRoot(configPath, opts)
			if err != nil {
				return err
			}

			moduleName := strings.TrimSpace(args[0])
			if err := deps.RemoveInternalDep(moduleName, dependency, modulesRoot); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "removed dependency %q -> %q\n", moduleName, dependency)
			return err
		},
	}
	removeCommand.Flags().StringVar(
		&removeDependsOn,
		"depends-on",
		"",
		"Dependency module interface name",
	)

	var addExternalURL string
	var addExternalVersion string
	var addExternalModule string
	var addExternalProducts []string
	var addExternalTargetSettings []string
	var addExternalAppTarget bool
	addExternalCommand := &cobra.Command{
		Use:   "add-external",
		Short: "Add an external Swift package dependency",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			url := strings.TrimSpace(addExternalURL)
			if url == "" {
				return fmt.Errorf("--url is required")
			}

			version := strings.TrimSpace(addExternalVersion)
			if version == "" {
				return fmt.Errorf("--version is required")
			}

			modulesRoot, err := resolveDependencyModulesRoot(configPath, opts)
			if err != nil {
				return err
			}

			targetModule := strings.TrimSpace(addExternalModule)
			if err := deps.AddExternalDep(url, version, "", targetModule, modulesRoot, addExternalProducts...); err != nil {
				return err
			}
			productNames := addExternalProducts
			if len(productNames) == 0 {
				productName, err := deps.InferExternalPackageName("", url)
				if err != nil {
					return err
				}
				productNames = []string{productName}
			}
			targetSettings, err := parseTargetSettings(addExternalTargetSettings)
			if err != nil {
				return err
			}
			if len(targetSettings) > 0 {
				if err := deps.AddExternalProductTargetSettings(modulesRoot, productNames, targetSettings); err != nil {
					return err
				}
			}
			if addExternalAppTarget {
				if err := deps.AddExternalProductsToAppTarget(modulesRoot, productNames...); err != nil {
					return err
				}
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "added external dependency %q\n", url)
			return err
		},
	}
	addExternalCommand.Flags().StringVar(
		&addExternalURL,
		"url",
		"",
		"Git URL of external Swift package",
	)
	addExternalCommand.Flags().StringVar(
		&addExternalVersion,
		"version",
		"",
		`Version requirement (1.0.0, from: "1.0.0", exact: "1.0.0", branch: "main", revision: "abc123")`,
	)
	addExternalCommand.Flags().StringVar(
		&addExternalModule,
		"module",
		"",
		"Optional target module to link with this external package",
	)
	addExternalCommand.Flags().StringArrayVar(
		&addExternalProducts,
		"product",
		nil,
		"Swift product name exposed by the package; repeat for multiple products",
	)
	addExternalCommand.Flags().StringArrayVar(
		&addExternalTargetSettings,
		"target-setting",
		nil,
		"Tuist PackageSettings target build setting KEY=VALUE for product(s); repeat for multiple settings",
	)
	addExternalCommand.Flags().BoolVar(
		&addExternalAppTarget,
		"app-target",
		false,
		"Link product(s) into the host app target in Project.swift",
	)

	var removeExternalPackage string
	removeExternalCommand := &cobra.Command{
		Use:   "remove-external",
		Short: "Remove an external Swift package dependency",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := strings.TrimSpace(removeExternalPackage)
			if packageName == "" {
				return fmt.Errorf("--package is required")
			}

			modulesRoot, err := resolveDependencyModulesRoot(configPath, opts)
			if err != nil {
				return err
			}

			if err := deps.RemoveExternalDep(packageName, modulesRoot); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "removed external dependency %q\n", packageName)
			return err
		},
	}
	removeExternalCommand.Flags().StringVar(
		&removeExternalPackage,
		"package",
		"",
		"External package name",
	)

	listCommand := &cobra.Command{
		Use:   "list [module]",
		Short: "List internal and external dependencies",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			modulesRoot, err := resolveDependencyModulesRoot(configPath, opts)
			if err != nil {
				return err
			}

			moduleName := ""
			if len(args) == 1 {
				moduleName = strings.TrimSpace(args[0])
			}

			lookup, err := deps.ListInternalDeps(moduleName, modulesRoot)
			if err != nil {
				return err
			}

			externalLookup, err := deps.ListExternalDeps(modulesRoot)
			if err != nil {
				return err
			}

			if len(args) == 1 {
				externalLookup = filterExternalDependenciesByModule(externalLookup, firstModuleNameFromLookup(lookup))
			}

			if err := printDependencyTable(cmd.OutOrStdout(), lookup); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
				return err
			}

			return printExternalDependencyTable(cmd.OutOrStdout(), externalLookup)
		},
	}

	cmd.AddCommand(
		addCommand,
		removeCommand,
		addExternalCommand,
		removeExternalCommand,
		listCommand,
	)

	return cmd
}

func parseTargetSettings(rawSettings []string) (map[string]string, error) {
	settings := make(map[string]string, len(rawSettings))
	for _, rawSetting := range rawSettings {
		trimmed := strings.TrimSpace(rawSetting)
		if trimmed == "" {
			continue
		}
		key, value, found := strings.Cut(trimmed, "=")
		if !found {
			return nil, fmt.Errorf("--target-setting must use KEY=VALUE format")
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			return nil, fmt.Errorf("--target-setting must use non-empty KEY=VALUE format")
		}
		settings[key] = value
	}
	return settings, nil
}

func resolveDependencyModulesRoot(configPath string, opts *RootOptions) (string, error) {
	selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
	cfg, err := config.LoadConfig(selectedConfigPath)
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	projectRoot := filepath.Dir(selectedConfigPath)
	normalizedModulesPath := normalizeCLIPath(cfg.ModulesPath)
	return resolveModulesRoot(projectRoot, normalizedModulesPath), nil
}

func printDependencyTable(output io.Writer, lookup map[string][]string) error {
	if len(lookup) == 0 {
		_, err := fmt.Fprintln(output, "no internal dependencies found")
		return err
	}

	modules := make([]string, 0, len(lookup))
	for moduleName := range lookup {
		modules = append(modules, moduleName)
	}
	sort.Strings(modules)

	writer := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "MODULE\tDEPENDS_ON"); err != nil {
		return err
	}

	for _, moduleName := range modules {
		dependencies := append([]string(nil), lookup[moduleName]...)
		sort.Strings(dependencies)

		display := "-"
		if len(dependencies) > 0 {
			display = strings.Join(dependencies, ", ")
		}

		if _, err := fmt.Fprintf(writer, "%s\t%s\n", moduleName, display); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func firstModuleNameFromLookup(lookup map[string][]string) string {
	for moduleName := range lookup {
		return moduleName
	}
	return ""
}

func filterExternalDependenciesByModule(
	allDependencies []deps.ExternalDependency,
	moduleName string,
) []deps.ExternalDependency {
	selectedModule := strings.TrimSpace(moduleName)
	if selectedModule == "" {
		return append([]deps.ExternalDependency(nil), allDependencies...)
	}

	filtered := make([]deps.ExternalDependency, 0, len(allDependencies))
	for _, dependency := range allDependencies {
		if dependency.Scope == depsExternalScopeRoot || dependency.Scope == selectedModule {
			filtered = append(filtered, dependency)
		}
	}
	return filtered
}

func printExternalDependencyTable(output io.Writer, lookup []deps.ExternalDependency) error {
	if len(lookup) == 0 {
		_, err := fmt.Fprintln(output, "no external dependencies found")
		return err
	}

	writer := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "SCOPE\tPACKAGE\tVERSION\tURL"); err != nil {
		return err
	}

	for _, dependency := range lookup {
		scope := strings.TrimSpace(dependency.Scope)
		if scope == "" {
			scope = depsExternalScopeRoot
		}

		requirement := strings.TrimSpace(dependency.Requirement)
		if requirement == "" {
			requirement = "-"
		}

		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			scope,
			dependency.PackageName,
			requirement,
			dependency.URL,
		); err != nil {
			return err
		}
	}

	return writer.Flush()
}

const depsExternalScopeRoot = "root"
