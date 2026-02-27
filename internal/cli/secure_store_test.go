package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestSecureStoreSetupIntegration(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	// Create minimal project scaffold for config loading.
	writeProjectScaffold(t, projectRoot, cfg)

	output, err := executeRootCommand("--config", configPath, "secure-store", "setup")
	if err != nil {
		t.Fatalf("executeRootCommand(secure-store setup) error = %v", err)
	}

	if !strings.Contains(output, "SecureStore setup complete") {
		t.Fatalf("output = %q, want setup confirmation", output)
	}

	// Verify interface package files.
	interfaceDir := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources", "SecureStore")

	namespacePath := filepath.Join(interfaceDir, "SecureStore.swift")
	requireFileExists(t, namespacePath)
	namespaceContent := readTestFile(t, namespacePath)
	if !strings.Contains(namespaceContent, "public enum SecureStore") {
		t.Fatalf("SecureStore.swift missing namespace:\n%s", namespaceContent)
	}

	modulePath := filepath.Join(interfaceDir, "Module", "SecureStore.Module.swift")
	requireFileExists(t, modulePath)

	interfacePath := filepath.Join(interfaceDir, "Module", "SecureStore.Module+Interface.swift")
	requireFileExists(t, interfacePath)
	interfaceContent := readTestFile(t, interfacePath)
	for _, expected := range []string{
		"protocol Interface: Sendable",
		"func save(key: String, data: Data) throws",
		"func load(key: String) throws -> Data?",
		"func delete(key: String) throws",
		"func clear() throws",
		"SecureStoring",
	} {
		if !strings.Contains(interfaceContent, expected) {
			t.Fatalf("Interface missing %q:\n%s", expected, interfaceContent)
		}
	}

	// Verify impl package files.
	implPath := filepath.Join(projectRoot, "Packages", "SecureStoreImpl", "Sources", "SecureStoreImpl", "Module", "SecureStore.Module+Impl.swift")
	requireFileExists(t, implPath)
	implContent := readTestFile(t, implPath)
	for _, expected := range []string{
		"import SecureStore",
		"import Security",
		"public actor Impl: SecureStoring",
		"kSecClassGenericPassword",
	} {
		if !strings.Contains(implContent, expected) {
			t.Fatalf("Impl missing %q:\n%s", expected, implContent)
		}
	}
}

func TestSecureStoreSetupIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	if _, err := executeRootCommand("--config", configPath, "secure-store", "setup"); err != nil {
		t.Fatalf("first secure-store setup error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "secure-store", "setup"); err != nil {
		t.Fatalf("second secure-store setup error = %v", err)
	}

	// Files should still exist after second run.
	namespacePath := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources", "SecureStore", "SecureStore.swift")
	requireFileExists(t, namespacePath)
}

func TestSecureStoreHelpShowsSubcommands(t *testing.T) {
	output, err := executeRootCommand("secure-store", "--help")
	if err != nil {
		t.Fatalf("executeRootCommand(secure-store --help) error = %v", err)
	}

	if !strings.Contains(output, "setup") {
		t.Fatalf("secure-store help output missing 'setup':\n%s", output)
	}
}

func TestSecureStoreSetupNoConfig(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()

	// No config file exists.
	configPath := filepath.Join(projectRoot, config.DefaultConfigPath)

	_, err := executeRootCommand("--config", configPath, "secure-store", "setup")
	if err == nil {
		t.Fatal("expected error when config missing, got nil")
	}
}

func TestSecureStoreIoCDiscovery(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// Create SecureStore module via secure-store setup.
	if _, err := executeRootCommand("--config", configPath, "secure-store", "setup"); err != nil {
		t.Fatalf("secure-store setup error = %v", err)
	}

	// Create Package.swift files for SecureStore and SecureStoreImpl so IoC can discover them.
	for _, pkg := range []string{"SecureStore", "SecureStoreImpl"} {
		pkgDir := filepath.Join(projectRoot, "Packages", pkg)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", pkgDir, err)
		}
	}

	// Run IoC setup — it should discover SecureStore/SecureStoreImpl pair.
	output, err := executeRootCommand("--config", configPath, "ioc", "setup")
	if err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}

	if !strings.Contains(output, "SwiftIoC setup complete") {
		t.Fatalf("output = %q, want setup confirmation", output)
	}

	// Verify Registry.swift contains SecureStore registration.
	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App", cfg.AppName+".Registry.swift")
	requireFileExists(t, registryPath)

	registryContent := readTestFile(t, registryPath)
	for _, expected := range []string{
		"import SecureStore",
		"import SecureStoreImpl",
		"SecureStore.Module.Interface.self",
		"SecureStore.Module.Impl()",
	} {
		if !strings.Contains(registryContent, expected) {
			t.Fatalf("Registry.swift missing %q:\n%s", expected, registryContent)
		}
	}
}
