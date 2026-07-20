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
	cfg.RuntimeProfiles.TestAction = config.RuntimeProfileTestActionConfig{
		Targets:         []string{"DemoAppTests", "DemoAppUITests"},
		LaunchArguments: []string{"--demo-hosted-tests"},
	}
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

func TestMatureProjectRuntimeAdoptionPreservesCustomCompositionAndConverges(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "runtime-profiles-config.json")
	cfg, err := config.LoadConfig(fixturePath)
	if err != nil {
		t.Fatalf("config.LoadConfig(%q) error = %v", fixturePath, err)
	}
	cfg.AppName = "DemoApp"
	cfg.ProductName = "DemoApp"
	cfg.BundleID = "com.example.demo"
	cfg.AppGroups = []string{"group.com.example.demo"}
	cfg.RuntimeProfiles.TestAction = config.RuntimeProfileTestActionConfig{
		Targets:         []string{"DemoAppTests", "DemoAppUITests"},
		LaunchArguments: []string{"--demo-hosted-tests"},
	}
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
	writeTestFile(t, filepath.Join(projectRoot, "Project.swift"), readMatureRuntimeFixture(t, "Project.swift"))
	writeTestFile(t, filepath.Join(projectRoot, "Package.swift"), readMatureRuntimeFixture(t, "Package.swift"))
	registryPath := filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "App", cfg.AppName+".Registry.swift")
	writeTestFile(t, registryPath, readMatureRuntimeFixture(t, "Registry.swift"))

	if _, err := executeRootCommand(
		"--config", configPath,
		"secure-store", "setup",
		"--access-group", "group.com.example.demo",
		"--yes",
	); err != nil {
		t.Fatalf("secure-store setup against mature project error = %v", err)
	}
	output, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes")
	if err != nil {
		t.Fatalf("app-config setup against mature project error = %v", err)
	}
	for _, forbidden := range []string{
		"public-client-validation-key",
		filepath.Join(projectRoot, ".firebase-inputs"),
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("app-config output leaked validation material %q: %s", forbidden, output)
		}
	}

	projectManifestPath := filepath.Join(projectRoot, "Project.swift")
	packageManifestPath := filepath.Join(projectRoot, "Package.swift")
	runtimeHelperPath := filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers", "RuntimeProfiles.swift")
	projectManifest := readTestFile(t, projectManifestPath)
	packageManifest := readTestFile(t, packageManifestPath)
	runtimeHelper := readTestFile(t, runtimeHelperPath)
	registry := readTestFile(t, registryPath)

	for _, orphaned := range []string{
		`name: "Staging"`,
		"SWIFT_ACTIVE_COMPILATION_CONDITIONS",
	} {
		if strings.Contains(projectManifest, orphaned) {
			t.Fatalf("Project.swift retained orphaned legacy configuration %q:\n%s", orphaned, projectManifest)
		}
	}
	if count := strings.Count(projectManifest, "let configurations: [Configuration]"); count != 1 {
		t.Fatalf("Project.swift configuration declaration count = %d, want 1:\n%s", count, projectManifest)
	}
	if !strings.Contains(
		projectManifest,
		".external(name: \"MatureFeature\"),\n                .external(name: \"SharedConfig\"),",
	) {
		t.Fatalf("Project.swift did not terminate the mature UI-test dependency before SharedConfig:\n%s", projectManifest)
	}
	if strings.Contains(projectManifest, "configuration: .debug") {
		t.Fatalf("Project.swift retained the legacy app scheme after runtime-profile adoption:\n%s", projectManifest)
	}
	for _, want := range []string{
		`.testableTarget(target: .target("DemoAppTests"))`,
		`.testableTarget(target: .target("DemoAppUITests"))`,
		`.launchArgument(name: "--demo-hosted-tests", isEnabled: true)`,
	} {
		if !strings.Contains(runtimeHelper, want) {
			t.Fatalf("RuntimeProfiles.swift missing mature test action value %q:\n%s", want, runtimeHelper)
		}
	}
	if strings.Contains(runtimeHelper, ".targets([]") {
		t.Fatalf("RuntimeProfiles.swift retained an empty test action:\n%s", runtimeHelper)
	}

	baseSettingsLine := ""
	for _, line := range strings.Split(packageManifest, "\n") {
		if strings.Contains(line, "baseSettings:") {
			baseSettingsLine = line
			break
		}
	}
	if count := strings.Count(baseSettingsLine, "configurations:"); count != 1 {
		t.Fatalf("Package.swift baseSettings configurations count = %d, want 1:\n%s", count, baseSettingsLine)
	}
	if !strings.Contains(baseSettingsLine, "RuntimeProfilesProjectDescription.configurations") {
		t.Fatalf("Package.swift baseSettings did not adopt runtime configurations:\n%s", packageManifest)
	}

	for _, preserved := range []string{
		"import MatureFeature",
		"private(set) static var runtimeMode",
		"MatureFeature.Persistence.self",
		"MatureFeature.APIClient.self",
		"preconditionFailure(\"Unregistered custom dependency\")",
		"func buildPersistence()",
		"func buildAPIClient()",
	} {
		if !strings.Contains(registry, preserved) {
			t.Fatalf("Registry.swift lost mature custom composition %q:\n%s", preserved, registry)
		}
	}
	for _, integrated := range []string{
		"SecureStore.Module.Interface.self",
		"func buildSecureStore()",
		"IApiConfigManager.self",
		"func buildAppConfigManager()",
	} {
		if !strings.Contains(registry, integrated) {
			t.Fatalf("Registry.swift missing supported integration %q:\n%s", integrated, registry)
		}
	}

	trackedPaths := []string{
		projectManifestPath,
		packageManifestPath,
		runtimeHelperPath,
		registryPath,
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "AppConfig", "AppConfig.Manager.swift"),
		filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Configuration", "Runtime", "RuntimeProfiles.swift"),
	}
	first := make(map[string]string, len(trackedPaths))
	for _, path := range trackedPaths {
		first[path] = readTestFile(t, path)
	}

	if _, err := executeRootCommand(
		"--config", configPath,
		"secure-store", "setup",
		"--access-group", "group.com.example.demo",
		"--yes",
	); err != nil {
		t.Fatalf("second secure-store setup against mature project error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "app-config", "setup", "--yes"); err != nil {
		t.Fatalf("second app-config setup against mature project error = %v", err)
	}
	for _, path := range trackedPaths {
		if got := readTestFile(t, path); got != first[path] {
			t.Fatalf("second mature-project adoption changed %s:\n%s", path, got)
		}
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

func readMatureRuntimeFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "mature-runtime", name)
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mature runtime fixture %q: %v", path, err)
	}
	return string(payload)
}
