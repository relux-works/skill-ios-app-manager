package testtargets

import (
	"fmt"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const (
	unitTargetArgKey = "unit-target"
	uiTargetArgKey   = "ui-target"
)

const usageGuide = `## Usage

  Add one or both test targets with explicit target names:

    ios-app-manager test-targets setup --unit-target DemoAppTests --ui-target DemoAppUITests --yes

  Unit test targets are generated with Swift Testing starter code.
  UI test targets are generated with Swift XCUITest starter code and are ready
  for Page Object/accessibility-id scaffolding.`

func init() {
	registry.Register(&registry.Module{
		ID:          registry.TestTargets,
		Name:        "Test Targets",
		Category:    registry.Infra,
		Description: "Unit/UI test target scaffold plugin",
		Dependencies: []registry.ModuleID{
			registry.Init,
		},

		Plan:       planSetup,
		Setup:      runSetup,
		UsageGuide: usageGuide,

		CLIUse:     "test-targets",
		CLIShort:   "Manage test target scaffolding",
		SetupShort: "Create unit and/or UI test targets",
		ExtraFlags: []registry.ExtraFlag{
			{Name: "unit-target", Usage: "unit test target name to create/update", Required: false, ArgKey: unitTargetArgKey},
			{Name: "ui-target", Usage: "UI test target name to create/update", Required: false, ArgKey: uiTargetArgKey},
		},
	})
}

func runSetup(input registry.SetupInput) error {
	return Setup(SetupInput{
		ProjectRoot:    input.ProjectRoot,
		AppName:        input.AppName,
		UnitTargetName: input.ExtraArgs[unitTargetArgKey],
		UITargetName:   input.ExtraArgs[uiTargetArgKey],
	})
}

func planSetup(input registry.SetupInput) (string, error) {
	plan, err := NewPlan(SetupInput{
		ProjectRoot:    input.ProjectRoot,
		AppName:        input.AppName,
		UnitTargetName: input.ExtraArgs[unitTargetArgKey],
		UITargetName:   input.ExtraArgs[uiTargetArgKey],
	})
	if err != nil {
		return "", err
	}

	lines := []string{
		"## Test Targets Setup Plan",
		"",
	}
	if plan.UnitTargetName != "" {
		lines = append(lines,
			"  Unit test subplugin:",
			fmt.Sprintf("    - create/update target %s", plan.UnitTargetName),
			fmt.Sprintf("    - create Targets/%s/Sources/%s.swift", plan.UnitTargetName, plan.UnitTargetName),
			"    - product .unitTests, Swift Testing starter",
			"",
		)
	}
	if plan.UITargetName != "" {
		lines = append(lines,
			"  UI test subplugin:",
			fmt.Sprintf("    - create/update target %s", plan.UITargetName),
			fmt.Sprintf("    - create Targets/%s/Sources/%s.swift", plan.UITargetName, plan.UITestTypeName),
			"    - product .uiTests, XCUITest starter",
			"",
		)
	}
	lines = append(lines,
		"  Patch:",
		"    Project.swift — add requested test targets idempotently",
		"",
		fmt.Sprintf("  App: %s", input.AppName),
	)
	return strings.Join(lines, "\n"), nil
}
