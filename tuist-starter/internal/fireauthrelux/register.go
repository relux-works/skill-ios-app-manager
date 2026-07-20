package fireauthrelux

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  FireAuthRelux is registered container-scoped and attached to the existing
  Relux instance without replacing the app's Relux builder.

  Unit and hosted in-process tests may install a custom module before
  Registry.configure():

    App.Registry.installFireAuthReluxModuleFactoryForTesting { descriptor in
        TestFireAuthModule(descriptor: descriptor)
    }

  Call resetFireAuthReluxModuleFactoryForTesting() during test cleanup.
  UI tests run in a separate process. Select the generated deterministic,
  network-disabled module through the generated launch helper:

    app.launchArguments +=
        GeneratedFireAuthReluxTestLaunch.deterministicLaunchArguments
    app.launch()

  The equivalent generated launch environment is also available. App.swift
  evaluates the process selection before Registry.configure(...).
  Live configuration reads the selected runtime descriptor and the matching
  bundled plist. Missing fixture configuration is explicitly unconfigured.`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.FireAuthRelux,
		Name:         "FireAuthRelux",
		Description:  "Firebase REST authentication Relux module",
		Category:     registry.Foundation,
		Dependencies: []registry.ModuleID{registry.Relux, registry.TokenProvider, registry.AppConfig},

		Plan:       Plan,
		Setup:      SetupFromRegistry,
		UsageGuide: usageGuide,

		CLIUse:     "fireauth-relux",
		CLIShort:   "Manage FireAuthRelux composition",
		SetupShort: "Add FireAuthRelux without replacing custom Registry composition",
	})
}

// SetupFromRegistry loads the selected validated project configuration.
func SetupFromRegistry(input registry.SetupInput) error {
	cfg, err := config.LoadConfig(input.ConfigPath)
	if err != nil {
		return fmt.Errorf("load FireAuthRelux config: %w", err)
	}
	return Setup(SetupInput{
		ProjectRoot: input.ProjectRoot,
		AppName:     input.AppName,
		ModulesPath: input.ModulesPath,
		Config:      cfg,
	})
}

// Plan describes the exact managed output. Dependency and Firebase validation
// occurs in Setup before any mutation.
func Plan(input registry.SetupInput) (string, error) {
	return fmt.Sprintf(`## FireAuthRelux Setup Plan

  Create:
    Targets/%[1]s/Sources/Configuration/FireAuth/GeneratedFireAuthRelux.swift
      — environment-keyed Firebase bundle loader, live module, deterministic
        network-disabled module, and injectable in-process factory
    Targets/%[1]s/Sources/Configuration/FireAuth/GeneratedFireAuthReluxProcess.swift
      — launch-argument/environment selection parsed in the app process
    configured test targets/Support/GeneratedFireAuthReluxTestLaunch.swift
      — typed XCUITest launch arguments and environment

  Converge:
    Package.swift  — exact FireAuthRelux 1.2.1 and FireAuthKit 1.1.0 pins
                     plus framework and iOS target overrides
    Project.swift  — FireAuthRelux, FireAuthKit, and FireAuthProvider products
    Registry.swift — focused container registration, builder, deterministic
                     test factory, and wrapper around the existing Relux builder
    App.swift      — managed process-selection call immediately before the
                     existing Registry.configure(...) call

  Existing runtime mode, Relux body, persistence, sync, API, AppConfig, and
  hosted-test composition remain user-owned and unchanged.

  App: %[1]s`, input.AppName), nil
}
