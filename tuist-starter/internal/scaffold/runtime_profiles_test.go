package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	templaterenderer "github.com/relux-works/ios-app-manager/internal/template"
	"github.com/relux-works/ios-app-manager/internal/testutil"
)

func TestRuntimeProfilesGeneratorAndSubpluginsRegistered(t *testing.T) {
	t.Parallel()

	var generator *GeneratorPlugin
	for _, candidate := range AllGenerators() {
		if candidate.Name == "runtime-profiles" {
			generator = candidate
			break
		}
	}
	if generator == nil {
		t.Fatal("runtime-profiles generator is not registered")
	}
	if want := []string{"init", "application-configuration"}; !reflect.DeepEqual(generator.Dependencies, want) {
		t.Fatalf("runtime-profiles dependencies = %#v, want %#v", generator.Dependencies, want)
	}

	plugins, err := runtimeProfilePluginsInDependencyOrder()
	if err != nil {
		t.Fatalf("runtimeProfilePluginsInDependencyOrder() error = %v", err)
	}
	got := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		got = append(got, plugin.Name)
	}
	want := []string{
		"runtime-profile-schema",
		"firebase-client-inputs",
		"runtime-descriptors",
		"tuist-project",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runtime profile plugin order = %#v, want %#v", got, want)
	}
}

func TestGenerateRuntimeProfilesGoldenOutput(t *testing.T) {
	t.Parallel()

	cfg := loadRuntimeProfilesFixture(t)
	testutil.AssertGoldenFile(t, "runtimeprofiles/runtime-profiles-swift", GenerateRuntimeProfilesSwift(cfg))
	testutil.AssertGoldenFile(t, "runtimeprofiles/runtime-profiles-project-description-swift", GenerateRuntimeProfilesProjectDescriptionSwift(cfg))
}

func TestGenerateRuntimeProfilesIsTypedDeterministicAndUsesExactOrigins(t *testing.T) {
	t.Parallel()

	cfg := loadRuntimeProfilesFixture(t)
	first := GenerateRuntimeProfilesSwift(cfg)
	second := GenerateRuntimeProfilesSwift(cfg)
	if first != second {
		t.Fatal("GenerateRuntimeProfilesSwift() is not deterministic")
	}
	for _, want := range []string{
		"public enum DistributionProfile: String",
		"public enum BackendEnvironment: String",
		"import SharedConfig",
		"case pilotTestFlight",
		"case `internal`",
		"case production",
		`apiOrigin: URL(string: "https://api.example.com")!`,
		"Configuration.ApplicationConfiguration.current.distributionProfile",
	} {
		if !strings.Contains(first, want) {
			t.Fatalf("RuntimeProfiles.swift missing %q:\n%s", want, first)
		}
	}
	for _, forbidden := range []string{"/api/v1", "apiKey", "API_KEY", "validation_input_environment_variable"} {
		if strings.Contains(first, forbidden) {
			t.Fatalf("RuntimeProfiles.swift contains forbidden generated value %q:\n%s", forbidden, first)
		}
	}
}

func TestSyncRuntimeProfilePackageManifestRepairsLegacyGeneratedArgumentOrder(t *testing.T) {
	t.Parallel()

	legacy := `// swift-tools-version: 6.2
import PackageDescription
#if TUIST
import ProjectDescription
import ProjectDescriptionHelpers // ios-app-manager:runtime-profiles

let packageSettings = PackageSettings(
    baseSettings: .settings(configurations: RuntimeProfilesProjectDescription.configurations), // ios-app-manager:runtime-profile-package-configurations
    targetSettings: [
        "SharedConfig": swiftPackageTargetSettings,
    ],
    productTypes: [
        "SharedConfig": .framework,
    ],
)
#endif
`

	got, err := syncRuntimeProfilePackageManifestContent(legacy, config.ProjectConfig{}, true)
	if err != nil {
		t.Fatalf("syncRuntimeProfilePackageManifestContent() error = %v", err)
	}
	productTypesIndex := strings.Index(got, "    productTypes:")
	baseSettingsIndex := strings.Index(got, "    baseSettings:")
	targetSettingsIndex := strings.Index(got, "    targetSettings:")
	if productTypesIndex < 0 || baseSettingsIndex < 0 || targetSettingsIndex < 0 ||
		!(productTypesIndex < baseSettingsIndex && baseSettingsIndex < targetSettingsIndex) {
		t.Fatalf("PackageSettings arguments are not repaired into Tuist initializer order:\n%s", got)
	}

	converged, err := syncRuntimeProfilePackageManifestContent(got, config.ProjectConfig{}, true)
	if err != nil {
		t.Fatalf("second syncRuntimeProfilePackageManifestContent() error = %v", err)
	}
	if converged != got {
		t.Fatalf("second Package.swift sync changed converged output:\n%s", converged)
	}
}

