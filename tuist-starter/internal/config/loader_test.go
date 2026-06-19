package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config-alpha.json")

	want := validProjectConfig()
	if err := WriteProjectConfig(path, want); err != nil {
		t.Fatalf("WriteProjectConfig() error = %v", err)
	}

	got, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadConfig() = %#v, want %#v", got, want)
	}
}

func TestLoadConfigDefaultPath(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	want := validProjectConfig()
	if err := WriteProjectConfig(DefaultConfigPath, want); err != nil {
		t.Fatalf("WriteProjectConfig() error = %v", err)
	}

	got, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig(\"\") error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadConfig(\"\") = %#v, want %#v", got, want)
	}
}

func TestLoadProjectConfigCompatibilityWrapper(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config-fc.json")
	want := validProjectConfig()

	if err := WriteProjectConfig(path, want); err != nil {
		t.Fatalf("WriteProjectConfig() error = %v", err)
	}

	got, err := LoadProjectConfig(path)
	if err != nil {
		t.Fatalf("LoadProjectConfig() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadProjectConfig() = %#v, want %#v", got, want)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	t.Parallel()

	_, err := LoadConfig("nope.json")
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want missing file error")
	}

	if !strings.Contains(err.Error(), `config file "nope.json" does not exist`) {
		t.Fatalf("LoadConfig() error = %q, want missing file message", err.Error())
	}
}

func TestLoadConfigAppliesDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config-defaults.json")
	content := `{
  "app_name": "DemoApp",
  "bundle_id": "com.example.demo",
  "team_id": "ABCDE12345",
  "marketing_version": "1.0.0",
  "project_version": "1",
  "swift_version": "6.2",
  "min_target": "17.0"
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.ProductName != cfg.AppName {
		t.Fatalf("ProductName = %q, want %q", cfg.ProductName, cfg.AppName)
	}
	if cfg.ModulesPath != "Packages" {
		t.Fatalf("ModulesPath = %q, want %q", cfg.ModulesPath, "Packages")
	}
	if cfg.SharedConfig.ModuleName != "SharedConfig" {
		t.Fatalf("SharedConfig.ModuleName = %q, want %q", cfg.SharedConfig.ModuleName, "SharedConfig")
	}
	if cfg.Theme != ThemeAutomatic {
		t.Fatalf("Theme = %q, want %q", cfg.Theme, ThemeAutomatic)
	}
	if cfg.Orientation != OrientationAutomatic {
		t.Fatalf("Orientation = %q, want %q", cfg.Orientation, OrientationAutomatic)
	}
	if cfg.ProjectSettings.Swift.LanguageMode != "v6" {
		t.Fatalf("ProjectSettings.Swift.LanguageMode = %q, want %q", cfg.ProjectSettings.Swift.LanguageMode, "v6")
	}
	if cfg.ProjectSettings.Swift.StrictMemorySafety != "yes" {
		t.Fatalf("ProjectSettings.Swift.StrictMemorySafety = %q, want %q", cfg.ProjectSettings.Swift.StrictMemorySafety, "yes")
	}
	if cfg.ProjectSettings.Swift.Concurrency.Approachable == nil || *cfg.ProjectSettings.Swift.Concurrency.Approachable {
		t.Fatalf("Approachable = %#v, want false", cfg.ProjectSettings.Swift.Concurrency.Approachable)
	}
	if cfg.ProjectSettings.Swift.Concurrency.MemberImportVisibility != "yes" {
		t.Fatalf("MemberImportVisibility = %q, want %q", cfg.ProjectSettings.Swift.Concurrency.MemberImportVisibility, "yes")
	}
	if cfg.ProjectSettings.Swift.Concurrency.ExistentialAny != "yes" {
		t.Fatalf("ExistentialAny = %q, want %q", cfg.ProjectSettings.Swift.Concurrency.ExistentialAny, "yes")
	}
}

func TestLoadConfigParsesExplicitExportComplianceFalse(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config-export-compliance.json")
	content := `{
  "app_name": "DemoApp",
  "bundle_id": "com.example.demo",
  "team_id": "ABCDE12345",
  "marketing_version": "1.0.0",
  "project_version": "1",
  "swift_version": "6.2",
  "min_target": "17.0",
  "uses_non_exempt_encryption": false
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.UsesNonExemptEncryption == nil {
		t.Fatal("UsesNonExemptEncryption = nil, want explicit false pointer")
	}
	if *cfg.UsesNonExemptEncryption {
		t.Fatal("UsesNonExemptEncryption = true, want false")
	}
}

