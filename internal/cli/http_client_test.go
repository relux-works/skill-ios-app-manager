package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestHttpClientSetupIntegration(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// IoC setup first — creates Registry.swift with anchors.
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}

	output, err := executeRootCommand("--config", configPath, "http-client", "setup")
	if err != nil {
		t.Fatalf("http-client setup error = %v", err)
	}

	if !strings.Contains(output, "HttpClient setup complete") {
		t.Fatalf("output = %q, want setup confirmation", output)
	}

	// Verify Configuration+HttpClient.swift.
	configFilePath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "Configuration+HttpClient.swift")
	requireFileExists(t, configFilePath)
	configContent := readTestFile(t, configFilePath)
	for _, expected := range []string{
		"Configuration",
		"HttpClient",
		"timeoutForResponse",
		"timeoutResourceInterval",
	} {
		if !strings.Contains(configContent, expected) {
			t.Fatalf("Configuration+HttpClient.swift missing %q:\n%s", expected, configContent)
		}
	}

	// Verify Registry.swift patching.
	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App", cfg.AppName+".Registry.swift")
	requireFileExists(t, registryPath)
	registryContent := readTestFile(t, registryPath)

	for _, expected := range []string{
		"import HttpClient",
		"IRpcAsyncClient.self",
		"buildHttpClient",
		"Configuration.HttpClient.timeoutForResponse",
		"Configuration.HttpClient.timeoutResourceInterval",
		"RpcClient(",
	} {
		if !strings.Contains(registryContent, expected) {
			t.Fatalf("Registry.swift missing %q:\n%s", expected, registryContent)
		}
	}

	// Verify external dep in Package.swift.
	packageContent := readTestFile(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(packageContent, "swift-httpclient") {
		t.Fatalf("Package.swift missing swift-httpclient:\n%s", packageContent)
	}

	// Verify Project.swift dependency.
	projectContent := readTestFile(t, filepath.Join(projectRoot, "Project.swift"))
	if !strings.Contains(projectContent, "HttpClient") {
		t.Fatalf("Project.swift missing HttpClient:\n%s", projectContent)
	}
}

func TestHttpClientSetupIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// IoC setup first.
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}

	if _, err := executeRootCommand("--config", configPath, "http-client", "setup"); err != nil {
		t.Fatalf("first http-client setup error = %v", err)
	}

	// Second run should be idempotent.
	if _, err := executeRootCommand("--config", configPath, "http-client", "setup"); err != nil {
		t.Fatalf("second http-client setup error = %v", err)
	}

	// Registration should appear exactly once.
	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App", cfg.AppName+".Registry.swift")
	registryContent := readTestFile(t, registryPath)
	count := strings.Count(registryContent, "IRpcAsyncClient.self")
	if count != 1 {
		t.Fatalf("IRpcAsyncClient.self appears %d times, want 1:\n%s", count, registryContent)
	}
}

func TestHttpClientHelpShowsSubcommands(t *testing.T) {
	output, err := executeRootCommand("http-client", "--help")
	if err != nil {
		t.Fatalf("http-client --help error = %v", err)
	}

	if !strings.Contains(output, "setup") {
		t.Fatalf("http-client help missing 'setup':\n%s", output)
	}
}

func TestHttpClientSetupNoConfig(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := filepath.Join(projectRoot, config.DefaultConfigPath)

	_, err := executeRootCommand("--config", configPath, "http-client", "setup")
	if err == nil {
		t.Fatal("expected error when config missing, got nil")
	}
}

func TestHttpClientSetupWithoutIoC(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// Don't run IoC setup — Registry.swift won't exist.
	_, err := executeRootCommand("--config", configPath, "http-client", "setup")
	if err == nil {
		t.Fatal("expected error when Registry.swift missing, got nil")
	}
}

func TestHttpClientSetupRegistryNetworkSection(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// IoC setup creates Registry with empty network section.
	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}

	if _, err := executeRootCommand("--config", configPath, "http-client", "setup"); err != nil {
		t.Fatalf("http-client setup error = %v", err)
	}

	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App", cfg.AppName+".Registry.swift")
	content := readTestFile(t, registryPath)

	// Registration should be in the Network section.
	networkIdx := strings.Index(content, "// MARK: - Network (scaffolding anchor: network)")
	utilsIdx := strings.Index(content, "// MARK: - Utils (scaffolding anchor: utils)")
	rpcIdx := strings.Index(content, "IRpcAsyncClient.self")

	if networkIdx < 0 || utilsIdx < 0 || rpcIdx < 0 {
		t.Fatalf("missing anchors or registration in Registry.swift:\n%s", content)
	}

	if rpcIdx < networkIdx || rpcIdx > utilsIdx {
		t.Fatalf("IRpcAsyncClient registration not in Network section (network=%d, rpc=%d, utils=%d):\n%s",
			networkIdx, rpcIdx, utilsIdx, content)
	}

	// Builder should be in Network Builders section.
	networkBuildersIdx := strings.Index(content, "// MARK: - Network Builders (scaffolding anchor: network-builders)")
	utilsBuildersIdx := strings.Index(content, "// MARK: - Utils Builders (scaffolding anchor: utils-builders)")
	builderIdx := strings.Index(content, "func buildHttpClient()")

	if networkBuildersIdx < 0 || utilsBuildersIdx < 0 || builderIdx < 0 {
		t.Fatalf("missing builder anchors or builder in Registry.swift:\n%s", content)
	}

	if builderIdx < networkBuildersIdx || builderIdx > utilsBuildersIdx {
		t.Fatalf("buildHttpClient not in Network Builders section:\n%s", content)
	}
}

func TestHttpClientSetupE2EWithModules(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	configPath := writeModuleConfig(t, projectRoot, cfg)

	writeProjectScaffold(t, projectRoot, cfg)

	// Full pipeline: module create → IoC setup → http-client setup.
	if _, err := executeRootCommand("--config", configPath, "module", "create", "Auth", "--type", "feature"); err != nil {
		t.Fatalf("module create error = %v", err)
	}

	if _, err := executeRootCommand("--config", configPath, "ioc", "setup"); err != nil {
		t.Fatalf("ioc setup error = %v", err)
	}

	if _, err := executeRootCommand("--config", configPath, "http-client", "setup"); err != nil {
		t.Fatalf("http-client setup error = %v", err)
	}

	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App", cfg.AppName+".Registry.swift")
	content := readTestFile(t, registryPath)

	// Should have both module registration and HttpClient.
	for _, expected := range []string{
		"Auth.Module.Interface.self",
		"IRpcAsyncClient.self",
		"import HttpClient",
		"import Auth",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("Registry.swift missing %q:\n%s", expected, content)
		}
	}

	// Verify Configuration+HttpClient exists.
	configFilePath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "Configuration+HttpClient.swift")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		t.Fatalf("Configuration+HttpClient.swift not found at %q", configFilePath)
	}
}
