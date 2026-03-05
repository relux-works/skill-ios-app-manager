package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDepAddRemoveAndList(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())
	writeProjectDependencyManifestForDepTest(t, projectRoot)

	if _, err := executeRootCommand("--config", configPath, "module", "create", "Auth", "--type", "utility"); err != nil {
		t.Fatalf("create Auth error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "module", "create", "CoreKit", "--type", "utility"); err != nil {
		t.Fatalf("create CoreKit error = %v", err)
	}

	addOutput, err := executeRootCommand(
		"--config",
		configPath,
		"dep",
		"add",
		"Auth",
		"--depends-on",
		"CoreKit",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(dep add) error = %v", err)
	}
	if !strings.Contains(addOutput, `added dependency "Auth" -> "CoreKit"`) {
		t.Fatalf("add output = %q, want success message", addOutput)
	}

	authManifestPath := filepath.Join(projectRoot, "Packages", "Auth", "Package.swift")
	moduleAssertManifestDependencies(t, authManifestPath, []string{"CoreKit"})
	authManifest := readFileString(t, authManifestPath)
	if !strings.Contains(authManifest, `.product(name: "CoreKit", package: "CoreKit"),`) {
		t.Fatalf("Auth manifest missing product dependency for CoreKit:\n%s", authManifest)
	}

	listOutput, err := executeRootCommand("--config", configPath, "dep", "list", "Auth")
	if err != nil {
		t.Fatalf("executeRootCommand(dep list Auth) error = %v", err)
	}
	for _, expected := range []string{"MODULE", "DEPENDS_ON", "Auth", "CoreKit"} {
		if !strings.Contains(listOutput, expected) {
			t.Fatalf("dep list output missing %q:\n%s", expected, listOutput)
		}
	}

	removeOutput, err := executeRootCommand(
		"--config",
		configPath,
		"dep",
		"remove",
		"Auth",
		"--depends-on",
		"CoreKit",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(dep remove) error = %v", err)
	}
	if !strings.Contains(removeOutput, `removed dependency "Auth" -> "CoreKit"`) {
		t.Fatalf("remove output = %q, want success message", removeOutput)
	}

	moduleAssertManifestDependencies(t, authManifestPath, []string{})
	authManifest = readFileString(t, authManifestPath)
	if strings.Contains(authManifest, `.product(name: "CoreKit", package: "CoreKit"),`) {
		t.Fatalf("Auth manifest still contains product dependency for CoreKit:\n%s", authManifest)
	}
}

func TestDepAddDetectsCircularDependency(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	if _, err := executeRootCommand("--config", configPath, "module", "create", "Auth", "--type", "utility"); err != nil {
		t.Fatalf("create Auth error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "module", "create", "CoreKit", "--type", "utility"); err != nil {
		t.Fatalf("create CoreKit error = %v", err)
	}

	if _, err := executeRootCommand(
		"--config",
		configPath,
		"dep",
		"add",
		"CoreKit",
		"--depends-on",
		"Auth",
	); err != nil {
		t.Fatalf("executeRootCommand(dep add CoreKit -> Auth) error = %v", err)
	}

	_, err := executeRootCommand(
		"--config",
		configPath,
		"dep",
		"add",
		"Auth",
		"--depends-on",
		"CoreKit",
	)
	if err == nil {
		t.Fatal("executeRootCommand(dep add Auth -> CoreKit) error = nil, want cycle error")
	}
	if !strings.Contains(err.Error(), "circular dependency: Auth → CoreKit → Auth") {
		t.Fatalf("error = %q, want cycle path", err.Error())
	}
}

func TestDepListAllShowsAllModules(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())

	if _, err := executeRootCommand("--config", configPath, "module", "create", "Auth", "--type", "utility"); err != nil {
		t.Fatalf("create Auth error = %v", err)
	}
	if _, err := executeRootCommand("--config", configPath, "module", "create", "CoreKit", "--type", "utility"); err != nil {
		t.Fatalf("create CoreKit error = %v", err)
	}
	if _, err := executeRootCommand(
		"--config",
		configPath,
		"dep",
		"add",
		"Auth",
		"--depends-on",
		"CoreKit",
	); err != nil {
		t.Fatalf("executeRootCommand(dep add Auth -> CoreKit) error = %v", err)
	}

	output, err := executeRootCommand("--config", configPath, "dep", "list")
	if err != nil {
		t.Fatalf("executeRootCommand(dep list) error = %v", err)
	}

	for _, expected := range []string{"MODULE", "DEPENDS_ON", "Auth", "CoreKit"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("dep list output missing %q:\n%s", expected, output)
		}
	}
}

