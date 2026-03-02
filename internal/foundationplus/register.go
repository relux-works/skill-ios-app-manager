package foundationplus

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Import FoundationPlus in any module that needs Foundation re-export or utility types:

    import FoundationPlus

  Transitive Foundation import (no need to import Foundation separately):

    let data = try JSONEncoder().encode(value)  // Foundation available via FoundationPlus

  Loading state wrapper:

    var items: MaybeData<[Item], MyError> = .initial
    items = .loading
    items = .success([item1, item2])
    items = .failure(.networkError)

  Completion status:

    var status: CompletionStatus = .idle
    status = .inProgress
    status = .completed

  FoundationPlus is a single package (no Interface/Impl split).`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.FoundationPlus,
		Name:         "FoundationPlus",
		Description:  "Foundation re-export with utility types (MaybeData, CompletionStatus)",
		Category:     registry.Utils,
		Dependencies: []registry.ModuleID{},

		Plan:       Plan,
		Setup:      SetupFromRegistry,
		UsageGuide: usageGuide,

		CLIUse:     "foundation-plus",
		CLIShort:   "Manage FoundationPlus module",
		SetupShort: "Create FoundationPlus utility module",
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
	plan := fmt.Sprintf(`## FoundationPlus Setup Plan

  Create:
    Packages/FoundationPlus/Sources/FoundationPlus/
      FoundationPlus.swift     — @_exported import Foundation + namespace enum
      MaybeData.swift          — MaybeData<Success, Failure> loading state wrapper
      CompletionStatus.swift   — CompletionStatus enum

  Patch:
    Package.swift   — add FoundationPlus package path
    Project.swift   — add FoundationPlus dependency
    Workspace.swift — add FoundationPlus package path

  App: %s`, input.AppName)
	return plan, nil
}
