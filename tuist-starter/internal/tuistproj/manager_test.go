package tuistproj

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/components"
	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestTuistProjectManagerDelegatesRunnerCommands(t *testing.T) {
	t.Parallel()

	type call struct {
		command string
		args    []string
	}

	var calls []call
	runner := mockRunner{
		runFn: func(_ context.Context, command string, extraArgs ...string) (RunResult, error) {
			calls = append(calls, call{
				command: command,
				args:    append([]string(nil), extraArgs...),
			})

			if command == CommandGraph {
				return RunResult{
					Stdout: `{"targets":[],"dependencies":[]}`,
				}, nil
			}

			return RunResult{}, nil
		},
	}

	manager := NewTuistProjectManager(WithRunner(runner))
	if err := manager.Generate(context.Background(), components.GenerateOpts{
		ConfigPath: "Project.swift",
	}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if err := manager.Install(context.Background()); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	if _, err := manager.Graph(context.Background(), "json"); err != nil {
		t.Fatalf("Graph() error = %v", err)
	}

	if err := manager.Clean(context.Background()); err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	wantCalls := []call{
		{command: CommandGenerate, args: []string{"--no-open", "--path", "Project.swift"}},
		{command: CommandInstall, args: nil},
		{command: CommandGraph, args: []string{"--format", "json"}},
		{command: CommandClean, args: nil},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", calls, wantCalls)
	}
}

func TestTuistProjectManagerGenerateOpenOptIn(t *testing.T) {
	t.Parallel()

	var gotArgs []string
	runner := mockRunner{
		runFn: func(_ context.Context, command string, extraArgs ...string) (RunResult, error) {
			if command == CommandGenerate {
				gotArgs = append([]string(nil), extraArgs...)
			}
			return RunResult{}, nil
		},
	}

	manager := NewTuistProjectManager(WithRunner(runner))
	if err := manager.Generate(context.Background(), components.GenerateOpts{Open: true}); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	wantArgs := []string{"--open"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestTuistProjectManagerGraphReturnsParseError(t *testing.T) {
	t.Parallel()

	manager := NewTuistProjectManager(WithRunner(mockRunner{
		runFn: func(_ context.Context, _ string, _ ...string) (RunResult, error) {
			return RunResult{Stdout: "{not-json"}, nil
		},
	}))

	_, err := manager.Graph(context.Background(), "json")
	if err == nil {
		t.Fatalf("Graph() error = nil, want non-nil")
	}

	var parseErr *GraphParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error type = %T, want *GraphParseError", err)
	}
}

func TestTuistProjectManagerCreateModuleProductScaffoldsTwoPackages(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := NewTuistProjectManager(
		WithRootDir(root),
		WithModulesDir("Packages"),
		WithPackagePlatform("iOS(.v16)"),
	)

	err := manager.CreateModule(context.Background(), components.ModuleOpts{
		Name: "Auth",
		Type: "feature",
		Config: config.ProjectConfig{
			MinTarget: "16.0",
		},
	})
	if err != nil {
		t.Fatalf("CreateModule() error = %v", err)
	}

	interfaceRoot := filepath.Join(root, "Packages", "Auth")
	requireDir(t, interfaceRoot)
	requireFile(t, filepath.Join(interfaceRoot, "Package.swift"))
	requireDir(t, filepath.Join(interfaceRoot, "Sources"))
	requireDir(t, filepath.Join(interfaceRoot, "Tests", "AuthTests"))

	implRoot := filepath.Join(root, "Packages", "AuthImpl")
	requireDir(t, implRoot)
	requireFile(t, filepath.Join(implRoot, "Package.swift"))
	requireDir(t, filepath.Join(implRoot, "Sources"))
	requireDir(t, filepath.Join(implRoot, "Tests", "AuthImplTests"))

	interfaceManifest, err := ReadManifestFile(filepath.Join(interfaceRoot, "Package.swift"))
	if err != nil {
		t.Fatalf("ReadManifestFile(interface) error = %v", err)
	}
	if !reflect.DeepEqual(collectManifestNames(interfaceManifest.Products), []string{"Auth"}) {
		t.Fatalf("interface products = %#v, want %#v", collectManifestNames(interfaceManifest.Products), []string{"Auth"})
	}

	implManifest, err := ReadManifestFile(filepath.Join(implRoot, "Package.swift"))
	if err != nil {
		t.Fatalf("ReadManifestFile(impl) error = %v", err)
	}
	if !reflect.DeepEqual(collectManifestNames(implManifest.Dependencies), []string{"Auth"}) {
		t.Fatalf("impl dependencies = %#v, want %#v", collectManifestNames(implManifest.Dependencies), []string{"Auth"})
	}

	implManifestRaw := readFileStringForManagerTest(t, filepath.Join(implRoot, "Package.swift"))
	if !strings.Contains(implManifestRaw, `.library(name: "AuthImpl", type: .dynamic, targets: ["AuthImpl"])`) {
		t.Fatalf("impl Package.swift missing dynamic library product:\n%s", implManifestRaw)
	}
}

func TestTuistProjectManagerCreateModuleRequiresPlatformTargets(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := NewTuistProjectManager(
		WithRootDir(root),
		WithModulesDir("Packages"),
	)

	err := manager.CreateModule(context.Background(), components.ModuleOpts{
		Name: "Auth",
		Type: "feature",
	})
	if err == nil {
		t.Fatal("CreateModule() error = nil, want platform requirement")
	}
	if !strings.Contains(err.Error(), "module package platforms are required") {
		t.Fatalf("CreateModule() error = %q, want platform requirement", err.Error())
	}
	requireNotExists(t, filepath.Join(root, "Packages", "Auth"))
}

func TestTuistProjectManagerCreateModuleRejectsInvalidPlatformVersion(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := NewTuistProjectManager(
		WithRootDir(root),
		WithModulesDir("Packages"),
	)

	err := manager.CreateModule(context.Background(), components.ModuleOpts{
		Name: "Auth",
		Type: "feature",
		Platforms: []components.PlatformTarget{
			{Platform: components.PlatformIOS, MinTarget: "16"},
		},
	})
	if err == nil {
		t.Fatal("CreateModule() error = nil, want invalid min target")
	}
	if !strings.Contains(err.Error(), `min target "16" must use major.minor format`) {
		t.Fatalf("CreateModule() error = %q, want invalid min target message", err.Error())
	}
	requireNotExists(t, filepath.Join(root, "Packages", "Auth"))
}

func TestTuistProjectManagerCreateModuleUsesModuleConfigForSwiftSettings(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := NewTuistProjectManager(
		WithRootDir(root),
		WithModulesDir("Packages"),
		WithPackagePlatform("iOS(.v16)"),
	)

	err := manager.CreateModule(context.Background(), components.ModuleOpts{
		Name: "Auth",
		Type: "feature",
		Config: config.ProjectConfig{
			MinTarget:    "16.0",
			SwiftVersion: "6.0",
			ProjectSettings: config.ProjectSettings{
				Swift: config.SwiftProjectSettings{
					LanguageMode: "v6",
					Concurrency: config.SwiftConcurrencySettings{
						ExistentialAny: "no",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateModule() error = %v", err)
	}

	interfaceManifest := readFileStringForManagerTest(t, filepath.Join(root, "Packages", "Auth", "Package.swift"))
	if !strings.Contains(interfaceManifest, `swiftSettings: [`) {
		t.Fatalf("interface Package.swift missing swiftSettings:\n%s", interfaceManifest)
	}
	if !strings.Contains(interfaceManifest, `.swiftLanguageMode(.v6)`) {
		t.Fatalf("interface Package.swift missing Swift 6 language mode:\n%s", interfaceManifest)
	}
	if strings.Contains(interfaceManifest, `ExistentialAny`) {
		t.Fatalf("interface Package.swift unexpectedly contains ExistentialAny despite config override:\n%s", interfaceManifest)
	}
}

func TestTuistProjectManagerCreateModuleReluxFeatureScaffoldsTwoPackagesWithExternalDeps(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Create root Package.swift so external deps can be added
	rootPkgPath := filepath.Join(root, "Package.swift")
	writeFileForManagerTest(t, rootPkgPath, []byte(`// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "App",
    products: [
        .library(name: "App", type: .dynamic, targets: ["App"]),
    ],
    dependencies: [
    ],
    targets: [
        .target(name: "App"),
    ]
)
`))

	manager := NewTuistProjectManager(
		WithRootDir(root),
		WithModulesDir("Packages"),
		WithPackagePlatform("iOS(.v17)"),
	)

	err := manager.CreateModule(context.Background(), components.ModuleOpts{
		Name: "Auth",
		Type: "relux-feature",
		Config: config.ProjectConfig{
			MinTarget: "17.0",
		},
		ExternalDeps: []components.ExternalDep{{
			PackageName: "swift-relux",
			ProductName: "Relux",
			URL:         "https://github.com/relux-works/swift-relux.git",
			Version:     `from: "9.0.3"`,
		}},
	})
	if err != nil {
		t.Fatalf("CreateModule() error = %v", err)
	}

	// Two packages created
	interfaceRoot := filepath.Join(root, "Packages", "Auth")
	requireDir(t, interfaceRoot)
	implRoot := filepath.Join(root, "Packages", "AuthImpl")
	requireDir(t, implRoot)

	// Interface Package.swift has swift-relux
	interfaceManifestRaw := readFileStringForManagerTest(t, filepath.Join(interfaceRoot, "Package.swift"))
	if !strings.Contains(interfaceManifestRaw, "swift-relux") {
		t.Fatalf("interface Package.swift missing swift-relux dependency:\n%s", interfaceManifestRaw)
	}
	if !strings.Contains(interfaceManifestRaw, `"Relux"`) {
		t.Fatalf("interface Package.swift missing Relux product:\n%s", interfaceManifestRaw)
	}

	// Impl Package.swift has swift-relux
	implManifestRaw := readFileStringForManagerTest(t, filepath.Join(implRoot, "Package.swift"))
	if !strings.Contains(implManifestRaw, "swift-relux") {
		t.Fatalf("impl Package.swift missing swift-relux dependency:\n%s", implManifestRaw)
	}
	if !strings.Contains(implManifestRaw, `"Relux"`) {
		t.Fatalf("impl Package.swift missing Relux product:\n%s", implManifestRaw)
	}

	// Root Package.swift has swift-relux
	rootManifestRaw := readFileStringForManagerTest(t, rootPkgPath)
	if !strings.Contains(rootManifestRaw, "swift-relux") {
		t.Fatalf("root Package.swift missing swift-relux dependency:\n%s", rootManifestRaw)
	}
	if !strings.Contains(rootManifestRaw, `"Relux": .framework`) {
		t.Fatalf("root Package.swift missing Relux framework product type:\n%s", rootManifestRaw)
	}
}

func TestTuistProjectManagerCreateModuleUtilityScaffoldsSinglePackage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manager := NewTuistProjectManager(
		WithRootDir(root),
		WithModulesDir("Packages"),
		WithPackagePlatform("iOS(.v16)"),
	)

	err := manager.CreateModule(context.Background(), components.ModuleOpts{
		Name: "CoreKit",
		Type: "utility",
		Config: config.ProjectConfig{
			MinTarget: "16.0",
		},
	})
	if err != nil {
		t.Fatalf("CreateModule() error = %v", err)
	}

	utilityRoot := filepath.Join(root, "Packages", "CoreKit")
	requireDir(t, utilityRoot)
	requireFile(t, filepath.Join(utilityRoot, "Package.swift"))
	requireDir(t, filepath.Join(utilityRoot, "Sources"))
	requireDir(t, filepath.Join(utilityRoot, "Tests", "CoreKitTests"))

	requireNotExists(t, filepath.Join(root, "Packages", "CoreKitImpl"))
}

func TestTuistProjectManagerCreateModuleUpdatesProjectManifestDependencies(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectManifest := filepath.Join(root, "Project.swift")
	writeFileForManagerTest(t, projectManifest, []byte(`// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "App",
    products: [
        .library(name: "App", type: .dynamic, targets: ["App"]),
    ],
    dependencies: [
        .package(path: "Packages/CoreKit"),
    ],
    targets: [
        .target(name: "App"),
    ]
)
`))

	manager := NewTuistProjectManager(
		WithRootDir(root),
		WithModulesDir("Packages"),
		WithPackagePlatform("iOS(.v16)"),
	)

	err := manager.CreateModule(context.Background(), components.ModuleOpts{
		Name: "Auth",
		Type: "feature",
		Config: config.ProjectConfig{
			MinTarget: "16.0",
		},
	})
	if err != nil {
		t.Fatalf("CreateModule() error = %v", err)
	}

	manifest, err := ReadManifestFile(projectManifest)
	if err != nil {
		t.Fatalf("ReadManifestFile(project) error = %v", err)
	}

	wantDependencies := []string{"Auth", "AuthImpl", "CoreKit"}
	if !reflect.DeepEqual(collectManifestNames(manifest.Dependencies), wantDependencies) {
		t.Fatalf(
			"project dependencies = %#v, want %#v",
			collectManifestNames(manifest.Dependencies),
			wantDependencies,
		)
	}
}

func TestTuistProjectManagerDeleteModuleRemovesDirectoriesAndManifestReferences(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "Packages", "Auth"),
		filepath.Join(root, "Packages", "AuthImpl"),
		filepath.Join(root, "Packages", "CoreKit"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("os.MkdirAll(%q) error = %v", dir, err)
		}
	}

	projectManifest := filepath.Join(root, "Project.swift")
	writeFileForManagerTest(t, projectManifest, []byte(`// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "App",
    products: [
        .library(name: "Auth", type: .dynamic, targets: ["Auth"]),
        .library(name: "AuthImpl", type: .dynamic, targets: ["AuthImpl"]),
        .library(name: "CoreKit", type: .dynamic, targets: ["CoreKit"]),
    ],
    dependencies: [
        .package(path: "Packages/Auth"),
        .package(path: "Packages/AuthImpl"),
        .package(path: "Packages/CoreKit"),
    ],
    targets: [
        .target(name: "Auth"),
        .target(name: "AuthImpl"),
        .target(name: "CoreKit"),
    ]
)
`))

	manager := NewTuistProjectManager(
		WithRootDir(root),
		WithModulesDir("Packages"),
	)

	if err := manager.DeleteModule(context.Background(), "Auth"); err != nil {
		t.Fatalf("DeleteModule() error = %v", err)
	}

	requireNotExists(t, filepath.Join(root, "Packages", "Auth"))
	requireNotExists(t, filepath.Join(root, "Packages", "AuthImpl"))
	requireDir(t, filepath.Join(root, "Packages", "CoreKit"))

	manifest, err := ReadManifestFile(projectManifest)
	if err != nil {
		t.Fatalf("ReadManifestFile(project) error = %v", err)
	}

	wantNames := []string{"CoreKit"}
	if !reflect.DeepEqual(collectManifestNames(manifest.Dependencies), wantNames) {
		t.Fatalf("dependencies = %#v, want %#v", collectManifestNames(manifest.Dependencies), wantNames)
	}
	if !reflect.DeepEqual(collectManifestNames(manifest.Targets), wantNames) {
		t.Fatalf("targets = %#v, want %#v", collectManifestNames(manifest.Targets), wantNames)
	}
	if !reflect.DeepEqual(collectManifestNames(manifest.Products), wantNames) {
		t.Fatalf("products = %#v, want %#v", collectManifestNames(manifest.Products), wantNames)
	}
}

func TestTuistProjectManagerEditManifestDeleteModuleOperation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectManifest := filepath.Join(root, "Project.swift")
	writeFileForManagerTest(t, projectManifest, []byte(`// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "App",
    products: [
        .library(name: "Auth", type: .dynamic, targets: ["Auth"]),
        .library(name: "AuthImpl", type: .dynamic, targets: ["AuthImpl"]),
        .library(name: "CoreKit", type: .dynamic, targets: ["CoreKit"]),
    ],
    dependencies: [
        .package(path: "Packages/Auth"),
        .package(path: "Packages/AuthImpl"),
        .package(path: "Packages/CoreKit"),
    ],
    targets: [
        .target(name: "Auth"),
        .target(name: "AuthImpl"),
        .target(name: "CoreKit"),
    ]
)
`))

	manager := NewTuistProjectManager(WithRootDir(root))
	err := manager.EditManifest(context.Background(), "Project.swift", []components.ManifestEdit{
		{
			Operation: "delete_module",
			Path:      "Auth",
		},
	})
	if err != nil {
		t.Fatalf("EditManifest() error = %v", err)
	}

	manifest, err := ReadManifestFile(projectManifest)
	if err != nil {
		t.Fatalf("ReadManifestFile(project) error = %v", err)
	}

	wantNames := []string{"CoreKit"}
	if !reflect.DeepEqual(collectManifestNames(manifest.Dependencies), wantNames) {
		t.Fatalf("dependencies = %#v, want %#v", collectManifestNames(manifest.Dependencies), wantNames)
	}
}

func requireDir(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", path)
	}
}

func requireFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("%q is a directory, want file", path)
	}
}

func requireNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("path %q exists, want not exists", path)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(%q) error = %v, want os.ErrNotExist", path, err)
	}
}

func writeFileForManagerTest(t *testing.T, path string, content []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readFileStringForManagerTest(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func collectManifestNames(items []ManifestItem) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		names = append(names, item.Name)
	}
	sort.Strings(names)
	return names
}
