package widgetbase

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

func TestSetupValidatesInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input SetupInput
		want  string
	}{
		{
			name:  "empty project root",
			input: SetupInput{AppName: "DemoApp"},
			want:  "project root is required",
		},
		{
			name:  "empty app name",
			input: SetupInput{ProjectRoot: "/tmp"},
			want:  "app name is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := Setup(tc.input)
			if err == nil {
				t.Fatal("Setup() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Setup() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestSetupCreatesWidgetExtensionAndPatchesHost(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, true)
	writeConfigFile(t, projectRoot, "DemoApp", "com.demo.app", []string{"group.com.demo.shared"})

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	extensionTargetName := widgetExtensionTargetName("DemoApp")
	extensionRoot := filepath.Join(projectRoot, "Extensions", extensionTargetName)
	requireDir(t, extensionRoot)
	requireFile(t, filepath.Join(extensionRoot, "Project.swift"))
	requireFile(t, filepath.Join(extensionRoot, "Sources", extensionTargetName+".swift"))
	requireFile(t, filepath.Join(extensionRoot, "Sources", "DemoAppWidgetBundle.swift"))
	requireFile(t, filepath.Join(extensionRoot, extensionTargetName+"Core", "Package.swift"))
	requireFile(t, filepath.Join(extensionRoot, extensionTargetName+"Core", "Sources", extensionTargetName+"Core.swift"))
	requireFile(t, filepath.Join(extensionRoot, extensionTargetName+"Core", "Tests", extensionTargetName+"CoreTests", extensionTargetName+"CoreTests.swift"))

	extensionProject := readFile(t, filepath.Join(extensionRoot, "Project.swift"))
	for _, want := range []string{
		`let developmentTeam = "TEAM123456"`,
		`bundleId: "\(hostBundleId).widget"`,
		`let marketingVersion = "1.0.0"`,
		`"CFBundleShortVersionString": .string(marketingVersion)`,
		`"DEVELOPMENT_TEAM": .string(developmentTeam)`,
		`"MARKETING_VERSION": .string(marketingVersion)`,
		`"NSExtensionPointIdentifier": .string("com.apple.widgetkit-extension")`,
		`.external(name: "DemoAppWidgetCore")`,
		`.sdk(name: "WidgetKit", type: .framework)`,
		`"com.apple.security.application-groups": .array([`,
		`.string("group.com.demo.shared")`,
	} {
		if !strings.Contains(extensionProject, want) {
			t.Fatalf("extension Project.swift missing %q:\n%s", want, extensionProject)
		}
	}

	widgetBundle := readFile(t, filepath.Join(extensionRoot, "Sources", "DemoAppWidgetBundle.swift"))
	for _, want := range []string{
		"import WidgetKit",
		"import SwiftUI",
		"@main",
		"struct DemoAppWidgetBundle: WidgetBundle",
	} {
		if !strings.Contains(widgetBundle, want) {
			t.Fatalf("WidgetBundle file missing %q:\n%s", want, widgetBundle)
		}
	}

	extensionSource := readFile(t, filepath.Join(extensionRoot, "Sources", extensionTargetName+".swift"))
	for _, want := range []string{
		"import DemoAppWidgetCore",
		"public static let core = DemoAppWidgetCore.self",
	} {
		if !strings.Contains(extensionSource, want) {
			t.Fatalf("extension entry point missing %q:\n%s", want, extensionSource)
		}
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	if !strings.Contains(projectSwift, `.project(target: "DemoAppWidget", path: "Extensions/DemoAppWidget")`) {
		t.Fatalf("Project.swift missing widget extension dependency:\n%s", projectSwift)
	}

	rootPackageSwift := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(rootPackageSwift, `.package(path: "Extensions/DemoAppWidget/DemoAppWidgetCore")`) {
		t.Fatalf("Package.swift missing widget Core package dependency:\n%s", rootPackageSwift)
	}

	workspaceSwift := readFile(t, filepath.Join(projectRoot, "Workspace.swift"))
	if !strings.Contains(workspaceSwift, `"Extensions/DemoAppWidget"`) {
		t.Fatalf("Workspace.swift missing widget extension project:\n%s", workspaceSwift)
	}

	appCapabilities := readFile(t, filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers", "AppCapabilities.swift"))
	if !strings.Contains(appCapabilities, `.appGroups(group: .custom(id: "group.com.demo.shared"))`) {
		t.Fatalf("AppCapabilities.swift missing app group capability:\n%s", appCapabilities)
	}
}

func TestSetupDerivesAppGroupFromBundleIDWhenNotConfigured(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, true)
	writeConfigFile(t, projectRoot, "DemoApp", "com.demo.app", nil)

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	extensionProject := readFile(
		t,
		filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Project.swift"),
	)
	if !strings.Contains(extensionProject, `.string("group.com.demo.app")`) {
		t.Fatalf("Project.swift missing derived app group:\n%s", extensionProject)
	}

	appCapabilities := readFile(t, filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers", "AppCapabilities.swift"))
	if !strings.Contains(appCapabilities, `.appGroups(group: .custom(id: "group.com.demo.app"))`) {
		t.Fatalf("AppCapabilities.swift missing derived app group:\n%s", appCapabilities)
	}
}

func TestSetupIsIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, true)
	writeConfigFile(t, projectRoot, "DemoApp", "com.demo.app", []string{"group.com.demo.shared"})

	input := SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}

	if err := Setup(input); err != nil {
		t.Fatalf("first Setup() error = %v", err)
	}
	if err := Setup(input); err != nil {
		t.Fatalf("second Setup() error = %v", err)
	}

	rootProject := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	if got := strings.Count(rootProject, `.project(target: "DemoAppWidget", path: "Extensions/DemoAppWidget")`); got != 1 {
		t.Fatalf("widget extension host dependency appears %d times, want 1:\n%s", got, rootProject)
	}

	workspaceSwift := readFile(t, filepath.Join(projectRoot, "Workspace.swift"))
	if got := strings.Count(workspaceSwift, `"Extensions/DemoAppWidget"`); got != 1 {
		t.Fatalf("workspace extension ref appears %d times, want 1:\n%s", got, workspaceSwift)
	}

	extensionProject := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Project.swift"))
	if got := strings.Count(extensionProject, `.sdk(name: "WidgetKit", type: .framework)`); got != 1 {
		t.Fatalf("WidgetKit dependency appears %d times, want 1:\n%s", got, extensionProject)
	}
	if got := strings.Count(extensionProject, `.external(name: "DemoAppWidgetCore")`); got != 1 {
		t.Fatalf("Core dependency appears %d times, want 1:\n%s", got, extensionProject)
	}
	if got := strings.Count(extensionProject, appGroupsEntitlementKey); got != 1 {
		t.Fatalf("app groups entitlement appears %d times, want 1:\n%s", got, extensionProject)
	}

	rootPackageSwift := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if got := strings.Count(rootPackageSwift, `.package(path: "Extensions/DemoAppWidget/DemoAppWidgetCore")`); got != 1 {
		t.Fatalf("Core package dependency appears %d times, want 1:\n%s", got, rootPackageSwift)
	}

	corePackageSwift := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "DemoAppWidgetCore", "Package.swift"))
	if got := strings.Count(corePackageSwift, `name: "DemoAppWidgetCoreTests"`); got != 1 {
		t.Fatalf("Core test target appears %d times, want 1:\n%s", got, corePackageSwift)
	}

	appCapabilities := readFile(t, filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers", "AppCapabilities.swift"))
	if got := strings.Count(appCapabilities, `.appGroups(group: .custom(id: "group.com.demo.shared"))`); got != 1 {
		t.Fatalf("appGroups capability appears %d times, want 1:\n%s", got, appCapabilities)
	}
}

func setupProjectFiles(t *testing.T, projectRoot string, includeWorkspace bool) {
	t.Helper()

	projectSwift := `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            destinations: .iOS,
            product: .app,
            bundleId: "com.demo.app",
            dependencies: [
            ]
        )
    ]
)
`
	writeTestFile(t, filepath.Join(projectRoot, "Project.swift"), projectSwift)

	rootPackageSwift := `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoAppDependencies",
    dependencies: [
    ],
    targets: []
)
`
	writeTestFile(t, filepath.Join(projectRoot, "Package.swift"), rootPackageSwift)

	if includeWorkspace {
		workspace := `import ProjectDescription

let workspace = Workspace(
    name: "DemoApp",
    projects: [
        "."
    ]
)
`
		writeTestFile(t, filepath.Join(projectRoot, "Workspace.swift"), workspace)
	}

	appCapabilitiesPath := filepath.Join(
		projectRoot,
		"Tuist",
		"ProjectDescriptionHelpers",
		"AppCapabilities.swift",
	)
	writeTestFile(t, appCapabilitiesPath, scaffold.GenerateAppCapabilities())
}

func writeConfigFile(t *testing.T, projectRoot, appName, bundleID string, appGroups []string) {
	t.Helper()

	cfg := config.ProjectConfig{
		AppName:          appName,
		BundleID:         bundleID,
		TeamID:           "TEAM123456",
		SwiftVersion:     "6.0",
		MinTarget:        "17.0",
		MarketingVersion: "1.0.0",
		ProjectVersion:   "1",
		AppGroups:        appGroups,
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal(config) error = %v", err)
	}

	writeTestFile(t, filepath.Join(projectRoot, config.DefaultConfigPath), string(raw))
}

func requireDir(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("path %q is not a directory", path)
	}
}

func requireFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("path %q is a directory, want file", path)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
