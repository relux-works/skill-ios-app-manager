package scaffold

import (
	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Initialize a new iOS project:

    ios-app-manager init

  This scaffolds Tuist manifests, app target, configuration,
  entitlements, Makefile, SwiftLint, and Periphery configs.

  Configure via ios-app-manager.json before running init.`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.Init,
		Name:         "Init",
		Description:  "Project scaffolding — Tuist manifests, app target, configuration",
		Category:     registry.Infra,
		Dependencies: []registry.ModuleID{},

		UsageGuide: usageGuide,
	})
}
