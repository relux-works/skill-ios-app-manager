package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestPlatformDestinationsGeneratorRegisteredWithInitDependency(t *testing.T) {
	t.Parallel()

	var plugin *GeneratorPlugin
	for _, candidate := range AllGenerators() {
		if candidate.Name == "platform-destinations" {
			plugin = candidate
			break
		}
	}
	if plugin == nil {
		t.Fatal("platform-destinations generator is not registered")
	}
	if len(plugin.Dependencies) != 1 || plugin.Dependencies[0] != "init" {
		t.Fatalf("platform-destinations dependencies = %#v, want [init]", plugin.Dependencies)
	}
}

func TestSyncPlatformDestinationsUpdatesOnlyHostAppTarget(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectPath := filepath.Join(projectRoot, "Project.swift")
	writeAppGroupsTestFile(t, projectPath, `import ProjectDescription
import ProjectDescriptionHelpers

let bundleID = "com.example.demo.app"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            destinations: .iOS,
            product: .app,
            bundleId: bundleID,
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: [],
            entitlements: EntitlementsFactory.make(hostBundleId: bundleID, destinations: .iOS, capabilities: AppCapabilities.app),
            dependencies: []
        ),
        .target(
            name: "DemoAppUITests",
            destinations: .iOS,
            product: .uiTests,
            bundleId: "com.example.demo.app.uitests",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: [],
            dependencies: [
                .target(name: "DemoApp"),
            ]
        )
    ]
)
`)

	iPadEnabled := false
	cfg := config.ProjectConfig{
		Platforms: &config.PlatformDestinationsConfig{
			IOS: config.PlatformDestinationConfig{
				Orientation: config.OrientationPortrait,
			},
			IPad: config.PlatformDestinationConfig{
				Enabled: &iPadEnabled,
			},
		},
	}

	result, err := SyncPlatformDestinations(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncPlatformDestinations() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("SyncPlatformDestinations() updated %#v, want one Project.swift", result.Updated)
	}

	projectManifest := readFile(t, projectPath)
	for _, want := range []string{
		`destinations: [.iPhone],`,
		`EntitlementsFactory.make(hostBundleId: bundleID, destinations: [.iPhone], capabilities: AppCapabilities.app)`,
		`name: "DemoAppUITests",
            destinations: .iOS,`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectManifest)
		}
	}

	secondResult, err := SyncPlatformDestinations(projectRoot, cfg)
	if err != nil {
		t.Fatalf("second SyncPlatformDestinations() error = %v", err)
	}
	if len(secondResult.Updated) != 0 {
		t.Fatalf("second SyncPlatformDestinations() updated %#v, want none", secondResult.Updated)
	}
}

func TestSyncPlatformDestinationsSupportsIPadAndMacDesignedDestination(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectPath := filepath.Join(projectRoot, "Project.swift")
	writeAppGroupsTestFile(t, projectPath, `import ProjectDescription
import ProjectDescriptionHelpers

let bundleID = "com.example.demo.app"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            destinations: [.iPhone],
            product: .app,
            bundleId: bundleID,
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: [],
            entitlements: EntitlementsFactory.make(hostBundleId: bundleID, destinations: [.iPhone], capabilities: AppCapabilities.app),
            dependencies: []
        )
    ]
)
`)

	iPadEnabled := true
	macWithIPadDesign := true
	cfg := config.ProjectConfig{
		Platforms: &config.PlatformDestinationsConfig{
			IOS: config.PlatformDestinationConfig{
				MacWithIPadDesign: &macWithIPadDesign,
			},
			IPad: config.PlatformDestinationConfig{
				Enabled: &iPadEnabled,
			},
		},
	}

	result, err := SyncPlatformDestinations(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncPlatformDestinations() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("SyncPlatformDestinations() updated %#v, want one Project.swift", result.Updated)
	}

	projectManifest := readFile(t, projectPath)
	for _, want := range []string{
		`destinations: [.iPhone, .iPad, .macWithiPadDesign],`,
		`EntitlementsFactory.make(hostBundleId: bundleID, destinations: [.iPhone, .iPad, .macWithiPadDesign], capabilities: AppCapabilities.app)`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectManifest)
		}
	}
}