func TestScaffoldCreatesRuntimeProfileOutputsOnInitialGeneration(t *testing.T) {
	cfg := loadRuntimeProfilesFixture(t)
	projectRoot := t.TempDir()
	configureRuntimeFirebaseInputs(t, projectRoot, cfg, "public-test-client-key")

	scaffolder := New(templaterenderer.NewRenderer(templaterenderer.WithRootDir(projectRoot)))
	if _, err := scaffolder.Scaffold(cfg, projectRoot, false); err != nil {
		t.Fatalf("Scaffold() runtime project error = %v", err)
	}
	requireFile(t, runtimeProfilesSwiftPath(projectRoot, cfg.AppName))
	requireFile(t, runtimeProfilesProjectDescriptionPath(projectRoot))

	projectManifest := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		runtimeProfileConfigurationsBegin,
		runtimeProfileSchemesBegin,
		`"distributionProfile": .string("$(DISTRIBUTION_PROFILE)")`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("initial runtime Project.swift missing %q:\n%s", want, projectManifest)
		}
	}
	rootPackage := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	for _, want := range []string{
		"import ProjectDescriptionHelpers " + runtimeProfileHelpersImportMarker,
		"configurations: RuntimeProfilesProjectDescription.configurations",
	} {
		if !strings.Contains(rootPackage, want) {
			t.Fatalf("initial runtime Package.swift missing %q:\n%s", want, rootPackage)
		}
	}
	if count := strings.Count(rootPackage, `.package(path: "Packages/SharedConfig")`); count != 1 {
		t.Fatalf("SharedConfig package dependency count = %d, want 1:\n%s", count, rootPackage)
	}
	if _, err := scaffolder.Scaffold(cfg, projectRoot, true); err != nil {
		t.Fatalf("forced Scaffold() runtime rerun error = %v", err)
	}
	if got := readFile(t, filepath.Join(projectRoot, "Project.swift")); got != projectManifest {
		t.Fatalf("forced Scaffold() changed converged Project.swift:\n%s", got)
	}
	if got := readFile(t, filepath.Join(projectRoot, "Package.swift")); got != rootPackage {
		t.Fatalf("forced Scaffold() changed converged Package.swift:\n%s", got)
	}
}

