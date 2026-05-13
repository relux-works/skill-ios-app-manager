package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

func TestGenerateHelpShowsMakefileSubcommand(t *testing.T) {
	t.Parallel()

	output, err := executeRootCommand("generate", "--help")
	if err != nil {
		t.Fatalf("executeRootCommand(generate --help) error = %v", err)
	}

	for _, expected := range []string{"makefile", "swiftlint", "Generate project artifacts"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("generate --help output missing %q:\n%s", expected, output)
		}
	}

	for _, expected := range []string{"versions", "min-target", "application-configuration", "app-capabilities", "build-flags", "project-config"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("generate --help output missing %q:\n%s", expected, output)
		}
	}
}

func TestGenerateAppCapabilitiesAddsConfiguredAppGroups(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)

	writeGenerateVersionManifest(
		t,
		filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers", "AppCapabilities.swift"),
		scaffold.GenerateAppCapabilities(),
	)
	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            infoPlist: .extendingDefault(
                with: [
                    "UILaunchScreen": .dictionary([:]),
                ]
            )
            dependencies: []
        )
    ]
)
`)
	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)
`)

	output, err := executeRootCommand("generate", "--config", configPath, "app-capabilities")
	if err != nil {
		t.Fatalf("executeRootCommand(generate app-capabilities) error = %v", err)
	}
	if !strings.Contains(output, "regenerated app capabilities via 1 enabled subplugin(s), updated 7 file(s)") {
		t.Fatalf("generate app-capabilities output = %q, want regenerate message", output)
	}

	content, err := os.ReadFile(filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers", "AppCapabilities.swift"))
	if err != nil {
		t.Fatalf("ReadFile(AppCapabilities.swift) error = %v", err)
	}
	if !strings.Contains(string(content), `.appGroups(group: .custom(id: "group.com.example.demo"))`) {
		t.Fatalf("AppCapabilities.swift missing app group:\n%s", string(content))
	}
}

func TestGenerateVersionsUpdatesHostAndExtensionManifests(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)

	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            infoPlist: .extendingDefault(with: [
                "CFBundleShortVersionString": .string(marketingVersion),
                "CFBundleVersion": .string(currentProjectVersion),
            ]),
            settings: .settings(
                base: [
                    "MARKETING_VERSION": .string(marketingVersion),
                    "CURRENT_PROJECT_VERSION": .string(currentProjectVersion),
                ]
            )
        )
    ]
)
`)
	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "DemoWidget",
    targets: [
        .target(
            name: "DemoWidget",
            infoPlist: .extendingDefault(with: [
                "CFBundleShortVersionString": .string(marketingVersion),
                "CFBundleVersion": .string(currentProjectVersion),
            ]),
            settings: .settings(
                base: [
                    "MARKETING_VERSION": .string(marketingVersion),
                    "CURRENT_PROJECT_VERSION": .string(currentProjectVersion),
                ]
            )
        )
    ]
)
`)

	updatedCfg := cfg
	updatedCfg.MarketingVersion = "2.0.0"
	updatedCfg.ProjectVersion = "55"
	if err := config.WriteProjectConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", configPath, err)
	}

	output, err := executeRootCommand("generate", "--config", configPath, "versions")
	if err != nil {
		t.Fatalf("executeRootCommand(generate versions) error = %v", err)
	}
	if !strings.Contains(output, "regenerated version manifests in 2 file(s)") {
		t.Fatalf("generate versions output = %q, want regenerate message", output)
	}

	for _, manifestPath := range []string{
		filepath.Join(projectRoot, "Project.swift"),
		filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"),
	} {
		content, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", manifestPath, err)
		}
		for _, want := range []string{
			`let marketingVersion = "2.0.0"`,
			`let currentProjectVersion = "55"`,
		} {
			if !strings.Contains(string(content), want) {
				t.Fatalf("%s missing %q:\n%s", manifestPath, want, string(content))
			}
		}
	}
}

func TestGenerateVersionsReportsUpToDateManifests(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)

	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"
