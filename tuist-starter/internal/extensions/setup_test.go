package extensions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/ioc"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
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

func TestSetupCreatesSharedKitAndExtensionsRoot(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages", false)

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	requireDir(t, filepath.Join(projectRoot, "Extensions"))

	sharedKitDir := filepath.Join(projectRoot, "Packages", sharedKitModuleName)
	requireDir(t, sharedKitDir)
	requireFile(t, filepath.Join(sharedKitDir, "Package.swift"))
	requireFile(t, filepath.Join(sharedKitDir, "Sources", "SharedKit.swift"))
	requireFile(t, filepath.Join(sharedKitDir, ioc.ModuleTypeFile))

	moduleType := readFile(t, filepath.Join(sharedKitDir, ioc.ModuleTypeFile))
	if !strings.Contains(moduleType, "utility") {
		t.Fatalf(".module-type = %q, want to contain %q", moduleType, "utility")
	}

	sharedKitNamespace := readFile(t, filepath.Join(sharedKitDir, "Sources", "SharedKit.swift"))
	if !strings.Contains(sharedKitNamespace, "public enum SharedKit {}") {
		t.Fatalf("SharedKit.swift missing namespace:\n%s", sharedKitNamespace)
	}
}

func TestSetupPatchesProjectAndPackages(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages", true)

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: "Packages",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	if !strings.Contains(projectSwift, `.external(name: "SharedKit")`) {
		t.Fatalf("Project.swift missing SharedKit dependency:\n%s", projectSwift)
	}

	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(rootPackage, `.package(path: "Packages/SharedKit")`) {
		t.Fatalf("Package.swift missing SharedKit dependency:\n%s", rootPackage)
	}

	workspaceSwift := readFile(t, filepath.Join(projectRoot, "Workspace.swift"))
	if !strings.Contains(workspaceSwift, `.package(path: "Packages/SharedKit")`) {
		t.Fatalf("Workspace.swift missing SharedKit dependency:\n%s", workspaceSwift)
	}
}

func TestSetupWithCustomModulesPath(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Modules", false)

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: "Modules",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	requireDir(t, filepath.Join(projectRoot, "Modules", sharedKitModuleName))

	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(rootPackage, `.package(path: "Modules/SharedKit")`) {
		t.Fatalf("Package.swift missing custom SharedKit path:\n%s", rootPackage)
	}
}

func TestSetupIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages", false)

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

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	if got := strings.Count(projectSwift, `.external(name: "SharedKit")`); got != 1 {
		t.Fatalf(".external(name: \"SharedKit\") appears %d times, want 1:\n%s", got, projectSwift)
	}

	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if got := strings.Count(rootPackage, `.package(path: "Packages/SharedKit")`); got != 1 {
		t.Fatalf(".package(path: \"Packages/SharedKit\") appears %d times, want 1:\n%s", got, rootPackage)
	}
}

