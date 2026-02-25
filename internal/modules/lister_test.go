package modules_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/modules"
)

func TestListerListDetectsTypesAndDependencies(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	modulesRoot := filepath.Join(root, "Packages")

	listerWritePackageManifest(t, modulesRoot, "Auth", nil)
	listerWritePackageManifest(t, modulesRoot, "AuthImpl", []string{"Auth", "CoreKit"})
	listerWriteReluxMarkers(t, modulesRoot, "Auth", []string{
		"AuthImpl/Sources/store.swift",
		"AuthImpl/Sources/reducer.swift",
		"AuthImpl/Sources/actions.swift",
		"AuthImpl/Sources/state.swift",
		"AuthImpl/Sources/middleware.swift",
		"AuthImpl/Sources/view.swift",
		"AuthInterface/Sources/protocol.swift",
		"AuthInterface/Sources/dto.swift",
	})

	listerWritePackageManifest(t, modulesRoot, "Billing", nil)
	listerWritePackageManifest(t, modulesRoot, "BillingImpl", []string{"Billing", "Auth", "SharedSpace"})
	listerWriteReluxMarkers(t, modulesRoot, "Billing", []string{
		"BillingImpl/Sources/store.swift",
		"BillingImpl/Sources/reducer.swift",
		"BillingImpl/Sources/actions.swift",
		"BillingImpl/Sources/state.swift",
		"BillingImpl/Sources/middleware.swift",
		"BillingInterface/Sources/protocol.swift",
	})

	listerWritePackageManifest(t, modulesRoot, "SharedSpace", nil)
	listerWritePackageManifest(t, modulesRoot, "SharedSpaceImpl", []string{"SharedSpace"})
	listerWriteReluxMarkers(t, modulesRoot, "SharedSpace", []string{
		"SharedSpaceInterface/Sources/protocol.swift",
		"SharedSpaceInterface/Sources/dto.swift",
		"SharedSpaceImpl/Sources/ioc_registration.swift",
		"SharedSpaceImpl/Sources/ioc_resolver.swift",
	})

	listerWritePackageManifest(t, modulesRoot, "Design", nil)
	listerWritePackageManifest(t, modulesRoot, "DesignImpl", []string{"Design"})

	listerWritePackageManifest(t, modulesRoot, "Logger", []string{"Auth"})

	if err := os.MkdirAll(filepath.Join(modulesRoot, ".hidden"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.hidden) error = %v", err)
	}

	lister := modules.NewLister()
	modulesList, err := lister.List(context.Background(), modulesRoot)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(modulesList) != 5 {
		t.Fatalf("len(modulesList) = %d, want %d", len(modulesList), 5)
	}

	byName := make(map[string]modules.ModuleInfo, len(modulesList))
	for _, moduleInfo := range modulesList {
		byName[moduleInfo.Name] = moduleInfo
	}

	listerAssertTypeAndDeps(t, byName["Auth"], modules.ModuleTypeFeature, 1, true)
	listerAssertTypeAndDeps(t, byName["Billing"], modules.ModuleTypeKit, 2, true)
	listerAssertTypeAndDeps(t, byName["SharedSpace"], modules.ModuleTypeShared, 0, true)
	listerAssertTypeAndDeps(t, byName["Design"], modules.ModuleTypeUI, 0, true)
	listerAssertTypeAndDeps(t, byName["Logger"], modules.ModuleTypeUtility, 1, false)
}

func TestListerListReturnsEmptyWhenModulesPathMissing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	modulesRoot := filepath.Join(root, "Packages")

	lister := modules.NewLister()
	modulesList, err := lister.List(context.Background(), modulesRoot)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(modulesList) != 0 {
		t.Fatalf("len(modulesList) = %d, want %d", len(modulesList), 0)
	}
}

func listerAssertTypeAndDeps(
	t *testing.T,
	info modules.ModuleInfo,
	wantType modules.ModuleType,
	wantDeps int,
	wantImpl bool,
) {
	t.Helper()

	if info.Type != wantType {
		t.Fatalf("%s type = %q, want %q", info.Name, info.Type, wantType)
	}
	if info.DependencyCount != wantDeps {
		t.Fatalf("%s dependency count = %d, want %d", info.Name, info.DependencyCount, wantDeps)
	}

	hasImpl := info.ImplementationPath != ""
	if hasImpl != wantImpl {
		t.Fatalf("%s has implementation path = %t, want %t", info.Name, hasImpl, wantImpl)
	}
}

func listerWritePackageManifest(t *testing.T, modulesRoot string, packageName string, dependencies []string) {
	t.Helper()

	packagePath := filepath.Join(modulesRoot, packageName)
	if err := os.MkdirAll(packagePath, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", packagePath, err)
	}

	dependenciesBlock := ""
	for _, dependency := range dependencies {
		dependenciesBlock += fmt.Sprintf("        .package(path: \"../%s\"),\n", dependency)
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

	manifestPath := filepath.Join(packagePath, "Package.swift")
	if err := os.WriteFile(manifestPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", manifestPath, err)
	}
}

func listerWriteReluxMarkers(t *testing.T, modulesRoot string, moduleName string, relativePaths []string) {
	t.Helper()

	moduleRoot := filepath.Join(modulesRoot, moduleName)
	for _, relativePath := range relativePaths {
		path := filepath.Join(moduleRoot, relativePath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte("// marker\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}
}
