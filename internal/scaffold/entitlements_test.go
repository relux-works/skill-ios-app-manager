package scaffold

import (
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestGenerateEntitlementsIncludesPushAndAppGroups(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		BundleID:  "com.example.demo",
		AppGroups: []string{"group.com.example.demo", "group.com.example.shared"},
	}

	entitlements := GenerateEntitlements(cfg)

	requiredSnippets := []string{
		`<key>aps-environment</key>`,
		`<string>development</string>`,
		`<key>com.apple.security.application-groups</key>`,
		`<string>group.com.example.demo</string>`,
		`<string>group.com.example.shared</string>`,
		`<key>keychain-access-groups</key>`,
		`<string>$(AppIdentifierPrefix)com.example.demo.shared</string>`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(entitlements, snippet) {
			t.Fatalf("GenerateEntitlements() missing %q:\n%s", snippet, entitlements)
		}
	}
}

func TestGenerateEntitlementsOmitsAppGroupsWhenEmpty(t *testing.T) {
	t.Parallel()

	entitlements := GenerateEntitlements(config.ProjectConfig{})
	if strings.Contains(entitlements, "com.apple.security.application-groups") {
		t.Fatalf("GenerateEntitlements() should omit app groups when empty:\n%s", entitlements)
	}
}

func TestGenerateEntitlementsOmitsKeychainGroupsWhenNoBundleID(t *testing.T) {
	t.Parallel()

	entitlements := GenerateEntitlements(config.ProjectConfig{})
	if strings.Contains(entitlements, "keychain-access-groups") {
		t.Fatalf("GenerateEntitlements() should omit keychain groups when bundle ID is empty:\n%s", entitlements)
	}
}

func TestGenerateEntitlementsIncludesKeychainGroups(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{BundleID: "com.example.app"}
	entitlements := GenerateEntitlements(cfg)

	requiredSnippets := []string{
		`<key>keychain-access-groups</key>`,
		`<string>$(AppIdentifierPrefix)com.example.app.shared</string>`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(entitlements, snippet) {
			t.Fatalf("GenerateEntitlements() missing %q:\n%s", snippet, entitlements)
		}
	}
}