`)

	output, err := executeRootCommand("generate", "--config", configPath, "versions")
	if err != nil {
		t.Fatalf("executeRootCommand(generate versions) error = %v", err)
	}
	if !strings.Contains(output, "version manifests already up to date in 1 file(s)") {
		t.Fatalf("generate versions output = %q, want up-to-date message", output)
	}
}

func TestGenerateMinTargetUpdatesHostAndExtensionManifests(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)

	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.demo.app",
            deploymentTargets: .iOS("16.0"),
            infoPlist: .extendingDefault(
                with: [
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            settings: .settings(
                base: [
                    "IPHONEOS_DEPLOYMENT_TARGET": .string("16.0"),
                ]
            )
        )
    ]
)
`)
	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "DemoWidget",
    targets: [
        .target(
            name: "DemoWidget",
            product: .appExtension,
            bundleId: "com.demo.app.widget",
            settings: .settings(
                base: [
                    "MARKETING_VERSION": .string(marketingVersion),
                ]
            )
        )
    ]
)
`)

	updatedCfg := cfg
	updatedCfg.MinTarget = "18.0"
	if err := config.WriteProjectConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", configPath, err)
	}

	output, err := executeRootCommand("generate", "--config", configPath, "min-target")
	if err != nil {
		t.Fatalf("executeRootCommand(generate min-target) error = %v", err)
	}
	if !strings.Contains(output, "regenerated min target manifests in 2 file(s)") {
		t.Fatalf("generate min-target output = %q, want regenerate message", output)
	}

	for _, manifestPath := range []string{
		filepath.Join(projectRoot, "Project.swift"),
		filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"),
	} {
		content, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", manifestPath, err)
		}
		for _, want := range []string{
			`let minTarget = "18.0"`,
			`deploymentTargets: .iOS(minTarget)`,
			`"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget)`,
		} {
			if !strings.Contains(string(content), want) {
				t.Fatalf("%s missing %q:\n%s", manifestPath, want, string(content))
			}
		}
	}
}

func TestGenerateMinTargetReportsUpToDateManifests(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)

	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let minTarget = "17.0"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.demo.app",
            deploymentTargets: .iOS(minTarget),
            settings: .settings(
                base: [
                    "IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),
                ]
            )
        )
    ]
)
`)

	output, err := executeRootCommand("generate", "--config", configPath, "min-target")
	if err != nil {
		t.Fatalf("executeRootCommand(generate min-target) error = %v", err)
	}
	if !strings.Contains(output, "min target manifests already up to date in 1 file(s)") {
		t.Fatalf("generate min-target output = %q, want up-to-date message", output)
	}
}

func TestGenerateBuildFlagsUpdatesHostAndExtensionManifests(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)

	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let minTarget = "17.0"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.demo.app",
            deploymentTargets: .iOS(minTarget),
            settings: .settings(
                base: [
                    "SWIFT_VERSION": .string("6.0"),
                    "IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),
                ]
            )
        )
    ]
)
`)
	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"), `import ProjectDescription

let minTarget = "17.0"

