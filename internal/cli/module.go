package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/ioc"
	"github.com/relux-works/ios-app-manager/internal/modules"
	"github.com/relux-works/ios-app-manager/internal/relux"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
	"github.com/spf13/cobra"
)

const defaultModulesPath = "Packages"

func newModuleCommand(opts *RootOptions) *cobra.Command {
	configPath := config.DefaultConfigPath
	if opts != nil && strings.TrimSpace(opts.ConfigPath) != "" {
		configPath = strings.TrimSpace(opts.ConfigPath)
	}

	cmd := &cobra.Command{
		Use:   "module",
		Short: "Manage modules",
		RunE:  runNotImplemented,
	}

	cmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		configPath,
		"Path to project config JSON file",
	)

	var moduleType string
	createCommand := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a module",
		RunE: func(cmd *cobra.Command, args []string) error {
			moduleName, moduleKind, selectedConfigPath, err := parseCreateModuleInput(args, moduleType, configPath)
			if err != nil {
				return err
			}

			moduleName, err = modules.ValidateModuleName(moduleName)
			if err != nil {
				return err
			}

			descriptor, err := modules.GetModuleType(moduleKind)
			if err != nil {
				return err
			}

			selectedConfigPath = resolveSelectedConfigPath(selectedConfigPath, opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			normalizedModulesPath := normalizeCLIPath(cfg.ModulesPath)
			projectRoot := filepath.Dir(selectedConfigPath)
			modulesRoot := resolveModulesRoot(projectRoot, normalizedModulesPath)

			tuistManager := tuistproj.NewTuistProjectManager(
				tuistproj.WithRootDir(projectRoot),
				tuistproj.WithModulesDir(normalizedModulesPath),
			)

			reluxManager, err := relux.NewReluxManager(modulesRoot)
			if err != nil {
				return fmt.Errorf("initialize relux manager: %w", err)
			}

			creator := modules.NewCreator(tuistManager, reluxManager)
			cfg.ModulesPath = modulesRoot
			if err := creator.Create(context.Background(), moduleName, string(descriptor.Type), cfg); err != nil {
				return err
			}

			// Re-scaffold Registry.swift if IoC is set up.
			if err := regenerateRegistryIfExists(projectRoot, cfg.AppName, modulesRoot); err != nil {
				return err
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"created module %q of type %q\n",
				moduleName,
				string(descriptor.Type),
			)
			return err
		},
	}
	createCommand.PersistentFlags().StringVar(
		&moduleType,
		"type",
		"",
		"Module type: feature|relux-feature|kit|shared|ui|utility",
	)

	var forceDelete bool
	listCommand := &cobra.Command{
		Use:   "list",
		Short: "List modules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			normalizedModulesPath := normalizeCLIPath(cfg.ModulesPath)
			projectRoot := filepath.Dir(selectedConfigPath)
			modulesRoot := resolveModulesRoot(projectRoot, normalizedModulesPath)

			lister := modules.NewLister()
			moduleList, err := lister.List(context.Background(), modulesRoot)
			if err != nil {
				return err
			}

			return printModuleListTable(cmd.OutOrStdout(), moduleList)
		},
	}

	deleteCommand := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			normalizedModulesPath := normalizeCLIPath(cfg.ModulesPath)
			projectRoot := filepath.Dir(selectedConfigPath)
			modulesRoot := resolveModulesRoot(projectRoot, normalizedModulesPath)

			deleter := modules.NewDeleter()
			result, err := deleter.Delete(context.Background(), args[0], modules.DeleteOptions{
				ModulesPath: modulesRoot,
				ProjectRoot: projectRoot,
				Force:       forceDelete,
				Confirm:     cliDeleteConfirmationPrompt(cmd),
			})
			if err != nil {
				if errors.Is(err, modules.ErrDeleteModuleCanceled) {
					_, writeErr := fmt.Fprintln(cmd.OutOrStdout(), "module delete canceled")
					return writeErr
				}
				return err
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"deleted module %q (packages: %s)\n",
				result.Module.Name,
				strings.Join(result.Module.PackageNames(), ", "),
			)
			return err
		},
	}
	deleteCommand.Flags().BoolVar(
		&forceDelete,
		"force",
		false,
		"Delete module without confirmation prompt",
	)

	cmd.AddCommand(
		createCommand,
		listCommand,
		deleteCommand,
	)

	return cmd
}

