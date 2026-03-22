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

	result, err := SyncBuildFlags(projectRoot, config.ProjectConfig{})
	if err != nil {
		t.Fatalf("SyncBuildFlags() error = %v", err)
	}
	if len(result.Updated) != 0 {
		t.Fatalf("updated len = %d, want 0", len(result.Updated))
	}
}
