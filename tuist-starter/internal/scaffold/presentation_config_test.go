package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestPresentationConfigGeneratorRegisteredWithInitDependency(t *testing.T) {
	t.Parallel()

	var plugin *GeneratorPlugin
	for _, candidate := range AllGenerators() {
		if candidate.Name == "presentation-config" {
			plugin = candidate
			break
		}
	}
	if plugin == nil {
		t.Fatal("presentation-config generator is not registered")
	}
	if len(plugin.Dependencies) != 1 || plugin.Dependencies[0] != "init" {
		t.Fatalf("presentation-config dependencies = %#v, want [init]", plugin.Dependencies)
	}
}

func TestSyncPresentationConfigUpdatesOnlyHostAppTarget(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectPath := filepath.Join(projectRoot, "Project.swift")
	writeAppGroupsTestFile(t, projectPath, `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            product: .app,
            bundleId: "com.example.demo.app",
            infoPlist: .extendingDefault(
                with: [
                    "UIUserInterfaceStyle": .string("Dark"),
                    "UISupportedInterfaceOrientations": .array([
                        .string("UIInterfaceOrientationLandscapeLeft"),
                    ]),
                    "ITSAppUsesNonExemptEncryption": .boolean(false),
                    "ApplicationConfiguration": .dictionary([
                        "appName": .string("DemoApp"),
                    ]),
                    "AppGroups": .dictionary([
                        "shared": .string("group.com.example.demo.app.shared"),
                    ]),
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            dependencies: []
        ),
        .target(
            name: "DemoAppUITests",
            product: .uiTests,
            bundleId: "com.example.demo.app.uitests",
            infoPlist: .extendingDefault(
                with: [
                    "UIUserInterfaceStyle": .string("Dark"),
                ]
            ),
            dependencies: [
                .target(name: "DemoApp"),
            ]
        )
    ]
)
`)

	cfg := config.ProjectConfig{
		Theme:       config.ThemeLight,
		Orientation: config.OrientationPortrait,
	}
	result, err := SyncPresentationConfig(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncPresentationConfig() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("SyncPresentationConfig() updated %#v, want one Project.swift", result.Updated)
	}

	projectManifest := readFile(t, projectPath)
	for _, want := range []string{
		`"UIUserInterfaceStyle": .string("Light")`,
		`"UISupportedInterfaceOrientations": .array([`,
		`.string("UIInterfaceOrientationPortrait")`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectManifest)
		}
	}
	if count := strings.Count(projectManifest, `"UIUserInterfaceStyle":`); count != 1 {
		t.Fatalf("UIUserInterfaceStyle count = %d, want host app only:\n%s", count, projectManifest)
	}
	if count := strings.Count(projectManifest, `"UISupportedInterfaceOrientations":`); count != 1 {
		t.Fatalf("UISupportedInterfaceOrientations count = %d, want host app only:\n%s", count, projectManifest)
	}
	if strings.Contains(projectManifest, "UIInterfaceOrientationLandscapeLeft") {
		t.Fatalf("Project.swift kept stale landscape orientation:\n%s", projectManifest)
	}
	if !strings.Contains(
		projectManifest,
		`"UISupportedInterfaceOrientations": .array([
                        .string("UIInterfaceOrientationPortrait"),
                    ]),
                    "ITSAppUsesNonExemptEncryption": .boolean(false),
                    "ApplicationConfiguration": .dictionary([`,
	) {
		t.Fatalf("presentation config should stay before export compliance and ApplicationConfiguration:\n%s", projectManifest)
	}

	secondResult, err := SyncPresentationConfig(projectRoot, cfg)
	if err != nil {
		t.Fatalf("second SyncPresentationConfig() error = %v", err)
	}
	if len(secondResult.Updated) != 0 {
		t.Fatalf("second SyncPresentationConfig() updated %#v, want none", secondResult.Updated)
	}
}

func TestSyncPresentationConfigAutomaticRemovesOwnedInfoPlistKeys(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectPath := filepath.Join(projectRoot, "Project.swift")
	writeAppGroupsTestFile(t, projectPath, `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            product: .app,
            bundleId: "com.example.demo.app",
            infoPlist: .extendingDefault(
                with: [
                    "UIUserInterfaceStyle": .string("Light"),
                    "UISupportedInterfaceOrientations": .array([
                        .string("UIInterfaceOrientationPortrait"),
                    ]),
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            dependencies: []
        )
    ]
)
`)

	result, err := SyncPresentationConfig(projectRoot, config.ProjectConfig{
		Theme:       config.ThemeAutomatic,
		Orientation: config.OrientationAutomatic,
	})
	if err != nil {
		t.Fatalf("SyncPresentationConfig() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("SyncPresentationConfig() updated %#v, want one Project.swift", result.Updated)
	}

	projectManifest := readFile(t, projectPath)
	for _, stale := range []string{
		"UIUserInterfaceStyle",
		"UISupportedInterfaceOrientations",
		"UIInterfaceOrientationPortrait",
	} {
		if strings.Contains(projectManifest, stale) {
			t.Fatalf("Project.swift kept stale %q:\n%s", stale, projectManifest)
		}
	}
}