func TestLoadConfigParsesPrivacyUsageDescriptions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config-privacy.json")
	content := `{
  "app_name": "DemoApp",
  "bundle_id": "com.example.demo",
  "team_id": "ABCDE12345",
  "marketing_version": "1.0.0",
  "project_version": "1",
  "swift_version": "6.2",
  "min_target": "17.0",
  "privacy_usage_descriptions": {
    "bluetooth_always": "Find nearby transfer receivers.",
    "bluetooth_peripheral": "Advertise nearby transfer availability."
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.PrivacyUsageDescriptions.BluetoothAlways != "Find nearby transfer receivers." {
		t.Fatalf("BluetoothAlways = %q", cfg.PrivacyUsageDescriptions.BluetoothAlways)
	}
	if cfg.PrivacyUsageDescriptions.BluetoothPeripheral != "Advertise nearby transfer availability." {
		t.Fatalf("BluetoothPeripheral = %q", cfg.PrivacyUsageDescriptions.BluetoothPeripheral)
	}
}

func TestLoadConfigParsesScripts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config-scripts.json")
	content := `{
  "app_name": "DemoApp",
  "bundle_id": "com.example.demo",
  "team_id": "ABCDE12345",
  "marketing_version": "1.0.0",
  "project_version": "1",
  "swift_version": "6.2",
  "min_target": "17.0",
  "scripts": {
    "pre_generate": [
      {
        "path": "scripts/patch-package.sh",
        "language": "bash",
        "description": "Patch remote package resources"
      }
    ]
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Scripts.PreGenerate) != 1 {
		t.Fatalf("Scripts.PreGenerate count = %d, want 1", len(cfg.Scripts.PreGenerate))
	}
	script := cfg.Scripts.PreGenerate[0]
	if script.Path != "scripts/patch-package.sh" {
		t.Fatalf("Path = %q", script.Path)
	}
	if script.Language != "bash" {
		t.Fatalf("Language = %q", script.Language)
	}
	if script.Description != "Patch remote package resources" {
		t.Fatalf("Description = %q", script.Description)
	}
}

func TestSampleConfigIsValid(t *testing.T) {
	t.Parallel()

	path := filepath.Join(goModuleRoot(t), "testdata", "sample-config.json")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", path, err)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("sample config Validate() error = %v", err)
	}
}

func TestXFlowConfigIsValid(t *testing.T) {
	t.Parallel()

	path := filepath.Join(goModuleRoot(t), "testdata", "xflow-config.json")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", path, err)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("xflow config Validate() error = %v", err)
	}

	if cfg.BundleID != "org.xflow.app" {
		t.Fatalf("BundleID = %q, want %q", cfg.BundleID, "org.xflow.app")
	}
	if cfg.TeamID != "H446YY77RR" {
		t.Fatalf("TeamID = %q, want %q", cfg.TeamID, "H446YY77RR")
	}
	if cfg.SwiftVersion != "6.0" {
		t.Fatalf("SwiftVersion = %q, want %q", cfg.SwiftVersion, "6.0")
	}
	if cfg.MinTarget != "17.6" {
		t.Fatalf("MinTarget = %q, want %q", cfg.MinTarget, "17.6")
	}
	if cfg.MarketingVersion != "0.1.9" {
		t.Fatalf("MarketingVersion = %q, want %q", cfg.MarketingVersion, "0.1.9")
	}
}

func validProjectConfig() ProjectConfig {
	return ProjectConfig{
		AppName:          "DemoApp",
		BundleID:         "com.example.demo",
		TeamID:           "ABCDE12345",
		OrgName:          "Example Org",
		MarketingVersion: "1.2.3",
		ProjectVersion:   "123",
		SwiftVersion:     "6.2",
		MinTarget:        "17.0",
		URLScheme:        "demoapp",
		AppGroups:        []string{"group.com.example.demo"},
		Theme:            ThemeLight,
		Orientation:      OrientationPortrait,
		ProductName:      "Demo Product",
		Configurations:   []string{"Debug", "Release"},
		ModulesPath:      "Packages",
		SharedConfig:     SharedConfigConfig{ModuleName: "SharedConfig"},
		PushKeyPath:      "certs/AuthKey_ABC123.p8",
		PushKeyID:        "ABC123DEF4",
		ProjectSettings: ProjectSettings{
			Swift: SwiftProjectSettings{
				LanguageMode:       "v6",
				StrictMemorySafety: "yes",
				Concurrency: SwiftConcurrencySettings{
					Approachable:                      boolPtr(false),
					DefaultActorIsolation:             "nonisolated",
					StrictChecking:                    "complete",
					ConciseMagicFile:                  boolPtr(true),
					DisableOutwardActorIsolation:      boolPtr(true),
					GlobalActorIsolatedTypesUsability: boolPtr(true),
					InferIsolatedConformances:         boolPtr(true),
					InferSendableFromCaptures:         boolPtr(true),
					GlobalConcurrency:                 boolPtr(true),
					MemberImportVisibility:            "yes",
					NonfrozenEnumExhaustivity:         boolPtr(true),
					RegionBasedIsolation:              boolPtr(true),
					ExistentialAny:                    "yes",
					NonisolatedNonsendingByDefault:    boolPtr(true),
				},
			},
		},
	}
}
