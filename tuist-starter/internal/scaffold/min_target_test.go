package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.0
import PackageDescription

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    targetSettings: [
        "DemoDependency": .settings(base: [
            "IPHONEOS_DEPLOYMENT_TARGET": "16.0",
        ]),
    ]
)
#endif

let package = Package(
    name: "DemoDependencies",
    dependencies: [
        .package(url: "https://example.invalid/pinned.git", exact: "1.2.3"),
    ],
    targets: []
)
`)
	writeVersionManifest(t, filepath.Join(projectRoot, "Packages", "Utilities", "Package.swift"), `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "Utilities",
    platforms: [
        .iOS("16.0"),
        .macOS(.v13)
    ],
    products: [
        .library(name: "Utilities", targets: ["Utilities"]),
    ],
    targets: [
        .target(name: "Utilities"),
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

	if len(result.Scanned) != 4 {
		t.Fatalf("scanned len = %d, want 4", len(result.Scanned))
	}
	if len(result.Updated) != 4 {
		t.Fatalf("updated len = %d, want 4", len(result.Updated))
	}

	for _, manifestPath := range []string{
		filepath.Join(projectRoot, "Project.swift"),
		filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"),
	} {
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

	packageContent := readVersionManifest(t, filepath.Join(projectRoot, "Packages", "Utilities", "Package.swift"))
	if !strings.Contains(packageContent, `.iOS("17.0")`) {
		t.Fatalf("package manifest missing synced iOS platform:\n%s", packageContent)
	}
	if !strings.Contains(packageContent, `.macOS(.v13)`) {
		t.Fatalf("package manifest should preserve non-iOS platforms:\n%s", packageContent)
	}
	rootPackageContent := readVersionManifest(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(rootPackageContent, `"IPHONEOS_DEPLOYMENT_TARGET": "17.0"`) {
		t.Fatalf("root package manifest missing synced deployment override:\n%s", rootPackageContent)
	}
	if !strings.Contains(rootPackageContent, `exact: "1.2.3"`) {
		t.Fatalf("root package manifest should preserve dependency pins:\n%s", rootPackageContent)
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
	writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.0
import PackageDescription

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    targetSettings: [
        "DemoDependency": .settings(base: [
            "IPHONEOS_DEPLOYMENT_TARGET": "17.0",
        ]),
    ]
)
#endif

let package = Package(name: "DemoDependencies")
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

func TestSyncMinTargetSyncsPackageManifestsToConfiguredMinimum(t *testing.T) {
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
	writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.0
import PackageDescription

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    targetSettings: [
        "SharedKit": .settings(base: [
            "IPHONEOS_DEPLOYMENT_TARGET": "16.0",
        ]),
    ]
)
#endif

let package = Package(name: "DemoDependencies")
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
	if len(result.Updated) != 3 {
		t.Fatalf("updated len = %d, want 3", len(result.Updated))
	}

	content := readVersionManifest(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		`let minTarget = "17.0"`,
		`deploymentTargets: .iOS(minTarget)`,
		`"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget)`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, content)
		}
	}

	packageContent := readVersionManifest(t, filepath.Join(projectRoot, "Packages", "SharedKit", "Package.swift"))
	if !strings.Contains(packageContent, `.iOS("17.0")`) {
		t.Fatalf("package manifest missing configured iOS minimum:\n%s", packageContent)
	}
	if strings.Contains(packageContent, `.iOS(.v18)`) {
		t.Fatalf("package manifest kept stale iOS minimum:\n%s", packageContent)
	}
	rootPackageContent := readVersionManifest(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(rootPackageContent, `"IPHONEOS_DEPLOYMENT_TARGET": "17.0"`) {
		t.Fatalf("root package manifest missing configured iOS minimum:\n%s", rootPackageContent)
	}
}

func TestSyncMinTargetMigratesConverterShapedProjectAndIsByteIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectPath := filepath.Join(projectRoot, "Project.swift")
	rootPackagePath := filepath.Join(projectRoot, "Package.swift")
	corePackagePath := filepath.Join(projectRoot, "Packages", "ConverterCore", "Package.swift")
	featurePackagePath := filepath.Join(projectRoot, "Packages", "Features", "ConverterFeature", "Package.swift")
	utilityPackagePath := filepath.Join(projectRoot, "Packages", "Utilities", "Package.swift")
	vendorPackagePath := filepath.Join(projectRoot, "Packages", "ConverterCore", ".build", "checkouts", "PinnedVendor", "Package.swift")

	writeVersionManifest(t, projectPath, converterShapedProjectManifest("17.0"))
	writeVersionManifest(t, rootPackagePath, converterShapedRootPackageManifest("17.0"))
	writeVersionManifest(t, corePackagePath, converterShapedLocalPackageManifest("ConverterCore", `
        .iOS(.v17),
        .macOS(.v14),
`))
	writeVersionManifest(t, featurePackagePath, converterShapedLocalPackageManifest("ConverterFeature", `
        .iOS(.v17_6),
`))
	writeVersionManifest(t, utilityPackagePath, converterShapedLocalPackageManifest("Utilities", `
        .macOS(.v14),
`))
	writeVersionManifest(t, vendorPackagePath, converterShapedLocalPackageManifest("PinnedVendor", `
        .iOS(.v17),
`))

	result, err := SyncMinTarget(projectRoot, config.ProjectConfig{
		MinTarget:   "26.0",
		ModulesPath: "Packages",
	})
	if err != nil {
		t.Fatalf("SyncMinTarget() error = %v", err)
	}
	if len(result.Scanned) != 5 {
		t.Fatalf("scanned len = %d, want 5", len(result.Scanned))
	}
	if len(result.Updated) != 4 {
		t.Fatalf("updated len = %d, want 4", len(result.Updated))
	}

	projectContent := readVersionManifest(t, projectPath)
	if strings.Count(projectContent, `deploymentTargets: .iOS(minTarget),`) != 3 {
		t.Fatalf("project deploymentTargets count = %d, want app/unit/UI targets:\n%s", strings.Count(projectContent, `deploymentTargets: .iOS(minTarget),`), projectContent)
	}
	if strings.Count(projectContent, `"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),`) != 3 {
		t.Fatalf("project deployment setting count = %d, want app/unit/UI targets:\n%s", strings.Count(projectContent, `"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),`), projectContent)
	}

	rootPackageContent := readVersionManifest(t, rootPackagePath)
	for _, want := range []string{
		`"IPHONEOS_DEPLOYMENT_TARGET": "26.0",`,
		`exact: "1.2.3"`,
		`.revision("0123456789abcdef")`,
	} {
		if !strings.Contains(rootPackageContent, want) {
			t.Fatalf("root Package.swift missing preserved/synced value %q:\n%s", want, rootPackageContent)
		}
	}
	if strings.Contains(rootPackageContent, `"IPHONEOS_DEPLOYMENT_TARGET": "17.0"`) {
		t.Fatalf("root Package.swift retained stale deployment override:\n%s", rootPackageContent)
	}

	for _, packagePath := range []string{corePackagePath, featurePackagePath} {
		content := readVersionManifest(t, packagePath)
		if !strings.Contains(content, `.iOS("26.0")`) {
			t.Fatalf("%s missing synced iOS platform:\n%s", packagePath, content)
		}
		if strings.Contains(content, `.iOS(.v17`) {
			t.Fatalf("%s retained stale iOS platform:\n%s", packagePath, content)
		}
	}
	if utilityContent := readVersionManifest(t, utilityPackagePath); strings.Contains(utilityContent, `.iOS(`) {
		t.Fatalf("macOS-only package unexpectedly gained an iOS declaration:\n%s", utilityContent)
	}
	if vendorContent := readVersionManifest(t, vendorPackagePath); !strings.Contains(vendorContent, `.iOS(.v17)`) {
		t.Fatalf("generated SwiftPM checkout should not be treated as a first-party package:\n%s", vendorContent)
	}

	paths := []string{projectPath, rootPackagePath, corePackagePath, featurePackagePath, utilityPackagePath}
	firstRun := make(map[string]string, len(paths))
	for _, path := range paths {
		firstRun[path] = readVersionManifest(t, path)
	}

	secondResult, err := SyncMinTarget(projectRoot, config.ProjectConfig{
		MinTarget:   "26.0",
		ModulesPath: "Packages",
	})
	if err != nil {
		t.Fatalf("second SyncMinTarget() error = %v", err)
	}
	if len(secondResult.Updated) != 0 {
		t.Fatalf("second SyncMinTarget() updated %d manifests, want 0: %v", len(secondResult.Updated), secondResult.Updated)
	}
	for _, path := range paths {
		if got := readVersionManifest(t, path); got != firstRun[path] {
			t.Fatalf("second SyncMinTarget() changed %s:\nfirst:\n%s\nsecond:\n%s", path, firstRun[path], got)
		}
	}
}

