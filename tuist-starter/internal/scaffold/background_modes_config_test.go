package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestBackgroundModesConfigGeneratorRegisteredWithInitDependency(t *testing.T) {
	t.Parallel()

	var plugin *GeneratorPlugin
	for _, candidate := range AllGenerators() {
		if candidate.Name == "background-modes-config" {
			plugin = candidate
			break
		}
	}
	if plugin == nil {
		t.Fatal("background-modes-config generator is not registered")
	}
	if len(plugin.Dependencies) != 1 || plugin.Dependencies[0] != "init" {
		t.Fatalf("background-modes-config dependencies = %#v, want [init]", plugin.Dependencies)
	}
}

func TestSyncBackgroundModesConfigUpdatesOnlyHostAppTarget(t *testing.T) {
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
                    "UIBackgroundModes": .array([
                        .string("voip"),
                    ]),
                    "NSMicrophoneUsageDescription": .string("Join audio calls."),
                    "UIUserInterfaceStyle": .string("Light"),
                    "ITSAppUsesNonExemptEncryption": .boolean(false),
                    "ApplicationConfiguration": .dictionary([
                        "appName": .string("DemoApp"),
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
                    "UIBackgroundModes": .array([
                        .string("voip"),
                    ]),
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
		BackgroundModes: []string{
			config.BackgroundModeAudio,
			config.BackgroundModeRemoteNotification,
			config.BackgroundModeVoIP,
		},
	}
	result, err := SyncBackgroundModesConfig(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncBackgroundModesConfig() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("SyncBackgroundModesConfig() updated %#v, want one Project.swift", result.Updated)
	}

	projectManifest := readFile(t, projectPath)
	for _, want := range []string{
		`"UIBackgroundModes": .array([`,
		`.string("audio"),`,
		`.string("remote-notification"),`,
		`.string("voip"),`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectManifest)
		}
	}
	if count := strings.Count(projectManifest, `"UIBackgroundModes":`); count != 1 {
		t.Fatalf("UIBackgroundModes count = %d, want host app only:\n%s", count, projectManifest)
	}
	if !strings.Contains(
		projectManifest,
		`"UIBackgroundModes": .array([
                        .string("audio"),
                        .string("remote-notification"),
                        .string("voip"),
                    ]),
                    "NSMicrophoneUsageDescription": .string("Join audio calls."),
                    "UIUserInterfaceStyle": .string("Light"),
                    "ITSAppUsesNonExemptEncryption": .boolean(false),
                    "ApplicationConfiguration": .dictionary([`,
	) {
		t.Fatalf("background modes config should stay before privacy, presentation, export compliance, and ApplicationConfiguration:\n%s", projectManifest)
	}

	secondResult, err := SyncBackgroundModesConfig(projectRoot, cfg)
	if err != nil {
		t.Fatalf("second SyncBackgroundModesConfig() error = %v", err)
	}
	if len(secondResult.Updated) != 0 {
		t.Fatalf("second SyncBackgroundModesConfig() updated %#v, want none", secondResult.Updated)
	}
}

func TestSyncBackgroundModesConfigAudioOnlyRemovesStaleVoIP(t *testing.T) {
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
                    "UIBackgroundModes": .array([
                        .string("audio"),
                        .string("remote-notification"),
                        .string("voip"),
                    ]),
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            dependencies: []
        )
    ]
)
`)

	result, err := SyncBackgroundModesConfig(projectRoot, config.ProjectConfig{
		BackgroundModes: []string{config.BackgroundModeAudio},
	})
	if err != nil {
		t.Fatalf("SyncBackgroundModesConfig() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("SyncBackgroundModesConfig() updated %#v, want one Project.swift", result.Updated)
	}

	projectManifest := readFile(t, projectPath)
	if !strings.Contains(projectManifest, `.string("audio"),`) {
		t.Fatalf("Project.swift missing audio background mode:\n%s", projectManifest)
	}
	if strings.Contains(projectManifest, `.string("voip"),`) {
		t.Fatalf("Project.swift kept voip without explicit config:\n%s", projectManifest)
	}
	if strings.Contains(projectManifest, `.string("remote-notification"),`) {
		t.Fatalf("Project.swift kept remote-notification without explicit config:\n%s", projectManifest)
	}
}

func TestSyncBackgroundModesConfigEmptyOrOmittedRemovesOwnedInfoPlistKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  config.ProjectConfig
	}{
		{name: "omitted", cfg: config.ProjectConfig{}},
		{name: "empty", cfg: config.ProjectConfig{BackgroundModes: []string{}}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
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
                    "UIBackgroundModes": .array([
                        .string("audio"),
                        .string("remote-notification"),
                        .string("voip"),
                    ]),
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            dependencies: []
        )
    ]
)
`)

			result, err := SyncBackgroundModesConfig(projectRoot, tc.cfg)
			if err != nil {
				t.Fatalf("SyncBackgroundModesConfig() error = %v", err)
			}
			if len(result.Updated) != 1 {
				t.Fatalf("SyncBackgroundModesConfig() updated %#v, want one Project.swift", result.Updated)
			}

			projectManifest := readFile(t, projectPath)
			if strings.Contains(projectManifest, "UIBackgroundModes") {
				t.Fatalf("Project.swift kept stale background modes key:\n%s", projectManifest)
			}
		})
	}
}
