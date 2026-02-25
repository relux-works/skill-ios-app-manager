package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	templaterenderer "github.com/relux-works/ios-app-manager/internal/template"
	"github.com/relux-works/ios-app-manager/internal/testutil"
)

func TestScaffoldCreatesExpectedLayoutAndFiles(t *testing.T) {
	t.Parallel()

	cfg := loadConfigFixture(t, "sample-config.json")
	outputDir := filepath.Join(t.TempDir(), "generated")

	scaffolder := New(templaterenderer.NewRenderer(templaterenderer.WithRootDir(outputDir)))
	written, err := scaffolder.Scaffold(cfg, outputDir, false)
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}
	if len(written) == 0 {
		t.Fatal("Scaffold() wrote 0 files, want > 0")
	}

	requiredDirs := []string{
		filepath.Join(outputDir, "Targets", cfg.AppName, "Sources"),
		filepath.Join(outputDir, "Targets", cfg.AppName, "Resources"),
		filepath.Join(outputDir, cfg.ModulesPath),
	}
	for _, dir := range requiredDirs {
		requireDir(t, dir)
	}

	requiredFiles := []string{
		filepath.Join(outputDir, "Tuist.swift"),
		filepath.Join(outputDir, "Project.swift"),
		filepath.Join(outputDir, "Workspace.swift"),
		filepath.Join(outputDir, "Package.swift"),
		filepath.Join(outputDir, "Makefile"),
		filepath.Join(outputDir, ".periphery.yml"),
		filepath.Join(outputDir, ".swiftlint.yml"),
		filepath.Join(outputDir, ".gitignore"),
		filepath.Join(outputDir, cfg.AppName+".entitlements"),
		filepath.Join(outputDir, "Targets", cfg.AppName, "Sources", "App.swift"),
	}
	for _, path := range requiredFiles {
		requireFile(t, path)
	}

	projectManifest := readFile(t, filepath.Join(outputDir, "Project.swift"))
	projectChecks := []string{
		cfg.AppName,
		cfg.BundleID,
		cfg.TeamID,
		cfg.MarketingVersion,
		cfg.ProjectVersion,
		`Targets/` + cfg.AppName + `/Sources/**`,
		`Targets/` + cfg.AppName + `/Resources/**`,
		cfg.AppName + `.entitlements`,
	}
	for _, want := range projectChecks {
		if !strings.Contains(projectManifest, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectManifest)
		}
	}

	tuistConfig := readFile(t, filepath.Join(outputDir, "Tuist.swift"))
	if !strings.Contains(tuistConfig, cfg.SwiftVersion) {
		t.Fatalf("Tuist.swift missing SwiftVersion %q:\n%s", cfg.SwiftVersion, tuistConfig)
	}

	workspaceManifest := readFile(t, filepath.Join(outputDir, "Workspace.swift"))
	if !strings.Contains(workspaceManifest, cfg.AppName) {
		t.Fatalf("Workspace.swift missing AppName %q:\n%s", cfg.AppName, workspaceManifest)
	}

	packageManifest := readFile(t, filepath.Join(outputDir, "Package.swift"))
	if !strings.Contains(packageManifest, cfg.ModulesPath) {
		t.Fatalf("Package.swift missing ModulesPath %q:\n%s", cfg.ModulesPath, packageManifest)
	}

	makefile := readFile(t, filepath.Join(outputDir, "Makefile"))
	for _, target := range []string{
		"setup",
		"resetup",
		"generate",
		"build",
		"test",
		"clean",
		"deep-clean",
		"lint",
		"format",
		"validate",
		"install-tools",
		"periphery",
		"help",
		"push-token",
		"push-send",
	} {
		if !strings.Contains(makefile, target+":") {
			t.Fatalf("Makefile missing target %q:\n%s", target, makefile)
		}
	}
	helpOutput := runMakeHelp(t, outputDir)
	for _, target := range []string{"setup", "build", "test", "validate", "periphery", "push-send"} {
		if !strings.Contains(helpOutput, target) {
			t.Fatalf("make help output missing target %q:\n%s", target, helpOutput)
		}
	}
	testutil.AssertGoldenFile(t, "golden/makefile", makefile)

	periphery := readFile(t, filepath.Join(outputDir, ".periphery.yml"))
	peripheryChecks := []string{
		`workspace: "` + cfg.AppName + `.xcworkspace"`,
		"schemes:",
		`  - "` + cfg.AppName + `"`,
		"retain_public: true",
	}
	for _, want := range peripheryChecks {
		if !strings.Contains(periphery, want) {
			t.Fatalf(".periphery.yml missing %q:\n%s", want, periphery)
		}
	}

	swiftlint := readFile(t, filepath.Join(outputDir, ".swiftlint.yml"))
	swiftlintChecks := []string{
		"included:",
		"  - Targets/",
		"  - " + cfg.ModulesPath + "/",
		"excluded:",
		"  - Derived/",
		"  - DerivedData/",
		`  - "*.generated.swift"`,
		"  - Tuist/Dependencies/",
	}
	for _, want := range swiftlintChecks {
		if !strings.Contains(swiftlint, want) {
			t.Fatalf(".swiftlint.yml missing %q:\n%s", want, swiftlint)
		}
	}

	gitignore := readFile(t, filepath.Join(outputDir, ".gitignore"))
	for _, pattern := range []string{"*.xcodeproj", "*.xcworkspace", "Derived/", ".DS_Store"} {
		if !strings.Contains(gitignore, pattern) {
			t.Fatalf(".gitignore missing pattern %q:\n%s", pattern, gitignore)
		}
	}

	entitlements := readFile(t, filepath.Join(outputDir, cfg.AppName+".entitlements"))
	if !strings.Contains(entitlements, "<key>aps-environment</key>") {
		t.Fatalf("entitlements missing aps-environment:\n%s", entitlements)
	}
	if len(cfg.AppGroups) > 0 && !strings.Contains(entitlements, cfg.AppGroups[0]) {
		t.Fatalf("entitlements missing app group %q:\n%s", cfg.AppGroups[0], entitlements)
	}

	appStub := readFile(t, filepath.Join(outputDir, "Targets", cfg.AppName, "Sources", "App.swift"))
	if !strings.Contains(appStub, `Text("Hello, World!")`) {
		t.Fatalf("App.swift missing hello world view:\n%s", appStub)
	}
}

