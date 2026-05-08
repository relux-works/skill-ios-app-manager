package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestAppGroupsCapabilityPluginRegisteredWithInitDependency(t *testing.T) {
	t.Parallel()

	var plugin *AppCapabilityPlugin
	for _, candidate := range AllAppCapabilityPlugins() {
		if candidate.Name == "app-groups" {
			plugin = candidate
			break
		}
	}
	if plugin == nil {
		t.Fatal("app-groups capability subplugin is not registered")
	}
	if len(plugin.Dependencies) != 1 || plugin.Dependencies[0] != "init" {
		t.Fatalf("app-groups dependencies = %#v, want [init]", plugin.Dependencies)
	}
}

func TestSyncAppGroupsUpdatesOwnedScaffoldSlices(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	helpersDir := filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "AppCapabilities.swift"), []byte(GenerateAppCapabilities()), 0o644); err != nil {
		t.Fatal(err)
	}
	writeAppGroupsRootPackage(t, projectRoot)
	writeAppGroupsTestFile(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            infoPlist: .extendingDefault(
                with: [
                    "GROUP_COM_EXAMPLE-DEMO_OLD": .string("group.com.example-demo.old"),
                    "APP_GROUP_OLD": .string("group.com.example-demo.old"),
                    "UILaunchScreen": .dictionary([:]),
                ]
            )
            dependencies: []
        ),
        .target(
            name: "DemoAppUITests",
            product: .uiTests,
            bundleId: "com.example-demo.app.uitests",
            dependencies: [
                .target(name: "DemoApp"),
            ]
        )
    ]
)
`)

	cfg := config.ProjectConfig{
		AppName:  "DemoApp",
		BundleID: "com.example-demo.app",
		AppGroups: []string{
			"group.com.example-demo.app.shared",
			"group.com.example-demo.app.sso",
		},
	}
	result, err := SyncAppGroups(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncAppGroups() error = %v", err)
	}
	if !result.Enabled {
		t.Fatal("SyncAppGroups() Enabled = false, want true")
	}
	if len(result.Updated) < 6 {
		t.Fatalf("SyncAppGroups() updated %d file(s), want at least 6: %#v", len(result.Updated), result.Updated)
	}

	appCapabilities := readFile(t, filepath.Join(helpersDir, "AppCapabilities.swift"))
	for _, want := range []string{
		`.appGroups(group: .custom(id: "group.com.example-demo.app.shared"))`,
		`.appGroups(group: .custom(id: "group.com.example-demo.app.sso"))`,
	} {
		if !strings.Contains(appCapabilities, want) {
			t.Fatalf("AppCapabilities.swift missing %q:\n%s", want, appCapabilities)
		}
	}

	configuration := readFile(t, filepath.Join(projectRoot, "Targets", "DemoApp", "Sources", "Configuration", "Configuration+AppGroups.swift"))
	if !strings.Contains(configuration, "static let shared: String") {
		t.Fatalf("Configuration+AppGroups.swift missing Swift property:\n%s", configuration)
	}
	if !strings.Contains(configuration, "DemoAppAppGroups.read(from: .main)") {
		t.Fatalf("Configuration+AppGroups.swift missing shared configuration read:\n%s", configuration)
	}
	if strings.Contains(configuration, "GROUP_COM_EXAMPLE_DEMO_APP_SHARED") {
		t.Fatalf("Configuration+AppGroups.swift kept app-group value shaped property:\n%s", configuration)
	}
	if strings.Contains(configuration, "EXAMPLE-DEMO") {
		t.Fatalf("Configuration+AppGroups.swift kept hyphenated identifier:\n%s", configuration)
	}

	projectManifest := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		`"AppGroups": .dictionary([`,
		`"shared": .string("group.com.example-demo.app.shared")`,
		`"sso": .string("group.com.example-demo.app.sso")`,
		`infoPlist: .extendingDefault(`,
		`.external(name: "SharedConfig")`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectManifest)
		}
	}
	for _, want := range []string{
		`"shared": .string("group.com.example-demo.app.shared")`,
		`"sso": .string("group.com.example-demo.app.sso")`,
	} {
		if count := strings.Count(projectManifest, want); count != 2 {
			t.Fatalf("Project.swift %q count = %d, want 2 app/test targets:\n%s", want, count, projectManifest)
		}
	}
	if count := strings.Count(projectManifest, `.external(name: "SharedConfig")`); count != 2 {
		t.Fatalf("Project.swift shared configuration dependency count = %d, want 2:\n%s", count, projectManifest)
	}
	for _, stale := range []string{
		"GROUP_COM_EXAMPLE-DEMO_OLD",
		"APP_GROUP_OLD",
		"APP_GROUP_SHARED",
		"APP_GROUP_SSO",
		"GROUP_COM_EXAMPLE_DEMO_APP_SHARED",
	} {
		if strings.Contains(projectManifest, stale) {
			t.Fatalf("Project.swift kept stale app group key %q:\n%s", stale, projectManifest)
		}
	}

	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	for _, want := range []string{
		`.package(path: "Packages/SharedConfig")`,
		`"SharedConfig": .framework`,
	} {
		if !strings.Contains(rootPackage, want) {
			t.Fatalf("Package.swift missing %q:\n%s", want, rootPackage)
		}
	}

	sharedConfiguration := readFile(t, filepath.Join(projectRoot, "Packages", "SharedConfig", "Sources", "SharedConfig.swift"))
	for _, want := range []string{
		`case appGroups = "AppGroups"`,
		`case shared = "shared"`,
		`case sso = "sso"`,
		"public struct DemoAppAppGroups",
		"public static func read(from bundle: Bundle = .main) throws -> Self",
	} {
		if !strings.Contains(sharedConfiguration, want) {
			t.Fatalf("SharedConfig.swift missing %q:\n%s", want, sharedConfiguration)
		}
	}
	for _, forbidden := range []string{"CaseIterable", "static let all", "var all"} {
		if strings.Contains(sharedConfiguration, forbidden) {
			t.Fatalf("SharedConfig.swift contains forbidden all-style API %q:\n%s", forbidden, sharedConfiguration)
		}
	}
}

func TestSyncAppGroupsPrefersExistingRuntimeConfigurationPath(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	helpersDir := filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "AppCapabilities.swift"), []byte(GenerateAppCapabilities()), 0o644); err != nil {
		t.Fatal(err)
	}
	writeAppGroupsRootPackage(t, projectRoot)
	writeAppGroupsTestFile(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            infoPlist: .extendingDefault(
                with: [
                    "GROUP_COM_EXAMPLE_DEMO_APP_SHARED": .string("group.com.example.demo.app.shared"),
                    "UILaunchScreen": .dictionary([:]),
                ]
            )
            dependencies: []
        )
    ]
)
`)

	runtimePath := filepath.Join(projectRoot, "Targets", "DemoApp", "Sources", "Configuration", "Runtime", "Configuration+AppGroups.swift")
	defaultPath := filepath.Join(projectRoot, "Targets", "DemoApp", "Sources", "Configuration", "Configuration+AppGroups.swift")
	writeAppGroupsTestFile(t, runtimePath, `import Foundation

extension Configuration {
    enum AppGroups {
        static let shared: String = {
            Bundle.main.readInfoPlistValue(by: "GROUP_COM_EXAMPLE_DEMO_APP_SHARED")
        }()
    }
}
`)
	writeAppGroupsTestFile(t, defaultPath, `extension Configuration {
    enum AppGroups {}
}
`)

	cfg := config.ProjectConfig{
		AppName:  "DemoApp",
		BundleID: "com.example.demo.app",
		AppGroups: []string{
			"group.com.example.demo.app.shared",
		},
	}
	if _, err := SyncAppGroups(projectRoot, cfg); err != nil {
		t.Fatalf("SyncAppGroups() error = %v", err)
	}

	configuration := readFile(t, runtimePath)
	for _, want := range []string{
		"import Foundation",
		"import SharedConfig",
		"static let shared: String",
		"resolved.shared",
	} {
		if !strings.Contains(configuration, want) {
			t.Fatalf("runtime Configuration+AppGroups.swift missing %q:\n%s", want, configuration)
		}
	}
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("default Configuration+AppGroups.swift should be removed, stat error = %v", err)
	}
}

