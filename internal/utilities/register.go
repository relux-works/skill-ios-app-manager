package utilities

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Import Utilities in any module that needs HTTP helpers:

    import Utilities

  JSON encoding/decoding:

    let encoder = BaseEncoder()
    let decoder = BaseDecoder()

  Standard HTTP headers:

    HeaderMaps.jsonHeaders          // Content-Type + Accept for JSON
    HeaderMaps.formHeaders          // form-encoded headers
    HeaderMaps.authHeader(token: t) // Bearer authorization header

  Utilities is a single package (no Interface/Impl split).`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.Utilities,
		Name:         "Utilities",
		Description:  "Shared utility helpers",
		Category:     registry.Utils,
		Dependencies: []registry.ModuleID{},

		Plan:       Plan,
		Setup:      SetupFromRegistry,
		UsageGuide: usageGuide,

		CLIUse:     "utilities",
		CLIShort:   "Manage Utilities module",
		SetupShort: "Create shared Utilities module",
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
	plan := fmt.Sprintf(`## Utilities Setup Plan

  Create:
    Packages/Utilities/Sources/Utilities/
      HttpClientUtils/HeaderMaps.swift   — HTTP header constants
      HttpClientUtils/BaseEncoder.swift  — JSON encoder helper
      HttpClientUtils/BaseDecoder.swift  — JSON decoder helper

  Patch:
    Package.swift   — add Utilities package path
    Project.swift   — add Utilities dependency
    Registry.swift  — will be updated on next ioc setup

  App: %s`, input.AppName)
	return plan, nil
}
