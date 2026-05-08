package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

func TestModuleCreateFeature(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	output, err := executeRootCommand(
		"--config",
		configPath,
		"module",
		"create",
		"Auth",
		"--type",
		"feature",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(module create Auth) error = %v", err)
	}

	if !strings.Contains(output, `created module "Auth" of type "feature"`) {
		t.Fatalf("output = %q, want create confirmation", output)
	}

	requireDirExists(t, filepath.Join(projectRoot, "Packages", "Auth"))
	requireDirExists(t, filepath.Join(projectRoot, "Packages", "AuthImpl"))
	requireFileExists(t, filepath.Join(projectRoot, "Packages", "Auth", "Sources", "Auth.swift"))
	requireFileExists(t, filepath.Join(projectRoot, "Packages", "Auth", "Sources", "Module", "Auth.Module+Interface.swift"))
	requireFileExists(t, filepath.Join(projectRoot, "Packages", "AuthImpl", "Sources", "Module", "Auth.Module+Impl.swift"))
}

func TestModuleCreateUsesConfiguredMinTargetByDefault(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	cfg := testProjectConfig()
	cfg.MinTarget = "16.0"
	configPath := writeModuleConfig(t, projectRoot, cfg)

	_, err := executeRootCommand(
		"--config",
		configPath,
		"module",
		"create",
		"Logger",
		"--type",
		"utility",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(module create Logger) error = %v", err)
	}

	manifest := readFileString(t, filepath.Join(projectRoot, "Packages", "Logger", "Package.swift"))
	if !strings.Contains(manifest, `.iOS(.v16)`) {
		t.Fatalf("Package.swift missing configured iOS min target:\n%s", manifest)
	}
	if strings.Contains(manifest, `.iOS(.v17)`) {
		t.Fatalf("Package.swift kept stale default iOS min target:\n%s", manifest)
	}
}

func TestModuleCreateAcceptsPlatformTuples(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	_, err := executeRootCommand(
		"--config",
		configPath,
		"module",
		"create",
		"Logger",
		"--type",
		"utility",
		"--platform",
		"iOS:16.0",
		"--platform",
		"macOS:13.0",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(module create Logger) error = %v", err)
	}

	manifest := readFileString(t, filepath.Join(projectRoot, "Packages", "Logger", "Package.swift"))
	for _, want := range []string{`.iOS(.v16)`, `.macOS(.v13)`} {
		if !strings.Contains(manifest, want) {
			t.Fatalf("Package.swift missing %q:\n%s", want, manifest)
		}
	}
}

func TestModuleCreateRejectsEmptyPlatformTuple(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	_, err := executeRootCommand(
		"--config",
		configPath,
		"module",
		"create",
		"Logger",
		"--type",
		"utility",
		"--platform",
		"",
	)
	if err == nil {
		t.Fatal("executeRootCommand(module create --platform '') error = nil, want error")
	}
	if !strings.Contains(err.Error(), "module platform tuple #1 is empty") {
		t.Fatalf("error = %q, want empty tuple message", err.Error())
	}
}

func TestModuleCreateRejectsUnknownPlatform(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	_, err := executeRootCommand(
		"--config",
		configPath,
		"module",
		"create",
		"Logger",
		"--type",
		"utility",
		"--platform",
		"linux:1.0",
	)
	if err == nil {
		t.Fatal("executeRootCommand(module create --platform linux:1.0) error = nil, want error")
	}
	for _, want := range []string{
		"module platform tuple #1",
		"unsupported platform",
		"supported platforms: iOS, macOS, tvOS, watchOS, visionOS",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	}
}

func TestModuleCreateUtility(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	_, err := executeRootCommand(
		"--config",
		configPath,
		"module",
		"create",
		"Logger",
		"--type",
		"utility",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(module create Logger) error = %v", err)
	}

	requireDirExists(t, filepath.Join(projectRoot, "Packages", "Logger"))
	requireDirExists(t, filepath.Join(projectRoot, "Packages", "Logger", "Sources"))
	requirePathMissing(t, filepath.Join(projectRoot, "Packages", "LoggerImpl"))
	requirePathMissing(t, filepath.Join(projectRoot, "Packages", "Logger", "Sources", "Logger.swift"))
}

func TestModuleCreateValidatesType(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	_, err := executeRootCommand(
		"--config",
		configPath,
		"module",
		"create",
		"Auth",
		"--type",
		"product",
	)
	if err == nil {
		t.Fatal("executeRootCommand(module create --type product) error = nil, want error")
	}
	if !strings.Contains(err.Error(), `unknown module type "product"`) {
		t.Fatalf("error = %q, want unknown type message", err.Error())
	}
}

func TestModuleCreateValidatesName(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	_, err := executeRootCommand(
		"--config",
		configPath,
		"module",
		"create",
		"auth",
		"--type",
		"feature",
	)
	if err == nil {
		t.Fatal("executeRootCommand(module create auth) error = nil, want error")
	}
	if !strings.Contains(err.Error(), "PascalCase") {
		t.Fatalf("error = %q, want PascalCase message", err.Error())
	}
}

func TestModuleCreateDetectsConflicts(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	_, err := executeRootCommand(
		"--config",
		configPath,
		"module",
		"create",
		"Auth",
		"--type",
		"feature",
	)
	if err != nil {
		t.Fatalf("first create error = %v", err)
	}

	_, err = executeRootCommand(
		"--config",
		configPath,
		"module",
		"create",
		"Auth",
		"--type",
		"feature",
	)
	if err == nil {
		t.Fatal("second create error = nil, want conflict error")
	}
	if !strings.Contains(err.Error(), "module package already exists") {
		t.Fatalf("error = %q, want conflict message", err.Error())
	}
}

