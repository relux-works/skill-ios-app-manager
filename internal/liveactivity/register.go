package liveactivity

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Run after app-extensions + widget-base setup:

    ios-app-manager live-activity setup --yes

  This creates:
    - SharedKit ActivityAttributes + ContentState model
    - Widget extension ActivityConfiguration + Dynamic Island UI
    - App-side LiveActivityManager (start/update/end + push token stub)

  It also patches:
    - Host app Info.plist dictionary in Project.swift:
      - NSSupportsLiveActivities = true
      - NSSupportsLiveActivitiesFrequentUpdates = true
    - WidgetBundle body to register the Live Activity widget.`

func init() {
	registry.Register(&registry.Module{
		ID:          registry.LiveActivity,
		Name:        "Live Activity",
		Category:    registry.Infra,
		Description: "Live Activity with ActivityKit + Dynamic Island",
		Dependencies: []registry.ModuleID{
			registry.WidgetBase,
		},

		Plan:       planSetup,
		Setup:      runSetup,
		UsageGuide: usageGuide,

		CLIUse:     "live-activity",
		CLIShort:   "Manage Live Activity scaffolding",
		SetupShort: "Create ActivityKit lifecycle + Live Activity widget files",
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
	plan := fmt.Sprintf(`## Live Activity Setup Plan

  Create:
    Packages/SharedKit/Sources/%sActivityAttributes.swift
      — ActivityAttributes + ContentState shared by app and widget extension
    Extensions/%sWidget/Sources/%sLiveActivityWidget.swift
      — ActivityConfiguration + Dynamic Island presentation
    Targets/%s/Sources/LiveActivityManager.swift
      — app lifecycle manager (start/update/end + push token updates stub)

  Patch:
    Project.swift                                  — add ActivityKit SDK dependency
    Project.swift                                  — set NSSupportsLiveActivities keys in app Info.plist dictionary
    Extensions/%sWidget/Project.swift              — add ActivityKit SDK dependency
    Extensions/%sWidget/Sources/%sWidgetBundle.swift — register %sLiveActivityWidget() in WidgetBundle body

  App: %s`, input.AppName, input.AppName, input.AppName, input.AppName, input.AppName, input.AppName, input.AppName, input.AppName, input.AppName)

	return plan, nil
}
