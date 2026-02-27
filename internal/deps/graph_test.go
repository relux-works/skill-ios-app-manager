package deps

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

func TestBuildDependencyGraphUsesInterfaceModulesOnly(t *testing.T) {
	t.Parallel()

	modulesRoot := filepath.Join(t.TempDir(), "Packages")
	writeInterfaceModuleManifest(t, modulesRoot, "Auth", nil)
	writeInterfaceModuleManifest(t, modulesRoot, "Feed", []string{"Auth", "RemoteSDK"})
	writeImplementationModuleManifest(t, modulesRoot, "Auth")

	graph, err := manifestGraphSource(modulesRoot)
	if err != nil {
		t.Fatalf("manifestGraphSource() error = %v", err)
	}

	if _, exists := graph["AuthImpl"]; exists {
		t.Fatalf("graph unexpectedly contains implementation package key: %#v", graph)
	}

	if got := graph["Feed"]; !reflect.DeepEqual(got, []string{"Auth"}) {
		t.Fatalf(`graph["Feed"] = %#v, want %#v`, got, []string{"Auth"})
	}
	if got := graph["Auth"]; !reflect.DeepEqual(got, []string{}) {
		t.Fatalf(`graph["Auth"] = %#v, want empty slice`, got)
	}
}

func TestDetectCircularDependenciesReportsPath(t *testing.T) {
	t.Parallel()

	modulesRoot := filepath.Join(t.TempDir(), "Packages")
	writeInterfaceModuleManifest(t, modulesRoot, "A", []string{"B"})
	writeInterfaceModuleManifest(t, modulesRoot, "B", []string{"C"})
	writeInterfaceModuleManifest(t, modulesRoot, "C", []string{"A"})

	err := detectCircularDependencies(modulesRoot, manifestGraphSource)
	if err == nil {
		t.Fatal("detectCircularDependencies() error = nil, want cycle error")
	}

	const expected = "circular dependency: A → B → C → A"
	if err.Error() != expected {
		t.Fatalf("error = %q, want %q", err.Error(), expected)
	}
}

func writeImplementationModuleManifest(t *testing.T, modulesRoot string, moduleName string) {
	t.Helper()

	implName := moduleName + moduleImplSuffix
	packagePath := filepath.Join(modulesRoot, implName)
	if err := os.MkdirAll(packagePath, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", packagePath, err)
	}

	manifest, err := tuistproj.GeneratePackageSwift(tuistproj.PackageGenerationInput{
		ModuleName: moduleName,
		Type:       tuistproj.PackageTypeImpl,
		Platform:   "iOS(.v16)",
	})
	if err != nil {
		t.Fatalf("GeneratePackageSwift(%q impl) error = %v", moduleName, err)
	}

	manifestPath := filepath.Join(packagePath, moduleManifestName)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", manifestPath, err)
	}
}
