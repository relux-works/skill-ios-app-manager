package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestExportComplianceConfigGeneratorRegisteredWithInitDependency(t *testing.T) {
	t.Parallel()

	var plugin *GeneratorPlugin
	for _, candidate := range AllGenerators() {
		if candidate.Name == "export-compliance-config" {
			plugin = candidate
			break
		}
	}
	if plugin == nil {
		t.Fatal("export-compliance-config generator is not registered")
	}
	if len(plugin.Dependencies) != 1 || plugin.Dependencies[0] != "init" {
		t.Fatalf("export-compliance-config dependencies = %#v, want [init]", plugin.Dependencies)
	}
}

func TestSyncExportComplianceConfigUpdatesOnlyHostAppTarget(t *testing.T) {
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
                    "ITSAppUsesNonExemptEncryption": .boolean(true),
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
                    "ITSAppUsesNonExemptEncryption": .boolean(true),
                ]
            ),
            dependencies: [
                .target(name: "DemoApp"),
            ]
        )
    ]
)
`)

	usesNonExemptEncryption := false
	result, err := SyncExportComplianceConfig(projectRoot, config.ProjectConfig{
		UsesNonExemptEncryption: &usesNonExemptEncryption,
	})
	if err != nil {
		t.Fatalf("SyncExportComplianceConfig() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("SyncExportComplianceConfig() updated %#v, want one Project.swift", result.Updated)
	}

	projectManifest := readFile(t, projectPath)
	if !strings.Contains(projectManifest, `"ITSAppUsesNonExemptEncryption": .boolean(false),`) {
		t.Fatalf("Project.swift missing export compliance false entry:\n%s", projectManifest)
	}
	if strings.Contains(projectManifest, `"ITSAppUsesNonExemptEncryption": .boolean(true),`) {
		t.Fatalf("Project.swift kept stale export compliance true entry:\n%s", projectManifest)
	}
	if count := strings.Count(projectManifest, `"ITSAppUsesNonExemptEncryption":`); count != 1 {
		t.Fatalf("ITSAppUsesNonExemptEncryption count = %d, want host app only:\n%s", count, projectManifest)
	}
	if !strings.Contains(
		projectManifest,
		`"ITSAppUsesNonExemptEncryption": .boolean(false),
                    "ApplicationConfiguration": .dictionary([`,
	) {
		t.Fatalf("export compliance config should stay before ApplicationConfiguration:\n%s", projectManifest)
	}

	secondResult, err := SyncExportComplianceConfig(projectRoot, config.ProjectConfig{
		UsesNonExemptEncryption: &usesNonExemptEncryption,
	})
	if err != nil {
		t.Fatalf("second SyncExportComplianceConfig() error = %v", err)
	}
	if len(secondResult.Updated) != 0 {
		t.Fatalf("second SyncExportComplianceConfig() updated %#v, want none", secondResult.Updated)
	}
}

func TestSyncExportComplianceConfigNilRemovesOwnedInfoPlistKey(t *testing.T) {
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
                    "ITSAppUsesNonExemptEncryption": .boolean(false),
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            dependencies: []
        )
    ]
)
`)

	result, err := SyncExportComplianceConfig(projectRoot, config.ProjectConfig{})
	if err != nil {
		t.Fatalf("SyncExportComplianceConfig() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("SyncExportComplianceConfig() updated %#v, want one Project.swift", result.Updated)
	}

	projectManifest := readFile(t, projectPath)
	if strings.Contains(projectManifest, "ITSAppUsesNonExemptEncryption") {
		t.Fatalf("Project.swift kept stale export compliance key:\n%s", projectManifest)
	}
}
