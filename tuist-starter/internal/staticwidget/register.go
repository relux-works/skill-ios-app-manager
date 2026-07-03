package staticwidget

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

const usageGuide = `## Usage

  Run after app-extensions + widget-base setup:

    ios-app-manager static-widget setup --yes

  This creates:
    - <AppName>Widget.swift with StaticConfiguration in <AppName>WidgetCore
    - <AppName>TimelineProvider.swift for placeholder/snapshot/timeline in <AppName>WidgetCore
    - <AppName>TimelineEntry.swift timeline entry model in <AppName>WidgetCore
    - <AppName>WidgetView.swift SwiftUI widget view in <AppName>WidgetCore

  It also patches WidgetBundle to import Core and register the new static widget.`

func init() {
	registry.Register(&registry.Module{
		ID:          registry.StaticWidget,
		Name:        "Static Widget",
		Category:    registry.Infra,
		Description: "Static WidgetKit widget with timeline provider",
		Dependencies: []registry.ModuleID{
			registry.WidgetBase,
			registry.AppIntents,
		},

		Plan:       planSetup,
		Setup:      runSetup,
		UsageGuide: usageGuide,

		CLIUse:     "static-widget",
		CLIShort:   "Manage static widget scaffolding",
		SetupShort: "Create static WidgetKit timeline widget files",
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

	plan := fmt.Sprintf(`## Static Widget Setup Plan

  Create:
    Extensions/%s/%sCore/Sources/%sWidget.swift
      — Widget struct with StaticConfiguration
    Extensions/%s/%sCore/Sources/%sTimelineProvider.swift
      — TimelineProvider implementation (placeholder/snapshot/timeline)
    Extensions/%s/%sCore/Sources/%sTimelineEntry.swift
      — timeline entry model (date + sample data)
    Extensions/%s/%sCore/Sources/%sWidgetView.swift
      — SwiftUI view for widget entry rendering

  Patch:
    Extensions/%s/Sources/%sBundle.swift
      — import %sCore and register %sWidget() in WidgetBundle body

  App: %s`, widgetTargetName, widgetTargetName, appTypeName, widgetTargetName, widgetTargetName, appTypeName, widgetTargetName, widgetTargetName, appTypeName, widgetTargetName, widgetTargetName, appTypeName, widgetTargetName, widgetTargetName, widgetTargetName, appTypeName, input.AppName)

	return plan, nil
}