let project = Project(
    name: "DemoWidget",
    targets: [
        .target(
            name: "DemoWidget",
            product: .appExtension,
            bundleId: "com.demo.app.widget",
            deploymentTargets: .iOS(minTarget),
            settings: .settings(
                base: [
                    "IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),
                    "SWIFT_APPROACHABLE_CONCURRENCY": "YES",
                ]
            )
        )
    ]
)
`)

	output, err := executeRootCommand("generate", "--config", configPath, "build-flags")
	if err != nil {
		t.Fatalf("executeRootCommand(generate build-flags) error = %v", err)
	}
	if !strings.Contains(output, "regenerated build flag manifests in 2 file(s)") {
		t.Fatalf("generate build-flags output = %q, want regenerate message", output)
	}

	for _, manifestPath := range []string{
		filepath.Join(projectRoot, "Project.swift"),
		filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"),
	} {
		content, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", manifestPath, err)
		}
		for _, want := range []string{
			`"SWIFT_APPROACHABLE_CONCURRENCY": "NO"`,
			`"SWIFT_DEFAULT_ACTOR_ISOLATION": "nonisolated"`,
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
		} {
			if !strings.Contains(string(content), want) {
				t.Fatalf("%s missing %q:\n%s", manifestPath, want, string(content))
			}
		}
	}
}

func TestGenerateBuildFlagsReportsUpToDateManifests(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)

	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let minTarget = "17.0"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.demo.app",
            deploymentTargets: .iOS(minTarget),
            settings: .settings(
                base: [
                    "SWIFT_VERSION": "6.0",
                    "SWIFT_APPROACHABLE_CONCURRENCY": "NO",
                    "SWIFT_DEFAULT_ACTOR_ISOLATION": "nonisolated",
                    "SWIFT_STRICT_CONCURRENCY": "complete",
                    "SWIFT_UPCOMING_FEATURE_CONCISE_MAGIC_FILE": "YES",
                    "SWIFT_UPCOMING_FEATURE_DISABLE_OUTWARD_ACTOR_ISOLATION": "YES",
                    "SWIFT_UPCOMING_FEATURE_GLOBAL_ACTOR_ISOLATED_TYPES_USABILITY": "YES",
                    "SWIFT_UPCOMING_FEATURE_INFER_ISOLATED_CONFORMANCES": "YES",
                    "SWIFT_UPCOMING_FEATURE_INFER_SENDABLE_FROM_CAPTURES": "YES",
                    "SWIFT_UPCOMING_FEATURE_GLOBAL_CONCURRENCY": "YES",
                    "SWIFT_UPCOMING_FEATURE_MEMBER_IMPORT_VISIBILITY": "YES",
                    "SWIFT_UPCOMING_FEATURE_NONFROZEN_ENUM_EXHAUSTIVITY": "YES",
                    "SWIFT_UPCOMING_FEATURE_REGION_BASED_ISOLATION": "YES",
                    "SWIFT_UPCOMING_FEATURE_EXISTENTIAL_ANY": "YES",
                    "SWIFT_UPCOMING_FEATURE_NONISOLATED_NONSENDING_BY_DEFAULT": "YES",
                    "IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),
                ]
            )
        )
    ]
)
`)

	output, err := executeRootCommand("generate", "--config", configPath, "build-flags")
	if err != nil {
		t.Fatalf("executeRootCommand(generate build-flags) error = %v", err)
	}
	if !strings.Contains(output, "build flag manifests already up to date in 1 file(s)") {
		t.Fatalf("generate build-flags output = %q, want up-to-date message", output)
	}
}

func TestGenerateBuildFlagsIsIdempotent(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)
	projectSwiftPath := filepath.Join(projectRoot, "Project.swift")

	writeGenerateVersionManifest(t, projectSwiftPath, `import ProjectDescription

let minTarget = "17.0"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.demo.app",
            deploymentTargets: .iOS(minTarget),
            settings: .settings(
                base: [
                    "SWIFT_VERSION": .string("6.0"),
                    "IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),
                ]
            )
        )
    ]
)
`)

	firstOutput, err := executeRootCommand("generate", "--config", configPath, "build-flags")
	if err != nil {
		t.Fatalf("first executeRootCommand(generate build-flags) error = %v", err)
	}
	if !strings.Contains(firstOutput, "regenerated build flag manifests in 1 file(s)") {
		t.Fatalf("first generate build-flags output = %q, want regenerate message", firstOutput)
	}

	firstContent, err := os.ReadFile(projectSwiftPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) after first run error = %v", projectSwiftPath, err)
	}

	secondOutput, err := executeRootCommand("generate", "--config", configPath, "build-flags")
	if err != nil {
		t.Fatalf("second executeRootCommand(generate build-flags) error = %v", err)
	}
	if !strings.Contains(secondOutput, "build flag manifests already up to date in 1 file(s)") {
		t.Fatalf("second generate build-flags output = %q, want up-to-date message", secondOutput)
	}

	secondContent, err := os.ReadFile(projectSwiftPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) after second run error = %v", projectSwiftPath, err)
	}
	if string(firstContent) != string(secondContent) {
		t.Fatalf("build-flags second run changed manifest:\nfirst:\n%s\n\nsecond:\n%s", string(firstContent), string(secondContent))
	}
}