func TestSyncMinTargetRejectsUnsupportedPackageLayoutBeforeMutation(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectPath := filepath.Join(projectRoot, "Project.swift")
	rootPackagePath := filepath.Join(projectRoot, "Package.swift")
	validPackagePath := filepath.Join(projectRoot, "Packages", "Valid", "Package.swift")
	unsupportedPackagePath := filepath.Join(projectRoot, "Packages", "Unsupported", "Package.swift")

	writeVersionManifest(t, projectPath, converterShapedProjectManifest("17.0"))
	writeVersionManifest(t, rootPackagePath, converterShapedRootPackageManifest("17.0"))
	writeVersionManifest(t, validPackagePath, converterShapedLocalPackageManifest("Valid", `
        .iOS(.v17),
`))
	writeVersionManifest(t, unsupportedPackagePath, `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "Unsupported",
    platforms: [.iOS(.v17, .v18)],
    targets: []
)
`)

	before := map[string]string{
		projectPath:      readVersionManifest(t, projectPath),
		rootPackagePath:  readVersionManifest(t, rootPackagePath),
		validPackagePath: readVersionManifest(t, validPackagePath),
	}
	_, err := SyncMinTarget(projectRoot, config.ProjectConfig{MinTarget: "26.0", ModulesPath: "Packages"})
	if err == nil {
		t.Fatal("SyncMinTarget() error = nil, want unsupported platform layout error")
	}
	if !strings.Contains(err.Error(), "unsupported iOS platform declaration") {
		t.Fatalf("SyncMinTarget() error = %q, want unsupported iOS platform diagnostic", err.Error())
	}
	for path, want := range before {
		if got := readVersionManifest(t, path); got != want {
			t.Fatalf("SyncMinTarget() changed %s before rejecting unsupported layout:\nwant:\n%s\ngot:\n%s", path, want, got)
		}
	}
}

