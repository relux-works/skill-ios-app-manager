package fireauthrelux

import (
	"slices"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

func TestModuleDeclaresDurableSessionStorePrerequisite(t *testing.T) {
	t.Parallel()

	module := registry.Get(registry.FireAuthRelux)
	if module == nil {
		t.Fatal("FireAuthRelux module not registered")
	}
	if !slices.Contains(module.Dependencies, registry.SecureStore) {
		t.Fatalf("FireAuthRelux dependencies = %v, want SecureStore", module.Dependencies)
	}
}
