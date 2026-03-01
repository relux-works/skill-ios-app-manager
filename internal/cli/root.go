package cli

import (
	"github.com/relux-works/ios-app-manager/internal/components"
	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/registry"
	"github.com/spf13/cobra"

	// Import module packages for init() registration.
	_ "github.com/relux-works/ios-app-manager/internal/appconfig"
	_ "github.com/relux-works/ios-app-manager/internal/httpclient"
	_ "github.com/relux-works/ios-app-manager/internal/ioc"
	_ "github.com/relux-works/ios-app-manager/internal/relux"
	_ "github.com/relux-works/ios-app-manager/internal/securestore"
	_ "github.com/relux-works/ios-app-manager/internal/tokenprovider"
	_ "github.com/relux-works/ios-app-manager/internal/utilities"
)

const defaultVersion = "dev"

var version = defaultVersion

// RootOptions captures root-level flag values shared by subcommands.
type RootOptions struct {
	ConfigPath string
	Verbose    bool
}

// SetVersion sets the version string rendered by --version.
func SetVersion(v string) {
	if v == "" {
		version = defaultVersion
		return
	}

	version = v
}

// Execute builds and executes the root command.
func Execute() error {
	return NewRootCommand().Execute()
}

// NewRootCommand builds the ios-app-manager command tree.
func NewRootCommand() *cobra.Command {
	return NewRootCommandWithAppManager(components.NewAppManager(nil, nil))
}

// NewRootCommandWithAppManager builds the command tree with an injected AppManager.
func NewRootCommandWithAppManager(appManager components.AppManager) *cobra.Command {
	if appManager == nil {
		appManager = components.NewAppManager(nil, nil)
	}

	opts := &RootOptions{}

	cmd := &cobra.Command{
		Use:          "ios-app-manager",
		Short:        "Manage iOS app project scaffolding",
		SilenceUsage: true,
	}

	cmd.Version = version
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.PersistentFlags().BoolVarP(
		&opts.Verbose,
		"verbose",
		"v",
		false,
		"Enable verbose output",
	)
	cmd.PersistentFlags().StringVarP(
		&opts.ConfigPath,
		"config",
		"c",
		config.DefaultConfigPath,
		"Path to project config JSON file",
	)

	cmd.AddCommand(
		newInitCommand(opts),
		newStatusCommand(opts, appManager),
		newModuleCommand(opts),
		newDependencyCommand(opts),
		newEntitlementsCommand(opts),
		newPushCommand(opts),
		newGenerateCommand(opts),
		newCleanCommand(opts),
		newQueryCommand(opts),
		newMutationCommand(opts),
	)

	// Registry-driven module commands (two-phase setup).
	for _, mod := range registry.AllSorted() {
		cmd.AddCommand(NewSetupCommand(mod, opts))
	}

	return cmd
}
