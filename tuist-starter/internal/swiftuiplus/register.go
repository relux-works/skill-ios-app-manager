package swiftuiplus

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Import SwiftUIPlus in any module that needs SwiftUI re-export or UI components:

    import SwiftUIPlus

  Transitive SwiftUI import (no need to import SwiftUI separately):

    struct MyView: View { ... }  // SwiftUI available via SwiftUIPlus

  Async button with auto-disable during action:

    AsyncButton("Save") {
        await viewModel.save()
    }

    AsyncButton(action: { await refresh() }) {
        Label("Refresh", systemImage: "arrow.clockwise")
    }

  SwiftUIPlus is a single package (no Interface/Impl split).`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.SwiftUIPlus,
		Name:         "SwiftUIPlus",
		Description:  "SwiftUI re-export with UI components (AsyncButton)",
		Category:     registry.Utils,
		Dependencies: []registry.ModuleID{},

		Plan:       Plan,
		Setup:      SetupFromRegistry,
		UsageGuide: usageGuide,

		CLIUse:     "swiftui-plus",
		CLIShort:   "Manage SwiftUIPlus module",
		SetupShort: "Create SwiftUIPlus utility module",
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

// Plan returns a description of what setup will create/patch.
func Plan(input registry.SetupInput) (string, error) {
	plan := fmt.Sprintf(`## SwiftUIPlus Setup Plan

  Create:
    Packages/SwiftUIPlus/Sources/SwiftUIPlus/
      SwiftUIPlus.swift                   — @_exported import SwiftUI + namespace enum
      Components/AsyncButton.swift        — AsyncButton with auto-disable and progress

  Patch:
    Package.swift   — add SwiftUIPlus package path
    Project.swift   — add SwiftUIPlus dependency
    Workspace.swift — add SwiftUIPlus package path

  App: %s`, input.AppName)
	return plan, nil
}
