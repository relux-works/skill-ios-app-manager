package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestSyncMinTargetCanonicalizesAppAndExtensionManifests(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.demo.app",
            deploymentTargets: .iOS("16.0"),
            settings: .settings(
                base: [
                    "IPHONEOS_DEPLOYMENT_TARGET": .string("16.0"),
                    "MARKETING_VERSION": .string(marketingVersion),
                ]
            )
        )
    ]
)
`)
	writeVersionManifest(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"), `import ProjectDescription

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

	result, err := SyncMinTarget(projectRoot, config.ProjectConfig{
		MinTarget: "17.0",
	})
	if err != nil {
		t.Fatalf("SyncMinTarget() error = %v", err)
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
			`let minTarget = "17.0"`,
			`deploymentTargets: .iOS(minTarget)`,
			`"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget)`,
		} {
			if !strings.Contains(content, want) {
				t.Fatalf("%s missing %q:\n%s", manifestPath, want, content)
			}
		}
	}
}

func TestSyncMinTargetReportsUpToDateManifest(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectSwiftPath := filepath.Join(projectRoot, "Project.swift")
	writeVersionManifest(t, projectSwiftPath, `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"
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

	result, err := SyncMinTarget(projectRoot, config.ProjectConfig{MinTarget: "17.0"})
	if err != nil {
		t.Fatalf("SyncMinTarget() error = %v", err)
	}
	if len(result.Updated) != 0 {
		t.Fatalf("updated len = %d, want 0", len(result.Updated))
	}
}

func TestSyncMinTargetRequiresMinTarget(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoApp"
)
`)

	_, err := SyncMinTarget(projectRoot, config.ProjectConfig{})
	if err == nil {
		t.Fatal("SyncMinTarget() error = nil, want min target requirement")
	}
	if !strings.Contains(err.Error(), "min target is required") {
		t.Fatalf("SyncMinTarget() error = %q, want min target requirement", err.Error())
	}
}

func TestSyncMinTargetDoesNotDropBelowPackageMinimum(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.demo.app",
            deploymentTargets: .iOS("16.0"),
            settings: .settings(
                base: [
                    "IPHONEOS_DEPLOYMENT_TARGET": .string("16.0"),
                ]
            )
        )
    ]
)
`)
	writeVersionManifest(t, filepath.Join(projectRoot, "Packages", "SharedKit", "Package.swift"), `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "SharedKit",
    platforms: [
        .iOS(.v18),
        .macOS(.v14)
    ],
    products: [
        .library(name: "SharedKit", type: .dynamic, targets: ["SharedKit"]),
    ],
    dependencies: [],
    targets: [
        .target(
            name: "SharedKit",
            dependencies: [],
            path: "Sources"
        ),
    ]
)
`)

	result, err := SyncMinTarget(projectRoot, config.ProjectConfig{
		MinTarget:   "17.0",
		ModulesPath: "Packages",
	})
	if err != nil {
		t.Fatalf("SyncMinTarget() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("updated len = %d, want 1", len(result.Updated))
	}

	content := readVersionManifest(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		`let minTarget = "18.0"`,
		`deploymentTargets: .iOS(minTarget)`,
		`"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget)`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, content)
		}
	}
}
