package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const testEntitlementsPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>aps-environment</key>
	<string>development</string>
	<key>com.apple.security.application-groups</key>
	<array>
		<string>group.com.example.demo</string>
	</array>
</dict>
</plist>
`

func TestEntitlementsListUsingConfigDefaultPath(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	entitlementsPath := filepath.Join(filepath.Dir(configPath), cfg.AppName+".entitlements")

	if err := os.WriteFile(entitlementsPath, []byte(testEntitlementsPlist), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", entitlementsPath, err)
	}

	listOutput, err := executeRootCommand("entitlements", "--config", configPath, "list")
	if err != nil {
		t.Fatalf("executeRootCommand(list) error = %v", err)
	}

	for _, expected := range []string{
		"aps-environment = development",
		"com.apple.security.application-groups",
	} {
		if !strings.Contains(listOutput, expected) {
			t.Fatalf("list output missing %q:\n%s", expected, listOutput)
		}
	}
}

func TestEntitlementsListSupportsExplicitPathFlag(t *testing.T) {
	t.Parallel()

	entitlementsPath := filepath.Join(t.TempDir(), "Custom.entitlements")
	if err := os.WriteFile(entitlementsPath, []byte(testEntitlementsPlist), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", entitlementsPath, err)
	}

	listOutput, err := executeRootCommand("entitlements", "--path", entitlementsPath, "list")
	if err != nil {
		t.Fatalf("executeRootCommand(list --path) error = %v", err)
	}
	if !strings.Contains(listOutput, "aps-environment = development") {
		t.Fatalf("list output missing aps-environment:\n%s", listOutput)
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
