package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestApplicationConfigurationGeneratorRegisteredWithInitDependency(t *testing.T) {
	t.Parallel()

	var plugin *GeneratorPlugin
	for _, candidate := range AllGenerators() {
		if candidate.Name == "application-configuration" {
			plugin = candidate
			break
		}
	}
	if plugin == nil {
		t.Fatal("application-configuration generator is not registered")
	}
	if len(plugin.Dependencies) != 1 || plugin.Dependencies[0] != "init" {
		t.Fatalf("application-configuration dependencies = %#v, want [init]", plugin.Dependencies)
	}
}

func TestSyncApplicationConfigurationUpdatesOwnedScaffoldSlices(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeAppGroupsRootPackage(t, projectRoot)
	writeAppGroupsTestFile(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.example.demo.app",
            infoPlist: .extendingDefault(
                with: [
                    "ApplicationConfiguration": .dictionary([
                        "appName": .string("Old"),
                    ]),
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            dependencies: []
        ),
        .target(
            name: "DemoAppUITests",
            bundleId: "com.example.demo.app.uitests",
            dependencies: [
                .target(name: "DemoApp"),
            ]
        )
    ]
)
`)
	writeAppGroupsTestFile(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoWidget",
    targets: [
        .target(
            name: "DemoWidget",
            product: .appExtension,
            bundleId: "com.example.demo.app.widget",
            dependencies: []
        )
    ]
)
`)

	cfg := config.ProjectConfig{
		AppName:   "DemoApp",
		BundleID:  "com.example.demo.app",
		TeamID:    "ABCDE12345",
		URLScheme: "demoapp",
	}
	result, err := SyncApplicationConfiguration(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncApplicationConfiguration() error = %v", err)
	}
	if len(result.Updated) < 7 {
		t.Fatalf("SyncApplicationConfiguration() updated %d file(s), want at least 7: %#v", len(result.Updated), result.Updated)
	}

	projectManifest := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		`"ApplicationConfiguration": .dictionary([`,
		`"appName": .string("DemoApp")`,
		`"applicationBundleIdentifier": .string("com.example.demo.app")`,
		`"developmentTeamID": .string("ABCDE12345")`,
		`"urlScheme": .string("demoapp")`,
		`.external(name: "SharedConfig")`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectManifest)
		}
	}
	if strings.Contains(projectManifest, `.string("Old")`) {
		t.Fatalf("Project.swift kept stale ApplicationConfiguration:\n%s", projectManifest)
	}
	if count := strings.Count(projectManifest, `"ApplicationConfiguration": .dictionary([`); count != 2 {
		t.Fatalf("Project.swift ApplicationConfiguration count = %d, want 2 app/test targets:\n%s", count, projectManifest)
	}

	extensionManifest := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"))
	for _, want := range []string{
		`bundleId: "com.example.demo.app.widget"`,
		`"applicationBundleIdentifier": .string("com.example.demo.app")`,
		`.external(name: "SharedConfig")`,
	} {
		if !strings.Contains(extensionManifest, want) {
			t.Fatalf("extension Project.swift missing %q:\n%s", want, extensionManifest)
		}
	}

	configuration := readFile(t, filepath.Join(projectRoot, "Targets", "DemoApp", "Sources", "Configuration", "Configuration+ApplicationConfiguration.swift"))
	for _, want := range []string{
		"import SharedConfig",
		"enum ApplicationConfiguration",
		"DemoAppApplicationConfiguration.read(from: .main)",
		"Could not read ApplicationConfiguration",
	} {
		if !strings.Contains(configuration, want) {
			t.Fatalf("Configuration+ApplicationConfiguration.swift missing %q:\n%s", want, configuration)
		}
	}

	sharedConfiguration := readFile(t, filepath.Join(projectRoot, "Packages", "SharedConfig", "Sources", "ApplicationConfiguration.swift"))
	for _, want := range []string{
		"public enum DemoAppApplicationConfigurationField",
		"public struct DemoAppApplicationConfiguration",
		"applicationBundleIdentifier",
		"developmentTeamID",
		"urlScheme",
	} {
		if !strings.Contains(sharedConfiguration, want) {
			t.Fatalf("ApplicationConfiguration.swift missing %q:\n%s", want, sharedConfiguration)
		}
	}
	for lineNumber, line := range strings.Split(sharedConfiguration, "\n") {
		if len(line) > 160 {
			t.Fatalf("ApplicationConfiguration.swift line %d is %d characters, want at most 160:\n%s", lineNumber+1, len(line), line)
		}
	}
	infoPlistReading := readFile(t, filepath.Join(projectRoot, "Packages", "SharedConfig", "Sources", "InfoPlistReading.swift"))
	if !strings.Contains(infoPlistReading, "missingInfoPlistDictionary") {
		t.Fatalf("InfoPlistReading.swift missing typed error:\n%s", infoPlistReading)
	}

	secondResult, err := SyncApplicationConfiguration(projectRoot, cfg)
	if err != nil {
		t.Fatalf("second SyncApplicationConfiguration() error = %v", err)
	}
	if len(secondResult.Updated) != 0 {
		t.Fatalf("second SyncApplicationConfiguration() updated %#v, want none", secondResult.Updated)
	}
}

