package scaffold

import (
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestGenerateConfigurationAppGroupsSingleGroup(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		AppName:   "Demo",
		BundleID:  "com.example.demo",
		AppGroups: []string{"group.com.example.demo"},
	}

	content := GenerateConfigurationAppGroups(cfg)

	checks := []string{
		"import Foundation",
		"extension Configuration",
		"enum AppGroups",
		"import SharedConfig",
		`serviceName: String = "com.example.demo"`,
		"private static let resolved: DemoAppGroups",
		"DemoAppGroups.read(from: .main)",
		"static let main: String",
		"resolved.main",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Fatalf("GenerateConfigurationAppGroups() missing %q:\n%s", want, content)
		}
	}
}

func TestGenerateConfigurationAppGroupsKeepsKnownAcronymSuffixes(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		AppName:   "Demo",
		BundleID:  "com.example.demo",
		AppGroups: []string{"group.com.example.demo.pushsdk"},
	}

	content := GenerateConfigurationAppGroups(cfg)

	if !strings.Contains(content, "static let pushSDK: String") {
		t.Fatalf("GenerateConfigurationAppGroups() missing pushSDK property:\n%s", content)
	}
	if !strings.Contains(content, "resolved.pushSDK") {
		t.Fatalf("GenerateConfigurationAppGroups() missing shared configuration read:\n%s", content)
	}
}

func TestGenerateConfigurationAppGroupsMultipleGroups(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		AppName:  "FullDemo",
		BundleID: "com.example.fulldemo",
		AppGroups: []string{
			"group.com.example.fulldemo",
			"group.com.example.shared",
		},
	}

	content := GenerateConfigurationAppGroups(cfg)

	checks := []string{
		`serviceName: String = "com.example.fulldemo"`,
		"FullDemoAppGroups.read(from: .main)",
		"static let main: String",
		"static let comExampleShared: String",
		"resolved.main",
		"resolved.comExampleShared",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Fatalf("GenerateConfigurationAppGroups() missing %q:\n%s", want, content)
		}
	}
}

func TestGenerateConfigurationAppGroupsSanitizesHyphenatedGroupIDs(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		AppName:   "ExampleDemo",
		BundleID:  "com.example-demo.app",
		AppGroups: []string{"group.com.example-demo.app.shared"},
	}

	content := GenerateConfigurationAppGroups(cfg)

	checks := []string{
		"static let shared: String",
		"resolved.shared",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Fatalf("GenerateConfigurationAppGroups() missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "EXAMPLE-DEMO") {
		t.Fatalf("GenerateConfigurationAppGroups() kept hyphen in Swift identifier:\n%s", content)
	}
}
