package scaffold

import (
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestGenerateConfigurationKeychainContainsConstants(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		BundleID: "com.example.demo",
		TeamID:   "ABCDE12345",
	}

	content := GenerateConfigurationKeychain(cfg)

	requiredSnippets := []string{
		"extension Configuration",
		"enum Keychain",
		`serviceName = "com.example.demo"`,
		`accessGroup = "ABCDE12345.com.example.demo.shared"`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("GenerateConfigurationKeychain() missing %q:\n%s", snippet, content)
		}
	}
}
