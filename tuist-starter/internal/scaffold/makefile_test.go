package scaffold

import (
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestGenerateMakefileContainsRequiredVariablesAndTargets(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		AppName:     "DemoApp",
		BundleID:    "com.example.demo",
		TeamID:      "ABCDE12345",
		ModulesPath: "Packages",
		MinTarget:   "17.0",
		PushKeyPath: "certs/AuthKey_ABC123.p8",
		PushKeyID:   "ABC123DEF4",
	}

	makefile := GenerateMakefile(cfg)

	requiredSnippets := []string{
		"APP_NAME := DemoApp",
		"BUNDLE_ID := com.example.demo",
		"TEAM_ID := ABCDE12345",
		"MODULES_PATH := Packages",
		"SCHEME ?= DemoApp",
		"DESTINATION ?= platform=iOS Simulator,name=iPhone 16,OS=17.0",
		"setup: ##",
		"resetup: ##",
		"generate: ##",
		"build: ##",
		"test: ##",
		"clean: ##",
		"deep-clean: ##",
		"lint: ##",
		"format: ##",
		"validate: lint build ##",
		"install-tools: ##",
		"periphery: ##",
		"help: ##",
		"push-token: ##",
		"push-send: ##",
		".PHONY:",
		generatedTargetsMarker,
		customTargetsMarker,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(makefile, snippet) {
			t.Fatalf("GenerateMakefile() missing %q:\n%s", snippet, makefile)
		}
	}
}

func TestGenerateMakefileOmitsPushTargetsWhenPushConfigMissing(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		AppName:     "DemoApp",
		BundleID:    "com.example.demo",
		TeamID:      "ABCDE12345",
		ModulesPath: "Packages",
		MinTarget:   "17.0",
	}

	makefile := GenerateMakefile(cfg)

	disallowedSnippets := []string{
		"PUSH_KEY_PATH :=",
		"PUSH_KEY_ID :=",
		"push-token: ##",
		"push-send: ##",
	}
	for _, snippet := range disallowedSnippets {
		if strings.Contains(makefile, snippet) {
			t.Fatalf("GenerateMakefile() unexpectedly contains %q:\n%s", snippet, makefile)
		}
	}
}

func TestGenerateMakefilePreservesCustomSection(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		AppName:     "UpdatedApp",
		BundleID:    "com.example.updated",
		TeamID:      "ABCDE12345",
		ModulesPath: "VendorPackages",
		MinTarget:   "18.0",
	}

	existing := strings.Join([]string{
		"APP_NAME := OldApp",
		generatedTargetsMarker,
		"",
		customTargetsMarker,
		"custom-target: ## Custom action",
		"\t@echo \"custom\"",
		"",
	}, "\n")

	makefile := GenerateMakefilePreservingCustom(cfg, existing)

	requiredSnippets := []string{
		"APP_NAME := UpdatedApp",
		"MODULES_PATH := VendorPackages",
		generatedTargetsMarker,
		customTargetsMarker,
		"custom-target: ## Custom action",
		"\t@echo \"custom\"",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(makefile, snippet) {
			t.Fatalf("GenerateMakefilePreservingCustom() missing %q:\n%s", snippet, makefile)
		}
	}
}

func TestGenerateMakefilePreservesSectionAfterGeneratedBoundaryWithoutCustomMarker(t *testing.T) {
	t.Parallel()

	cfg := config.ProjectConfig{
		AppName:     "DemoApp",
		BundleID:    "com.example.demo",
		TeamID:      "ABCDE12345",
		ModulesPath: "Packages",
		MinTarget:   "17.0",
	}

	existing := strings.Join([]string{
		"# legacy generated section",
		generatedTargetsMarker,
		"",
		"custom-target: ## Custom action",
		"\t@echo \"custom\"",
		"",
	}, "\n")

	makefile := GenerateMakefilePreservingCustom(cfg, existing)

	requiredSnippets := []string{
		"APP_NAME := DemoApp",
		generatedTargetsMarker,
		customTargetsMarker,
		"custom-target: ## Custom action",
		"\t@echo \"custom\"",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(makefile, snippet) {
			t.Fatalf("GenerateMakefilePreservingCustom() missing %q:\n%s", snippet, makefile)
		}
	}
}
