package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestAppConfigSetupIntegration(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// IoC setup first — creates Registry.swift with anchors.
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup", "--yes"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}

	// SecureStore setup — required for AppConfig.
	if _, err := executeRootCommand("--config", configPath, "secure-store", "setup", "--access-group", "group.com.example.demo", "--yes"); err != nil {
		t.Fatalf("secure-store setup error = %v", err)
	}

	// Re-run IoC setup to regenerate Registry with SecureStore.
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup", "--yes"); err != nil {
		t.Fatalf("ioc setup (re-run) error = %v", err)
	}

	output, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes")
	if err != nil {
		t.Fatalf("app-config setup error = %v", err)
	}

	if !strings.Contains(output, "AppConfig setup complete") {
		t.Fatalf("output = %q, want setup confirmation", output)
	}

	// Verify all 8 AppConfig files.
	appConfigDir := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "AppConfig")
	expectedFiles := []string{
		"AppConfig.swift",
		"AppConfig.Env.swift",
		"AppConfig.Env+Configuration.swift",
		"AppConfig.Env+Configuration+Presets.swift",
		"AppConfig.Manager+Protocols.swift",
		"AppConfig.Manager.swift",
		"AppConfig.ApiConfigurator.swift",
		"AppConfig.UrlComponents.swift",
	}
	for _, name := range expectedFiles {
		requireFileExists(t, filepath.Join(appConfigDir, name))
	}

	// Verify key content in generated files.
	namespaceContent := readTestFile(t, filepath.Join(appConfigDir, "AppConfig.swift"))
	if !strings.Contains(namespaceContent, "enum AppConfig") {
		t.Fatalf("AppConfig.swift missing namespace enum:\n%s", namespaceContent)
	}

	managerContent := readTestFile(t, filepath.Join(appConfigDir, "AppConfig.Manager.swift"))
	for _, expected := range []string{
		"IApiConfigManager",
		"SecureStoring",
		"NSLock",
		"resolver()",
	} {
		if !strings.Contains(managerContent, expected) {
			t.Fatalf("AppConfig.Manager.swift missing %q:\n%s", expected, managerContent)
		}
	}

	// Verify Registry.swift patching.
	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App", cfg.AppName+".Registry.swift")
	registryContent := readTestFile(t, registryPath)

	for _, expected := range []string{
		"IApiConfigManager.self",
		"buildAppConfigManager",
		"resolve(SecureStoring.self)",
	} {
		if !strings.Contains(registryContent, expected) {
			t.Fatalf("Registry.swift missing %q:\n%s", expected, registryContent)
		}
	}
}

func TestAppConfigSetupIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// Setup prerequisites.
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup", "--yes"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "secure-store", "setup", "--access-group", "group.com.example.demo", "--yes"); err != nil {
		t.Fatalf("secure-store setup error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup", "--yes"); err != nil {
		t.Fatalf("ioc setup (re-run) error = %v", err)
	}

	// First run.
	if _, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes"); err != nil {
		t.Fatalf("first app-config setup error = %v", err)
	}

	// Second run should be idempotent.
	if _, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes"); err != nil {
		t.Fatalf("second app-config setup error = %v", err)
	}

	// Registration should appear exactly once.
	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App", cfg.AppName+".Registry.swift")
	registryContent := readTestFile(t, registryPath)
	count := strings.Count(registryContent, "IApiConfigManager.self")
	if count != 1 {
		t.Fatalf("IApiConfigManager.self appears %d times, want 1:\n%s", count, registryContent)
	}
}

