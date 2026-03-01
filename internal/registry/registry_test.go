package registry

import (
	"testing"
)

func TestRegister(t *testing.T) {
	Reset()
	m := &Module{
		ID:       IoC,
		Name:     "IoC",
		Category: Infra,
	}
	Register(m)

	got := Get(IoC)
	if got == nil {
		t.Fatal("expected module, got nil")
	}
	if got.ID != IoC {
		t.Errorf("expected ID %s, got %s", IoC, got.ID)
	}
	if got.Name != "IoC" {
		t.Errorf("expected Name IoC, got %s", got.Name)
	}
}

func TestGetUnknown(t *testing.T) {
	Reset()
	got := Get("nonexistent")
	if got != nil {
		t.Errorf("expected nil for unknown ID, got %+v", got)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	Reset()
	m := &Module{ID: IoC, Name: "IoC"}
	Register(m)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if msg != "module ioc already registered" {
			t.Errorf("unexpected panic message: %s", msg)
		}
	}()
	Register(&Module{ID: IoC, Name: "IoC duplicate"})
}

func TestAll(t *testing.T) {
	Reset()
	Register(&Module{ID: IoC, Name: "IoC"})
	Register(&Module{ID: Relux, Name: "Relux"})
	Register(&Module{ID: Utilities, Name: "Utilities"})

	all := All()
	if len(all) != 3 {
		t.Fatalf("expected 3 modules, got %d", len(all))
	}
	for _, id := range []ModuleID{IoC, Relux, Utilities} {
		if _, ok := all[id]; !ok {
			t.Errorf("missing module %s", id)
		}
	}
}

func TestAllSorted(t *testing.T) {
	Reset()

	// IoC has no deps
	Register(&Module{ID: IoC, Name: "IoC", Category: Infra})
	// Relux depends on IoC
	Register(&Module{ID: Relux, Name: "Relux", Category: Infra, Dependencies: []ModuleID{IoC}})
	// AppConfig depends on IoC and SecureStore
	Register(&Module{ID: AppConfig, Name: "AppConfig", Category: Foundation, Dependencies: []ModuleID{IoC, SecureStore}})
	// SecureStore has no deps
	Register(&Module{ID: SecureStore, Name: "SecureStore", Category: Foundation})

	sorted := AllSorted()
	if len(sorted) != 4 {
		t.Fatalf("expected 4 modules, got %d", len(sorted))
	}

	// Build position map
	pos := make(map[ModuleID]int)
	for i, m := range sorted {
		pos[m.ID] = i
	}

	// IoC must come before Relux
	if pos[IoC] >= pos[Relux] {
		t.Errorf("IoC (pos %d) should come before Relux (pos %d)", pos[IoC], pos[Relux])
	}
	// IoC must come before AppConfig
	if pos[IoC] >= pos[AppConfig] {
		t.Errorf("IoC (pos %d) should come before AppConfig (pos %d)", pos[IoC], pos[AppConfig])
	}
	// SecureStore must come before AppConfig
	if pos[SecureStore] >= pos[AppConfig] {
		t.Errorf("SecureStore (pos %d) should come before AppConfig (pos %d)", pos[SecureStore], pos[AppConfig])
	}
}

func TestAllSortedEmpty(t *testing.T) {
	Reset()
	sorted := AllSorted()
	if len(sorted) != 0 {
		t.Errorf("expected empty slice, got %d items", len(sorted))
	}
}

func TestCheckDependencies(t *testing.T) {
	Reset()
	Register(&Module{ID: IoC, Name: "IoC", Category: Infra})
	Register(&Module{ID: SecureStore, Name: "SecureStore", Category: Foundation})
	Register(&Module{ID: AppConfig, Name: "AppConfig", Category: Foundation, Dependencies: []ModuleID{IoC, SecureStore}})

	registryContent := `
// MARK: - infra
IoC.Module.register(container)
// MARK: - foundation
SecureStore.Module.register(container)
`

	// All deps present — should pass
	err := CheckDependencies(AppConfig, registryContent)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckDependenciesMissing(t *testing.T) {
	Reset()
	Register(&Module{ID: IoC, Name: "IoC", Category: Infra})
	Register(&Module{ID: SecureStore, Name: "SecureStore", Category: Foundation})
	Register(&Module{ID: AppConfig, Name: "AppConfig", Category: Foundation, Dependencies: []ModuleID{IoC, SecureStore}})

	// Only IoC present, SecureStore missing
	registryContent := `
// MARK: - infra
IoC.Module.register(container)
`

	err := CheckDependencies(AppConfig, registryContent)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
	if got := err.Error(); got != "missing dependencies for AppConfig: SecureStore — run their setup first" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestCheckDependenciesNoDeps(t *testing.T) {
	Reset()
	Register(&Module{ID: IoC, Name: "IoC", Category: Infra})

	err := CheckDependencies(IoC, "")
	if err != nil {
		t.Errorf("expected no error for module with no deps, got: %v", err)
	}
}

func TestCheckDependenciesUnknownModule(t *testing.T) {
	Reset()
	err := CheckDependencies("nonexistent", "")
	if err == nil {
		t.Fatal("expected error for unknown module")
	}
}

func TestModuleFields(t *testing.T) {
	Reset()

	planCalled := false
	setupCalled := false

	m := &Module{
		ID:          SecureStore,
		Name:        "SecureStore",
		Description: "Keychain wrapper with interface/impl split",
		Category:    Foundation,
		Dependencies: []ModuleID{IoC},
		Plan: func(input SetupInput) (string, error) {
			planCalled = true
			return "will create SecureStore", nil
		},
		Setup: func(input SetupInput) error {
			setupCalled = true
			return nil
		},
		UsageGuide: "Use SecureStore via IoC",
		CLIUse:     "secure-store",
		CLIShort:   "Manage SecureStore module",
		SetupShort: "Create SecureStore kit module",
		ExtraFlags: []ExtraFlag{
			{Name: "access-group", Usage: "app group", Required: true, ArgKey: "access-group"},
		},
	}
	Register(m)

	got := Get(SecureStore)
	if got.Description != "Keychain wrapper with interface/impl split" {
		t.Errorf("unexpected Description: %s", got.Description)
	}
	if got.CLIUse != "secure-store" {
		t.Errorf("unexpected CLIUse: %s", got.CLIUse)
	}
	if len(got.ExtraFlags) != 1 {
		t.Fatalf("expected 1 ExtraFlag, got %d", len(got.ExtraFlags))
	}
	if got.ExtraFlags[0].ArgKey != "access-group" {
		t.Errorf("unexpected ArgKey: %s", got.ExtraFlags[0].ArgKey)
	}

	input := SetupInput{ProjectRoot: "/tmp", AppName: "Test"}
	plan, err := got.Plan(input)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if plan != "will create SecureStore" {
		t.Errorf("unexpected plan: %s", plan)
	}
	if !planCalled {
		t.Error("Plan was not called")
	}

	err = got.Setup(input)
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	if !setupCalled {
		t.Error("Setup was not called")
	}
}