func TestSyncMinTargetPackagePlatformsCompileWithPackageDescriptionSix(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("SwiftPM platform syntax regression requires macOS")
	}
	if _, err := exec.LookPath("xcrun"); err != nil {
		t.Skip("SwiftPM platform syntax regression requires xcrun")
	}

	for _, testCase := range []struct {
		name       string
		minTarget  string
		platform   string
		wantOutput string
	}{
		{
			name:       "17.6",
			minTarget:  "17.6",
			platform:   `.iOS(.v17_6)`,
			wantOutput: `.iOS("17.6")`,
		},
		{
			name:       "26.0",
			minTarget:  "26.0",
			platform:   `.iOS(.v17)`,
			wantOutput: `.iOS("26.0")`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			projectRoot := t.TempDir()
			writeVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), converterShapedProjectManifest("17.0"))
			writeVersionManifest(t, filepath.Join(projectRoot, "Package.swift"), `// swift-tools-version: 6.0
import PackageDescription

let package = Package(name: "Root")
`)
			packageRoot := filepath.Join(projectRoot, "Packages", "Probe")
			packagePath := filepath.Join(packageRoot, "Package.swift")
			writeVersionManifest(t, packagePath, compilerShapedLocalPackageManifest(testCase.platform))

			if _, err := SyncMinTarget(projectRoot, config.ProjectConfig{
				MinTarget:   testCase.minTarget,
				ModulesPath: "Packages",
			}); err != nil {
				t.Fatalf("SyncMinTarget() error = %v", err)
			}

			content := readVersionManifest(t, packagePath)
			if !strings.Contains(content, testCase.wantOutput) {
				t.Fatalf("package manifest missing canonical platform %q:\n%s", testCase.wantOutput, content)
			}

			command := exec.Command("xcrun", "swift", "package", "dump-package")
			command.Dir = packageRoot
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("xcrun swift package dump-package error = %v\n%s\nmanifest:\n%s", err, output, content)
			}
		})
	}
}

