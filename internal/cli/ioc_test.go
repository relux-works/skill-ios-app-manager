package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestIocSetupIntegration(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// Create two feature modules (with Interface/Impl split).
	for _, name := range []string{"Auth", "TodoList"} {
		if _, err := executeRootCommand(
			"--config", configPath,
			"module", "create", name, "--type", "feature",
		); err != nil {
			t.Fatalf("module create %s error = %v", name, err)
		}
	}

	output, err := executeRootCommand("--config", configPath, "ioc", "setup")
	if err != nil {
		t.Fatalf("executeRootCommand(ioc setup) error = %v", err)
	}

	if !strings.Contains(output, "SwiftIoC setup complete") {
		t.Fatalf("output = %q, want setup confirmation", output)
	}

	// Verify Registry.swift was created.
	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Registry.swift")
	requireFileExists(t, registryPath)

	registryContent := readTestFile(t, registryPath)
	for _, expected := range []string{
		"import SwiftIoC",
		"import Auth",
		"import AuthImpl",
		"import TodoList",
		"import TodoListImpl",
		"extension DemoApp",
		"Auth.Module.Interface.self",
		"Auth.Module.Impl()",
		"TodoList.Module.Interface.self",
		"TodoList.Module.Impl()",
		"static func configure()",
		"static func resolve",
	} {
		if !strings.Contains(registryContent, expected) {
			t.Fatalf("Registry.swift missing %q:\n%s", expected, registryContent)
		}
	}

	// Verify App.swift was updated.
	appSwiftPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App.swift")
	appContent := readTestFile(t, appSwiftPath)
	for _, expected := range []string{
		"import SwiftIoC",
		"Registry.configure()",
		"init() {",
	} {
		if !strings.Contains(appContent, expected) {
			t.Fatalf("App.swift missing %q:\n%s", expected, appContent)
		}
	}

	// Verify Package.swift has SwiftIoC.
	packageSwiftPath := filepath.Join(projectRoot, "Package.swift")
	packageContent := readTestFile(t, packageSwiftPath)
	if !strings.Contains(packageContent, "swift-ioc") {
		t.Fatalf("Package.swift missing SwiftIoC dependency:\n%s", packageContent)
	}

	// Verify Project.swift has .external(name: "SwiftIoC").
	projectSwiftPath := filepath.Join(projectRoot, "Project.swift")
	projectContent := readTestFile(t, projectSwiftPath)
	if !strings.Contains(projectContent, `.external(name: "SwiftIoC")`) {
		t.Fatalf("Project.swift missing SwiftIoC dependency:\n%s", projectContent)
	}
}

func TestIocSetupNoModules(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	output, err := executeRootCommand("--config", configPath, "ioc", "setup")
	if err != nil {
		t.Fatalf("executeRootCommand(ioc setup) error = %v", err)
	}

	if !strings.Contains(output, "SwiftIoC setup complete") {
		t.Fatalf("output = %q, want setup confirmation", output)
	}

	// Registry.swift should exist but without module registrations.
	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Registry.swift")
	requireFileExists(t, registryPath)

	registryContent := readTestFile(t, registryPath)
	if strings.Contains(registryContent, "ioc.register") {
		t.Fatalf("Registry.swift should not contain registrations for 0 modules:\n%s", registryContent)
	}
}

func TestIocSetupIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	if _, err := executeRootCommand(
		"--config", configPath,
		"module", "create", "Auth", "--type", "feature",
	); err != nil {
		t.Fatalf("module create Auth error = %v", err)
	}

	// Run setup twice.
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("first ioc setup error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("second ioc setup error = %v", err)
	}

	// Verify App.swift has exactly one Registry.configure().
	appSwiftPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App.swift")
	appContent := readTestFile(t, appSwiftPath)
	count := strings.Count(appContent, "Registry.configure()")
	if count != 1 {
		t.Fatalf("Registry.configure() appears %d times, want 1:\n%s", count, appContent)
	}
}

func TestIocHelpShowsSubcommands(t *testing.T) {
	output, err := executeRootCommand("ioc", "--help")
	if err != nil {
		t.Fatalf("executeRootCommand(ioc --help) error = %v", err)
	}

	if !strings.Contains(output, "setup") {
		t.Fatalf("ioc help output missing 'setup':\n%s", output)
	}
}

func writeProjectScaffold(t *testing.T, projectRoot string, cfg config.ProjectConfig) {
	t.Helper()

	// Create directories.
	for _, dir := range []string{
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Resources"),
		filepath.Join(projectRoot, "Packages"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
	}

	// Write Package.swift.
	packageSwift := `// swift-tools-version: 6.2
import PackageDescription

let modulesPath = "Packages"

let package = Package(
    name: "DemoAppDependencies",
    dependencies: [
    ],
    targets: []
)
`
	writeTestFile(t, filepath.Join(projectRoot, "Package.swift"), packageSwift)

	// Write Project.swift.
	projectSwift := `import ProjectDescription

let appName = "DemoApp"
let bundleID = "com.example.demo"

let project = Project(
    name: appName,
    targets: [
        .target(
            name: appName,
            destinations: .iOS,
            product: .app,
            bundleId: bundleID,
            sources: ["Targets/DemoApp/Sources/**"],
            dependencies: [
            ]
        )
    ]
)
`
	writeTestFile(t, filepath.Join(projectRoot, "Project.swift"), projectSwift)

	// Write App.swift.
	appSwift := `import SwiftUI

@main
struct DemoApp: App {
    var body: some Scene {
        WindowGroup {
            Text("Hello, World!")
        }
    }
}
`
	writeTestFile(t, filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App.swift"), appSwift)
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

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}
