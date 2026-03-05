package ioc

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestDiscoverModulesFindsInterfaceImplPairs(t *testing.T) {
	t.Parallel()

	modulesRoot := t.TempDir()
	mkdirs(t,
		filepath.Join(modulesRoot, "Auth"),
		filepath.Join(modulesRoot, "AuthImpl"),
		filepath.Join(modulesRoot, "TodoList"),
		filepath.Join(modulesRoot, "TodoListImpl"),
		filepath.Join(modulesRoot, "Logger"),
	)

	modules, err := DiscoverModules(modulesRoot)
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v", err)
	}

	if len(modules) != 2 {
		t.Fatalf("DiscoverModules() = %d modules, want 2", len(modules))
	}

	names := []string{modules[0].Name, modules[1].Name}
	sort.Strings(names)
	if names[0] != "Auth" || names[1] != "TodoList" {
		t.Fatalf("module names = %v, want [Auth, TodoList]", names)
	}
}

func TestDiscoverModulesReturnsEmptyForNoModules(t *testing.T) {
	t.Parallel()

	modulesRoot := t.TempDir()

	modules, err := DiscoverModules(modulesRoot)
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v", err)
	}

	if len(modules) != 0 {
		t.Fatalf("DiscoverModules() = %d modules, want 0", len(modules))
	}
}

func TestDiscoverModulesIgnoresNonSplitModules(t *testing.T) {
	t.Parallel()

	modulesRoot := t.TempDir()
	mkdirs(t,
		filepath.Join(modulesRoot, "Logger"),
		filepath.Join(modulesRoot, "Utils"),
	)

	modules, err := DiscoverModules(modulesRoot)
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v", err)
	}

	if len(modules) != 0 {
		t.Fatalf("DiscoverModules() = %d modules, want 0 (no split modules)", len(modules))
	}
}

func TestDiscoverModulesIgnoresHiddenDirs(t *testing.T) {
	t.Parallel()

	modulesRoot := t.TempDir()
	mkdirs(t,
		filepath.Join(modulesRoot, ".Hidden"),
		filepath.Join(modulesRoot, ".HiddenImpl"),
	)

	modules, err := DiscoverModules(modulesRoot)
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v", err)
	}

	if len(modules) != 0 {
		t.Fatalf("DiscoverModules() = %d modules, want 0", len(modules))
	}
}

func TestDiscoverModulesNonExistentPath(t *testing.T) {
	t.Parallel()

	modules, err := DiscoverModules("/nonexistent/path")
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v, want nil for nonexistent path", err)
	}

	if modules != nil {
		t.Fatalf("DiscoverModules() = %v, want nil", modules)
	}
}

func TestRenderRegistryNoModules(t *testing.T) {
	t.Parallel()

	content, err := RenderRegistry("DemoApp", nil)
	if err != nil {
		t.Fatalf("RenderRegistry() error = %v", err)
	}

	if !strings.Contains(content, "extension DemoApp") {
		t.Fatalf("content missing extension declaration:\n%s", content)
	}
	if !strings.Contains(content, "import SwiftIoC") {
		t.Fatalf("content missing SwiftIoC import:\n%s", content)
	}
	if !strings.Contains(content, "static func configure()") {
		t.Fatalf("content missing configure():\n%s", content)
	}
	if !strings.Contains(content, "static func resolve") {
		t.Fatalf("content missing resolve():\n%s", content)
	}
	// No module registrations should be present.
	if strings.Contains(content, "ioc.register") {
		t.Fatalf("content should not contain registrations for 0 modules:\n%s", content)
	}
}

func TestRenderRegistryWithModules(t *testing.T) {
	t.Parallel()

	modules := []DiscoveredModule{
		{Name: "Auth", InterfacePackage: "Auth", ImplPackage: "AuthImpl"},
		{Name: "TodoList", InterfacePackage: "TodoList", ImplPackage: "TodoListImpl"},
	}

	content, err := RenderRegistry("DemoApp", modules)
	if err != nil {
		t.Fatalf("RenderRegistry() error = %v", err)
	}

	for _, expected := range []string{
		"import SwiftIoC",
		"import Auth",
		"import AuthImpl",
		"import TodoList",
		"import TodoListImpl",
		"extension DemoApp",
		"Auth.Module.Interface.self",
		"Auth.Module.Impl()",
		"TodoList.Module.Interface.self",
		"TodoList.Module.Impl()",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}
}