func TestGenerateProjectConfigRunsLeafPlugins(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)

	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.demo.app",
            deploymentTargets: .iOS("16.0"),
            infoPlist: .extendingDefault(
                with: [
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            settings: .settings(
                base: [
                    "IPHONEOS_DEPLOYMENT_TARGET": .string("16.0"),
                    "MARKETING_VERSION": .string(marketingVersion),
                    "CURRENT_PROJECT_VERSION": .string(currentProjectVersion),
                ]
            )
        )
    ]
)
`)
	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "DemoWidget",
    targets: [
        .target(
            name: "DemoWidget",
            product: .appExtension,
            bundleId: "com.demo.app.widget",
            infoPlist: .extendingDefault(with: [
                "CFBundleShortVersionString": .string(marketingVersion),
                "CFBundleVersion": .string(currentProjectVersion),
            ]),
            settings: .settings(
                base: [
                    "MARKETING_VERSION": .string(marketingVersion),
                    "CURRENT_PROJECT_VERSION": .string(currentProjectVersion),
                ]
            )
        )
    ]
)
`)
	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.2
import PackageDescription

let modulesPath = "Packages"

let package = Package(
    name: "DemoDependencies",
    dependencies: [
        .package(path: "./Packages/Auth"),
    ],
    targets: []
)
`)
	writeGenerateVersionManifest(t, filepath.Join(projectRoot, "Packages", "Auth", "Package.swift"), `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "Auth",
    platforms: [
        .iOS(.v17),
    ],
    products: [
        .library(name: "Auth", type: .dynamic, targets: ["Auth"]),
    ],
    dependencies: [
    ],
    targets: [
        .target(
            name: "Auth",
            dependencies: [
            ],
            path: "Sources"
        ),
    ]
)
`)
	writeGenerateVersionManifest(
		t,
		filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers", "AppCapabilities.swift"),
		scaffold.GenerateAppCapabilities(),
	)

	updatedCfg := cfg
	updatedCfg.MarketingVersion = "2.0.0"
	updatedCfg.ProjectVersion = "55"
	updatedCfg.MinTarget = "18.0"
	if err := config.WriteProjectConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", configPath, err)
	}

	output, err := executeRootCommand("generate", "--config", configPath, "project-config")
	if err != nil {
		t.Fatalf("executeRootCommand(generate project-config) error = %v", err)
	}

	for _, want := range []string{
		"project config sync summary:",
		"- versions: regenerated version manifests in 2 file(s)",
		"- min-target: regenerated min target manifests in 3 file(s)",
		"- application-configuration: regenerated application configuration in 7 file(s)",
		"- app-capabilities: regenerated app capabilities via 1 enabled subplugin(s), updated 5 file(s)",
		"- build-flags: regenerated build flag manifests in 2 file(s)",
		"- package-strictness: regenerated package strictness manifests in 1 file(s)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("generate project-config output missing %q:\n%s", want, output)
		}
	}

	for _, manifestPath := range []string{
		filepath.Join(projectRoot, "Project.swift"),
		filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"),
	} {
		content, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", manifestPath, err)
		}
		for _, want := range []string{
			`let marketingVersion = "2.0.0"`,
			`let currentProjectVersion = "55"`,
			`let minTarget = "18.0"`,
			`deploymentTargets: .iOS(minTarget)`,
			`"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget)`,
			`"SWIFT_APPROACHABLE_CONCURRENCY": "NO"`,
			`"SWIFT_DEFAULT_ACTOR_ISOLATION": "nonisolated"`,
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
		} {
			if !strings.Contains(string(content), want) {
				t.Fatalf("%s missing %q:\n%s", manifestPath, want, string(content))
			}
		}
	}

	packageManifest, err := os.ReadFile(filepath.Join(projectRoot, "Packages", "Auth", "Package.swift"))
	if err != nil {
		t.Fatalf("ReadFile(Packages/Auth/Package.swift) error = %v", err)
	}
	if !strings.Contains(string(packageManifest), `.iOS(.v18)`) {
		t.Fatalf("Package.swift missing synced iOS minimum:\n%s", string(packageManifest))
	}

	appCapabilities, err := os.ReadFile(filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers", "AppCapabilities.swift"))
	if err != nil {
		t.Fatalf("ReadFile(AppCapabilities.swift) error = %v", err)
	}
	if !strings.Contains(string(appCapabilities), `.appGroups(group: .custom(id: "group.com.example.demo"))`) {
		t.Fatalf("AppCapabilities.swift missing app group:\n%s", string(appCapabilities))
	}
}

