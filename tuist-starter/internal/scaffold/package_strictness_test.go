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

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    productTypes: [
        "SwiftIoC": .framework,
        "SwiftUIRelux": .framework,
    ],
    baseSettings: .settings(
        base: [
            "OTHER_LDFLAGS": .string("$(inherited) -framework AppIntents"),
        ]
    )
)
#endif

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
	if len(result.Updated) != 1 {
		t.Fatalf("updated len = %d, want 1", len(result.Updated))
	}

	rootPackage := readVersionManifest(t, filepath.Join(projectRoot, "Package.swift"))
	for _, want := range []string{
		`let packageSettings = PackageSettings(`,
		`"SwiftIoC": .framework,`,
		`"OTHER_LDFLAGS": .string("$(inherited) -framework AppIntents"),`,
	} {
		if !strings.Contains(rootPackage, want) {
			t.Fatalf("root Package.swift missing %q:\n%s", want, rootPackage)
		}
	}
	if strings.Contains(rootPackage, `strictPackageBaseSettings`) {
		t.Fatalf("root Package.swift unexpectedly received generated strict root settings:\n%s", rootPackage)
	}

	modulePackage := readVersionManifest(t, filepath.Join(projectRoot, "Packages", "Auth", "Package.swift"))
	for _, want := range []string{
		`// swift-tools-version: 6.2`,
		`swiftSettings: [`,
		`.swiftLanguageMode(.v6),`,
		`.enableUpcomingFeature("StrictConcurrency"),`,
		`.enableUpcomingFeature("InferSendableFromCaptures"),`,
	} {
		if !strings.Contains(modulePackage, want) {
			t.Fatalf("module Package.swift missing %q:\n%s", want, modulePackage)
		}
	}
}