func TestSyncAppGroupsMigratesLegacySharedConfigurationPackage(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	helpersDir := filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "AppCapabilities.swift"), []byte(GenerateAppCapabilities()), 0o644); err != nil {
		t.Fatal(err)
	}
	writeAppGroupsTestFile(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [
        .package(path: "Packages/DemoAppSharedConfiguration"),
    ],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    productTypes: [
        "DemoAppSharedConfiguration": .framework,
    ]
)
#endif
`)
	writeAppGroupsTestFile(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            dependencies: [
                .external(name: "DemoAppSharedConfiguration"),
            ]
        )
    ]
)
`)
	legacyPackagePath := filepath.Join(projectRoot, "Packages", "DemoAppSharedConfiguration")
	writeAppGroupsTestFile(t, filepath.Join(legacyPackagePath, "Package.swift"), "// legacy\n")

	cfg := config.ProjectConfig{
		AppName:   "DemoApp",
		BundleID:  "com.example.demo.app",
		AppGroups: []string{"group.com.example.demo.app.shared"},
	}
	if _, err := SyncAppGroups(projectRoot, cfg); err != nil {
		t.Fatalf("SyncAppGroups() error = %v", err)
	}

	if _, err := os.Stat(legacyPackagePath); !os.IsNotExist(err) {
		t.Fatalf("legacy shared configuration package should be removed, stat error = %v", err)
	}

	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	for _, forbidden := range []string{
		`Packages/DemoAppSharedConfiguration`,
		`"DemoAppSharedConfiguration": .framework`,
	} {
		if strings.Contains(rootPackage, forbidden) {
			t.Fatalf("Package.swift kept legacy shared config %q:\n%s", forbidden, rootPackage)
		}
	}
	for _, want := range []string{
		`.package(path: "Packages/SharedConfig")`,
		`"SharedConfig": .framework`,
	} {
		if !strings.Contains(rootPackage, want) {
			t.Fatalf("Package.swift missing %q:\n%s", want, rootPackage)
		}
	}

	projectManifest := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	if strings.Contains(projectManifest, `.external(name: "DemoAppSharedConfiguration")`) {
		t.Fatalf("Project.swift kept legacy shared config dependency:\n%s", projectManifest)
	}
	if !strings.Contains(projectManifest, `.external(name: "SharedConfig")`) {
		t.Fatalf("Project.swift missing SharedConfig dependency:\n%s", projectManifest)
	}
}

