package securestore

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/registry"
)

func TestSecureStoreModuleAccessGroupFlagIsOptional(t *testing.T) {
	t.Parallel()

	mod := registry.Get(registry.SecureStore)
	if mod == nil {
		t.Fatal("SecureStore module not registered")
	}

	for _, f := range mod.ExtraFlags {
		if f.Name == "access-group" {
			if f.Required {
				t.Fatal("access-group flag must be optional; Plan() performs contextual validation")
			}
			return
		}
	}

	t.Fatal("access-group flag not found in module ExtraFlags")
}

func TestPlanSuccessWithValidAccessGroup(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeSecureStoreConfig(t, projectRoot, []string{"group.com.example.demo"})

	plan, err := Plan(registry.SetupInput{
		ProjectRoot: projectRoot,
		ExtraArgs: map[string]string{
			"access-group": "group.com.example.demo",
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if !strings.Contains(plan, "SecureStore Setup Plan") {
		t.Fatalf("Plan() missing header, got: %s", plan)
	}
	if !strings.Contains(plan, "group.com.example.demo") {
		t.Fatalf("Plan() missing access group, got: %s", plan)
	}
}

func TestPlanReturnsConfigLoadError(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	_, err := Plan(registry.SetupInput{
		ProjectRoot: projectRoot,
		ExtraArgs: map[string]string{
			"access-group": "group.com.example.demo",
		},
	})
	if err == nil {
		t.Fatal("Plan() error = nil, want load config error")
	}
	if !strings.Contains(err.Error(), "load config:") {
		t.Fatalf("error = %q, want load config wrapper", err.Error())
	}
}

func TestPlanMissingAccessGroupNoConfigGroups(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeSecureStoreConfig(t, projectRoot, nil)

	_, err := Plan(registry.SetupInput{
		ProjectRoot: projectRoot,
		ExtraArgs:   map[string]string{},
	})
	if err == nil {
		t.Fatal("Plan() error = nil, want missing access-group error")
	}

	want := "--access-group is required but no app_groups defined in config\nadd groups via \"app_groups\" field in ios-app-manager.json, e.g.:\n  \"app_groups\": [\"group.com.example.app\"]"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestPlanMissingAccessGroupShowsAvailableGroups(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeSecureStoreConfig(t, projectRoot, []string{"group.com.example.demo"})

	_, err := Plan(registry.SetupInput{
		ProjectRoot: projectRoot,
		ExtraArgs:   map[string]string{},
	})
	if err == nil {
		t.Fatal("Plan() error = nil, want missing access-group error")
	}

	want := "--access-group is required\navailable groups in config: [group.com.example.demo]"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestPlanRejectsUnknownAccessGroup(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeSecureStoreConfig(t, projectRoot, []string{"group.com.example.demo"})

	_, err := Plan(registry.SetupInput{
		ProjectRoot: projectRoot,
		ExtraArgs: map[string]string{
			"access-group": "group.fake",
		},
	})
	if err == nil {
		t.Fatal("Plan() error = nil, want unknown access-group error")
	}

	want := "access group \"group.fake\" not found in config\navailable groups: [group.com.example.demo]"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestValidateAccessGroup(t *testing.T) {
	t.Parallel()

	if err := validateAccessGroup("group.com.example.demo", []string{"group.com.example.demo"}); err != nil {
		t.Fatalf("validateAccessGroup(valid) error = %v", err)
	}
}

func writeSecureStoreConfig(t *testing.T, projectRoot string, groups []string) {
	t.Helper()

	cfgPath := filepath.Join(projectRoot, config.DefaultConfigPath)
	cfg := config.ProjectConfig{
		AppName:          "DemoApp",
		BundleID:         "com.example.demo",
		TeamID:           "TEAM123456",
		MarketingVersion: "1.0.0",
		ProjectVersion:   "1",
		SwiftVersion:     "6.2",
		MinTarget:        "17.0",
		AppGroups:        groups,
	}
	if err := config.WriteProjectConfig(cfgPath, cfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", cfgPath, err)
	}
}