func TestSyncApplicationConfigurationValidationReturnsActionableError(t *testing.T) {
	t.Parallel()

	_, err := SyncApplicationConfiguration(t.TempDir(), config.ProjectConfig{
		AppName:  "DemoApp",
		BundleID: "com.example.demo.app",
		SharedConfig: config.SharedConfigConfig{
			ModuleName: "Bad-SharedConfig",
		},
	})
	if err == nil {
		t.Fatal("SyncApplicationConfiguration() error = nil, want validation error")
	}
	msg := err.Error()
	for _, want := range []string{
		"invalid application configuration config",
		"team_id is required",
		`shared_config.module_name "Bad-SharedConfig" must be a valid Swift module identifier`,
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("SyncApplicationConfiguration() error = %q, want %q", msg, want)
		}
	}
}

func TestGenerateApplicationConfigurationSuppressesDerivedTypeNameLint(t *testing.T) {
	t.Parallel()

	got := GenerateApplicationConfigurationSharedConfigurationSwift(config.ProjectConfig{
		AppName: "RuntimeExample",
	})
	want := "// swiftlint:disable:next type_name\npublic enum RuntimeExampleApplicationConfigurationField"
	if !strings.Contains(got, want) {
		t.Fatalf("ApplicationConfiguration.swift missing scoped type_name suppression:\n%s", got)
	}
	if strings.Contains(got, "// swiftlint:disable:next type_name\npublic struct RuntimeExampleApplicationConfiguration") {
		t.Fatalf("ApplicationConfiguration.swift contains superfluous struct type_name suppression:\n%s", got)
	}

	longName := GenerateApplicationConfigurationSharedConfigurationSwift(config.ProjectConfig{
		AppName: "ExtraordinaryRuntimeExample",
	})
	want = "// swiftlint:disable:next type_name\npublic struct ExtraordinaryRuntimeExampleApplicationConfiguration"
	if !strings.Contains(longName, want) {
		t.Fatalf("long ApplicationConfiguration.swift missing scoped struct type_name suppression:\n%s", longName)
	}
}

func TestSyncApplicationConfigurationPrefersExistingConfigurationRoot(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeAppGroupsRootPackage(t, projectRoot)
	writeAppGroupsTestFile(t, filepath.Join(projectRoot, "Targets", "ProductModule", "Sources", "Configuration", "Configuration.swift"), `enum Configuration {}
`)
	writeAppGroupsTestFile(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.example.demo.app",
            dependencies: []
        )
    ]
)
`)

	cfg := config.ProjectConfig{
		AppName:  "DemoApp",
		BundleID: "com.example.demo.app",
		TeamID:   "ABCDE12345",
	}
	if _, err := SyncApplicationConfiguration(projectRoot, cfg); err != nil {
		t.Fatalf("SyncApplicationConfiguration() error = %v", err)
	}

	requireFile(t, filepath.Join(projectRoot, "Targets", "ProductModule", "Sources", "Configuration", "Configuration+ApplicationConfiguration.swift"))
	if fileExists(filepath.Join(projectRoot, "Targets", "DemoApp", "Sources", "Configuration", "Configuration+ApplicationConfiguration.swift")) {
		t.Fatal("generated Configuration+ApplicationConfiguration.swift under target-name path instead of existing configuration root")
	}
}

func TestSyncProjectManifestExternalDependencyTerminatesExistingLastItem(t *testing.T) {
	t.Parallel()

	manifest := `import ProjectDescription

let project = Project(
    name: "MatureApp",
    targets: [
        .target(
            name: "MatureAppUITests",
            dependencies: [
                .target(name: "MatureApp"),
                .external(name: "MatureFeature")
            ]
        )
    ]
)
`

	got, changed, err := syncProjectManifestExternalDependencyContent(manifest, "SharedConfig")
	if err != nil {
		t.Fatalf("syncProjectManifestExternalDependencyContent() error = %v", err)
	}
	if !changed {
		t.Fatal("syncProjectManifestExternalDependencyContent() changed = false, want true")
	}
	for _, want := range []string{
		`.external(name: "MatureFeature"),`,
		`.external(name: "SharedConfig"),`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Project.swift missing comma-safe dependency %q:\n%s", want, got)
		}
	}

	converged, changed, err := syncProjectManifestExternalDependencyContent(got, "SharedConfig")
	if err != nil {
		t.Fatalf("second syncProjectManifestExternalDependencyContent() error = %v", err)
	}
	if changed || converged != got {
		t.Fatalf("second dependency sync did not converge (changed=%v):\n%s", changed, converged)
	}
}