func TestDepAddExternalRemoveExternalAndList(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	configPath := writeModuleConfig(t, projectRoot, testProjectConfig())
	writeProjectDependencyManifestForDepTest(t, projectRoot)

	if _, err := executeRootCommand("--config", configPath, "module", "create", "Auth", "--type", "utility"); err != nil {
		t.Fatalf("create Auth error = %v", err)
	}

	addOutput, err := executeRootCommand(
		"--config",
		configPath,
		"dep",
		"add-external",
		"--url",
		"https://github.com/apple/swift-collections.git",
		"--version",
		`branch: "main"`,
		"--module",
		"Auth",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(dep add-external) error = %v", err)
	}
	if !strings.Contains(addOutput, `added external dependency "https://github.com/apple/swift-collections.git"`) {
		t.Fatalf("add-external output = %q, want success message", addOutput)
	}

	projectManifestPath := filepath.Join(projectRoot, "Package.swift")
	projectManifest := readFileString(t, projectManifestPath)
	if !strings.Contains(
		projectManifest,
		`.package(name: "swift-collections", url: "https://github.com/apple/swift-collections.git", branch: "main"),`,
	) {
		t.Fatalf("project manifest missing external dependency:\n%s", projectManifest)
	}

	authManifestPath := filepath.Join(projectRoot, "Packages", "Auth", "Package.swift")
	authManifest := readFileString(t, authManifestPath)
	if !strings.Contains(
		authManifest,
		`.package(name: "swift-collections", url: "https://github.com/apple/swift-collections.git", branch: "main"),`,
	) {
		t.Fatalf("Auth manifest missing external dependency:\n%s", authManifest)
	}
	if !strings.Contains(authManifest, `.product(name: "swift-collections", package: "swift-collections"),`) {
		t.Fatalf("Auth manifest missing external target product dependency:\n%s", authManifest)
	}

	listOutput, err := executeRootCommand("--config", configPath, "dep", "list")
	if err != nil {
		t.Fatalf("executeRootCommand(dep list) error = %v", err)
	}
	for _, expected := range []string{
		"MODULE",
		"DEPENDS_ON",
		"SCOPE",
		"PACKAGE",
		"VERSION",
		"URL",
		"swift-collections",
		`branch: "main"`,
	} {
		if !strings.Contains(listOutput, expected) {
			t.Fatalf("dep list output missing %q:\n%s", expected, listOutput)
		}
	}

	removeOutput, err := executeRootCommand(
		"--config",
		configPath,
		"dep",
		"remove-external",
		"--package",
		"swift-collections",
	)
	if err != nil {
		t.Fatalf("executeRootCommand(dep remove-external) error = %v", err)
	}
	if !strings.Contains(removeOutput, `removed external dependency "swift-collections"`) {
		t.Fatalf("remove-external output = %q, want success message", removeOutput)
	}

	projectManifest = readFileString(t, projectManifestPath)
	if strings.Contains(projectManifest, `.package(name: "swift-collections",`) {
		t.Fatalf("project manifest still contains swift-collections:\n%s", projectManifest)
	}

	authManifest = readFileString(t, authManifestPath)
	if strings.Contains(authManifest, `.package(name: "swift-collections",`) {
		t.Fatalf("Auth manifest still contains swift-collections package dependency:\n%s", authManifest)
	}
	if strings.Contains(authManifest, `.product(name: "swift-collections", package: "swift-collections"),`) {
		t.Fatalf("Auth manifest still contains swift-collections product dependency:\n%s", authManifest)
	}

	postRemoveListOutput, err := executeRootCommand("--config", configPath, "dep", "list")
	if err != nil {
		t.Fatalf("executeRootCommand(dep list after remove-external) error = %v", err)
	}
	if !strings.Contains(postRemoveListOutput, "no external dependencies found") {
		t.Fatalf("dep list after remove-external missing empty-state message:\n%s", postRemoveListOutput)
	}
}

func writeProjectDependencyManifestForDepTest(t *testing.T, projectRoot string) {
	t.Helper()

	manifestPath := filepath.Join(projectRoot, "Package.swift")
	content := `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "AppDependencies",
    dependencies: [
    ],
    targets: []
)
`

	if err := os.WriteFile(manifestPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", manifestPath, err)
	}
}
