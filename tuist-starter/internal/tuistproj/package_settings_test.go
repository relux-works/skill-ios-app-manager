package tuistproj

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureFrameworkProductTypesAppendsPackageSettingsBlock(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "Package.swift")
	writePackageSettingsTestFile(t, path, `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)
`)

	if err := EnsureFrameworkProductTypes(path, "SwiftIoC", "SwiftUIRelux"); err != nil {
		t.Fatalf("EnsureFrameworkProductTypes() error = %v", err)
	}

	content := readPackageSettingsTestFile(t, path)
	for _, expected := range []string{
		"#if TUIST",
		"import ProjectDescription",
		"PackageSettings",
		`"SwiftIoC": .framework`,
		`"SwiftUIRelux": .framework`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}
}

func TestEnsureFrameworkProductTypesMergesIntoExistingBlock(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "Package.swift")
	writePackageSettingsTestFile(t, path, `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    productTypes: [
        "Relux": .framework,
    ]
)
#endif
`)

	if err := EnsureFrameworkProductTypes(path, "SwiftUIRelux", "ReluxRouter"); err != nil {
		t.Fatalf("EnsureFrameworkProductTypes() error = %v", err)
	}

	content := readPackageSettingsTestFile(t, path)
	for _, expected := range []string{
		`"Relux": .framework`,
		`"ReluxRouter": .framework`,
		`"SwiftUIRelux": .framework`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}
}

func TestEnsureFrameworkProductTypesInsertsProductTypesIntoExistingPackageSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "Package.swift")
	writePackageSettingsTestFile(t, path, `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    baseSettings: .settings(configurations: [])
)
#endif
`)

	if err := EnsureFrameworkProductTypes(path, "SwiftIoC"); err != nil {
		t.Fatalf("EnsureFrameworkProductTypes() error = %v", err)
	}

	content := readPackageSettingsTestFile(t, path)
	for _, expected := range []string{
		"PackageSettings(",
		"productTypes: [",
		`"SwiftIoC": .framework`,
		"baseSettings: .settings(configurations: [])",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}
}

func TestEnsureFrameworkProductTypesIsIdempotent(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "Package.swift")
	writePackageSettingsTestFile(t, path, `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    productTypes: [
        "SwiftIoC": .framework,
        "SwiftUIRelux": .framework,
    ]
)
#endif
`)

	if err := EnsureFrameworkProductTypes(path, "SwiftUIRelux", "SwiftIoC"); err != nil {
		t.Fatalf("EnsureFrameworkProductTypes() error = %v", err)
	}

	content := readPackageSettingsTestFile(t, path)
	if strings.Count(content, "PackageSettings") != 1 {
		t.Fatalf("PackageSettings count = %d, want 1:\n%s", strings.Count(content, "PackageSettings"), content)
	}
	if strings.Count(content, `"SwiftIoC": .framework`) != 1 {
		t.Fatalf("SwiftIoC framework entry count = %d, want 1:\n%s", strings.Count(content, `"SwiftIoC": .framework`), content)
	}
}

func TestEnsureTargetBuildSettingsInsertsTargetSettingsIntoExistingPackageSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "Package.swift")
	writePackageSettingsTestFile(t, path, `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    productTypes: [
        "ErrorHandlingModule": .framework,
    ]
)
#endif
`)

	if err := EnsureTargetBuildSettings(path, TargetBuildSetting{
		ProductName: "ErrorHandlingModule",
		Key:         "IPHONEOS_DEPLOYMENT_TARGET",
		Value:       "16.0",
	}); err != nil {
		t.Fatalf("EnsureTargetBuildSettings() error = %v", err)
	}

	content := readPackageSettingsTestFile(t, path)
	for _, expected := range []string{
		"targetSettings: [",
		`"ErrorHandlingModule": .settings(base: [`,
		`"IPHONEOS_DEPLOYMENT_TARGET": "16.0"`,
		`"ErrorHandlingModule": .framework`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}
}

