package tuistproj

import (
	"os"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/testutil"
)

func TestParseManifestExtractsTargetsDependenciesAndProducts(t *testing.T) {
	t.Parallel()

	payload, err := os.ReadFile("testdata/manifest_package.swift")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	manifest, err := ParseManifest(string(payload))
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}

	if len(manifest.Targets) != 2 {
		t.Fatalf("targets len = %d, want 2", len(manifest.Targets))
	}
	if manifest.Targets[0].Name != "Auth" || manifest.Targets[0].Kind != "target" {
		t.Fatalf("targets[0] = %#v, want name=Auth kind=target", manifest.Targets[0])
	}
	if manifest.Targets[1].Name != "AuthTests" || manifest.Targets[1].Kind != "testTarget" {
		t.Fatalf("targets[1] = %#v, want name=AuthTests kind=testTarget", manifest.Targets[1])
	}

	if len(manifest.Dependencies) != 2 {
		t.Fatalf("dependencies len = %d, want 2", len(manifest.Dependencies))
	}
	if manifest.Dependencies[0].Name != "CoreKit" || manifest.Dependencies[0].Kind != "package" {
		t.Fatalf("dependencies[0] = %#v, want name=CoreKit kind=package", manifest.Dependencies[0])
	}
	if manifest.Dependencies[1].Name != "ExternalSDK" || manifest.Dependencies[1].Kind != "package" {
		t.Fatalf("dependencies[1] = %#v, want name=ExternalSDK kind=package", manifest.Dependencies[1])
	}

	if len(manifest.Products) != 2 {
		t.Fatalf("products len = %d, want 2", len(manifest.Products))
	}
	if manifest.Products[0].Name != "Auth" || manifest.Products[0].Kind != "library" {
		t.Fatalf("products[0] = %#v, want name=Auth kind=library", manifest.Products[0])
	}
	if manifest.Products[1].Name != "AuthTesting" || manifest.Products[1].Kind != "library" {
		t.Fatalf("products[1] = %#v, want name=AuthTesting kind=library", manifest.Products[1])
	}
}

func TestApplyManifestEditsGolden(t *testing.T) {
	t.Parallel()

	payload, err := os.ReadFile("testdata/manifest_edit_input.swift")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	updated, err := ApplyManifestEdits(
		string(payload),
		ManifestEdit{
			Type:    AddDependency,
			Name:    "Networking",
			Content: `.package(path: "../Networking")`,
		},
		ManifestEdit{
			Type: AddTarget,
			Name: "AuthImpl",
			Content: `.target(
    name: "AuthImpl",
    dependencies: [
        .product(name: "Networking", package: "Networking"),
    ]
)`,
		},
		ManifestEdit{
			Type:    AddProduct,
			Name:    "AuthImpl",
			Content: `.library(name: "AuthImpl", type: .dynamic, targets: ["AuthImpl"])`,
		},
		ManifestEdit{
			Type: RemoveDependency,
			Name: "CoreKit",
		},
		ManifestEdit{
			Type: RemoveTarget,
			Name: "Auth",
		},
		ManifestEdit{
			Type: RemoveProduct,
			Name: "Auth",
		},
	)
	if err != nil {
		t.Fatalf("ApplyManifestEdits() error = %v", err)
	}

	testutil.AssertGoldenFile(t, "tuistproj/manifest_edit", updated)
}
