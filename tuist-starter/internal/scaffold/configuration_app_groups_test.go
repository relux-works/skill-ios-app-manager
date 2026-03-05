package scaffold

import (
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestGenerateConfigurationAppGroupsSingleGroup(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		BundleID:  "com.example.demo",
		AppGroups: []string{"group.com.example.demo"},
	}

	content := GenerateConfigurationAppGroups(cfg)

	checks := []string{
		"extension Configuration",
		"enum AppGroups",
		`serviceName: String = "com.example.demo"`,
		"GROUP_COM_EXAMPLE_DEMO",
		`readInfoPlistValue(by: "GROUP_COM_EXAMPLE_DEMO")`,
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Fatalf("GenerateConfigurationAppGroups() missing %q:\n%s", want, content)
		}
	}
}

func TestGenerateConfigurationAppGroupsMultipleGroups(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		BundleID: "com.example.fulldemo",
		AppGroups: []string{
			"group.com.example.fulldemo",
			"group.com.example.shared",
		},
	}

	content := GenerateConfigurationAppGroups(cfg)

	checks := []string{
		`serviceName: String = "com.example.fulldemo"`,
		"GROUP_COM_EXAMPLE_FULLDEMO",
		"GROUP_COM_EXAMPLE_SHARED",
		`readInfoPlistValue(by: "GROUP_COM_EXAMPLE_FULLDEMO")`,
		`readInfoPlistValue(by: "GROUP_COM_EXAMPLE_SHARED")`,
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Fatalf("GenerateConfigurationAppGroups() missing %q:\n%s", want, content)
		}
	}
}