func TestMakeAppExtensionProjectCreatesScaffold(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages", false)
	writeConfigFile(t, projectRoot, config.ProjectConfig{
		AppName:          "DemoApp",
		BundleID:         "com.demo.app",
		TeamID:           "TEAM123456",
		MarketingVersion: "2.3.4",
		ProjectVersion:   "42",
		SwiftVersion:     "6.0",
		MinTarget:        "17.0",
	})

	err := makeAppExtensionProject(ExtensionProjectInput{
		ProjectRoot:              projectRoot,
		ExtensionName:            "WidgetExtension",
		BundleIDSuffix:           "widget",
		ExtensionPointIdentifier: "com.apple.widgetkit-extension",
		HostBundleID:             "com.demo.app",
	})
	if err != nil {
		t.Fatalf("makeAppExtensionProject() error = %v", err)
	}

	extensionRoot := filepath.Join(projectRoot, "Extensions", "WidgetExtension")
	requireDir(t, extensionRoot)
	requireDir(t, filepath.Join(extensionRoot, "Sources"))
	requireFile(t, filepath.Join(extensionRoot, "Project.swift"))
	requireFile(t, filepath.Join(extensionRoot, "Sources", "WidgetExtension.swift"))
	requireFile(t, filepath.Join(extensionRoot, "WidgetExtensionCore", "Package.swift"))
	requireFile(t, filepath.Join(extensionRoot, "WidgetExtensionCore", "Sources", "WidgetExtensionCore.swift"))
	requireFile(t, filepath.Join(extensionRoot, "WidgetExtensionCore", "Tests", "WidgetExtensionCoreTests", "WidgetExtensionCoreTests.swift"))

	projectSwift := readFile(t, filepath.Join(extensionRoot, "Project.swift"))
	for _, want := range []string{
		`name: "WidgetExtension"`,
		`let developmentTeam = "TEAM123456"`,
		`let marketingVersion = "2.3.4"`,
		`let currentProjectVersion = "42"`,
		`let minTarget = "17.0"`,
		`bundleId: "\(hostBundleId).widget"`,
		`deploymentTargets: .iOS(minTarget)`,
		`"CFBundleShortVersionString": .string(marketingVersion)`,
		`"CFBundleVersion": .string(currentProjectVersion)`,
		`"NSExtensionPointIdentifier": .string("com.apple.widgetkit-extension")`,
		`"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget)`,
		`"DEVELOPMENT_TEAM": .string(developmentTeam)`,
		`"SWIFT_STRICT_MEMORY_SAFETY": "YES"`,
		`"SWIFT_APPROACHABLE_CONCURRENCY": "NO"`,
		`"SWIFT_DEFAULT_ACTOR_ISOLATION": "nonisolated"`,
		`"SWIFT_STRICT_CONCURRENCY_DEFAULT": "complete"`,
		`"SWIFT_STRICT_CONCURRENCY": "complete"`,
		`"SWIFT_UPCOMING_FEATURE_CONCISE_MAGIC_FILE": "YES"`,
		`"SWIFT_UPCOMING_FEATURE_DISABLE_OUTWARD_ACTOR_ISOLATION": "YES"`,
		`"SWIFT_UPCOMING_FEATURE_GLOBAL_ACTOR_ISOLATED_TYPES_USABILITY": "YES"`,
		`"SWIFT_UPCOMING_FEATURE_INFER_ISOLATED_CONFORMANCES": "YES"`,
		`"SWIFT_UPCOMING_FEATURE_INFER_SENDABLE_FROM_CAPTURES": "YES"`,
		`"SWIFT_UPCOMING_FEATURE_GLOBAL_CONCURRENCY": "YES"`,
		`"SWIFT_UPCOMING_FEATURE_MEMBER_IMPORT_VISIBILITY": "YES"`,
		`"SWIFT_UPCOMING_FEATURE_NONFROZEN_ENUM_EXHAUSTIVITY": "YES"`,
		`"SWIFT_UPCOMING_FEATURE_REGION_BASED_ISOLATION": "YES"`,
		`"SWIFT_UPCOMING_FEATURE_EXISTENTIAL_ANY": "YES"`,
		`"SWIFT_UPCOMING_FEATURE_NONISOLATED_NONSENDING_BY_DEFAULT": "YES"`,
		`"MARKETING_VERSION": .string(marketingVersion)`,
		`"CURRENT_PROJECT_VERSION": .string(currentProjectVersion)`,
		`.external(name: "WidgetExtensionCore")`,
	} {
		if !strings.Contains(projectSwift, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectSwift)
		}
	}

	extensionSource := readFile(t, filepath.Join(extensionRoot, "Sources", "WidgetExtension.swift"))
	for _, want := range []string{
		`import WidgetExtensionCore`,
		`public enum WidgetExtensionEntryPoint`,
		`public static let core = WidgetExtensionCore.self`,
	} {
		if !strings.Contains(extensionSource, want) {
			t.Fatalf("extension source missing %q:\n%s", want, extensionSource)
		}
	}

	corePackageSwift := readFile(t, filepath.Join(extensionRoot, "WidgetExtensionCore", "Package.swift"))
	for _, want := range []string{
		`name: "WidgetExtensionCore"`,
		`.library(name: "WidgetExtensionCore", type: .dynamic, targets: ["WidgetExtensionCore"])`,
		`.target(`,
		`name: "WidgetExtensionCore"`,
		`.testTarget(`,
		`name: "WidgetExtensionCoreTests"`,
		`dependencies: ["WidgetExtensionCore"]`,
		`.swiftLanguageMode(.v6)`,
	} {
		if !strings.Contains(corePackageSwift, want) {
			t.Fatalf("Core Package.swift missing %q:\n%s", want, corePackageSwift)
		}
	}

	coreSource := readFile(t, filepath.Join(extensionRoot, "WidgetExtensionCore", "Sources", "WidgetExtensionCore.swift"))
	if !strings.Contains(coreSource, "public enum WidgetExtensionCore {}") {
		t.Fatalf("Core source missing namespace:\n%s", coreSource)
	}

	coreTests := readFile(t, filepath.Join(extensionRoot, "WidgetExtensionCore", "Tests", "WidgetExtensionCoreTests", "WidgetExtensionCoreTests.swift"))
	for _, want := range []string{
		"import Testing",
		"@testable import WidgetExtensionCore",
		"@Suite",
		"#expect(true)",
	} {
		if !strings.Contains(coreTests, want) {
			t.Fatalf("Core tests missing %q:\n%s", want, coreTests)
		}
	}

	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(rootPackage, `.package(path: "Extensions/WidgetExtension/WidgetExtensionCore")`) {
		t.Fatalf("root Package.swift missing Core package dependency:\n%s", rootPackage)
	}

	manifest, err := tuistproj.ParseManifest(projectSwift)
	if err != nil {
		t.Fatalf("ParseManifest(Project.swift) error = %v", err)
	}
	if len(manifest.Targets) != 1 {
		t.Fatalf("targets len = %d, want 1", len(manifest.Targets))
	}
	if manifest.Targets[0].Name != "WidgetExtension" {
		t.Fatalf("targets[0].Name = %q, want %q", manifest.Targets[0].Name, "WidgetExtension")
	}
}

func TestMakeAppExtensionProjectIsIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages", false)
	writeConfigFile(t, projectRoot, config.ProjectConfig{
		AppName:          "DemoApp",
		BundleID:         "com.demo.app",
		TeamID:           "TEAM123456",
		MarketingVersion: "2.3.4",
		ProjectVersion:   "42",
		SwiftVersion:     "6.0",
		MinTarget:        "17.0",
	})

	input := ExtensionProjectInput{
		ProjectRoot:              projectRoot,
		ExtensionName:            "WidgetExtension",
		BundleIDSuffix:           "widget",
		ExtensionPointIdentifier: "com.apple.widgetkit-extension",
		HostBundleID:             "com.demo.app",
	}

	if err := makeAppExtensionProject(input); err != nil {
		t.Fatalf("first makeAppExtensionProject() error = %v", err)
	}
	if err := makeAppExtensionProject(input); err != nil {
		t.Fatalf("second makeAppExtensionProject() error = %v", err)
	}

	extensionRoot := filepath.Join(projectRoot, "Extensions", "WidgetExtension")
	projectSwift := readFile(t, filepath.Join(extensionRoot, "Project.swift"))
	if got := strings.Count(projectSwift, `.external(name: "WidgetExtensionCore")`); got != 1 {
		t.Fatalf("Core dependency appears %d times, want 1:\n%s", got, projectSwift)
	}

	corePackageSwift := readFile(t, filepath.Join(extensionRoot, "WidgetExtensionCore", "Package.swift"))
	if got := strings.Count(corePackageSwift, `name: "WidgetExtensionCoreTests"`); got != 1 {
		t.Fatalf("Core test target appears %d times, want 1:\n%s", got, corePackageSwift)
	}

	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if got := strings.Count(rootPackage, `.package(path: "Extensions/WidgetExtension/WidgetExtensionCore")`); got != 1 {
		t.Fatalf("root Core package dependency appears %d times, want 1:\n%s", got, rootPackage)
	}
}

func writeConfigFile(t *testing.T, projectRoot string, cfg config.ProjectConfig) {
	t.Helper()

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal(config) error = %v", err)
	}

	writeTestFile(t, filepath.Join(projectRoot, config.DefaultConfigPath), string(raw))
}

func TestMakeAppExtensionProjectValidatesInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input ExtensionProjectInput
		want  string
	}{
		{
			name:  "missing project root",
			input: ExtensionProjectInput{ExtensionName: "WidgetExtension", BundleIDSuffix: "widget", ExtensionPointIdentifier: "com.apple.widgetkit-extension"},
			want:  "project root is required",
		},
		{
			name:  "missing extension name",
			input: ExtensionProjectInput{ProjectRoot: "/tmp", BundleIDSuffix: "widget", ExtensionPointIdentifier: "com.apple.widgetkit-extension"},
			want:  "extension name is required",
		},
		{
			name:  "missing bundle ID suffix",
			input: ExtensionProjectInput{ProjectRoot: "/tmp", ExtensionName: "WidgetExtension", ExtensionPointIdentifier: "com.apple.widgetkit-extension"},
			want:  "bundle ID suffix is required",
		},
		{
			name:  "missing extension point identifier",
			input: ExtensionProjectInput{ProjectRoot: "/tmp", ExtensionName: "WidgetExtension", BundleIDSuffix: "widget"},
			want:  "extension point identifier is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := makeAppExtensionProject(tc.input)
			if err == nil {
				t.Fatal("makeAppExtensionProject() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func setupProjectFiles(t *testing.T, projectRoot, modulesPath string, includeWorkspace bool) {
	t.Helper()

	mkdirs(t, filepath.Join(projectRoot, modulesPath))

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

	rootPackage := `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoAppDependencies",
    dependencies: [
    ],
    targets: []
)
`
	writeTestFile(t, filepath.Join(projectRoot, "Package.swift"), rootPackage)

	if includeWorkspace {
		workspace := `import ProjectDescription

let workspace = Workspace(
    name: "DemoApp",
    projects: [
        "."
    ],
    dependencies: [
    ]
)
`
		writeTestFile(t, filepath.Join(projectRoot, "Workspace.swift"), workspace)
	}
}

func mkdirs(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", path, err)
		}
	}
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