func writeGenerateVersionManifest(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestGenerateMakefileCreatesFileWhenMissing(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	makefilePath := filepath.Join(filepath.Dir(configPath), "Makefile")

	output, err := executeRootCommand("generate", "--config", configPath, "makefile")
	if err != nil {
		t.Fatalf("executeRootCommand(generate makefile) error = %v", err)
	}
	if !strings.Contains(output, "created") {
		t.Fatalf("generate makefile output = %q, want creation message", output)
	}

	content, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", makefilePath, err)
	}
	expected := scaffold.GenerateMakefile(cfg)
	if string(content) != expected {
		t.Fatalf("generated Makefile mismatch:\nwant:\n%s\n\ngot:\n%s", expected, string(content))
	}
}

func TestGenerateMakefileRegeneratesAndPreservesCustomSection(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	makefilePath := filepath.Join(filepath.Dir(configPath), "Makefile")

	initialCfg := cfg
	initialCfg.ModulesPath = "OldPackages"
	customTarget := "custom-target: ## Custom workflow\n\t@echo \"custom\"\n"
	initialMakefile := scaffold.GenerateMakefile(initialCfg) + "\n" + customTarget
	if err := os.WriteFile(makefilePath, []byte(initialMakefile), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", makefilePath, err)
	}

	updatedCfg := cfg
	updatedCfg.ModulesPath = "VendorPackages"
	if err := config.WriteProjectConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", configPath, err)
	}

	output, err := executeRootCommand("generate", "--config", configPath, "makefile")
	if err != nil {
		t.Fatalf("executeRootCommand(generate makefile) error = %v", err)
	}
	for _, expected := range []string{"regenerated", "preserved custom section"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("generate makefile output missing %q:\n%s", expected, output)
		}
	}

	regenerated, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", makefilePath, err)
	}
	makefile := string(regenerated)
	if !strings.Contains(makefile, "MODULES_PATH := VendorPackages") {
		t.Fatalf("regenerated Makefile missing updated modules path:\n%s", makefile)
	}
	if !strings.Contains(makefile, customTarget) {
		t.Fatalf("regenerated Makefile missing preserved custom target:\n%s", makefile)
	}
}

func TestGenerateSwiftLintCreatesFileWhenMissing(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	swiftlintPath := filepath.Join(filepath.Dir(configPath), ".swiftlint.yml")

	output, err := executeRootCommand("generate", "--config", configPath, "swiftlint")
	if err != nil {
		t.Fatalf("executeRootCommand(generate swiftlint) error = %v", err)
	}
	if !strings.Contains(output, "created") {
		t.Fatalf("generate swiftlint output = %q, want creation message", output)
	}

	content, err := os.ReadFile(swiftlintPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", swiftlintPath, err)
	}

	expected := scaffold.GenerateSwiftLintConfig(cfg)
	if string(content) != expected {
		t.Fatalf("generated .swiftlint.yml mismatch:\nwant:\n%s\n\ngot:\n%s", expected, string(content))
	}
}

func TestGenerateSwiftLintRegeneratesWhenModulesPathChanges(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	cfg.ModulesPath = "OldPackages"
	configPath := writeTestConfig(t, cfg)
	swiftlintPath := filepath.Join(filepath.Dir(configPath), ".swiftlint.yml")

	initialConfig := scaffold.GenerateSwiftLintConfig(cfg)
	if err := os.WriteFile(swiftlintPath, []byte(initialConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", swiftlintPath, err)
	}

	updatedCfg := cfg
	updatedCfg.ModulesPath = "VendorPackages"
	if err := config.WriteProjectConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", configPath, err)
	}

	output, err := executeRootCommand("generate", "--config", configPath, "swiftlint")
	if err != nil {
		t.Fatalf("executeRootCommand(generate swiftlint) error = %v", err)
	}
	if !strings.Contains(output, "regenerated") {
		t.Fatalf("generate swiftlint output = %q, want regenerate message", output)
	}

	regenerated, err := os.ReadFile(swiftlintPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", swiftlintPath, err)
	}

	content := string(regenerated)
	if !strings.Contains(content, "  - VendorPackages/") {
		t.Fatalf("regenerated .swiftlint.yml missing updated modules path:\n%s", content)
	}
	if strings.Contains(content, "  - OldPackages/") {
		t.Fatalf("regenerated .swiftlint.yml should not contain old modules path:\n%s", content)
	}
}
