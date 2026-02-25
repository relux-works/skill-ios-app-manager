package template

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/testutil"
)

func TestRendererRenderWithSampleConfig(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeLocalPackageManifest(t, projectRoot, "Packages", "Auth")
	writeLocalPackageManifest(t, projectRoot, "Packages", "CoreKit")

	cfg := loadConfigFixture(t, "sample-config.json")
	renderer := NewRenderer(WithRootDir(projectRoot))

	rendered, err := renderer.Render(cfg)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	wantFiles := []string{"Package.swift", "Project.swift", "Tuist.swift", "Workspace.swift"}
	assertRenderedFiles(t, rendered, wantFiles)

	tuistSwift := rendered["Tuist.swift"]
	if !strings.Contains(tuistSwift, "import ProjectDescription") {
		t.Fatalf("Tuist.swift missing ProjectDescription import:\n%s", tuistSwift)
	}
	if !strings.Contains(tuistSwift, "let config = Config(") {
		t.Fatalf("Tuist.swift missing Config declaration:\n%s", tuistSwift)
	}
	if !strings.Contains(tuistSwift, cfg.SwiftVersion) {
		t.Fatalf("Tuist.swift missing SwiftVersion %q:\n%s", cfg.SwiftVersion, tuistSwift)
	}

	projectSwift := rendered["Project.swift"]
	requiredProjectValues := []string{
		cfg.AppName,
		cfg.BundleID,
		cfg.TeamID,
		cfg.OrgName,
		cfg.SwiftVersion,
		cfg.MinTarget,
		cfg.MarketingVersion,
		cfg.ProjectVersion,
		cfg.URLScheme,
		cfg.AppGroups[0],
		`.package(product: "Auth")`,
		`.package(product: "CoreKit")`,
	}
	for _, value := range requiredProjectValues {
		if !strings.Contains(projectSwift, value) {
			t.Fatalf("Project.swift missing %q:\n%s", value, projectSwift)
		}
	}

	workspaceSwift := rendered["Workspace.swift"]
	if !strings.Contains(workspaceSwift, "import ProjectDescription") {
		t.Fatalf("Workspace.swift missing ProjectDescription import:\n%s", workspaceSwift)
	}

	packageSwift := rendered["Package.swift"]
	requiredPackageValues := []string{
		"// swift-tools-version: " + cfg.SwiftVersion,
		".package(path: \"./Packages/Auth\")",
		".package(path: \"./Packages/CoreKit\")",
		cfg.ModulesPath,
	}
	for _, value := range requiredPackageValues {
		if !strings.Contains(packageSwift, value) {
			t.Fatalf("Package.swift missing %q:\n%s", value, packageSwift)
		}
	}

	for name, content := range rendered {
		assertNoTemplateArtifacts(t, name, content)
	}

	assertTemplatesMatchGolden(t, rendered)
}

func TestRendererRenderWithMinimalConfig(t *testing.T) {
	t.Parallel()

	cfg := loadConfigFixture(t, "minimal-config.json")
	renderer := NewRenderer(WithRootDir(t.TempDir()))

	rendered, err := renderer.Render(cfg)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	projectSwift := rendered["Project.swift"]
	if strings.Contains(projectSwift, "CFBundleURLTypes") {
		t.Fatalf("Project.swift should omit URL scheme block for minimal config:\n%s", projectSwift)
	}
	if strings.Contains(projectSwift, "com.apple.security.application-groups") {
		t.Fatalf("Project.swift should omit app groups for minimal config:\n%s", projectSwift)
	}
	if !strings.Contains(projectSwift, `.debug(name: "Debug")`) || !strings.Contains(projectSwift, `.release(name: "Release")`) {
		t.Fatalf("Project.swift missing default configurations:\n%s", projectSwift)
	}

	packageSwift := rendered["Package.swift"]
	if strings.Contains(packageSwift, ".package(path:") {
		t.Fatalf("Package.swift should not include local package dependencies for minimal config:\n%s", packageSwift)
	}
	if !strings.Contains(packageSwift, `let modulesPath = "Packages"`) {
		t.Fatalf("Package.swift missing default modules path:\n%s", packageSwift)
	}

	workspaceSwift := rendered["Workspace.swift"]
	if !strings.Contains(workspaceSwift, cfg.AppName) {
		t.Fatalf("Workspace.swift missing app name %q:\n%s", cfg.AppName, workspaceSwift)
	}

	for name, content := range rendered {
		assertNoTemplateArtifacts(t, name, content)
	}
}

