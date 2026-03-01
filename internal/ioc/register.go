package ioc

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Registry.swift is the IoC composition root.
  All module registrations live here — one per protocol.

  Resolve a dependency anywhere in the app:

    let service: MyProtocol = IoC.resolve(MyProtocol.self)

  Async resolution (for actor-based implementations):

    let service: MyProtocol = await IoC.resolveAsync(MyProtocol.self)

  Modules with Interface/Impl split are auto-discovered.
  Create new modules with:

    ios-app-manager module create <Name> --type feature
    ios-app-manager ioc setup   # regenerates Registry.swift`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.IoC,
		Name:         "IoC",
		Description:  "SwiftIoC dependency injection container setup",
		Category:     registry.Infra,
		Dependencies: []registry.ModuleID{},

		Plan:       Plan,
		Setup:      SetupFromRegistry,
		UsageGuide: usageGuide,

		CLIUse:     "ioc",
		CLIShort:   "Manage IoC container",
		SetupShort: "Create IoC container with Registry.swift",
	})
}

// SetupFromRegistry adapts registry.SetupInput to the local Setup function.
func SetupFromRegistry(input registry.SetupInput) error {
	return Setup(SetupInput{
		ProjectRoot: input.ProjectRoot,
		AppName:     input.AppName,
		ModulesPath: input.ModulesPath,
	})
}

// Plan returns a description of what setup will create/modify.
func Plan(input registry.SetupInput) (string, error) {
	plan := fmt.Sprintf(`## IoC Setup Plan

  Create:
    Targets/%s/Sources/App/%s.Registry.swift
      - IoC container with configure() + resolve helpers
      - Auto-discovers modules in Packages/ with Interface/Impl split
      - Groups registrations by category (infra, foundation, features, network, utils)

  Patch:
    Package.swift  — add SwiftIoC dependency (from: 1.0.1)
    Project.swift  — add .external(name: "SwiftIoC")
    App.swift      — inject init() { Registry.configure() }`, input.AppName, input.AppName)
	return plan, nil
}
