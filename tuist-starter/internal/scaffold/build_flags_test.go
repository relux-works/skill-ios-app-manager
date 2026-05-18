package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestSyncBuildFlagsCanonicalizesAppAndExtensionManifests(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

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
                    "IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),
                    "MARKETING_VERSION": .string("1.0.0"),
                ]
            )
        )
    ]
)
`)
	writeVersionManifest(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"), `import ProjectDescription

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
                    "MARKETING_VERSION": .string("1.0.0"),
                ]
            )
        )
    ]
)
`)

	result, err := SyncBuildFlags(projectRoot, config.ProjectConfig{})
	if err != nil {
		t.Fatalf("SyncBuildFlags() error = %v", err)
	}

	if len(result.Scanned) != 2 {
		t.Fatalf("scanned len = %d, want 2", len(result.Scanned))
	}
	if len(result.Updated) != 2 {
		t.Fatalf("updated len = %d, want 2", len(result.Updated))
	}

	for _, manifestPath := range result.Updated {
		content := readVersionManifest(t, manifestPath)
		for _, want := range []string{
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
		} {
			if !strings.Contains(content, want) {
				t.Fatalf("%s missing %q:\n%s", manifestPath, want, content)
			}
		}
	}
}

func TestSyncBuildFlagsReportsUpToDateManifest(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectSwiftPath := filepath.Join(projectRoot, "Project.swift")
	writeVersionManifest(t, projectSwiftPath, `import ProjectDescription

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
                    "SWIFT_STRICT_MEMORY_SAFETY": "YES",
                    "SWIFT_APPROACHABLE_CONCURRENCY": "NO",
                    "SWIFT_DEFAULT_ACTOR_ISOLATION": "nonisolated",
                    "SWIFT_STRICT_CONCURRENCY_DEFAULT": "complete",
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

	result, err := SyncBuildFlags(projectRoot, config.ProjectConfig{})
	if err != nil {
		t.Fatalf("SyncBuildFlags() error = %v", err)
	}
	if len(result.Updated) != 0 {
		t.Fatalf("updated len = %d, want 0", len(result.Updated))
	}
}