func TestRendererRenderWithFullConfig(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := loadConfigFixture(t, "full-config.json")
	writeLocalPackageManifest(t, projectRoot, cfg.ModulesPath, "AnalyticsKit")
	writeLocalPackageManifest(t, projectRoot, cfg.ModulesPath, "NetworkingKit")

	rendered, err := NewRenderer(WithRootDir(projectRoot)).Render(cfg)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	projectSwift := rendered["Project.swift"]
	fullConfigChecks := []string{
		cfg.AppName,
		cfg.BundleID,
		cfg.TeamID,
		cfg.OrgName,
		cfg.MarketingVersion,
		cfg.ProjectVersion,
		cfg.URLScheme,
		cfg.AppGroups[0],
		cfg.AppGroups[1],
		`.release(name: "Beta")`,
		`.package(product: "AnalyticsKit")`,
		`.package(product: "NetworkingKit")`,
	}
	for _, value := range fullConfigChecks {
		if !strings.Contains(projectSwift, value) {
			t.Fatalf("Project.swift missing %q:\n%s", value, projectSwift)
		}
	}

	workspaceSwift := rendered["Workspace.swift"]
	if !strings.Contains(workspaceSwift, cfg.AppName) {
		t.Fatalf("Workspace.swift missing app name %q:\n%s", cfg.AppName, workspaceSwift)
	}

	packageSwift := rendered["Package.swift"]
	if !strings.Contains(packageSwift, `.package(path: "./VendorPackages/AnalyticsKit")`) {
		t.Fatalf("Package.swift missing AnalyticsKit local package:\n%s", packageSwift)
	}
	if !strings.Contains(packageSwift, `.package(path: "./VendorPackages/NetworkingKit")`) {
		t.Fatalf("Package.swift missing NetworkingKit local package:\n%s", packageSwift)
	}

	for name, content := range rendered {
		assertNoTemplateArtifacts(t, name, content)
	}
}

func TestRendererRenderDefaultsModulesPathAndConfigurations(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeLocalPackageManifest(t, projectRoot, "Packages", "FoundationKit")

	cfg := config.ProjectConfig{
		AppName:          "DemoApp",
		BundleID:         "com.example.demo",
		TeamID:           "ABCDE12345",
		OrgName:          "Example Org",
		MarketingVersion: "1.0.0",
		ProjectVersion:   "1",
		SwiftVersion:     "6.2",
		MinTarget:        "17.0",
	}

	renderer := NewRenderer(WithRootDir(projectRoot))
	rendered, err := renderer.Render(cfg)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	projectSwift := rendered["Project.swift"]
	if !strings.Contains(projectSwift, `.debug(name: "Debug")`) {
		t.Fatalf("Project.swift missing default Debug configuration:\n%s", projectSwift)
	}
	if !strings.Contains(projectSwift, `.release(name: "Release")`) {
		t.Fatalf("Project.swift missing default Release configuration:\n%s", projectSwift)
	}

	workspaceSwift := rendered["Workspace.swift"]
	if !strings.Contains(workspaceSwift, `"DemoApp"`) {
		t.Fatalf("Workspace.swift missing app name:\n%s", workspaceSwift)
	}

	packageSwift := rendered["Package.swift"]
	if !strings.Contains(packageSwift, `.package(path: "./Packages/FoundationKit")`) {
		t.Fatalf("Package.swift missing discovered local package dependency:\n%s", packageSwift)
	}
}

func TestRendererRenderModulesPathReadError(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesPath := filepath.Join(projectRoot, "Packages")
	if err := os.WriteFile(modulesPath, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", modulesPath, err)
	}

	cfg := config.ProjectConfig{
		AppName:          "DemoApp",
		BundleID:         "com.example.demo",
		TeamID:           "ABCDE12345",
		OrgName:          "Example Org",
		MarketingVersion: "1.0.0",
		ProjectVersion:   "1",
		SwiftVersion:     "6.2",
		MinTarget:        "17.0",
		ModulesPath:      "Packages",
	}

	renderer := NewRenderer(WithRootDir(projectRoot))
	_, err := renderer.Render(cfg)
	if err == nil {
		t.Fatal("Render() error = nil, want read modules path error")
	}

	if !strings.Contains(err.Error(), "read modules directory") {
		t.Fatalf("Render() error = %q, want read modules directory failure", err.Error())
	}
}

func assertRenderedFiles(t *testing.T, rendered map[string]string, want []string) {
	t.Helper()

	got := make([]string, 0, len(rendered))
	for name := range rendered {
		got = append(got, name)
	}
	sort.Strings(got)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("rendered files count = %d, want %d (got=%v)", len(got), len(want), got)
	}

	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("rendered file[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func assertNoTemplateArtifacts(t *testing.T, fileName string, content string) {
	t.Helper()

	if strings.Contains(content, "{{") || strings.Contains(content, "}}") {
		t.Fatalf("%s still contains template markers:\n%s", fileName, content)
	}
}

func assertTemplatesMatchGolden(t *testing.T, rendered map[string]string) {
	t.Helper()

	goldenMap := map[string]string{
		"Tuist.swift":     "golden/tuist-swift",
		"Project.swift":   "golden/project-swift",
		"Workspace.swift": "golden/workspace-swift",
		"Package.swift":   "golden/package-swift",
	}
	for fileName, goldenName := range goldenMap {
		content, ok := rendered[fileName]
		if !ok {
			t.Fatalf("rendered output missing %q", fileName)
		}
		testutil.AssertGoldenFile(t, goldenName, content)
	}
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

func writeLocalPackageManifest(t *testing.T, projectRoot, modulesPath, packageName string) {
	t.Helper()

	packageDir := filepath.Join(projectRoot, modulesPath, packageName)
	if err := os.MkdirAll(packageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", packageDir, err)
	}

	manifestPath := filepath.Join(packageDir, "Package.swift")
	if err := os.WriteFile(manifestPath, []byte("// test package\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", manifestPath, err)
	}
}
