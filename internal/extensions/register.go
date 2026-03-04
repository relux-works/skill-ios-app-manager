package extensions

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Run once to enable shared app-extension infrastructure:

    ios-app-manager app-extensions setup --yes

  This creates:
    - SharedKit package for code shared between host app and extensions
    - Extensions/ root directory for extension targets

  Extension-specific setup modules can then scaffold concrete targets
  (widget, notification service/content, etc.) using this base.`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.AppExtensions,
		Name:         "App Extensions",
		Category:     registry.Infra,
		Description:  "Base infrastructure for app extension targets",
		Dependencies: []registry.ModuleID{},

		Plan:       planSetup,
		Setup:      runSetup,
		UsageGuide: usageGuide,

		CLIUse:     "app-extensions",
		CLIShort:   "Manage app extension scaffolding",
		SetupShort: "Create base app extension infrastructure",
	})
}

func runSetup(input registry.SetupInput) error {
	return Setup(SetupInput{
		ProjectRoot: input.ProjectRoot,
		AppName:     input.AppName,
		ModulesPath: input.ModulesPath,
	})
}

func planSetup(input registry.SetupInput) (string, error) {
	modulesRelPath := strings.TrimSpace(input.ModulesPath)
	if modulesRelPath == "" {
		modulesRelPath = defaultModulesRelPath
	}

	sharedKitRef := filepath.ToSlash(filepath.Join(modulesRelPath, sharedKitModuleName))

	plan := fmt.Sprintf(`## App Extensions Setup Plan

  Create:
    %s/
      Package.swift           — SharedKit package manifest
      .module-type            — utility marker for registry grouping
      Sources/SharedKit.swift — shared namespace for extension-safe code
    %s/                        — base directory for extension targets

  Patch:
    Project.swift   — add .external(name: "SharedKit")
    Package.swift   — add .package(path: "%s")
    Workspace.swift — add .package(path: "%s") when dependencies section exists

  App: %s`, sharedKitRef, extensionsDirectoryName, sharedKitRef, sharedKitRef, input.AppName)

	return plan, nil
}
