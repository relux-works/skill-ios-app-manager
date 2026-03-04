package widgetbase

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

const usageGuide = `## Usage

  Run once after app-extensions setup:

    ios-app-manager widget-base setup --yes

  This creates:
    - Extensions/<AppName>Widget app-extension target
    - <AppName>WidgetBundle.swift @main entry point

  Additional widget modules (static/configurable/live activity) can append
  their widgets to the generated WidgetBundle.`

func init() {
	registry.Register(&registry.Module{
		ID:          registry.WidgetBase,
		Name:        "Widget Base",
		Category:    registry.Infra,
		Description: "Base WidgetKit extension with WidgetBundle",
		Dependencies: []registry.ModuleID{
			registry.AppExtensions,
		},

		Plan:       planSetup,
		Setup:      runSetup,
		UsageGuide: usageGuide,

		CLIUse:     "widget-base",
		CLIShort:   "Manage WidgetKit base extension scaffolding",
		SetupShort: "Create base WidgetKit extension with WidgetBundle",
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
	widgetTargetName := widgetExtensionTargetName(input.AppName)
	appTypeName := scaffold.SwiftTypeName(input.AppName)

	plan := fmt.Sprintf(`## Widget Base Setup Plan

  Create:
    Extensions/%s/
      Project.swift                      — WidgetKit extension target
      Sources/%sWidgetBundle.swift       — @main WidgetBundle entry point

  Patch:
    Extensions/%s/Project.swift         — add WidgetKit dependency + App Groups entitlement
    Project.swift                        — embed widget extension in host app target
    Workspace.swift                      — add extension project path
    Tuist/ProjectDescriptionHelpers/AppCapabilities.swift — ensure app group capability

  App: %s`, widgetTargetName, appTypeName, widgetTargetName, input.AppName)

	return plan, nil
}
