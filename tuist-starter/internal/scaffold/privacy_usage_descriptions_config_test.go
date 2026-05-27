package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestPrivacyUsageDescriptionsConfigGeneratorRegisteredWithInitDependency(t *testing.T) {
	t.Parallel()

	var plugin *GeneratorPlugin
	for _, candidate := range AllGenerators() {
		if candidate.Name == "privacy-usage-descriptions-config" {
			plugin = candidate
			break
		}
	}
	if plugin == nil {
		t.Fatal("privacy-usage-descriptions-config generator is not registered")
	}
	if len(plugin.Dependencies) != 1 || plugin.Dependencies[0] != "init" {
		t.Fatalf("privacy-usage-descriptions-config dependencies = %#v, want [init]", plugin.Dependencies)
	}
}

func TestSyncPrivacyUsageDescriptionsConfigUpdatesOnlyHostAppTarget(t *testing.T) {
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
                    "NSBluetoothAlwaysUsageDescription": .string("stale"),
                    "NSBluetoothPeripheralUsageDescription": .string("stale"),
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
                    "NSBluetoothAlwaysUsageDescription": .string("stale"),
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
		PrivacyUsageDescriptions: config.PrivacyUsageDescriptionsConfig{
			BluetoothAlways:     "Find nearby transfer receivers.",
			BluetoothPeripheral: "Advertise nearby transfer availability.",
		},
	}
	result, err := SyncPrivacyUsageDescriptionsConfig(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncPrivacyUsageDescriptionsConfig() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("SyncPrivacyUsageDescriptionsConfig() updated %#v, want one Project.swift", result.Updated)
	}

	projectManifest := readFile(t, projectPath)
	for _, want := range []string{
		`"NSBluetoothAlwaysUsageDescription": .string("Find nearby transfer receivers."),`,
		`"NSBluetoothPeripheralUsageDescription": .string("Advertise nearby transfer availability."),`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectManifest)
		}
	}
	if strings.Contains(projectManifest, `.string("stale")`) {
		t.Fatalf("Project.swift kept stale privacy usage description:\n%s", projectManifest)
	}
	if count := strings.Count(projectManifest, `"NSBluetoothAlwaysUsageDescription":`); count != 1 {
		t.Fatalf("NSBluetoothAlwaysUsageDescription count = %d, want host app only:\n%s", count, projectManifest)
	}
	if !strings.Contains(
		projectManifest,
		`"NSBluetoothPeripheralUsageDescription": .string("Advertise nearby transfer availability."),
                    "UIUserInterfaceStyle": .string("Light"),
                    "ITSAppUsesNonExemptEncryption": .boolean(false),
                    "ApplicationConfiguration": .dictionary([`,
	) {
		t.Fatalf("privacy usage descriptions should stay before presentation, export compliance, and ApplicationConfiguration:\n%s", projectManifest)
	}

	secondResult, err := SyncPrivacyUsageDescriptionsConfig(projectRoot, cfg)
	if err != nil {
		t.Fatalf("second SyncPrivacyUsageDescriptionsConfig() error = %v", err)
	}
	if len(secondResult.Updated) != 0 {
		t.Fatalf("second SyncPrivacyUsageDescriptionsConfig() updated %#v, want none", secondResult.Updated)
	}
}

func TestSyncPrivacyUsageDescriptionsConfigEmptyRemovesOwnedInfoPlistKeys(t *testing.T) {
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
                    "NSBluetoothAlwaysUsageDescription": .string("Find nearby devices."),
                    "NSBluetoothPeripheralUsageDescription": .string("Advertise nearby state."),
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            dependencies: []
        )
    ]
)
`)

	result, err := SyncPrivacyUsageDescriptionsConfig(projectRoot, config.ProjectConfig{})
	if err != nil {
		t.Fatalf("SyncPrivacyUsageDescriptionsConfig() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("SyncPrivacyUsageDescriptionsConfig() updated %#v, want one Project.swift", result.Updated)
	}

	projectManifest := readFile(t, projectPath)
	for _, stale := range []string{
		"NSBluetoothAlwaysUsageDescription",
		"NSBluetoothPeripheralUsageDescription",
	} {
		if strings.Contains(projectManifest, stale) {
			t.Fatalf("Project.swift kept stale %q:\n%s", stale, projectManifest)
		}
	}
}
