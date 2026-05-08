package tuistproj

import (
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/components"
	"github.com/relux-works/ios-app-manager/internal/testutil"
)

func TestGeneratePackageSwiftInterfaceGolden(t *testing.T) {
	t.Parallel()

	rendered, err := GeneratePackageSwift(PackageGenerationInput{
		ModuleName:   "Auth",
		Type:         PackageTypeInterface,
		Dependencies: []string{"CoreKit", "Analytics"},
		Platform:     "iOS(.v16)",
	})
	if err != nil {
		t.Fatalf("GeneratePackageSwift() error = %v", err)
	}

	testutil.AssertGoldenFile(t, "tuistproj/package_interface", rendered)
}

func TestGeneratePackageSwiftImplGolden(t *testing.T) {
	t.Parallel()

	rendered, err := GeneratePackageSwift(PackageGenerationInput{
		ModuleName:   "Auth",
		Type:         PackageTypeImpl,
		Dependencies: []string{"CoreKit", "Analytics"},
		Platform:     ".iOS(.v16)",
	})
	if err != nil {
		t.Fatalf("GeneratePackageSwift() error = %v", err)
	}

	testutil.AssertGoldenFile(t, "tuistproj/package_impl", rendered)
}

func TestGeneratePackageSwiftSupportsPlatformTuples(t *testing.T) {
	t.Parallel()

	rendered, err := GeneratePackageSwift(PackageGenerationInput{
		ModuleName: "Auth",
		Type:       PackageTypeInterface,
		Platforms: []components.PlatformTarget{
			{Platform: "iOS", MinTarget: "16.0"},
			{Platform: "macOS", MinTarget: "13.0"},
		},
	})
	if err != nil {
		t.Fatalf("GeneratePackageSwift() error = %v", err)
	}

	for _, want := range []string{`.iOS(.v16)`, `.macOS(.v13)`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("Package.swift missing %q:\n%s", want, rendered)
		}
	}
}