func TestSyncMinTargetRejectsUnwritableManifestBeforeMutation(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectPath := filepath.Join(projectRoot, "Project.swift")
	rootPackagePath := filepath.Join(projectRoot, "Package.swift")
	packagePath := filepath.Join(projectRoot, "Packages", "Locked", "Package.swift")

	writeVersionManifest(t, projectPath, converterShapedProjectManifest("17.0"))
	writeVersionManifest(t, rootPackagePath, converterShapedRootPackageManifest("17.0"))
	writeVersionManifest(t, packagePath, converterShapedLocalPackageManifest("Locked", `
        .iOS(.v17),
`))
	beforeProject := readVersionManifest(t, projectPath)
	beforeRootPackage := readVersionManifest(t, rootPackagePath)

	if err := os.Chmod(packagePath, 0o444); err != nil {
		t.Fatalf("Chmod(%q) error = %v", packagePath, err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(packagePath, 0o644)
	})

	_, err := SyncMinTarget(projectRoot, config.ProjectConfig{MinTarget: "26.0", ModulesPath: "Packages"})
	if err == nil {
		t.Fatal("SyncMinTarget() error = nil, want unwritable manifest diagnostic")
	}
	if !strings.Contains(err.Error(), "not writable") || !strings.Contains(err.Error(), packagePath) {
		t.Fatalf("SyncMinTarget() error = %q, want actionable unwritable package diagnostic", err.Error())
	}
	if got := readVersionManifest(t, projectPath); got != beforeProject {
		t.Fatalf("SyncMinTarget() changed Project.swift before writable preflight failed:\nwant:\n%s\ngot:\n%s", beforeProject, got)
	}
	if got := readVersionManifest(t, rootPackagePath); got != beforeRootPackage {
		t.Fatalf("SyncMinTarget() changed root Package.swift before writable preflight failed:\nwant:\n%s\ngot:\n%s", beforeRootPackage, got)
	}
}

func TestSyncMinTargetRequiresRootPackageWhenLocalPackagesArePresent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectPath := filepath.Join(projectRoot, "Project.swift")
	writeVersionManifest(t, projectPath, converterShapedProjectManifest("17.0"))
	writeVersionManifest(t, filepath.Join(projectRoot, "Packages", "Core", "Package.swift"), converterShapedLocalPackageManifest("Core", `
        .iOS(.v17),
`))
	beforeProject := readVersionManifest(t, projectPath)

	_, err := SyncMinTarget(projectRoot, config.ProjectConfig{MinTarget: "26.0", ModulesPath: "Packages"})
	if err == nil {
		t.Fatal("SyncMinTarget() error = nil, want missing root Package.swift diagnostic")
	}
	if !strings.Contains(err.Error(), "root Package.swift") {
		t.Fatalf("SyncMinTarget() error = %q, want missing root Package.swift diagnostic", err.Error())
	}
	if got := readVersionManifest(t, projectPath); got != beforeProject {
		t.Fatalf("SyncMinTarget() changed Project.swift before rejecting missing root package:\nwant:\n%s\ngot:\n%s", beforeProject, got)
	}
}

func converterShapedProjectManifest(minTarget string) string {
	return strings.ReplaceAll(`import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "Converter",
    targets: [
        .target(
            name: "Converter",
            product: .app,
            bundleId: "com.example.converter",
            deploymentTargets: .iOS("__MIN_TARGET__"),
            settings: .settings(
                base: [
                    "IPHONEOS_DEPLOYMENT_TARGET": .string("__MIN_TARGET__"),
                ]
            )
        ),
        .target(
            name: "ConverterTests",
            product: .unitTests,
            bundleId: "com.example.converter.tests",
            deploymentTargets: .iOS("__MIN_TARGET__"),
            settings: .settings(
                base: [
                    "IPHONEOS_DEPLOYMENT_TARGET": .string("__MIN_TARGET__"),
                ]
            )
        ),
        .target(
            name: "ConverterUITests",
            product: .uiTests,
            bundleId: "com.example.converter.ui-tests",
            deploymentTargets: .iOS("__MIN_TARGET__"),
            settings: .settings(
                base: [
                    "IPHONEOS_DEPLOYMENT_TARGET": .string("__MIN_TARGET__"),
                ]
            )
        ),
    ]
)
`, "__MIN_TARGET__", minTarget)
}

func converterShapedRootPackageManifest(minTarget string) string {
	return strings.ReplaceAll(`// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "ConverterDependencies",
    dependencies: [
        .package(url: "https://example.invalid/pinned.git", exact: "1.2.3"),
        .package(url: "https://example.invalid/revision.git", .revision("0123456789abcdef")),
    ],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    targetSettings: [
        "PinnedDependency": .settings(base: [
            "IPHONEOS_DEPLOYMENT_TARGET": "__MIN_TARGET__",
        ]),
        "AnotherPinnedDependency": .settings(base: [
            "IPHONEOS_DEPLOYMENT_TARGET": "__MIN_TARGET__",
        ]),
    ]
)
#endif
`, "__MIN_TARGET__", minTarget)
}

func converterShapedLocalPackageManifest(name, platforms string) string {
	return `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "` + name + `",
    platforms: [` + platforms + `    ],
    products: [
        .library(name: "` + name + `", targets: ["` + name + `"]),
    ],
    targets: []
)
`
}

func compilerShapedLocalPackageManifest(platform string) string {
	return `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "Probe",
    platforms: [` + platform + `],
    targets: [
        .target(name: "Probe"),
    ]
)
`
}
