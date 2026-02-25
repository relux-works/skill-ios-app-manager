package deps

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

func TestAddInternalDepAddsPackageAndProduct(t *testing.T) {
	t.Parallel()

	modulesRoot := filepath.Join(t.TempDir(), "Packages")
	writeInterfaceModuleManifest(t, modulesRoot, "Auth", nil)
	writeInterfaceModuleManifest(t, modulesRoot, "CoreKit", nil)

	if err := AddInternalDep("Auth", "CoreKit", modulesRoot); err != nil {
		t.Fatalf("AddInternalDep(Auth, CoreKit) error = %v", err)
	}

	manifest := readStringFile(t, filepath.Join(modulesRoot, "Auth", moduleManifestName))
	if !strings.Contains(manifest, `.package(path: "../CoreKit"),`) {
		t.Fatalf("Auth Package.swift missing package dependency:\n%s", manifest)
	}
	if !strings.Contains(manifest, `.product(name: "CoreKit", package: "CoreKit"),`) {
		t.Fatalf("Auth Package.swift missing target dependency:\n%s", manifest)
	}
}

func TestAddInternalDepDetectsCircularDependency(t *testing.T) {
	t.Parallel()

	modulesRoot := filepath.Join(t.TempDir(), "Packages")
	writeInterfaceModuleManifest(t, modulesRoot, "Auth", nil)
	writeInterfaceModuleManifest(t, modulesRoot, "CoreKit", []string{"Auth"})

	before := readStringFile(t, filepath.Join(modulesRoot, "Auth", moduleManifestName))
	err := AddInternalDep("Auth", "CoreKit", modulesRoot)
	if err == nil {
		t.Fatal("AddInternalDep(Auth, CoreKit) error = nil, want cycle error")
	}
	if !strings.Contains(err.Error(), "circular dependency: Auth → CoreKit → Auth") {
		t.Fatalf("error = %q, want cycle path", err.Error())
	}

	after := readStringFile(t, filepath.Join(modulesRoot, "Auth", moduleManifestName))
	if after != before {
		t.Fatalf("Auth Package.swift changed on rejected cycle:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestAddInternalDepRejectsImplementationPackage(t *testing.T) {
	t.Parallel()

	modulesRoot := filepath.Join(t.TempDir(), "Packages")
	writeInterfaceModuleManifest(t, modulesRoot, "Auth", nil)

	err := AddInternalDep("Auth", "CoreKitImpl", modulesRoot)
	if err == nil {
		t.Fatal("AddInternalDep(Auth, CoreKitImpl) error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "interface package") {
		t.Fatalf("error = %q, want interface package validation", err.Error())
	}
}

func TestRemoveInternalDepRemovesPackageAndProduct(t *testing.T) {
	t.Parallel()

	modulesRoot := filepath.Join(t.TempDir(), "Packages")
	writeInterfaceModuleManifest(t, modulesRoot, "Auth", []string{"CoreKit"})
	writeInterfaceModuleManifest(t, modulesRoot, "CoreKit", nil)

	if err := RemoveInternalDep("Auth", "CoreKit", modulesRoot); err != nil {
		t.Fatalf("RemoveInternalDep(Auth, CoreKit) error = %v", err)
	}

	manifest := readStringFile(t, filepath.Join(modulesRoot, "Auth", moduleManifestName))
	if strings.Contains(manifest, `.package(path: "../CoreKit"),`) {
		t.Fatalf("Auth Package.swift still contains package dependency:\n%s", manifest)
	}
	if strings.Contains(manifest, `.product(name: "CoreKit", package: "CoreKit"),`) {
		t.Fatalf("Auth Package.swift still contains target dependency:\n%s", manifest)
	}
}

func TestListInternalDeps(t *testing.T) {
	t.Parallel()

	modulesRoot := filepath.Join(t.TempDir(), "Packages")
	writeInterfaceModuleManifest(t, modulesRoot, "Auth", []string{"CoreKit", "Analytics"})
	writeInterfaceModuleManifest(t, modulesRoot, "CoreKit", nil)
	writeInterfaceModuleManifest(t, modulesRoot, "Analytics", nil)

	allDeps, err := ListInternalDeps("", modulesRoot)
	if err != nil {
		t.Fatalf("ListInternalDeps(\"\") error = %v", err)
	}

	if got := allDeps["Auth"]; !reflect.DeepEqual(got, []string{"Analytics", "CoreKit"}) {
		t.Fatalf(`allDeps["Auth"] = %#v, want %#v`, got, []string{"Analytics", "CoreKit"})
	}
	if got := allDeps["CoreKit"]; !reflect.DeepEqual(got, []string{}) {
		t.Fatalf(`allDeps["CoreKit"] = %#v, want empty slice`, got)
	}
	if got := allDeps["Analytics"]; !reflect.DeepEqual(got, []string{}) {
		t.Fatalf(`allDeps["Analytics"] = %#v, want empty slice`, got)
	}

	authDeps, err := ListInternalDeps("Auth", modulesRoot)
	if err != nil {
		t.Fatalf("ListInternalDeps(Auth) error = %v", err)
	}
	if len(authDeps) != 1 {
		t.Fatalf("len(authDeps) = %d, want 1", len(authDeps))
	}
	if got := authDeps["Auth"]; !reflect.DeepEqual(got, []string{"Analytics", "CoreKit"}) {
		t.Fatalf(`authDeps["Auth"] = %#v, want %#v`, got, []string{"Analytics", "CoreKit"})
	}
}

func writeInterfaceModuleManifest(t *testing.T, modulesRoot string, moduleName string, dependencies []string) {
	t.Helper()

	packagePath := filepath.Join(modulesRoot, moduleName)
	if err := os.MkdirAll(packagePath, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", packagePath, err)
	}

	manifest, err := tuistproj.GeneratePackageSwift(tuistproj.PackageGenerationInput{
		ModuleName:   moduleName,
		Type:         tuistproj.PackageTypeInterface,
		Dependencies: dependencies,
		Platform:     "iOS(.v16)",
	})
	if err != nil {
		t.Fatalf("GeneratePackageSwift(%q) error = %v", moduleName, err)
	}

	manifestPath := filepath.Join(packagePath, moduleManifestName)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", manifestPath, err)
	}
}

func readStringFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}
