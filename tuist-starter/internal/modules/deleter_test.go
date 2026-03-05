package modules_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/modules"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

func TestDeleterDeleteRemovesPackagesAndCleansManifestReferences(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	modulesRoot := filepath.Join(root, "Packages")

	deleterWritePackageManifest(t, modulesRoot, "Auth", nil)
	deleterWritePackageManifest(t, modulesRoot, "AuthImpl", []string{"Auth", "CoreKit"})
	deleterWritePackageManifest(t, modulesRoot, "Feed", []string{"Auth", "AuthImpl", "CoreKit"})

	deleterWriteManifestLikePackage(
		t,
		filepath.Join(root, "Project.swift"),
		"ProjectDependencies",
		[]string{"Auth", "AuthImpl", "CoreKit"},
	)
	deleterWriteManifestLikePackage(
		t,
		filepath.Join(root, "Package.swift"),
		"WorkspacePackages",
		[]string{"Auth", "AuthImpl", "CoreKit"},
	)

	deleter := modules.NewDeleter()
	result, err := deleter.Delete(context.Background(), "Auth", modules.DeleteOptions{
		ModulesPath: modulesRoot,
		ProjectRoot: root,
		Force:       true,
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if result.Module.Name != "Auth" {
		t.Fatalf("result.Module.Name = %q, want %q", result.Module.Name, "Auth")
	}
	if !reflect.DeepEqual(result.Module.PackageNames(), []string{"Auth", "AuthImpl"}) {
		t.Fatalf("result.Module.PackageNames() = %#v, want %#v", result.Module.PackageNames(), []string{"Auth", "AuthImpl"})
	}

	deleterRequireMissingPath(t, filepath.Join(modulesRoot, "Auth"))
	deleterRequireMissingPath(t, filepath.Join(modulesRoot, "AuthImpl"))
	deleterRequireExistingDir(t, filepath.Join(modulesRoot, "Feed"))

	deleterAssertManifestDependencies(
		t,
		filepath.Join(modulesRoot, "Feed", "Package.swift"),
		[]string{"CoreKit"},
	)
	deleterAssertManifestDependencies(
		t,
		filepath.Join(root, "Project.swift"),
		[]string{"CoreKit"},
	)
	deleterAssertManifestDependencies(
		t,
		filepath.Join(root, "Package.swift"),
		[]string{"CoreKit"},
	)

	expectedUpdated := []string{
		filepath.Join(modulesRoot, "Feed", "Package.swift"),
		filepath.Join(root, "Package.swift"),
		filepath.Join(root, "Project.swift"),
	}
	sort.Strings(expectedUpdated)
	if !reflect.DeepEqual(result.UpdatedManifests, expectedUpdated) {
		t.Fatalf("result.UpdatedManifests = %#v, want %#v", result.UpdatedManifests, expectedUpdated)
	}
}

func TestDeleterDeleteCanceledByConfirmation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	modulesRoot := filepath.Join(root, "Packages")
	deleterWritePackageManifest(t, modulesRoot, "Logger", nil)

	confirmCalled := false
	deleter := modules.NewDeleter()
	_, err := deleter.Delete(context.Background(), "Logger", modules.DeleteOptions{
		ModulesPath: modulesRoot,
		ProjectRoot: root,
		Confirm: func(module modules.ModuleInfo) (bool, error) {
			confirmCalled = true
			if module.Name != "Logger" {
				return false, fmt.Errorf("unexpected module for confirmation: %s", module.Name)
			}
			return false, nil
		},
	})
	if !errors.Is(err, modules.ErrDeleteModuleCanceled) {
		t.Fatalf("Delete() error = %v, want %v", err, modules.ErrDeleteModuleCanceled)
	}
	if !confirmCalled {
		t.Fatal("confirmation callback was not called")
	}

	deleterRequireExistingDir(t, filepath.Join(modulesRoot, "Logger"))
}

func deleterWritePackageManifest(t *testing.T, modulesRoot string, packageName string, dependencies []string) {
	t.Helper()
	deleterWriteManifestLikePackage(t, filepath.Join(modulesRoot, packageName, "Package.swift"), packageName, dependencies)
}

func deleterWriteManifestLikePackage(t *testing.T, path string, packageName string, dependencies []string) {
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

func deleterAssertManifestDependencies(t *testing.T, manifestPath string, want []string) {
	t.Helper()

	manifest, err := tuistproj.ReadManifestFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadManifestFile(%q) error = %v", manifestPath, err)
	}

	got := make([]string, 0, len(manifest.Dependencies))
	for _, dependency := range manifest.Dependencies {
		if dependency.Name == "" {
			continue
		}
		got = append(got, dependency.Name)
	}
	sort.Strings(got)
	sort.Strings(want)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dependencies in %q = %#v, want %#v", manifestPath, got, want)
	}
}

func deleterRequireExistingDir(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", path)
	}
}

func deleterRequireMissingPath(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("path %q exists, want missing", path)
	}
}