func TestAppConfigSetupOrchestratesRuntimeProfilesAndRemoval(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "runtime-profiles-config.json")
	cfg, err := config.LoadConfig(fixturePath)
	if err != nil {
		t.Fatalf("config.LoadConfig(%q) error = %v", fixturePath, err)
	}
	cfg.AppName = "DemoApp"
	cfg.ProductName = "DemoApp"
	cfg.BundleID = "com.example.demo"
	cfg.AppGroups = []string{"group.com.example.demo"}
	for environment, descriptor := range cfg.RuntimeProfiles.BackendEnvironments {
		if descriptor.Firebase != nil {
			descriptor.Firebase.BundleID = cfg.BundleID
		}
		cfg.RuntimeProfiles.BackendEnvironments[environment] = descriptor
	}

	projectRoot := t.TempDir()
	configureCLIRuntimeFirebaseInputs(t, projectRoot, cfg)
	configPath := writeModuleConfig(t, projectRoot, cfg)
	writeProjectScaffold(t, projectRoot, cfg)

	if _, err := executeRootCommand("--config", configPath, "ioc", "setup", "--yes"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "secure-store", "setup", "--access-group", "group.com.example.demo", "--yes"); err != nil {
		t.Fatalf("secure-store setup error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup", "--yes"); err != nil {
		t.Fatalf("ioc setup re-run error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes"); err != nil {
		t.Fatalf("runtime app-config setup error = %v", err)
	}

	managerPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "AppConfig", "AppConfig.Manager.swift")
	manager := readTestFile(t, managerPath)
	for _, want := range []string{
		"GeneratedRuntimeProfiles.currentDistributionProfileDescriptor",
		"profileDescriptor.allowedEnvironments.contains(newEnv)",
		"profileDescriptor.selectionPersistence == .enabled",
	} {
		if !strings.Contains(manager, want) {
			t.Fatalf("runtime AppConfig manager missing %q:\n%s", want, manager)
		}
	}
	runtimeSwiftPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "Runtime", "RuntimeProfiles.swift")
	tuistHelperPath := filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers", "RuntimeProfiles.swift")
	requireFileExists(t, runtimeSwiftPath)
	requireFileExists(t, tuistHelperPath)
	projectManifest := readTestFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		"RuntimeProfilesProjectDescription.configurations",
		"RuntimeProfilesProjectDescription.schemes(appName: appName)",
		`"distributionProfile": .string("$(DISTRIBUTION_PROFILE)")`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("runtime Project.swift missing %q:\n%s", want, projectManifest)
		}
	}

	if _, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes"); err != nil {
		t.Fatalf("second runtime app-config setup error = %v", err)
	}
	if got := readTestFile(t, managerPath); got != manager {
		t.Fatal("second runtime app-config setup did not converge")
	}

	cfg.RuntimeProfiles = nil
	if err := config.WriteProjectConfig(configPath, cfg); err != nil {
		t.Fatalf("config.WriteProjectConfig() removal error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes"); err != nil {
		t.Fatalf("legacy app-config restoration error = %v", err)
	}
	if _, err := os.Stat(runtimeSwiftPath); !os.IsNotExist(err) {
		t.Fatalf("RuntimeProfiles.swift remains after removal: %v", err)
	}
	if _, err := os.Stat(tuistHelperPath); !os.IsNotExist(err) {
		t.Fatalf("Tuist RuntimeProfiles.swift remains after removal: %v", err)
	}
	restoredManager := readTestFile(t, managerPath)
	if !strings.Contains(restoredManager, "return .prod") || strings.Contains(restoredManager, "GeneratedRuntimeProfiles") {
		t.Fatalf("AppConfig manager was not restored to backward-compatible mode:\n%s", restoredManager)
	}
}

func TestAppConfigSetupMissingIoC(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// Don't run IoC setup — Registry.swift won't exist.
	_, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes")
	if err == nil {
		t.Fatal("expected error when Registry.swift missing, got nil")
	}

	if !strings.Contains(err.Error(), "ioc setup") {
		t.Fatalf("error = %q, want guidance about ioc setup", err.Error())
	}
}

func TestAppConfigSetupMissingSecureStore(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// IoC setup creates Registry without SecureStore.
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup", "--yes"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}

	// No secure-store setup — Registry won't have SecureStoring.
	_, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes")
	if err == nil {
		t.Fatal("expected error when SecureStore missing, got nil")
	}

	if !strings.Contains(err.Error(), "SecureStore") {
		t.Fatalf("error = %q, want missing SecureStore dependency message", err.Error())
	}
}

func TestAppConfigHelpShowsSubcommands(t *testing.T) {
	output, err := executeRootCommand("app-config", "--help")
	if err != nil {
		t.Fatalf("app-config --help error = %v", err)
	}

	if !strings.Contains(output, "setup") {
		t.Fatalf("app-config help missing 'setup':\n%s", output)
	}
}

func TestAppConfigSetupNoConfig(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := filepath.Join(projectRoot, config.DefaultConfigPath)

	_, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes")
	if err == nil {
		t.Fatal("expected error when config missing, got nil")
	}
}

func configureCLIRuntimeFirebaseInputs(t *testing.T, projectRoot string, cfg config.ProjectConfig) {
	t.Helper()
	inputDir := filepath.Join(projectRoot, ".firebase-inputs")
	if err := os.MkdirAll(inputDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for environment, descriptor := range cfg.RuntimeProfiles.BackendEnvironments {
		if descriptor.Firebase == nil {
			continue
		}
		path := filepath.Join(inputDir, string(environment)+".plist")
		payload := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>PROJECT_ID</key><string>%s</string>
<key>GOOGLE_APP_ID</key><string>%s</string>
<key>BUNDLE_ID</key><string>%s</string>
<key>API_KEY</key><string>public-client-validation-key</string>
</dict></plist>
`, descriptor.Firebase.ProjectID, descriptor.Firebase.GoogleAppID, descriptor.Firebase.BundleID)
		if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv(descriptor.Firebase.ValidationInputEnvironmentVar, path)
	}
}
