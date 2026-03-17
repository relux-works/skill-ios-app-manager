package deps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddExternalDepAddsProjectAndModuleDependency(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesRoot := filepath.Join(projectRoot, "Packages")
	writeProjectDependencyManifest(t, projectRoot)
	writeInterfaceModuleManifest(t, modulesRoot, "Auth", nil)

	url := "https://github.com/realm/realm-swift.git"
	if err := AddExternalDep(url, "1.0.0", "RealmSwift", "Auth", modulesRoot); err != nil {
		t.Fatalf("AddExternalDep(...) error = %v", err)
	}

	projectManifest := readStringFile(t, filepath.Join(projectRoot, moduleManifestName))
	projectSnippet := `.package(name: "RealmSwift", url: "https://github.com/realm/realm-swift.git", from: "1.0.0"),`
	if !strings.Contains(projectManifest, projectSnippet) {
		t.Fatalf("project Package.swift missing external dependency:\n%s", projectManifest)
	}
	if !strings.Contains(projectManifest, `"RealmSwift": .framework`) {
		t.Fatalf("project Package.swift missing framework product type override:\n%s", projectManifest)
	}

	authManifest := readStringFile(t, filepath.Join(modulesRoot, "Auth", moduleManifestName))
	if !strings.Contains(authManifest, projectSnippet) {
		t.Fatalf("Auth Package.swift missing external package dependency:\n%s", authManifest)
	}
	if !strings.Contains(authManifest, `.product(name: "RealmSwift", package: "RealmSwift"),`) {
		t.Fatalf("Auth Package.swift missing target product dependency:\n%s", authManifest)
	}
}

func TestAddExternalDepVersionRequirements(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		version    string
		wantClause string
	}{
		{name: "default from", version: "1.0.0", wantClause: `from: "1.0.0"`},
		{name: "explicit from", version: `from: "2.0.0"`, wantClause: `from: "2.0.0"`},
		{name: "exact", version: `exact: "3.1.4"`, wantClause: `exact: "3.1.4"`},
		{name: "branch", version: `branch: "main"`, wantClause: `branch: "main"`},
		{name: "revision", version: `revision: "abc123"`, wantClause: `revision: "abc123"`},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			projectRoot := t.TempDir()
			modulesRoot := filepath.Join(projectRoot, "Packages")
			writeProjectDependencyManifest(t, projectRoot)

			packageName := "SDK" + strings.ReplaceAll(tc.name, " ", "")
			url := fmt.Sprintf("https://github.com/example/%s.git", strings.ToLower(packageName))
			if err := AddExternalDep(url, tc.version, packageName, "", modulesRoot); err != nil {
				t.Fatalf("AddExternalDep(...) error = %v", err)
			}

			projectManifest := readStringFile(t, filepath.Join(projectRoot, moduleManifestName))
			wantSnippet := fmt.Sprintf(
				`.package(name: "%s", url: "%s", %s),`,
				packageName,
				url,
				tc.wantClause,
			)
			if !strings.Contains(projectManifest, wantSnippet) {
				t.Fatalf("project Package.swift missing version clause %q:\n%s", tc.wantClause, projectManifest)
			}
		})
	}
}

func TestAddExternalDepInfersPackageNameFromURL(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesRoot := filepath.Join(projectRoot, "Packages")
	writeProjectDependencyManifest(t, projectRoot)

	url := "https://github.com/apple/swift-collections.git"
	if err := AddExternalDep(url, "1.0.0", "", "", modulesRoot); err != nil {
		t.Fatalf("AddExternalDep(url, version, empty packageName, ...) error = %v", err)
	}

	projectManifest := readStringFile(t, filepath.Join(projectRoot, moduleManifestName))
	if !strings.Contains(projectManifest, `.package(name: "swift-collections", url: "https://github.com/apple/swift-collections.git", from: "1.0.0"),`) {
		t.Fatalf("project Package.swift missing inferred package name:\n%s", projectManifest)
	}
}