func TestSyncRuntimeProfilesCreateUpdateConvergeAndRemove(t *testing.T) {
	cfg := loadRuntimeProfilesFixture(t)
	projectRoot := t.TempDir()
	configureRuntimeFirebaseInputs(t, projectRoot, cfg, "public-test-client-key")

	legacyCfg := cfg
	legacyCfg.RuntimeProfiles = nil
	scaffolder := New(templaterenderer.NewRenderer(templaterenderer.WithRootDir(projectRoot)))
	if _, err := scaffolder.Scaffold(legacyCfg, projectRoot, false); err != nil {
		t.Fatalf("Scaffold() legacy project error = %v", err)
	}

	applicationResult, err := SyncApplicationConfiguration(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncApplicationConfiguration() creation error = %v", err)
	}
	runtimeResult, err := SyncRuntimeProfiles(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncRuntimeProfiles() creation error = %v", err)
	}
	if len(applicationResult.Updated) == 0 || len(runtimeResult.Updated) == 0 {
		t.Fatalf("creation updated application=%#v runtime=%#v, want both non-empty", applicationResult.Updated, runtimeResult.Updated)
	}

	runtimeSwiftPath := runtimeProfilesSwiftPath(projectRoot, cfg.AppName)
	tuistHelperPath := runtimeProfilesProjectDescriptionPath(projectRoot)
	requireFile(t, runtimeSwiftPath)
	requireFile(t, tuistHelperPath)
	projectManifest := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		runtimeProfileConfigurationsBegin,
		"RuntimeProfilesProjectDescription.configurations",
		runtimeProfileSchemesBegin,
		"RuntimeProfilesProjectDescription.schemes(appName: appName)",
		`"distributionProfile": .string("$(DISTRIBUTION_PROFILE)")`,
	} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("created Project.swift missing %q:\n%s", want, projectManifest)
		}
	}
	rootPackagePath := filepath.Join(projectRoot, "Package.swift")
	rootPackage := readFile(t, rootPackagePath)
	if !strings.Contains(rootPackage, "configurations: RuntimeProfilesProjectDescription.configurations") {
		t.Fatalf("created Package.swift missing runtime configurations:\n%s", rootPackage)
	}
	strictnessResult, err := SyncPackageStrictness(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncPackageStrictness() with runtime profiles error = %v", err)
	}
	if len(strictnessResult.Updated) == 0 {
		t.Fatal("SyncPackageStrictness() changed no initial package manifests")
	}
	rootPackage = readFile(t, rootPackagePath)
	for _, want := range []string{
		"configurations: RuntimeProfilesProjectDescription.configurations",
		`"SharedConfig": .framework`,
	} {
		if !strings.Contains(rootPackage, want) {
			t.Fatalf("package strictness removed %q:\n%s", want, rootPackage)
		}
	}
	productTypesIndex := strings.Index(rootPackage, "    productTypes:")
	baseSettingsIndex := strings.Index(rootPackage, "    baseSettings:")
	targetSettingsIndex := strings.Index(rootPackage, "    targetSettings:")
	if productTypesIndex < 0 || baseSettingsIndex < 0 || targetSettingsIndex < 0 ||
		!(productTypesIndex < baseSettingsIndex && baseSettingsIndex < targetSettingsIndex) {
		t.Fatalf("PackageSettings arguments are not in Tuist initializer order:\n%s", rootPackage)
	}

	applicationResult, err = SyncApplicationConfiguration(projectRoot, cfg)
	if err != nil {
		t.Fatalf("second SyncApplicationConfiguration() error = %v", err)
	}
	runtimeResult, err = SyncRuntimeProfiles(projectRoot, cfg)
	if err != nil {
		t.Fatalf("second SyncRuntimeProfiles() error = %v", err)
	}
	if len(applicationResult.Updated) != 0 || len(runtimeResult.Updated) != 0 {
		t.Fatalf("second sync updated application=%#v runtime=%#v, want convergence", applicationResult.Updated, runtimeResult.Updated)
	}

	internal := cfg.RuntimeProfiles.DistributionProfiles[config.DistributionProfileInternal]
	internal.AllowedEnvironments = []config.BackendEnvironment{config.BackendEnvironmentStaging}
	cfg.RuntimeProfiles.DistributionProfiles[config.DistributionProfileInternal] = internal
	if err := cfg.Validate(); err != nil {
		t.Fatalf("updated allowlist config error = %v", err)
	}
	runtimeResult, err = SyncRuntimeProfiles(projectRoot, cfg)
	if err != nil {
		t.Fatalf("SyncRuntimeProfiles() allowlist update error = %v", err)
	}
	if len(runtimeResult.Updated) == 0 {
		t.Fatal("allowlist update changed no generated files")
	}
	runtimeSwift := readFile(t, runtimeSwiftPath)
	if !strings.Contains(runtimeSwift, "allowedEnvironments: [.staging]") {
		t.Fatalf("updated RuntimeProfiles.swift missing staging-only allowlist:\n%s", runtimeSwift)
	}
	runtimeResult, err = SyncRuntimeProfiles(projectRoot, cfg)
	if err != nil {
		t.Fatalf("second allowlist SyncRuntimeProfiles() error = %v", err)
	}
	if len(runtimeResult.Updated) != 0 {
		t.Fatalf("second allowlist sync updated %#v, want convergence", runtimeResult.Updated)
	}

	invalidCfg := loadRuntimeProfilesFixture(t)
	invalidInternal := invalidCfg.RuntimeProfiles.DistributionProfiles[config.DistributionProfileInternal]
	invalidInternal.DefaultEnvironment = config.BackendEnvironmentProduction
	invalidCfg.RuntimeProfiles.DistributionProfiles[config.DistributionProfileInternal] = invalidInternal
	if _, err := SyncRuntimeProfiles(projectRoot, invalidCfg); err == nil || !strings.Contains(err.Error(), "must default to staging") {
		t.Fatalf("SyncRuntimeProfiles() changed default error = %v, want policy rejection", err)
	}

	applicationResult, err = SyncApplicationConfiguration(projectRoot, legacyCfg)
	if err != nil {
		t.Fatalf("SyncApplicationConfiguration() removal error = %v", err)
	}
	runtimeResult, err = SyncRuntimeProfiles(projectRoot, legacyCfg)
	if err != nil {
		t.Fatalf("SyncRuntimeProfiles() removal error = %v", err)
	}
	if len(applicationResult.Updated) == 0 || len(runtimeResult.Updated) == 0 {
		t.Fatalf("removal updated application=%#v runtime=%#v, want both non-empty", applicationResult.Updated, runtimeResult.Updated)
	}
	if fileExists(runtimeSwiftPath) || fileExists(tuistHelperPath) {
		t.Fatalf("runtime generated files remain after removal: swift=%v helper=%v", fileExists(runtimeSwiftPath), fileExists(tuistHelperPath))
	}
	projectManifest = readFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, forbidden := range []string{
		"RuntimeProfilesProjectDescription",
		runtimeProfileConfigurationsBegin,
		runtimeProfileSchemesBegin,
		`"distributionProfile"`,
	} {
		if strings.Contains(projectManifest, forbidden) {
			t.Fatalf("removed Project.swift retained %q:\n%s", forbidden, projectManifest)
		}
	}
	for _, want := range []string{`.debug(name: "Debug")`, `.release(name: "Release")`} {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("removed Project.swift missing legacy configuration %q:\n%s", want, projectManifest)
		}
	}
	rootPackage = readFile(t, rootPackagePath)
	for _, forbidden := range []string{runtimeProfilePackageConfigMarker, "RuntimeProfilesProjectDescription.configurations"} {
		if strings.Contains(rootPackage, forbidden) {
			t.Fatalf("removed Package.swift retained %q:\n%s", forbidden, rootPackage)
		}
	}
}

