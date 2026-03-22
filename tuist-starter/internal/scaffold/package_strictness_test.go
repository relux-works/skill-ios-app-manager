package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestSyncPackageStrictnessCanonicalizesRootAndModulePackageManifests(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.2
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
	writeVersionManifest(t, filepath.Join(projectRoot, "Packages", "Auth", "Package.swift"), `// swift-tools-version: 6.0
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

	result, err := SyncPackageStrictness(projectRoot, config.ProjectConfig{
		SwiftVersion: "6.2",
		ModulesPath:  "Packages",
	})
	if err != nil {
		t.Fatalf("SyncPackageStrictness() error = %v", err)
	}

	if len(result.Scanned) != 2 {
		t.Fatalf("scanned len = %d, want 2", len(result.Scanned))
	}
	if len(result.Updated) != 2 {
		t.Fatalf("updated len = %d, want 2", len(result.Updated))
	}

	rootPackage := readVersionManifest(t, filepath.Join(projectRoot, "Package.swift"))
	for _, want := range []string{
		`let packageSettings = PackageSettings(`,
		`"SWIFT_VERSION": "6.0",`,
		`"SWIFT_STRICT_CONCURRENCY": "complete",`,
	} {
		if !strings.Contains(rootPackage, want) {
			t.Fatalf("root Package.swift missing %q:\n%s", want, rootPackage)
		}
	}

	modulePackage := readVersionManifest(t, filepath.Join(projectRoot, "Packages", "Auth", "Package.swift"))
	for _, want := range []string{
		`// swift-tools-version: 6.2`,
		`swiftSettings: [`,
		`.swiftLanguageMode(.v6),`,
		`.enableUpcomingFeature("InferSendableFromCaptures"),`,
	} {
		if !strings.Contains(modulePackage, want) {
			t.Fatalf("module Package.swift missing %q:\n%s", want, modulePackage)
		}
	}
}

func TestSyncPackageStrictnessIsIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.2
import PackageDescription

let modulesPath = "Packages"

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)
`)
	writeVersionManifest(t, filepath.Join(projectRoot, "Packages", "Auth", "Package.swift"), `// swift-tools-version: 6.0
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

	cfg := config.ProjectConfig{
		SwiftVersion: "6.2",
		ModulesPath:  "Packages",
	}

	if _, err := SyncPackageStrictness(projectRoot, cfg); err != nil {
		t.Fatalf("first SyncPackageStrictness() error = %v", err)
	}

	result, err := SyncPackageStrictness(projectRoot, cfg)
	if err != nil {
		t.Fatalf("second SyncPackageStrictness() error = %v", err)
	}
	if len(result.Updated) != 0 {
		t.Fatalf("updated len = %d, want 0", len(result.Updated))
	}
}
