package relux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditAppSwiftForReluxAddsImportsAndResolver(t *testing.T) {
	t.Parallel()

	input := `import SwiftUI
import SwiftIoC

@main
struct DemoApp: App {
    init() {
        Registry.configure()
    }

    var body: some Scene {
        WindowGroup {
            Text("Hello, World!")
        }
    }
}
`

	result := EditAppSwiftForRelux(input, "DemoApp")

	for _, expected := range []string{
		"@_exported import Relux",
		"import SwiftUIRelux",
		"Relux.Resolver(",
		"splash: { DemoApp.Splash() }",
		"content: { relux in DemoApp.Content() }",
		"resolver: { await Registry.resolveAsync(Relux.self) }",
	} {
		if !strings.Contains(result, expected) {
			t.Fatalf("result missing %q:\n%s", expected, result)
		}
	}

	// Original WindowGroup should still be present.
	if !strings.Contains(result, "WindowGroup {") {
		t.Fatalf("result missing WindowGroup:\n%s", result)
	}

	// Old Text("Hello, World!") should be removed.
	if strings.Contains(result, `Text("Hello, World!")`) {
		t.Fatalf("result should not contain old Text body:\n%s", result)
	}
}

func TestEditAppSwiftForReluxIdempotent(t *testing.T) {
	t.Parallel()

	input := `@_exported import Relux
import SwiftUI
import SwiftUIRelux
import SwiftIoC

@main
struct DemoApp: App {
    init() {
        Registry.configure()
    }

    var body: some Scene {
        WindowGroup {
            Relux.Resolver(
                splash: { DemoApp.Splash() },
                content: { relux in DemoApp.Content() },
                resolver: { await Registry.resolveAsync(Relux.self) }
            )
        }
    }
}
`

	result := EditAppSwiftForRelux(input, "DemoApp")

	// Should have exactly one of each.
	for _, s := range []string{
		"@_exported import Relux",
		"import SwiftUIRelux",
		"Relux.Resolver(",
	} {
		count := strings.Count(result, s)
		if count != 1 {
			t.Fatalf("%q appears %d times, want 1:\n%s", s, count, result)
		}
	}
}

func TestEditAppSwiftForReluxReplacesExportedImport(t *testing.T) {
	t.Parallel()

	input := `import Relux
import SwiftUI

@main
struct DemoApp: App {
    var body: some Scene {
        WindowGroup {
            Text("Hello")
        }
    }
}
`

	result := EditAppSwiftForRelux(input, "DemoApp")

	if !strings.Contains(result, "@_exported import Relux") {
		t.Fatalf("result missing @_exported import Relux:\n%s", result)
	}

	// Plain "import Relux" should be replaced (not duplicated).
	plainImportCount := 0
	for _, line := range strings.Split(result, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "import Relux" {
			plainImportCount++
		}
	}
	if plainImportCount > 0 {
		t.Fatalf("plain 'import Relux' should be replaced by @_exported version:\n%s", result)
	}
}

func TestEnsureExportedImport(t *testing.T) {
	t.Parallel()

	input := "import SwiftUI\nimport Relux\n\nstruct Foo {}\n"
	result := ensureExportedImport(input, "Relux")

	if !strings.Contains(result, "@_exported import Relux") {
		t.Fatalf("result missing @_exported import Relux:\n%s", result)
	}

	// Should not have plain import Relux.
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "import Relux" {
			t.Fatalf("result should not contain plain 'import Relux':\n%s", result)
		}
	}
}

func TestEnsureExportedImportAlreadyPresent(t *testing.T) {
	t.Parallel()

	input := "@_exported import Relux\nimport SwiftUI\n"
	result := ensureExportedImport(input, "Relux")

	count := strings.Count(result, "@_exported import Relux")
	if count != 1 {
		t.Fatalf("@_exported import Relux appears %d times, want 1:\n%s", count, result)
	}
}

func TestReplaceWindowGroupBody(t *testing.T) {
	t.Parallel()

	input := `        WindowGroup {
            Text("Hello, World!")
        }`

	result := replaceWindowGroupBody(input, "DemoApp")

	if !strings.Contains(result, "Relux.Resolver(") {
		t.Fatalf("result missing Relux.Resolver:\n%s", result)
	}
	if strings.Contains(result, `Text("Hello, World!")`) {
		t.Fatalf("result should not contain old body:\n%s", result)
	}
}

