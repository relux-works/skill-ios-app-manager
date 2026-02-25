package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

func TestEntitlementsCommandAddListRemoveUsingConfigDefaultPath(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	entitlementsPath := filepath.Join(filepath.Dir(configPath), cfg.AppName+".entitlements")

	if err := os.WriteFile(entitlementsPath, []byte(scaffold.GenerateEntitlements(cfg)), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", entitlementsPath, err)
	}

	if _, err := executeRootCommand("entitlements", "--config", configPath, "add", "healthkit"); err != nil {
		t.Fatalf("executeRootCommand(add healthkit) error = %v", err)
	}

	listOutput, err := executeRootCommand("entitlements", "--config", configPath, "list")
	if err != nil {
		t.Fatalf("executeRootCommand(list) error = %v", err)
	}

	for _, expected := range []string{
		"push (aps-environment) = development",
		"app-groups (com.apple.security.application-groups)",
		"healthkit (com.apple.developer.healthkit) = true",
	} {
		if !strings.Contains(listOutput, expected) {
			t.Fatalf("list output missing %q:\n%s", expected, listOutput)
		}
	}

	if _, err := executeRootCommand("entitlements", "--config", configPath, "remove", "healthkit"); err != nil {
		t.Fatalf("executeRootCommand(remove healthkit) error = %v", err)
	}

	listOutput, err = executeRootCommand("entitlements", "--config", configPath, "list")
	if err != nil {
		t.Fatalf("executeRootCommand(list after remove) error = %v", err)
	}
	if strings.Contains(listOutput, "healthkit") {
		t.Fatalf("list output still contains healthkit after remove:\n%s", listOutput)
	}
}

func TestEntitlementsCommandSupportsExplicitPathFlag(t *testing.T) {
	t.Parallel()

	entitlementsPath := filepath.Join(t.TempDir(), "Custom.entitlements")
	if err := os.WriteFile(entitlementsPath, []byte(scaffold.GenerateEntitlements(testProjectConfig())), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", entitlementsPath, err)
	}

	if _, err := executeRootCommand(
		"entitlements",
		"--path",
		entitlementsPath,
		"add",
		"--value",
		"production",
		"push",
	); err != nil {
		t.Fatalf("executeRootCommand(add push --path) error = %v", err)
	}

	listOutput, err := executeRootCommand("entitlements", "--path", entitlementsPath, "list")
	if err != nil {
		t.Fatalf("executeRootCommand(list --path) error = %v", err)
	}
	if !strings.Contains(listOutput, "push (aps-environment) = production") {
		t.Fatalf("list output missing updated push entitlement:\n%s", listOutput)
	}
}

func TestEntitlementsCommandAddValidatesArguments(t *testing.T) {
	t.Parallel()

	if _, err := executeRootCommand("entitlements", "add"); err == nil {
		t.Fatal("executeRootCommand(entitlements add) error = nil, want argument validation error")
	}
}

func writeTestConfig(t *testing.T, cfg config.ProjectConfig) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, config.DefaultConfigPath)
	if err := config.WriteProjectConfig(path, cfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", path, err)
	}
	return path
}

func testProjectConfig() config.ProjectConfig {
	return config.ProjectConfig{
		AppName:          "DemoApp",
		BundleID:         "com.example.demo",
		TeamID:           "ABCDE12345",
		OrgName:          "Example Org",
		MarketingVersion: "1.0.0",
		ProjectVersion:   "1",
		SwiftVersion:     "6.2",
		MinTarget:        "17.0",
		AppGroups:        []string{"group.com.example.demo"},
	}
}