func TestSyncAppGroupsUsesConfiguredSharedConfigModuleName(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	helpersDir := filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "AppCapabilities.swift"), []byte(GenerateAppCapabilities()), 0o644); err != nil {
		t.Fatal(err)
	}
	writeAppGroupsRootPackage(t, projectRoot)
	writeAppGroupsTestFile(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            dependencies: []
        )
    ]
)
`)

	cfg := config.ProjectConfig{
		AppName:     "DemoApp",
		BundleID:    "com.example.demo.app",
		ModulesPath: "VendorPackages",
		SharedConfig: config.SharedConfigConfig{
			ModuleName: "AppSharedConfig",
		},
		AppGroups: []string{"group.com.example.demo.app.shared"},
	}
	if _, err := SyncAppGroups(projectRoot, cfg); err != nil {
		t.Fatalf("SyncAppGroups() error = %v", err)
	}

	configuration := readFile(t, filepath.Join(projectRoot, "Targets", "DemoApp", "Sources", "Configuration", "Configuration+AppGroups.swift"))
	if !strings.Contains(configuration, "import AppSharedConfig") {
		t.Fatalf("Configuration+AppGroups.swift missing configured import:\n%s", configuration)
	}

	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	for _, want := range []string{
		`.package(path: "VendorPackages/AppSharedConfig")`,
		`"AppSharedConfig": .framework`,
	} {
		if !strings.Contains(rootPackage, want) {
			t.Fatalf("Package.swift missing configured shared config %q:\n%s", want, rootPackage)
		}
	}

	projectManifest := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	if !strings.Contains(projectManifest, `.external(name: "AppSharedConfig")`) {
		t.Fatalf("Project.swift missing configured shared config dependency:\n%s", projectManifest)
	}

	requireFile(t, filepath.Join(projectRoot, "VendorPackages", "AppSharedConfig", "Package.swift"))
	requireFile(t, filepath.Join(projectRoot, "VendorPackages", "AppSharedConfig", "Sources", "AppSharedConfig.swift"))
}

func TestSyncAppGroupsValidationReturnsActionableError(t *testing.T) {
	t.Parallel()

	_, err := SyncAppGroups(t.TempDir(), config.ProjectConfig{
		AppName:  "DemoApp",
		BundleID: "com.example.app",
		SharedConfig: config.SharedConfigConfig{
			ModuleName: "Bad-SharedConfig",
		},
		AppGroups: []string{
			"com.example.app.shared",
			"group.com.example.app.shared",
			"group.com.example.app.shared",
			"group.com.example.app.shared!",
			"group.com.example.app.shared@",
		},
	})
	if err == nil {
		t.Fatal("SyncAppGroups() error = nil, want validation error")
	}

	msg := err.Error()
	for _, want := range []string{
		"invalid app_groups config",
		`shared_config.module_name "Bad-SharedConfig" must be a valid Swift module identifier`,
		`app_groups[0] "com.example.app.shared" must start with "group."`,
		`app_groups[2] "group.com.example.app.shared" duplicates app_groups[1]`,
		`both map to AppGroups key "shared"`,
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("SyncAppGroups() error = %q, want %q", msg, want)
		}
	}
}

func writeAppGroupsTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeAppGroupsRootPackage(t *testing.T, projectRoot string) {
	t.Helper()

	writeAppGroupsTestFile(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    productTypes: [:]
)
#endif
`)
}