func TestReplaceWindowGroupBodySkipsIfAlreadyPresent(t *testing.T) {
	t.Parallel()

	input := `        WindowGroup {
            Relux.Resolver(
                splash: { DemoApp.Splash() },
                content: { relux in DemoApp.Content() },
                resolver: { await Registry.resolveAsync(Relux.self) }
            )
        }`

	result := replaceWindowGroupBody(input, "DemoApp")

	// Should be unchanged.
	if result != input {
		t.Fatalf("expected no change, got:\n%s", result)
	}
}

func TestRenderSetupTemplate(t *testing.T) {
	t.Parallel()

	outputPath := t.TempDir() + "/Splash.swift"
	data := setupTemplateData{AppTypeName: "DemoApp"}

	if err := renderSetupTemplate("setup_templates/splash.swift.tmpl", outputPath, data); err != nil {
		t.Fatalf("renderSetupTemplate() error = %v", err)
	}

	content := readTestFile(t, outputPath)

	for _, expected := range []string{
		"import SwiftUI",
		"extension DemoApp",
		"struct Splash: View",
		"ProgressView()",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}
}

func TestRenderReluxLoggerTemplate(t *testing.T) {
	t.Parallel()

	outputPath := t.TempDir() + "/ReluxLogger.swift"
	data := setupTemplateData{AppTypeName: "DemoApp"}

	if err := renderSetupTemplate("setup_templates/relux_logger.swift.tmpl", outputPath, data); err != nil {
		t.Fatalf("renderSetupTemplate() error = %v", err)
	}

	content := readTestFile(t, outputPath)

	for _, expected := range []string{
		"import Relux",
		"extension DemoApp",
		"struct ReluxLogger: Relux.Logger",
		"func logAction(",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}
}

func TestValidateSetupInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input SetupInput
		want  string
	}{
		{
			name:  "empty project root",
			input: SetupInput{AppName: "Demo"},
			want:  "project root is required",
		},
		{
			name:  "empty app name",
			input: SetupInput{ProjectRoot: "/tmp"},
			want:  "app name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSetupInput(tc.input)
			if err == nil {
				t.Fatal("validateSetupInput() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestPatchPackageSwiftForRelux(t *testing.T) {
	t.Parallel()

	packageSwift := `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoDependencies",
    dependencies: [],
    targets: []
)
`
	path := filepath.Join(t.TempDir(), "Package.swift")
	if err := os.WriteFile(path, []byte(packageSwift), 0o644); err != nil {
		t.Fatalf("write Package.swift: %v", err)
	}

	if err := patchPackageSwiftForRelux(path); err != nil {
		t.Fatalf("patchPackageSwiftForRelux() error = %v", err)
	}

	content := readTestFile(t, path)

	for _, expected := range []string{
		"#if TUIST",
		"import ProjectDescription",
		"PackageSettings",
		`"Relux": .framework`,
		`"ReluxRouter": .framework`,
		`"SwiftUIRelux": .framework`,
		"#endif",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}

	// Original package should still be present.
	if !strings.Contains(content, "let package = Package(") {
		t.Fatalf("content missing original Package declaration:\n%s", content)
	}
}

func TestPatchPackageSwiftIdempotent(t *testing.T) {
	t.Parallel()

	packageSwift := `// swift-tools-version: 6.0
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
`
	path := filepath.Join(t.TempDir(), "Package.swift")
	if err := os.WriteFile(path, []byte(packageSwift), 0o644); err != nil {
		t.Fatalf("write Package.swift: %v", err)
	}

	if err := patchPackageSwiftForRelux(path); err != nil {
		t.Fatalf("patchPackageSwiftForRelux() error = %v", err)
	}

	content := readTestFile(t, path)
	count := strings.Count(content, "PackageSettings")
	if count != 1 {
		t.Fatalf("PackageSettings appears %d times, want 1:\n%s", count, content)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := readFile(path)
	if err != nil {
		t.Fatalf("readFile(%q) error = %v", path, err)
	}
	return string(data)
}