func TestRemoveExternalDepRemovesProjectAndModuleDependency(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesRoot := filepath.Join(projectRoot, "Packages")
	writeProjectDependencyManifest(t, projectRoot)
	writeInterfaceModuleManifest(t, modulesRoot, "Auth", nil)

	url := "https://github.com/realm/realm-swift.git"
	if err := AddExternalDep(url, "1.0.0", "RealmSwift", "Auth", modulesRoot); err != nil {
		t.Fatalf("AddExternalDep(...) error = %v", err)
	}

	if err := RemoveExternalDep("RealmSwift", modulesRoot); err != nil {
		t.Fatalf("RemoveExternalDep(RealmSwift) error = %v", err)
	}

	projectManifest := readStringFile(t, filepath.Join(projectRoot, moduleManifestName))
	if strings.Contains(projectManifest, `.package(name: "RealmSwift", url: "https://github.com/realm/realm-swift.git", from: "1.0.0"),`) {
		t.Fatalf("project Package.swift still contains RealmSwift dependency:\n%s", projectManifest)
	}
	if strings.Contains(projectManifest, `"RealmSwift": .framework`) {
		t.Fatalf("project Package.swift still contains RealmSwift framework override:\n%s", projectManifest)
	}

	authManifest := readStringFile(t, filepath.Join(modulesRoot, "Auth", moduleManifestName))
	if strings.Contains(authManifest, `.package(name: "RealmSwift", url: "https://github.com/realm/realm-swift.git", from: "1.0.0"),`) {
		t.Fatalf("Auth Package.swift still contains external package dependency:\n%s", authManifest)
	}
	if strings.Contains(authManifest, `.product(name: "RealmSwift", package: "RealmSwift"),`) {
		t.Fatalf("Auth Package.swift still contains target product dependency:\n%s", authManifest)
	}
}

func TestListExternalDepsScansRootAndModules(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesRoot := filepath.Join(projectRoot, "Packages")
	writeProjectDependencyManifest(t, projectRoot)
	writeInterfaceModuleManifest(t, modulesRoot, "Auth", nil)
	writeInterfaceModuleManifest(t, modulesRoot, "Payments", nil)

	if err := AddExternalDep(
		"https://github.com/realm/realm-swift.git",
		"1.0.0",
		"RealmSwift",
		"Auth",
		modulesRoot,
	); err != nil {
		t.Fatalf("AddExternalDep(RealmSwift) error = %v", err)
	}
	if err := AddExternalDep(
		"https://github.com/pointfreeco/swift-snapshot-testing.git",
		`branch: "main"`,
		"SnapshotTesting",
		"",
		modulesRoot,
	); err != nil {
		t.Fatalf("AddExternalDep(SnapshotTesting) error = %v", err)
	}
	if err := AddExternalDep(
		"https://github.com/getsentry/sentry-cocoa.git",
		`exact: "8.0.0"`,
		"Sentry",
		"Payments",
		modulesRoot,
	); err != nil {
		t.Fatalf("AddExternalDep(Sentry) error = %v", err)
	}

	dependencies, err := ListExternalDeps(modulesRoot)
	if err != nil {
		t.Fatalf("ListExternalDeps error = %v", err)
	}

	if len(dependencies) != 5 {
		t.Fatalf("len(ListExternalDeps) = %d, want 5", len(dependencies))
	}

	assertExternalDependencyPresent(
		t,
		dependencies,
		externalDependencyScopeRoot,
		"RealmSwift",
		"https://github.com/realm/realm-swift.git",
		`from: "1.0.0"`,
	)
	assertExternalDependencyPresent(
		t,
		dependencies,
		"Auth",
		"RealmSwift",
		"https://github.com/realm/realm-swift.git",
		`from: "1.0.0"`,
	)
	assertExternalDependencyPresent(
		t,
		dependencies,
		externalDependencyScopeRoot,
		"SnapshotTesting",
		"https://github.com/pointfreeco/swift-snapshot-testing.git",
		`branch: "main"`,
	)
	assertExternalDependencyPresent(
		t,
		dependencies,
		externalDependencyScopeRoot,
		"Sentry",
		"https://github.com/getsentry/sentry-cocoa.git",
		`exact: "8.0.0"`,
	)
	assertExternalDependencyPresent(
		t,
		dependencies,
		"Payments",
		"Sentry",
		"https://github.com/getsentry/sentry-cocoa.git",
		`exact: "8.0.0"`,
	)
}

func assertExternalDependencyPresent(
	t *testing.T,
	dependencies []ExternalDependency,
	scope string,
	packageName string,
	url string,
	requirement string,
) {
	t.Helper()

	for _, dependency := range dependencies {
		if dependency.Scope != scope {
			continue
		}
		if dependency.PackageName != packageName {
			continue
		}
		if dependency.URL != url {
			continue
		}
		if dependency.Requirement != requirement {
			continue
		}
		return
	}

	t.Fatalf(
		"dependency not found: scope=%q package=%q url=%q requirement=%q, got %#v",
		scope,
		packageName,
		url,
		requirement,
		dependencies,
	)
}

func writeProjectDependencyManifest(t *testing.T, projectRoot string) {
	t.Helper()

	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", projectRoot, err)
	}

	manifestPath := filepath.Join(projectRoot, moduleManifestName)
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