func parseCreateModuleInput(
	args []string,
	typeFromFlag string,
	configPathFromFlag string,
) (string, string, string, error) {
	moduleName := ""
	moduleType := strings.TrimSpace(typeFromFlag)
	configPath := strings.TrimSpace(configPathFromFlag)

	for i := 0; i < len(args); i++ {
		current := strings.TrimSpace(args[i])
		if current == "" {
			continue
		}

		if strings.HasPrefix(current, "--type=") {
			moduleType = strings.TrimSpace(strings.TrimPrefix(current, "--type="))
			continue
		}

		if current == "--type" {
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("module type is required (--type)")
			}
			i++
			moduleType = strings.TrimSpace(args[i])
			continue
		}

		if strings.HasPrefix(current, "--config=") {
			configPath = strings.TrimSpace(strings.TrimPrefix(current, "--config="))
			continue
		}

		if current == "--config" || current == "-c" {
			if i+1 >= len(args) {
				return "", "", "", fmt.Errorf("%s expects a value", current)
			}
			i++
			configPath = strings.TrimSpace(args[i])
			continue
		}

		if strings.HasPrefix(current, "-") {
			return "", "", "", fmt.Errorf("unknown flag %q", current)
		}

		if moduleName != "" {
			return "", "", "", fmt.Errorf("module create expects exactly one module name argument")
		}
		moduleName = current
	}

	if strings.TrimSpace(moduleName) == "" {
		return "", "", "", fmt.Errorf("module create expects exactly one module name argument")
	}
	if strings.TrimSpace(moduleType) == "" {
		return "", "", "", fmt.Errorf("module type is required (--type)")
	}

	return moduleName, moduleType, configPath, nil
}

func normalizeCLIPath(path string) string {
	value := strings.TrimSpace(path)
	if value == "" {
		return defaultModulesPath
	}
	return filepath.Clean(value)
}

func resolveModulesRoot(projectRoot string, modulesPath string) string {
	if filepath.IsAbs(modulesPath) {
		return filepath.Clean(modulesPath)
	}
	return filepath.Clean(filepath.Join(projectRoot, modulesPath))
}

func printModuleListTable(output io.Writer, modulesList []modules.ModuleInfo) error {
	if len(modulesList) == 0 {
		_, err := fmt.Fprintln(output, "no modules found")
		return err
	}

	writer := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "NAME\tTYPE\tPACKAGES\tDEPS"); err != nil {
		return err
	}

	for _, moduleInfo := range modulesList {
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%d\n",
			moduleInfo.Name,
			string(moduleInfo.Type),
			strings.Join(moduleInfo.PackageNames(), ", "),
			moduleInfo.DependencyCount,
		); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func regenerateRegistryIfExists(projectRoot, appName, modulesRoot string) error {
	appTypeName := scaffold.SwiftTypeName(appName)
	registryPath := filepath.Join(
		projectRoot, "Targets", appName, "Sources", "App",
		appTypeName+".Registry.swift",
	)

	if _, err := os.Stat(registryPath); err != nil {
		return nil // Registry doesn't exist yet — nothing to regenerate.
	}

	discoveredModules, err := ioc.DiscoverModules(modulesRoot)
	if err != nil {
		return fmt.Errorf("discover modules: %w", err)
	}

	hasRelux := registryHasRelux(registryPath)

	if err := ioc.ScaffoldRegistryWithData(registryPath, ioc.RegistryTemplateData{
		AppTypeName: appTypeName,
		Imports:     ioc.BuildModuleImports(discoveredModules),
		Modules:     discoveredModules,
		HasRelux:    hasRelux,
	}); err != nil {
		return fmt.Errorf("regenerate Registry.swift: %w", err)
	}

	return nil
}

func registryHasRelux(registryPath string) bool {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "import Relux") ||
		strings.Contains(string(data), "@_exported import Relux")
}

func cliDeleteConfirmationPrompt(cmd *cobra.Command) func(module modules.ModuleInfo) (bool, error) {
	return func(module modules.ModuleInfo) (bool, error) {
		if _, err := fmt.Fprintf(
			cmd.OutOrStdout(),
			"delete module %q (packages: %s)? [y/N]: ",
			module.Name,
			strings.Join(module.PackageNames(), ", "),
		); err != nil {
			return false, err
		}

		reader := bufio.NewReader(cmd.InOrStdin())
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}

		answer := strings.ToLower(strings.TrimSpace(line))
		return answer == "y" || answer == "yes", nil
	}
}
