package tuistproj

import (
	"testing"

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
