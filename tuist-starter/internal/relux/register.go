package relux

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  App.swift uses Relux.Resolver for state-driven navigation:

    Relux.Resolver(splash: Splash(), content: Content(), resolver: resolver)

  Create feature modules with full Relux business logic:

    ios-app-manager module create Auth --type relux-feature

  This scaffolds Actions, Effects (interface) + State, Flow (impl).
  After creating modules, re-run ioc setup to update Registry:

    ios-app-manager ioc setup`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.Relux,
		Name:         "Relux",
		Description:  "Relux state management infrastructure",
		Category:     registry.Infra,
		Dependencies: []registry.ModuleID{registry.IoC},
		ExternalDeps: []registry.ExternalDep{
			{
				URL:     "https://github.com/relux-works/swiftui-relux.git",
				Version: "9.0.0",
				Product: "SwiftUIRelux",
				Package: "swiftui-relux",
			},
		},
		AdditionalFrameworkProducts: []string{
			"Relux",
			"ReluxRouter",
		},

		Plan:       Plan,
		Setup:      SetupFromRegistry,
		UsageGuide: usageGuide,

		CLIUse:     "relux",
		CLIShort:   "Manage Relux setup",
		SetupShort: "Set up Relux composition root",
	})
}

// SetupFromRegistry adapts registry.SetupInput to the local Setup() function.
func SetupFromRegistry(input registry.SetupInput) error {
	return Setup(SetupInput{
		ProjectRoot: input.ProjectRoot,
		AppName:     input.AppName,
		ModulesPath: input.ModulesPath,
	})
}

// Plan returns a human-readable plan of what setup will do.
func Plan(input registry.SetupInput) (string, error) {
	plan := fmt.Sprintf(`## Relux Setup Plan

  Create:
    Targets/%[1]s/Sources/App/%[1]s.Splash.swift     — splash screen view
    Targets/%[1]s/Sources/App/%[1]s.Content.swift     — main content view
    Targets/%[1]s/Sources/App/%[1]s.ReluxLogger.swift — Relux logging

  Patch:
    Package.swift  — add swiftui-relux dependency (from: 9.0.0)
    Project.swift  — add .external(name: "SwiftUIRelux")
    App.swift      — inject Relux.Resolver, @_exported import Relux
    Registry.swift — add Relux infrastructure builders (Store, RootSaga)`, input.AppName)
	return plan, nil
}