func TestSyncPackageStrictnessAppliesToAllRegularTargetsInSwift6Modules(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.2
import PackageDescription

let modulesPath = "Packages"

let package = Package(
    name: "DemoDependencies",
    dependencies: [
        .package(path: "./Packages/AuthImpl"),
    ],
    targets: []
)
`)
	writeVersionManifest(t, filepath.Join(projectRoot, "Packages", "AuthImpl", "Package.swift"), `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "AuthImpl",
    products: [
        .library(name: "AuthImpl", type: .dynamic, targets: ["AuthImpl"]),
    ],
    dependencies: [],
    targets: [
        .target(
            name: "AuthImplCore",
            dependencies: [],
            path: "Sources/Core"
        ),
        .target(
            name: "AuthImplWireSupport",
            dependencies: [],
            path: "Sources/WireSupport"
        ),
        .target(
            name: "AuthImpl",
            dependencies: [
                "AuthImplCore",
                "AuthImplWireSupport",
            ],
            path: "Sources",
            exclude: [
                "Core",
                "WireSupport",
            ]
        ),
        .testTarget(
            name: "AuthImplTests",
            dependencies: ["AuthImpl"],
            path: "Tests"
        )
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
	if len(result.Updated) != 1 {
		t.Fatalf("updated len = %d, want 1", len(result.Updated))
	}

	modulePackage := readVersionManifest(t, filepath.Join(projectRoot, "Packages", "AuthImpl", "Package.swift"))
	if got := strings.Count(modulePackage, `swiftSettings: [`); got != 3 {
		t.Fatalf("swiftSettings block count = %d, want 3:\n%s", got, modulePackage)
	}
	if strings.Contains(modulePackage, `name: "AuthImplTests",
            dependencies: ["AuthImpl"],
            path: "Tests",
            swiftSettings: [`) {
		t.Fatalf("test target unexpectedly received swiftSettings:\n%s", modulePackage)
	}
	for _, want := range []string{
		`.target(
            name: "AuthImplCore",`,
		`.target(
            name: "AuthImplWireSupport",`,
		`.target(
            name: "AuthImpl",`,
		`.enableUpcomingFeature("ExistentialAny"),`,
		`exclude: [
                "Core",
                "WireSupport",
            ],
            swiftSettings: [`,
	} {
		if !strings.Contains(modulePackage, want) {
			t.Fatalf("module Package.swift missing %q:\n%s", want, modulePackage)
		}
	}
}

func TestSyncPackageStrictnessRepositionsSwiftSettingsAfterExclude(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.2
import PackageDescription

let modulesPath = "Packages"

let package = Package(
    name: "DemoDependencies",
    dependencies: [
        .package(path: "./Packages/AuthImpl"),
    ],
    targets: []
)
`)
	modulePath := filepath.Join(projectRoot, "Packages", "AuthImpl", "Package.swift")
	writeVersionManifest(t, modulePath, `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "AuthImpl",
    products: [
        .library(name: "AuthImpl", type: .dynamic, targets: ["AuthImpl"]),
    ],
    dependencies: [],
    targets: [
        .target(
            name: "AuthImpl",
            dependencies: [],
            path: "Sources",
            swiftSettings: [
                .swiftLanguageMode(.v6),
            ],
            exclude: [
                "Core",
            ]
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
	if len(result.Updated) != 1 {
		t.Fatalf("updated len = %d, want 1", len(result.Updated))
	}

	modulePackage := readVersionManifest(t, modulePath)
	excludeIndex := strings.Index(modulePackage, `exclude: [`)
	swiftSettingsIndex := strings.Index(modulePackage, `swiftSettings: [`)
	if excludeIndex < 0 || swiftSettingsIndex < 0 {
		t.Fatalf("module Package.swift missing exclude/swiftSettings:\n%s", modulePackage)
	}
	if excludeIndex > swiftSettingsIndex {
		t.Fatalf("swiftSettings still appears before exclude:\n%s", modulePackage)
	}
}

func TestSyncPackageStrictnessRemovesDuplicateSwiftSettingsForms(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.2
import PackageDescription

let modulesPath = "Packages"

let package = Package(
    name: "DemoDependencies",
    dependencies: [
        .package(path: "./Packages/Utilities"),
    ],
    targets: []
)
`)
	modulePath := filepath.Join(projectRoot, "Packages", "Utilities", "Package.swift")
	writeVersionManifest(t, modulePath, `// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "Utilities",
    targets: [
        .target(
            name: "Utilities",
            dependencies: [],
            path: "Sources",
            swiftSettings: strictSwiftSettings,
            swiftSettings: [
                .swiftLanguageMode(.v6),
            ]
        ),
        .testTarget(
            name: "UtilitiesTests",
            dependencies: ["Utilities"],
            path: "Tests",
            swiftSettings: strictSwiftSettings
        ),
    ]
)

let strictSwiftSettings: [SwiftSetting] = [
    .swiftLanguageMode(.v6),
]
`)

	result, err := SyncPackageStrictness(projectRoot, config.ProjectConfig{
		SwiftVersion: "6.2",
		ModulesPath:  "Packages",
	})
	if err != nil {
		t.Fatalf("SyncPackageStrictness() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("updated len = %d, want 1", len(result.Updated))
	}

	modulePackage := readVersionManifest(t, modulePath)
	targetStart := strings.Index(modulePackage, `.target(
            name: "Utilities"`)
	testTargetStart := strings.Index(modulePackage, `.testTarget(`)
	if targetStart < 0 || testTargetStart < 0 {
		t.Fatalf("module Package.swift missing target/testTarget:\n%s", modulePackage)
	}
	targetBlock := modulePackage[targetStart:testTargetStart]
	if strings.Contains(targetBlock, `swiftSettings: strictSwiftSettings`) {
		t.Fatalf("regular target kept strictSwiftSettings reference:\n%s", modulePackage)
	}
	if got := strings.Count(targetBlock, `swiftSettings:`); got != 1 {
		t.Fatalf("regular target swiftSettings count = %d, want 1:\n%s", got, modulePackage)
	}
	if !strings.Contains(modulePackage[testTargetStart:], `swiftSettings: strictSwiftSettings`) {
		t.Fatalf("test target should keep existing swiftSettings reference:\n%s", modulePackage)
	}
}

func TestSyncPackageStrictnessSkipsSwift5Modules(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.2
import PackageDescription

let modulesPath = "Packages"

let package = Package(
    name: "DemoDependencies",
    dependencies: [
        .package(path: "./Packages/LegacyKit"),
    ],
    targets: []
)
`)
	writeVersionManifest(t, filepath.Join(projectRoot, "Packages", "LegacyKit", "Package.swift"), `// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "LegacyKit",
    products: [
        .library(name: "LegacyKit", targets: ["LegacyKit"]),
    ],
    dependencies: [],
    targets: [
        .target(
            name: "LegacyKit",
            dependencies: [],
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
	if len(result.Updated) != 0 {
		t.Fatalf("updated len = %d, want 0", len(result.Updated))
	}

	modulePackage := readVersionManifest(t, filepath.Join(projectRoot, "Packages", "LegacyKit", "Package.swift"))
	if strings.Contains(modulePackage, `swiftSettings: [`) {
		t.Fatalf("swift 5 module unexpectedly received swiftSettings:\n%s", modulePackage)
	}
	if !strings.Contains(modulePackage, `// swift-tools-version: 5.9`) {
		t.Fatalf("swift 5 module tools version changed unexpectedly:\n%s", modulePackage)
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

func TestSyncPackageStrictnessRemovesGeneratedRootStrictnessBlock(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.0
import PackageDescription
#if TUIST
import ProjectDescription

let strictPackageBaseSettings: SettingsDictionary = [
    "SWIFT_VERSION": "6.0",
    "SWIFT_STRICT_CONCURRENCY": "complete",
]

let packageSettings = PackageSettings(
    baseSettings: .settings(base: strictPackageBaseSettings)
)
#endif

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
    products: [
        .library(name: "Auth", type: .dynamic, targets: ["Auth"]),
    ],
    dependencies: [],
    targets: [
        .target(
            name: "Auth",
            dependencies: [],
            path: "Sources"
        ),
    ]
)
`)

	result, err := SyncPackageStrictness(projectRoot, config.ProjectConfig{
		SwiftVersion: "6.0",
		ModulesPath:  "Packages",
	})
	if err != nil {
		t.Fatalf("SyncPackageStrictness() error = %v", err)
	}
	if len(result.Updated) != 2 {
		t.Fatalf("updated len = %d, want 2", len(result.Updated))
	}

	rootPackage := readVersionManifest(t, filepath.Join(projectRoot, "Package.swift"))
	if strings.Contains(rootPackage, `strictPackageBaseSettings`) {
		t.Fatalf("root Package.swift still contains generated strict root settings:\n%s", rootPackage)
	}
	if strings.Contains(rootPackage, `baseSettings: .settings(base: strictPackageBaseSettings)`) {
		t.Fatalf("root Package.swift still contains generated strict root PackageSettings:\n%s", rootPackage)
	}
	if !strings.Contains(rootPackage, `let modulesPath = "Packages"`) {
		t.Fatalf("root Package.swift lost package content unexpectedly:\n%s", rootPackage)
	}
}
