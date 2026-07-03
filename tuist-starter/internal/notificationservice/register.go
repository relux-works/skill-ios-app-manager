package notificationservice

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const (
	extensionTargetArgKey = "extension-target"
	bundleIDSuffixArgKey  = "bundle-id-suffix"
)

const usageGuide = `## Usage

  Run after app-extensions setup:

    ios-app-manager notification-service setup --yes

  Optional target/suffix override:

    ios-app-manager notification-service setup \
      --extension-target DemoNotificationServiceExtension \
      --bundle-id-suffix notification-service \
      --yes

  This creates a thin UNNotificationServiceExtension wrapper and keeps
  notification processing logic in <ExtensionName>Core for direct SwiftPM tests.`

func init() {
	registry.Register(&registry.Module{
		ID:          registry.NotificationService,
		Name:        "Notification Service",
		Category:    registry.Infra,
		Description: "Notification Service Extension with Core package",
		Dependencies: []registry.ModuleID{
			registry.AppExtensions,
		},

		Plan:       planSetup,
		Setup:      runSetup,
		UsageGuide: usageGuide,

		CLIUse:     "notification-service",
		CLIShort:   "Manage Notification Service Extension scaffolding",
		SetupShort: "Create Notification Service Extension target and Core package",
		ExtraFlags: []registry.ExtraFlag{
			{Name: "extension-target", Usage: "notification service extension target name", Required: false, ArgKey: extensionTargetArgKey},
			{Name: "bundle-id-suffix", Usage: "bundle ID suffix for the extension", Required: false, ArgKey: bundleIDSuffixArgKey},
		},
	})
}

func runSetup(input registry.SetupInput) error {
	return Setup(SetupInput{
		ProjectRoot:     input.ProjectRoot,
		AppName:         input.AppName,
		ExtensionTarget: input.ExtraArgs[extensionTargetArgKey],
		BundleIDSuffix:  input.ExtraArgs[bundleIDSuffixArgKey],
	})
}

func planSetup(input registry.SetupInput) (string, error) {
	plan, err := NewPlan(SetupInput{
		ProjectRoot:     input.ProjectRoot,
		AppName:         input.AppName,
		ExtensionTarget: input.ExtraArgs[extensionTargetArgKey],
		BundleIDSuffix:  input.ExtraArgs[bundleIDSuffixArgKey],
	})
	if err != nil {
		return "", err
	}

	extensionPath := filepath.ToSlash(filepath.Join("Extensions", plan.ExtensionTarget))
	corePath := filepath.ToSlash(filepath.Join(extensionPath, plan.CorePackageName))

	lines := []string{
		"## Notification Service Setup Plan",
		"",
		"  Create/update:",
		fmt.Sprintf("    %s/Project.swift", extensionPath),
		"      — Notification Service Extension target",
		fmt.Sprintf("    %s/Sources/%s.swift", extensionPath, plan.ExtensionTarget),
		"      — thin UNNotificationServiceExtension wrapper",
		fmt.Sprintf("    %s/Package.swift", corePath),
		"      — SwiftPM package for extension internals",
		fmt.Sprintf("    %s/Sources/%sNotificationServiceHandler.swift", corePath, plan.CorePackageName),
		"      — testable notification processing handler",
		"",
		"  Patch:",
		fmt.Sprintf("    Package.swift — add .package(path: %q)", corePath),
		fmt.Sprintf("    %s/Project.swift — add UserNotifications SDK dependency", extensionPath),
		"",
		fmt.Sprintf("  Bundle ID suffix: %s", plan.BundleIDSuffix),
		fmt.Sprintf("  App: %s", strings.TrimSpace(input.AppName)),
	}
	return strings.Join(lines, "\n"), nil
}
