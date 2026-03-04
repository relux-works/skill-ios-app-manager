package appintents

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

const usageGuide = `## Usage

  Run after widget-base setup:

    ios-app-manager app-intents setup --yes

  This creates:
    - <AppName>WidgetToggleIntent.swift — sample AppIntent with shared state mutation

  It also adds AppIntents SDK to the widget extension dependencies.`

func init() {
	registry.Register(&registry.Module{
		ID:          registry.AppIntents,
		Name:        "App Intents",
		Category:    registry.Infra,
		Description: "App Intents infrastructure for interactive widgets",
		Dependencies: []registry.ModuleID{
			registry.WidgetBase,
		},

		Plan:       planSetup,
		Setup:      runSetup,
		UsageGuide: usageGuide,

		CLIUse:     "app-intents",
		CLIShort:   "Manage App Intents scaffolding",
		SetupShort: "Create App Intents infrastructure for widgets",
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
	appTypeName := scaffold.SwiftTypeName(input.AppName)
	widgetTargetName := widgetExtensionTargetName(input.AppName)

	plan := fmt.Sprintf(`## App Intents Setup Plan

  Create:
    Extensions/%s/Sources/%sWidgetToggleIntent.swift
      — sample AppIntent: toggles shared state via App Groups UserDefaults

  Patch:
    Extensions/%s/Project.swift
      — add AppIntents SDK dependency

  App: %s`, widgetTargetName, appTypeName, widgetTargetName, input.AppName)

	return plan, nil
}
