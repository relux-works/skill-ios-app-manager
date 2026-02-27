package cli

import (
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
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}

	// SecureStore setup — required for AppConfig.
	if _, err := executeRootCommand("--config", configPath, "secure-store", "setup", "--access-group", "group.com.example.demo"); err != nil {
		t.Fatalf("secure-store setup error = %v", err)
	}

	// Re-run IoC setup to regenerate Registry with SecureStore.
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("ioc setup (re-run) error = %v", err)
	}

	output, err := executeRootCommand("--config", configPath, "app-config", "setup")
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
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "secure-store", "setup", "--access-group", "group.com.example.demo"); err != nil {
		t.Fatalf("secure-store setup error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("ioc setup (re-run) error = %v", err)
	}

	// First run.
	if _, err := executeRootCommand("--config", configPath, "app-config", "setup"); err != nil {
		t.Fatalf("first app-config setup error = %v", err)
	}

	// Second run should be idempotent.
	if _, err := executeRootCommand("--config", configPath, "app-config", "setup"); err != nil {
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

func TestAppConfigSetupMissingIoC(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// Don't run IoC setup — Registry.swift won't exist.
	_, err := executeRootCommand("--config", configPath, "app-config", "setup")
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
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}

	// No secure-store setup — Registry won't have SecureStoring.
	_, err := executeRootCommand("--config", configPath, "app-config", "setup")
	if err == nil {
		t.Fatal("expected error when SecureStore missing, got nil")
	}

	if !strings.Contains(err.Error(), "secure-store setup") {
		t.Fatalf("error = %q, want guidance about secure-store setup", err.Error())
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

	_, err := executeRootCommand("--config", configPath, "app-config", "setup")
	if err == nil {
		t.Fatal("expected error when config missing, got nil")
	}
}