func TestEnsureTargetBuildSettingsIsIdempotent(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "Package.swift")
	writePackageSettingsTestFile(t, path, `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    targetSettings: [
        "ErrorHandlingModule": .settings(base: [
            "IPHONEOS_DEPLOYMENT_TARGET": "16.0",
        ]),
    ]
)
#endif
`)

	setting := TargetBuildSetting{
		ProductName: "ErrorHandlingModule",
		Key:         "IPHONEOS_DEPLOYMENT_TARGET",
		Value:       "16.0",
	}
	if err := EnsureTargetBuildSettings(path, setting, setting); err != nil {
		t.Fatalf("EnsureTargetBuildSettings() error = %v", err)
	}

	content := readPackageSettingsTestFile(t, path)
	if strings.Count(content, `"ErrorHandlingModule": .settings`) != 1 {
		t.Fatalf("target settings entry count = %d, want 1:\n%s", strings.Count(content, `"ErrorHandlingModule": .settings`), content)
	}
	if strings.Count(content, `"IPHONEOS_DEPLOYMENT_TARGET": "16.0"`) != 1 {
		t.Fatalf("build setting count = %d, want 1:\n%s", strings.Count(content, `"IPHONEOS_DEPLOYMENT_TARGET": "16.0"`), content)
	}
}

func TestEnsureTargetBuildSettingsConvergesExistingValueWithoutDuplicateProduct(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "Package.swift")
	writePackageSettingsTestFile(t, path, `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    targetSettings: [
        "FireAuthRelux": .settings(base: [
            "IPHONEOS_DEPLOYMENT_TARGET": "18.0",
            "SWIFT_VERSION": "6.0",
        ]),
        "Unrelated": .settings(base: [
            "OTHER_SETTING": "preserved",
        ]),
    ]
)
#endif
`)

	setting := TargetBuildSetting{
		ProductName: "FireAuthRelux",
		Key:         "IPHONEOS_DEPLOYMENT_TARGET",
		Value:       "17.0",
	}
	if err := EnsureTargetBuildSettings(path, setting); err != nil {
		t.Fatalf("EnsureTargetBuildSettings() error = %v", err)
	}
	first := readPackageSettingsTestFile(t, path)
	if err := EnsureTargetBuildSettings(path, setting); err != nil {
		t.Fatalf("second EnsureTargetBuildSettings() error = %v", err)
	}
	second := readPackageSettingsTestFile(t, path)

	if second != first {
		t.Fatalf("EnsureTargetBuildSettings() is not byte-idempotent:\n%s", second)
	}
	if strings.Count(first, `"FireAuthRelux": .settings`) != 1 {
		t.Fatalf("FireAuthRelux settings count != 1:\n%s", first)
	}
	if strings.Contains(first, `"IPHONEOS_DEPLOYMENT_TARGET": "18.0"`) ||
		!strings.Contains(first, `"IPHONEOS_DEPLOYMENT_TARGET": "17.0"`) {
		t.Fatalf("deployment target did not converge:\n%s", first)
	}
	for _, preserved := range []string{
		`"SWIFT_VERSION": "6.0"`,
		`"Unrelated": .settings`,
		`"OTHER_SETTING": "preserved"`,
	} {
		if !strings.Contains(first, preserved) {
			t.Fatalf("convergence lost %q:\n%s", preserved, first)
		}
	}
}

func TestRemoveFrameworkProductTypesRemovesRequestedEntries(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "Package.swift")
	writePackageSettingsTestFile(t, path, `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)

#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    productTypes: [
        "Relux": .framework,
        "ReluxRouter": .framework,
        "SwiftUIRelux": .framework,
    ]
)
#endif
`)

	if err := RemoveFrameworkProductTypes(path, "ReluxRouter", "SwiftUIRelux"); err != nil {
		t.Fatalf("RemoveFrameworkProductTypes() error = %v", err)
	}

	content := readPackageSettingsTestFile(t, path)
	if !strings.Contains(content, `"Relux": .framework`) {
		t.Fatalf("content removed unrelated entry:\n%s", content)
	}
	for _, unexpected := range []string{
		`"ReluxRouter": .framework`,
		`"SwiftUIRelux": .framework`,
	} {
		if strings.Contains(content, unexpected) {
			t.Fatalf("content still contains %q:\n%s", unexpected, content)
		}
	}
}

func TestRemoveFrameworkProductTypesIsNoOpWithoutPackageSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "Package.swift")
	original := `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)
`
	writePackageSettingsTestFile(t, path, original)

	if err := RemoveFrameworkProductTypes(path, "SwiftIoC"); err != nil {
		t.Fatalf("RemoveFrameworkProductTypes() error = %v", err)
	}

	content := readPackageSettingsTestFile(t, path)
	if content != original {
		t.Fatalf("content changed unexpectedly:\n%s", content)
	}
}

func writePackageSettingsTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %q: %v", path, err)
	}
}

func readPackageSettingsTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}
	return string(data)
}