func TestScaffoldEntitlementsWithoutAppGroups(t *testing.T) {
	t.Parallel()

	cfg := loadConfigFixture(t, "minimal-config.json")
	outputDir := t.TempDir()

	scaffolder := New(templaterenderer.NewRenderer(templaterenderer.WithRootDir(outputDir)))
	if _, err := scaffolder.Scaffold(cfg, outputDir, false); err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	entitlementsPath := filepath.Join(outputDir, cfg.AppName+".entitlements")
	entitlements := readFile(t, entitlementsPath)

	if !strings.Contains(entitlements, "<key>aps-environment</key>") {
		t.Fatalf("entitlements missing aps-environment:\n%s", entitlements)
	}
	if strings.Contains(entitlements, "com.apple.security.application-groups") {
		t.Fatalf("entitlements should omit application groups when config has none:\n%s", entitlements)
	}
}

func TestScaffoldPreventsOverwriteWithoutForce(t *testing.T) {
	t.Parallel()

	cfg := loadConfigFixture(t, "sample-config.json")
	outputDir := t.TempDir()

	existingProjectPath := filepath.Join(outputDir, "Project.swift")
	if err := os.WriteFile(existingProjectPath, []byte("// existing"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", existingProjectPath, err)
	}

	scaffolder := New(templaterenderer.NewRenderer(templaterenderer.WithRootDir(outputDir)))
	_, err := scaffolder.Scaffold(cfg, outputDir, false)
	if err == nil {
		t.Fatal("Scaffold() error = nil, want overwrite protection error")
	}

	message := err.Error()
	if !strings.Contains(message, "--force") || !strings.Contains(message, "Project.swift") {
		t.Fatalf("Scaffold() error = %q, want --force and Project.swift", message)
	}
}

func TestScaffoldForceOverwritesExistingFiles(t *testing.T) {
	t.Parallel()

	cfg := loadConfigFixture(t, "sample-config.json")
	outputDir := t.TempDir()

	projectPath := filepath.Join(outputDir, "Project.swift")
	if err := os.WriteFile(projectPath, []byte("// old-content"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", projectPath, err)
	}

	scaffolder := New(templaterenderer.NewRenderer(templaterenderer.WithRootDir(outputDir)))
	if _, err := scaffolder.Scaffold(cfg, outputDir, true); err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	projectManifest := readFile(t, projectPath)
	if strings.Contains(projectManifest, "old-content") {
		t.Fatalf("Project.swift was not overwritten:\n%s", projectManifest)
	}
	if !strings.Contains(projectManifest, cfg.BundleID) {
		t.Fatalf("Project.swift missing rendered bundle id %q:\n%s", cfg.BundleID, projectManifest)
	}
}

func TestScaffoldForcePreservesMakefileCustomSection(t *testing.T) {
	t.Parallel()

	cfg := loadConfigFixture(t, "sample-config.json")
	outputDir := t.TempDir()

	scaffolder := New(templaterenderer.NewRenderer(templaterenderer.WithRootDir(outputDir)))
	if _, err := scaffolder.Scaffold(cfg, outputDir, false); err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	makefilePath := filepath.Join(outputDir, "Makefile")
	initialMakefile := readFile(t, makefilePath)
	customTarget := "custom-target: ## Custom workflow\n\t@echo \"custom\"\n"
	if err := os.WriteFile(makefilePath, []byte(initialMakefile+"\n"+customTarget), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", makefilePath, err)
	}

	cfg.ModulesPath = "VendorPackages"
	if _, err := scaffolder.Scaffold(cfg, outputDir, true); err != nil {
		t.Fatalf("Scaffold() force error = %v", err)
	}

	regeneratedMakefile := readFile(t, makefilePath)
	if !strings.Contains(regeneratedMakefile, "MODULES_PATH := VendorPackages") {
		t.Fatalf("regenerated Makefile missing updated modules path:\n%s", regeneratedMakefile)
	}
	if !strings.Contains(regeneratedMakefile, customTarget) {
		t.Fatalf("regenerated Makefile missing preserved custom target:\n%s", regeneratedMakefile)
	}
}

func requireDir(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("path %q is not a directory", path)
	}
}

func requireFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("path %q is a directory, want file", path)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	return string(content)
}

func loadConfigFixture(t *testing.T, fileName string) config.ProjectConfig {
	t.Helper()

	path := filepath.Join(repoRoot(t), "testdata", fileName)
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) error = %v", path, err)
	}
	return cfg
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("failed to find repo root from %q", dir)
		}
		dir = parent
	}
}

func runMakeHelp(t *testing.T, workdir string) string {
	t.Helper()

	if _, err := exec.LookPath("make"); err != nil {
		t.Skipf("make is not available: %v", err)
	}

	cmd := exec.Command("make", "-f", "Makefile", "help")
	cmd.Dir = workdir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make help error = %v\noutput:\n%s", err, string(output))
	}

	return string(output)
}
