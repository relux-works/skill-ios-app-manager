package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommandCreatesScaffoldFromConfig(t *testing.T) {
	t.Parallel()

	outputDir := filepath.Join(t.TempDir(), "DemoApp")
	configPath := filepath.Join(repoRoot(t), "testdata", "sample-config.json")

	output, err := executeRootCommand("init", "--config", configPath, "--output", outputDir)
	if err != nil {
		t.Fatalf("executeRootCommand(init) error = %v", err)
	}
	if !strings.Contains(output, "scaffolded") {
		t.Fatalf("init output = %q, want scaffold confirmation", output)
	}

	requiredPaths := []string{
		filepath.Join(outputDir, "Tuist.swift"),
		filepath.Join(outputDir, "Project.swift"),
		filepath.Join(outputDir, "Workspace.swift"),
		filepath.Join(outputDir, "Package.swift"),
		filepath.Join(outputDir, "Targets", "DemoApp", "Sources", "App.swift"),
		filepath.Join(outputDir, "Targets", "DemoApp", "Resources"),
		filepath.Join(outputDir, "Packages"),
		filepath.Join(outputDir, "Makefile"),
		filepath.Join(outputDir, ".periphery.yml"),
		filepath.Join(outputDir, ".swiftlint.yml"),
		filepath.Join(outputDir, ".gitignore"),
	}
	for _, path := range requiredPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected generated path %q: %v", path, err)
		}
	}

	projectManifest := readFileString(t, filepath.Join(outputDir, "Project.swift"))
	projectChecks := []string{
		"DemoApp",
		"com.example.demo",
		"ABCDE12345",
		`Targets/DemoApp/Sources/**`,
		`Targets/DemoApp/Resources/**`,
		"EntitlementsFactory.make(",
		"AppCapabilities.app",
	}
	for _, want := range projectChecks {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectManifest)
		}
	}

	peripheryConfig := readFileString(t, filepath.Join(outputDir, ".periphery.yml"))
	for _, want := range []string{
		`workspace: "DemoApp.xcworkspace"`,
		`  - "DemoApp"`,
		"retain_public: true",
	} {
		if !strings.Contains(peripheryConfig, want) {
			t.Fatalf(".periphery.yml missing %q:\n%s", want, peripheryConfig)
		}
	}
}

func TestInitCommandPreventsOverwriteWithoutForce(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	configPath := filepath.Join(repoRoot(t), "testdata", "sample-config.json")

	projectManifest := filepath.Join(outputDir, "Project.swift")
	if err := os.WriteFile(projectManifest, []byte("// existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", projectManifest, err)
	}

	_, err := executeRootCommand("init", "--config", configPath, "--output", outputDir)
	if err == nil {
		t.Fatal("executeRootCommand(init) error = nil, want overwrite protection error")
	}

	message := err.Error()
	if !strings.Contains(message, "--force") || !strings.Contains(message, "Project.swift") {
		t.Fatalf("init error = %q, want --force and Project.swift", message)
	}
}

func TestInitCommandForceOverwritesExistingFiles(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	configPath := filepath.Join(repoRoot(t), "testdata", "sample-config.json")

	projectManifestPath := filepath.Join(outputDir, "Project.swift")
	if err := os.WriteFile(projectManifestPath, []byte("// existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", projectManifestPath, err)
	}

	if _, err := executeRootCommand("init", "--config", configPath, "--output", outputDir, "--force"); err != nil {
		t.Fatalf("executeRootCommand(init --force) error = %v", err)
	}

	projectManifest := readFileString(t, projectManifestPath)
	if strings.Contains(projectManifest, "// existing") {
		t.Fatalf("Project.swift should be overwritten:\n%s", projectManifest)
	}
	if !strings.Contains(projectManifest, "com.example.demo") {
		t.Fatalf("Project.swift missing rendered content:\n%s", projectManifest)
	}
}

func TestInitCommandWithoutConfigReturnsDefaultConfigError(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()

	_, err := executeRootCommand("init", "--output", outputDir)
	if err == nil {
		t.Fatal("executeRootCommand(init) error = nil, want missing config error")
	}

	message := err.Error()
	if !strings.Contains(message, "load config:") {
		t.Fatalf("init error = %q, want load config prefix", message)
	}
	if !strings.Contains(message, "ios-app-manager.json") {
		t.Fatalf("init error = %q, want default config file name", message)
	}
}

func TestInitCommandWithInvalidConfigReturnsValidationErrors(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	invalidConfigPath := filepath.Join(repoRoot(t), "testdata", "invalid-config.json")

	_, err := executeRootCommand("init", "--config", invalidConfigPath, "--output", outputDir)
	if err == nil {
		t.Fatal("executeRootCommand(init --config invalid-config.json) error = nil, want validation error")
	}

	message := err.Error()
	requiredSnippets := []string{
		"load config:",
		"validate config file",
		"AppName is required",
		"BundleID must use reverse-domain format",
		"SwiftVersion must use major.minor format",
		"MinTarget must use major.minor format",
		"ProjectVersion is required",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(message, snippet) {
			t.Fatalf("init error missing %q:\n%s", snippet, message)
		}
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("failed to find repo root from %q", dir)
		}
		dir = parent
	}
}