func TestModuleListDisplaysTable(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	if _, err := executeRootCommand("--config", configPath, "module", "create", "Auth", "--type", "feature"); err != nil {
		t.Fatalf("create Auth error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "module", "create", "Logger", "--type", "utility"); err != nil {
		t.Fatalf("create Logger error = %v", err)
	}

	output, err := executeRootCommand("--config", configPath, "module", "list")
	if err != nil {
		t.Fatalf("executeRootCommand(module list) error = %v", err)
	}

	for _, expected := range []string{
		"NAME",
		"TYPE",
		"PACKAGES",
		"DEPS",
		"Auth",
		"feature",
		"Auth, AuthImpl",
		"Logger",
		"utility",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("module list output missing %q:\n%s", expected, output)
		}
	}
}

func TestModuleDeleteForceRemovesPackagesAndReferences(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	if _, err := executeRootCommand("--config", configPath, "module", "create", "Auth", "--type", "feature"); err != nil {
		t.Fatalf("create Auth error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "module", "create", "Feed", "--type", "utility"); err != nil {
		t.Fatalf("create Feed error = %v", err)
	}

	moduleWriteManifestLikePackage(
		t,
		filepath.Join(projectRoot, "Packages", "Feed", "Package.swift"),
		"Feed",
		[]string{"Auth", "AuthImpl", "CoreKit"},
	)
	moduleWriteManifestLikePackage(
		t,
		filepath.Join(projectRoot, "Project.swift"),
		"ProjectDependencies",
		[]string{"Auth", "AuthImpl", "CoreKit"},
	)
	moduleWriteManifestLikePackage(
		t,
		filepath.Join(projectRoot, "Package.swift"),
		"WorkspacePackages",
		[]string{"Auth", "AuthImpl", "CoreKit"},
	)

	output, err := executeRootCommand("--config", configPath, "module", "delete", "Auth", "--force")
	if err != nil {
		t.Fatalf("executeRootCommand(module delete --force) error = %v", err)
	}
	if !strings.Contains(output, `deleted module "Auth"`) {
		t.Fatalf("output = %q, want delete confirmation", output)
	}

	requirePathMissing(t, filepath.Join(projectRoot, "Packages", "Auth"))
	requirePathMissing(t, filepath.Join(projectRoot, "Packages", "AuthImpl"))
	requireDirExists(t, filepath.Join(projectRoot, "Packages", "Feed"))

	moduleAssertManifestDependencies(
		t,
		filepath.Join(projectRoot, "Packages", "Feed", "Package.swift"),
		[]string{"CoreKit"},
	)
	moduleAssertManifestDependencies(
		t,
		filepath.Join(projectRoot, "Project.swift"),
		[]string{"CoreKit"},
	)
	moduleAssertManifestDependencies(
		t,
		filepath.Join(projectRoot, "Package.swift"),
		[]string{"CoreKit"},
	)
}

func TestModuleDeleteConfirmationCancel(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	if _, err := executeRootCommand("--config", configPath, "module", "create", "Logger", "--type", "utility"); err != nil {
		t.Fatalf("create Logger error = %v", err)
	}

	output, err := executeRootCommandWithInput(
		"n\n",
		"--config",
		configPath,
		"module",
		"delete",
		"Logger",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(module delete Logger) error = %v", err)
	}

	if !strings.Contains(output, "module delete canceled") {
		t.Fatalf("output = %q, want cancel message", output)
	}
	requireDirExists(t, filepath.Join(projectRoot, "Packages", "Logger"))
}

func writeModuleConfig(t *testing.T, projectRoot string, cfg config.ProjectConfig) string {
	t.Helper()

	configPath := filepath.Join(projectRoot, config.DefaultConfigPath)
	if err := config.WriteProjectConfig(configPath, cfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", configPath, err)
	}
	return configPath
}

func requireDirExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", path)
	}
}

func requireFileExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("%q is a directory, want file", path)
	}
}

func requirePathMissing(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("path %q exists, want missing", path)
	}
}

func moduleWriteManifestLikePackage(t *testing.T, path string, packageName string, dependencies []string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	dependenciesBlock := ""
	for _, dependency := range dependencies {
		dependenciesBlock += fmt.Sprintf("        .package(path: \"Packages/%s\"),\n", dependency)
	}

	content := fmt.Sprintf(`// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "%s",
    products: [
        .library(name: "%s", type: .dynamic, targets: ["%s"]),
    ],
    dependencies: [
%s    ],
    targets: [
        .target(name: "%s"),
    ]
)
`, packageName, packageName, packageName, dependenciesBlock, packageName)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func moduleAssertManifestDependencies(t *testing.T, path string, want []string) {
	t.Helper()

	manifest, err := tuistproj.ReadManifestFile(path)
	if err != nil {
		t.Fatalf("ReadManifestFile(%q) error = %v", path, err)
	}

	got := make([]string, 0, len(manifest.Dependencies))
	for _, dependency := range manifest.Dependencies {
		if strings.TrimSpace(dependency.Name) == "" {
			continue
		}
		got = append(got, dependency.Name)
	}

	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dependencies(%q) = %#v, want %#v", path, got, want)
	}
}
