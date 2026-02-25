package scaffold

import (
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"gopkg.in/yaml.v3"
)

func TestGenerateSwiftLintConfigContainsRequiredStructure(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		ModulesPath: "VendorPackages",
	}

	content := GenerateSwiftLintConfig(cfg)

	requiredSnippets := []string{
		"included:",
		"  - Targets/",
		"  - VendorPackages/",
		"excluded:",
		"  - Derived/",
		"  - DerivedData/",
		`  - "*.generated.swift"`,
		"  - Tuist/Dependencies/",
		"disabled_rules:",
		"opt_in_rules:",
		"  - empty_count",
		"  - empty_string",
		"  - force_unwrapping",
		"line_length:",
		`reporter: "xcode"`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("GenerateSwiftLintConfig() missing %q:\n%s", snippet, content)
		}
	}
}

func TestGenerateSwiftLintConfigUsesDefaultModulesPath(t *testing.T) {
	t.Parallel()

	content := GenerateSwiftLintConfig(config.ProjectConfig{})
	if !strings.Contains(content, "  - Packages/") {
		t.Fatalf("GenerateSwiftLintConfig() should include default ModulesPath Packages:\n%s", content)
	}
}

func TestGenerateSwiftLintConfigIsValidYAML(t *testing.T) {
	t.Parallel()

	content := GenerateSwiftLintConfig(config.ProjectConfig{
		ModulesPath: "Packages",
	})

	var decoded map[string]any
	if err := yaml.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("GenerateSwiftLintConfig() should produce valid YAML: %v\n%s", err, content)
	}

	for _, key := range []string{"included", "excluded", "disabled_rules", "opt_in_rules"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("GenerateSwiftLintConfig() missing root key %q", key)
		}
	}
}
