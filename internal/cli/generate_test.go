package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

func TestGenerateHelpShowsMakefileSubcommand(t *testing.T) {
	t.Parallel()

	output, err := executeRootCommand("generate", "--help")
	if err != nil {
		t.Fatalf("executeRootCommand(generate --help) error = %v", err)
	}

	for _, expected := range []string{"makefile", "swiftlint", "Generate project artifacts"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("generate --help output missing %q:\n%s", expected, output)
		}
	}
}

func TestGenerateMakefileCreatesFileWhenMissing(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	makefilePath := filepath.Join(filepath.Dir(configPath), "Makefile")

	output, err := executeRootCommand("generate", "--config", configPath, "makefile")
	if err != nil {
		t.Fatalf("executeRootCommand(generate makefile) error = %v", err)
	}
	if !strings.Contains(output, "created") {
		t.Fatalf("generate makefile output = %q, want creation message", output)
	}

	content, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", makefilePath, err)
	}
	expected := scaffold.GenerateMakefile(cfg)
	if string(content) != expected {
		t.Fatalf("generated Makefile mismatch:\nwant:\n%s\n\ngot:\n%s", expected, string(content))
	}
}

func TestGenerateMakefileRegeneratesAndPreservesCustomSection(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	makefilePath := filepath.Join(filepath.Dir(configPath), "Makefile")

	initialCfg := cfg
	initialCfg.ModulesPath = "OldPackages"
	customTarget := "custom-target: ## Custom workflow\n\t@echo \"custom\"\n"
	initialMakefile := scaffold.GenerateMakefile(initialCfg) + "\n" + customTarget
	if err := os.WriteFile(makefilePath, []byte(initialMakefile), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", makefilePath, err)
	}

	updatedCfg := cfg
	updatedCfg.ModulesPath = "VendorPackages"
	if err := config.WriteProjectConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", configPath, err)
	}

	output, err := executeRootCommand("generate", "--config", configPath, "makefile")
	if err != nil {
		t.Fatalf("executeRootCommand(generate makefile) error = %v", err)
	}
	for _, expected := range []string{"regenerated", "preserved custom section"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("generate makefile output missing %q:\n%s", expected, output)
		}
	}

	regenerated, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", makefilePath, err)
	}
	makefile := string(regenerated)
	if !strings.Contains(makefile, "MODULES_PATH := VendorPackages") {
		t.Fatalf("regenerated Makefile missing updated modules path:\n%s", makefile)
	}
	if !strings.Contains(makefile, customTarget) {
		t.Fatalf("regenerated Makefile missing preserved custom target:\n%s", makefile)
	}
}

func TestGenerateSwiftLintCreatesFileWhenMissing(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	swiftlintPath := filepath.Join(filepath.Dir(configPath), ".swiftlint.yml")

	output, err := executeRootCommand("generate", "--config", configPath, "swiftlint")
	if err != nil {
		t.Fatalf("executeRootCommand(generate swiftlint) error = %v", err)
	}
	if !strings.Contains(output, "created") {
		t.Fatalf("generate swiftlint output = %q, want creation message", output)
	}

	content, err := os.ReadFile(swiftlintPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", swiftlintPath, err)
	}

	expected := scaffold.GenerateSwiftLintConfig(cfg)
	if string(content) != expected {
		t.Fatalf("generated .swiftlint.yml mismatch:\nwant:\n%s\n\ngot:\n%s", expected, string(content))
	}
}

func TestGenerateSwiftLintRegeneratesWhenModulesPathChanges(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	cfg.ModulesPath = "OldPackages"
	configPath := writeTestConfig(t, cfg)
	swiftlintPath := filepath.Join(filepath.Dir(configPath), ".swiftlint.yml")

	initialConfig := scaffold.GenerateSwiftLintConfig(cfg)
	if err := os.WriteFile(swiftlintPath, []byte(initialConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", swiftlintPath, err)
	}

	updatedCfg := cfg
	updatedCfg.ModulesPath = "VendorPackages"
	if err := config.WriteProjectConfig(configPath, updatedCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", configPath, err)
	}

	output, err := executeRootCommand("generate", "--config", configPath, "swiftlint")
	if err != nil {
		t.Fatalf("executeRootCommand(generate swiftlint) error = %v", err)
	}
	if !strings.Contains(output, "regenerated") {
		t.Fatalf("generate swiftlint output = %q, want regenerate message", output)
	}

	regenerated, err := os.ReadFile(swiftlintPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", swiftlintPath, err)
	}

	content := string(regenerated)
	if !strings.Contains(content, "  - VendorPackages/") {
		t.Fatalf("regenerated .swiftlint.yml missing updated modules path:\n%s", content)
	}
	if strings.Contains(content, "  - OldPackages/") {
		t.Fatalf("regenerated .swiftlint.yml should not contain old modules path:\n%s", content)
	}
}
