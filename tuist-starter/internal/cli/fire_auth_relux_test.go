package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

func TestFireAuthReluxSetupCommandConvergesMatureProject(t *testing.T) {
	fixtureConfig := filepath.Join("..", "..", "testdata", "runtime-profiles-config.json")
	cfg, err := config.LoadConfig(fixtureConfig)
	if err != nil {
		t.Fatalf("LoadConfig(%s) error = %v", fixtureConfig, err)
	}
	cfg.AppName = "MatureApp"
	cfg.ProductName = "MatureApp"
	cfg.BundleID = "com.example.mature"
	cfg.RuntimeProfiles.TestAction = config.RuntimeProfileTestActionConfig{
		Targets:         []string{"MatureAppTests", "MatureAppUITests"},
		LaunchArguments: []string{"--mature-hosted-tests"},
	}
	for environment, descriptor := range cfg.RuntimeProfiles.BackendEnvironments {
		if descriptor.Firebase != nil {
			descriptor.Firebase.BundleID = cfg.BundleID
		}
		cfg.RuntimeProfiles.BackendEnvironments[environment] = descriptor
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("mature FireAuth config validation error = %v", err)
	}

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, cfg)
	for _, name := range []string{"Package.swift", "Project.swift", "App.swift"} {
		destination := filepath.Join(projectRoot, name)
		if name == "App.swift" {
			destination = filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", name)
		}
		writeTestFile(t, destination, readMatureFireAuthFixture(t, name))
	}
	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App", cfg.AppName+".Registry.swift")
	writeTestFile(t, registryPath, readMatureFireAuthFixture(t, "Registry.swift"))
	writeTestFile(t, filepath.Join(
		projectRoot,
		"Targets", cfg.AppName, "Sources", "Configuration", "Runtime", "RuntimeProfiles.swift",
	), scaffold.GenerateRuntimeProfilesSwift(cfg))
	configureFireAuthCLIInputs(t, cfg, "protected-cli-api-key")

	dryRunBefore := readTestFile(t, filepath.Join(projectRoot, "Package.swift"))
	dryRunOutput, err := executeRootCommand("--config", configPath, "fireauth-relux", "setup", "--dry-run")
	if err != nil {
		t.Fatalf("fireauth-relux setup --dry-run error = %v", err)
	}
	if !strings.Contains(dryRunOutput, "FireAuthRelux Setup Plan") {
		t.Fatalf("dry-run output missing plan:\n%s", dryRunOutput)
	}
	if !strings.Contains(dryRunOutput, "EnvironmentScopedFireAuthSessionStore.swift") {
		t.Fatalf("dry-run output missing durable session-store output:\n%s", dryRunOutput)
	}
	if got := readTestFile(t, filepath.Join(projectRoot, "Package.swift")); got != dryRunBefore {
		t.Fatalf("dry-run mutated Package.swift:\n%s", got)
	}

	output, err := executeRootCommand("--config", configPath, "fireauth-relux", "setup", "--yes")
	if err != nil {
		t.Fatalf("fireauth-relux setup error = %v", err)
	}
	if !strings.Contains(output, "FireAuthRelux setup complete") {
		t.Fatalf("setup output missing confirmation:\n%s", output)
	}
	for _, forbidden := range []string{"protected-cli-api-key", os.TempDir()} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("setup output leaked protected input %q: %s", forbidden, output)
		}
	}

	trackedPaths := []string{
		filepath.Join(projectRoot, "Package.swift"),
		filepath.Join(projectRoot, "Project.swift"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App.swift"),
		registryPath,
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "FireAuth", "GeneratedFireAuthRelux.swift"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "FireAuth", "GeneratedFireAuthReluxProcess.swift"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "FireAuth", "EnvironmentScopedFireAuthSessionStore.swift"),
	}
	for _, target := range cfg.RuntimeProfiles.TestAction.Targets {
		trackedPaths = append(trackedPaths, filepath.Join(
			projectRoot,
			"Targets", target, "Sources", "Support", "GeneratedFireAuthReluxTestLaunch.swift",
		))
	}
	first := make(map[string]string, len(trackedPaths))
	for _, path := range trackedPaths {
		first[path] = readTestFile(t, path)
	}
	if _, err := executeRootCommand("--config", configPath, "fireauth-relux", "setup", "--yes"); err != nil {
		t.Fatalf("second fireauth-relux setup error = %v", err)
	}
	for _, path := range trackedPaths {
		if got := readTestFile(t, path); got != first[path] {
			t.Fatalf("second FireAuthRelux setup changed %s:\n%s", path, got)
		}
	}
}

func TestFireAuthReluxHelpShowsSetup(t *testing.T) {
	output, err := executeRootCommand("fireauth-relux", "--help")
	if err != nil {
		t.Fatalf("fireauth-relux --help error = %v", err)
	}
	if !strings.Contains(output, "setup") {
		t.Fatalf("fireauth-relux help missing setup:\n%s", output)
	}
}

func configureFireAuthCLIInputs(t *testing.T, cfg config.ProjectConfig, apiKey string) {
	t.Helper()
	inputRoot := t.TempDir()
	written := make(map[string]struct{})
	for _, environment := range cfg.OrderedBackendEnvironments() {
		descriptor := cfg.RuntimeProfiles.BackendEnvironments[environment]
		if descriptor.Firebase == nil {
			continue
		}
		firebase := descriptor.Firebase
		if _, exists := written[firebase.ValidationInputEnvironmentVar]; exists {
			continue
		}
		written[firebase.ValidationInputEnvironmentVar] = struct{}{}
		path := filepath.Join(inputRoot, string(environment)+".plist")
		payload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>PROJECT_ID</key><string>%s</string>
<key>GOOGLE_APP_ID</key><string>%s</string>
<key>BUNDLE_ID</key><string>%s</string>
<key>API_KEY</key><string>%s</string>
</dict></plist>
`, firebase.ProjectID, firebase.GoogleAppID, firebase.BundleID, apiKey)
		if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
		t.Setenv(firebase.ValidationInputEnvironmentVar, path)
	}
}

func readMatureFireAuthFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "mature-fireauth", name)
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(payload)
}
