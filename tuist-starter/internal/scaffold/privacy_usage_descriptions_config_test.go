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
                    "NSCameraUsageDescription": .string("stale"),
                    "NSMicrophoneUsageDescription": .string("stale"),
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
                    "NSCameraUsageDescription": .string("stale"),
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
			Camera:              "Join video calls.",
			Microphone:          "Join audio calls.",
			LocalNetwork:        "Connect calls on the local network when available.",
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
		`"NSCameraUsageDescription": .string("Join video calls."),`,
		`"NSMicrophoneUsageDescription": .string("Join audio calls."),`,
		`"NSLocalNetworkUsageDescription": .string("Connect calls on the local network when available."),`,
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
	if count := strings.Count(projectManifest, `"NSCameraUsageDescription":`); count != 1 {
		t.Fatalf("NSCameraUsageDescription count = %d, want host app only:\n%s", count, projectManifest)
	}
	if !strings.Contains(
		projectManifest,
		`"NSMicrophoneUsageDescription": .string("Join audio calls."),
                    "NSLocalNetworkUsageDescription": .string("Connect calls on the local network when available."),
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

func TestSyncPrivacyUsageDescriptionsConfigEmptyOrOmittedRemovesOwnedInfoPlistKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  config.ProjectConfig
	}{
		{name: "omitted", cfg: config.ProjectConfig{}},
		{
			name: "empty",
			cfg: config.ProjectConfig{
				PrivacyUsageDescriptions: config.PrivacyUsageDescriptionsConfig{
					BluetoothAlways:     " ",
					BluetoothPeripheral: "\t",
					Camera:              "",
					Microphone:          " \n ",
				},
			},
		},
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
                    "NSBluetoothAlwaysUsageDescription": .string("Find nearby devices."),
                    "NSBluetoothPeripheralUsageDescription": .string("Advertise nearby state."),
					"NSCameraUsageDescription": .string("Capture video."),
					"NSMicrophoneUsageDescription": .string("Capture audio."),
					"NSLocalNetworkUsageDescription": .string("Connect local peers."),
					"UILaunchScreen": .dictionary([:]),
                ]
            ),
            dependencies: []
        )
    ]
)
`)

			result, err := SyncPrivacyUsageDescriptionsConfig(projectRoot, tc.cfg)
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
				"NSCameraUsageDescription",
				"NSMicrophoneUsageDescription",
				"NSLocalNetworkUsageDescription",
			} {
				if strings.Contains(projectManifest, stale) {
					t.Fatalf("Project.swift kept stale %q:\n%s", stale, projectManifest)
				}
			}
		})
	}
}