func TestEditAppSwiftInjectsInit(t *testing.T) {
	t.Parallel()

	input := `import SwiftUI

@main
struct DemoApp: App {
    var body: some Scene {
        WindowGroup {
            Text("Hello, World!")
        }
    }
}
`

	modules := []DiscoveredModule{
		{Name: "Auth", InterfacePackage: "Auth", ImplPackage: "AuthImpl"},
	}

	result := EditAppSwift(input, modules)

	for _, expected := range []string{
		"import SwiftIoC",
		"Registry.configure()",
		"init() {",
	} {
		if !strings.Contains(result, expected) {
			t.Fatalf("result missing %q:\n%s", expected, result)
		}
	}

	// Module imports should NOT be in App.swift (they live in Registry.swift).
	for _, unexpected := range []string{
		"import Auth",
		"import AuthImpl",
	} {
		if strings.Contains(result, unexpected) {
			t.Fatalf("result should not contain %q (module imports belong in Registry.swift):\n%s", unexpected, result)
		}
	}

	// Struct declaration should still be present.
	if !strings.Contains(result, "struct DemoApp: App {") {
		t.Fatalf("result missing struct declaration:\n%s", result)
	}
}

func TestEditAppSwiftIdempotent(t *testing.T) {
	t.Parallel()

	input := `import SwiftUI
import SwiftIoC
import Auth
import AuthImpl

@main
struct DemoApp: App {
    init() {
        Registry.configure()
    }

    var body: some Scene {
        WindowGroup {
            Text("Hello, World!")
        }
    }
}
`

	modules := []DiscoveredModule{
		{Name: "Auth", InterfacePackage: "Auth", ImplPackage: "AuthImpl"},
	}

	result := EditAppSwift(input, modules)

	// Count occurrences of Registry.configure().
	count := strings.Count(result, "Registry.configure()")
	if count != 1 {
		t.Fatalf("Registry.configure() appears %d times, want 1:\n%s", count, result)
	}

	// Count occurrences of import SwiftIoC.
	importCount := strings.Count(result, "import SwiftIoC")
	if importCount != 1 {
		t.Fatalf("import SwiftIoC appears %d times, want 1:\n%s", importCount, result)
	}
}

func TestEditAppSwiftNoModules(t *testing.T) {
	t.Parallel()

	input := `import SwiftUI

@main
struct DemoApp: App {
    var body: some Scene {
        WindowGroup {
            Text("Hello, World!")
        }
    }
}
`

	result := EditAppSwift(input, nil)

	if !strings.Contains(result, "import SwiftIoC") {
		t.Fatalf("result missing SwiftIoC import:\n%s", result)
	}
	if !strings.Contains(result, "Registry.configure()") {
		t.Fatalf("result missing Registry.configure():\n%s", result)
	}
}

func TestSetupValidatesInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input SetupInput
		want  string
	}{
		{
			name:  "empty project root",
			input: SetupInput{AppName: "Demo"},
			want:  "project root is required",
		},
		{
			name:  "empty app name",
			input: SetupInput{ProjectRoot: "/tmp"},
			want:  "app name is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Setup(tc.input)
			if err == nil {
				t.Fatal("Setup() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestEnsureImport(t *testing.T) {
	t.Parallel()

	input := "import SwiftUI\n\n@main\nstruct App {}\n"

	result := EnsureImport(input, "SwiftIoC")
	if !strings.Contains(result, "import SwiftIoC") {
		t.Fatalf("result missing import SwiftIoC:\n%s", result)
	}

	// Should not duplicate.
	result2 := EnsureImport(result, "SwiftIoC")
	count := strings.Count(result2, "import SwiftIoC")
	if count != 1 {
		t.Fatalf("import SwiftIoC appears %d times, want 1:\n%s", count, result2)
	}
}

func TestBuildModuleImports(t *testing.T) {
	t.Parallel()

	modules := []DiscoveredModule{
		{Name: "TodoList", InterfacePackage: "TodoList", ImplPackage: "TodoListImpl"},
		{Name: "Auth", InterfacePackage: "Auth", ImplPackage: "AuthImpl"},
	}

	imports := BuildModuleImports(modules)

	expected := []string{"Auth", "AuthImpl", "TodoList", "TodoListImpl"}
	if len(imports) != len(expected) {
		t.Fatalf("imports = %v, want %v", imports, expected)
	}
	for i, imp := range imports {
		if imp != expected[i] {
			t.Fatalf("imports[%d] = %q, want %q", i, imp, expected[i])
		}
	}
}

func TestDiscoverModulesReadsModuleType(t *testing.T) {
	t.Parallel()

	modulesRoot := t.TempDir()
	mkdirs(t,
		filepath.Join(modulesRoot, "SecureStore"),
		filepath.Join(modulesRoot, "SecureStoreImpl"),
		filepath.Join(modulesRoot, "Auth"),
		filepath.Join(modulesRoot, "AuthImpl"),
		filepath.Join(modulesRoot, "Utilities"),
		filepath.Join(modulesRoot, "UtilitiesImpl"),
	)

	// Write .module-type markers.
	writeModuleType(t, filepath.Join(modulesRoot, "SecureStore"), "kit")
	writeModuleType(t, filepath.Join(modulesRoot, "Auth"), "feature")

	modules, err := DiscoverModules(modulesRoot)
	if err != nil {
		t.Fatalf("DiscoverModules() error = %v", err)
	}

	catMap := make(map[string]ModuleCategory, len(modules))
	for _, m := range modules {
		catMap[m.Name] = m.Category
	}

	if catMap["SecureStore"] != CategoryFoundation {
		t.Fatalf("SecureStore category = %q, want %q", catMap["SecureStore"], CategoryFoundation)
	}
	if catMap["Auth"] != CategoryFeature {
		t.Fatalf("Auth category = %q, want %q", catMap["Auth"], CategoryFeature)
	}
	// No .module-type file → defaults to feature.
	if catMap["Utilities"] != CategoryFeature {
		t.Fatalf("Utilities category = %q, want %q (default)", catMap["Utilities"], CategoryFeature)
	}
}

func TestGroupModulesByCategory(t *testing.T) {
	t.Parallel()

	modules := []DiscoveredModule{
		{Name: "Auth", Category: CategoryFeature},
		{Name: "SecureStore", Category: CategoryFoundation},
		{Name: "TokenProvider", Category: CategoryFoundation},
		{Name: "TodoList", Category: CategoryFeature},
	}

	groups := GroupModulesByCategory(modules)

	if len(groups.Foundation) != 2 {
		t.Fatalf("Foundation = %d, want 2", len(groups.Foundation))
	}
	if len(groups.Features) != 2 {
		t.Fatalf("Features = %d, want 2", len(groups.Features))
	}
	if len(groups.Network) != 0 {
		t.Fatalf("Network = %d, want 0", len(groups.Network))
	}
	if len(groups.Utils) != 0 {
		t.Fatalf("Utils = %d, want 0", len(groups.Utils))
	}
}

func TestMapModuleTypeToCategory(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  ModuleCategory
	}{
		{"kit", CategoryFoundation},
		{"shared", CategoryFoundation},
		{"feature", CategoryFeature},
		{"relux-feature", CategoryFeature},
		{"ui", CategoryFeature},
		{"utility", CategoryUtils},
		{"unknown", CategoryFeature},
		{"", CategoryFeature},
	}

	for _, tc := range cases {
		got := MapModuleTypeToCategory(tc.input)
		if got != tc.want {
			t.Fatalf("MapModuleTypeToCategory(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestRenderRegistryWithSections(t *testing.T) {
	t.Parallel()

	modules := []DiscoveredModule{
		{Name: "Auth", InterfacePackage: "Auth", ImplPackage: "AuthImpl", Category: CategoryFeature},
		{Name: "SecureStore", InterfacePackage: "SecureStore", ImplPackage: "SecureStoreImpl", Category: CategoryFoundation},
	}

	content, err := RenderRegistryWithData(RegistryTemplateData{
		AppTypeName: "DemoApp",
		Imports:     BuildModuleImports(modules),
		Modules:     modules,
		HasRelux:    false,
	})
	if err != nil {
		t.Fatalf("RenderRegistryWithData() error = %v", err)
	}

	for _, expected := range []string{
		"// MARK: - Foundation (scaffolding anchor: foundation)",
		"// MARK: - Features (scaffolding anchor: features)",
		"SecureStore.Module.Interface.self",
		"Auth.Module.Interface.self",
		"Foundation Builders (scaffolding anchor: foundation-builders)",
		"Feature Builders (scaffolding anchor: feature-builders)",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}
}

func writeModuleType(t *testing.T, moduleDir, moduleType string) {
	t.Helper()
	path := filepath.Join(moduleDir, ModuleTypeFile)
	if err := os.WriteFile(path, []byte(moduleType+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func mkdirs(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", path, err)
		}
	}
}