func TestSyncBuildFlagsCanonicalizesEveryBaseBlockInManifest(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectSwiftPath := filepath.Join(projectRoot, "Project.swift")
	writeVersionManifest(t, projectSwiftPath, `import ProjectDescription

let project = Project(
    name: "DemoApp",
    settings: .settings(
        base: [
            "LOCALIZATION_PREFERS_STRING_CATALOGS": .string("YES"),
        ]
    ),
    targets: [
        .target(
            name: "DemoApp",
            settings: .settings(
                base: [
                    "SWIFT_VERSION": "6.0",
                ]
            )
        ),
        .target(
            name: "DemoAppUITests",
            settings: .settings(
                base: [
                    "SWIFT_VERSION": .string("6.0"),
                ]
            )
        ),
        .target(
            name: "DemoAppTests",
            settings: .settings(
                base: [
                    "SWIFT_VERSION": .string("6.0"),
                ]
            )
        ),
    ]
)
`)

	result, err := SyncBuildFlags(projectRoot, config.ProjectConfig{})
	if err != nil {
		t.Fatalf("SyncBuildFlags() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("updated len = %d, want 1", len(result.Updated))
	}

	content := readVersionManifest(t, projectSwiftPath)
	if got := strings.Count(content, `"SWIFT_STRICT_CONCURRENCY": "complete"`); got != 4 {
		t.Fatalf("SWIFT_STRICT_CONCURRENCY count = %d, want 4:\n%s", got, content)
	}
	if got := strings.Count(content, `"SWIFT_STRICT_CONCURRENCY_DEFAULT": "complete"`); got != 4 {
		t.Fatalf("SWIFT_STRICT_CONCURRENCY_DEFAULT count = %d, want 4:\n%s", got, content)
	}
	if got := strings.Count(content, `"SWIFT_STRICT_MEMORY_SAFETY": "YES"`); got != 4 {
		t.Fatalf("SWIFT_STRICT_MEMORY_SAFETY count = %d, want 4:\n%s", got, content)
	}
	if got := strings.Count(content, `"SWIFT_APPROACHABLE_CONCURRENCY": "NO"`); got != 4 {
		t.Fatalf("SWIFT_APPROACHABLE_CONCURRENCY count = %d, want 4:\n%s", got, content)
	}
	if got := strings.Count(content, `"SWIFT_VERSION": "6.0"`); got != 4 {
		t.Fatalf("SWIFT_VERSION count = %d, want 4:\n%s", got, content)
	}
}

func TestSyncBuildFlagsCanonicalizesSharedSettingsDictionaryBlocks(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectSwiftPath := filepath.Join(projectRoot, "Project.swift")
	writeVersionManifest(t, projectSwiftPath, `import ProjectDescription

let appBuildSettings: SettingsDictionary = [
    "SWIFT_VERSION": "6.0",
]

let testBuildSettings: SettingsDictionary = [
    "SWIFT_VERSION": "6.0",
    "SWIFT_STRICT_CONCURRENCY": "minimal",
]

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            settings: .settings(base: appBuildSettings)
        ),
        .target(
            name: "DemoAppTests",
            settings: .settings(base: testBuildSettings)
        )
    ]
)
`)

	result, err := SyncBuildFlags(projectRoot, config.ProjectConfig{})
	if err != nil {
		t.Fatalf("SyncBuildFlags() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("updated len = %d, want 1", len(result.Updated))
	}

	content := readVersionManifest(t, projectSwiftPath)
	if got := strings.Count(content, `"SWIFT_APPROACHABLE_CONCURRENCY": "NO"`); got != 2 {
		t.Fatalf("SWIFT_APPROACHABLE_CONCURRENCY count = %d, want 2:\n%s", got, content)
	}
	if got := strings.Count(content, `"SWIFT_STRICT_CONCURRENCY": "complete"`); got != 2 {
		t.Fatalf("SWIFT_STRICT_CONCURRENCY count = %d, want 2:\n%s", got, content)
	}
	if got := strings.Count(content, `"SWIFT_STRICT_CONCURRENCY_DEFAULT": "complete"`); got != 2 {
		t.Fatalf("SWIFT_STRICT_CONCURRENCY_DEFAULT count = %d, want 2:\n%s", got, content)
	}
	if got := strings.Count(content, `"SWIFT_STRICT_MEMORY_SAFETY": "YES"`); got != 2 {
		t.Fatalf("SWIFT_STRICT_MEMORY_SAFETY count = %d, want 2:\n%s", got, content)
	}
}

func TestSyncBuildFlagsScansNestedAppProjectManifests(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            settings: .settings(
                base: [
                    "SWIFT_VERSION": "6.0",
                ]
            )
        )
    ]
)
`)
	nestedAppPath := filepath.Join(projectRoot, "Apps", "DemoConfigurator", "Project.swift")
	writeVersionManifest(t, nestedAppPath, `import ProjectDescription

let project = Project(
    name: "DemoConfigurator",
    targets: [
        .target(
            name: "DemoConfigurator",
            settings: .settings(
                base: [
                    "SWIFT_VERSION": .string("6.0"),
                ]
            )
        )
    ]
)
`)

	result, err := SyncBuildFlags(projectRoot, config.ProjectConfig{})
	if err != nil {
		t.Fatalf("SyncBuildFlags() error = %v", err)
	}
	if len(result.Scanned) != 2 {
		t.Fatalf("scanned len = %d, want 2", len(result.Scanned))
	}
	if len(result.Updated) != 2 {
		t.Fatalf("updated len = %d, want 2", len(result.Updated))
	}

	nestedContent := readVersionManifest(t, nestedAppPath)
	if !strings.Contains(nestedContent, `"SWIFT_STRICT_CONCURRENCY": "complete"`) {
		t.Fatalf("nested app manifest missing strict concurrency:\n%s", nestedContent)
	}
	if !strings.Contains(nestedContent, `"SWIFT_STRICT_CONCURRENCY_DEFAULT": "complete"`) {
		t.Fatalf("nested app manifest missing strict concurrency default:\n%s", nestedContent)
	}
	if !strings.Contains(nestedContent, `"SWIFT_STRICT_MEMORY_SAFETY": "YES"`) {
		t.Fatalf("nested app manifest missing strict memory safety:\n%s", nestedContent)
	}
}