func TestValidateFirebaseClientConfigurationInputsDoesNotLeakInputMaterial(t *testing.T) {
	cfg := loadRuntimeProfilesFixture(t)
	projectRoot := t.TempDir()
	production := cfg.RuntimeProfiles.BackendEnvironments[config.BackendEnvironmentProduction]
	inputPath := filepath.Join(projectRoot, "local-credential-material.plist")
	payload := firebasePlist(
		"different-public-project",
		production.Firebase.GoogleAppID,
		production.Firebase.BundleID,
		"secret-api-key-must-not-appear",
	)
	if err := os.WriteFile(inputPath, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(production.Firebase.ValidationInputEnvironmentVar, inputPath)

	err := ValidateFirebaseClientConfigurationInputs(projectRoot, cfg)
	if err == nil || !strings.Contains(err.Error(), "PROJECT_ID does not match") {
		t.Fatalf("ValidateFirebaseClientConfigurationInputs() error = %v, want public metadata mismatch", err)
	}
	for _, secret := range []string{inputPath, "local-credential-material.plist", "secret-api-key-must-not-appear"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("validation error leaked %q: %v", secret, err)
		}
	}
}

func TestValidateFirebaseClientConfigurationInputsRequiresConfiguredHook(t *testing.T) {
	cfg := loadRuntimeProfilesFixture(t)
	production := cfg.RuntimeProfiles.BackendEnvironments[config.BackendEnvironmentProduction]
	t.Setenv(production.Firebase.ValidationInputEnvironmentVar, "")

	err := ValidateFirebaseClientConfigurationInputs(t.TempDir(), cfg)
	if err == nil || !strings.Contains(err.Error(), production.Firebase.ValidationInputEnvironmentVar) {
		t.Fatalf("ValidateFirebaseClientConfigurationInputs() error = %v, want missing hook name", err)
	}
}

func loadRuntimeProfilesFixture(t *testing.T) config.ProjectConfig {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "runtime-profiles-config.json")
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatalf("config.LoadConfig(%q) error = %v", path, err)
	}
	return cfg
}

func configureRuntimeFirebaseInputs(t *testing.T, projectRoot string, cfg config.ProjectConfig, apiKey string) {
	t.Helper()
	inputDir := filepath.Join(projectRoot, ".firebase-validation-inputs")
	if err := os.MkdirAll(inputDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, environment := range cfg.OrderedBackendEnvironments() {
		descriptor := cfg.RuntimeProfiles.BackendEnvironments[environment]
		if descriptor.Firebase == nil {
			continue
		}
		inputPath := filepath.Join(inputDir, string(environment)+".plist")
		payload := firebasePlist(
			descriptor.Firebase.ProjectID,
			descriptor.Firebase.GoogleAppID,
			descriptor.Firebase.BundleID,
			apiKey,
		)
		if err := os.WriteFile(inputPath, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv(descriptor.Firebase.ValidationInputEnvironmentVar, inputPath)
	}
}

func firebasePlist(projectID string, googleAppID string, bundleID string, apiKey string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PROJECT_ID</key>
    <string>%s</string>
    <key>GOOGLE_APP_ID</key>
    <string>%s</string>
    <key>BUNDLE_ID</key>
    <string>%s</string>
    <key>API_KEY</key>
    <string>%s</string>
</dict>
</plist>
`, projectID, googleAppID, bundleID, apiKey)
}
