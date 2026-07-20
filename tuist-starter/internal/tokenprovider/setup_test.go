package tokenprovider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

func TestUsageAndPlanNameGeneratedTokenProviderInterface(t *testing.T) {
	t.Parallel()

	plan, err := Plan(registry.SetupInput{AppName: "DemoApp"})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	for name, rendered := range map[string]string{
		"usage": usageGuide,
		"plan":  plan,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(rendered, "TokenProvider.Module.Interface") {
				t.Fatalf("%s guidance does not name generated interface:\n%s", name, rendered)
			}
			if strings.Contains(rendered, "TokenProviding") {
				t.Fatalf("%s guidance names ungenerated TokenProviding type:\n%s", name, rendered)
			}
		})
	}
}

func TestSetupValidatesInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
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

	for _, tc := range tests {
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

func TestSetupCreatesModulePackages(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesPath := "Packages"
	modulesRoot := filepath.Join(projectRoot, modulesPath)

	setupProjectFiles(t, projectRoot, modulesPath)

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: modulesPath,
		Platform:    "iOS(.v17)",
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	// Verify interface package.
	requireDir(t, filepath.Join(modulesRoot, "TokenProvider"))
	requireFile(t, filepath.Join(modulesRoot, "TokenProvider", "Package.swift"))
	requireFile(t, filepath.Join(modulesRoot, "TokenProvider", "Sources", "TokenProvider.swift"))
	requireFile(t, filepath.Join(modulesRoot, "TokenProvider", "Sources", "TokenProvider.AuthData.swift"))
	requireFile(t, filepath.Join(modulesRoot, "TokenProvider", "Sources", "Module", "TokenProvider.Module.swift"))
	requireFile(t, filepath.Join(modulesRoot, "TokenProvider", "Sources", "Module", "TokenProvider.Module+Interface.swift"))

	// Verify impl package.
	requireDir(t, filepath.Join(modulesRoot, "TokenProviderImpl"))
	requireFile(t, filepath.Join(modulesRoot, "TokenProviderImpl", "Package.swift"))
	requireFile(t, filepath.Join(modulesRoot, "TokenProviderImpl", "Sources", "Module", "TokenProvider.Module+Impl.swift"))
}

func TestSetupSwiftFileContents(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesPath := "Packages"
	modulesRoot := filepath.Join(projectRoot, modulesPath)

	setupProjectFiles(t, projectRoot, modulesPath)

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: modulesPath,
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	// Check namespace.
	ns := readFile(t, filepath.Join(modulesRoot, "TokenProvider", "Sources", "TokenProvider.swift"))
	if !strings.Contains(ns, "public enum TokenProvider {}") {
		t.Fatalf("namespace missing enum declaration:\n%s", ns)
	}

	// Check protocol.
	proto := readFile(t, filepath.Join(modulesRoot, "TokenProvider", "Sources", "Module", "TokenProvider.Module+Interface.swift"))
	for _, want := range []string{
		"public protocol Interface: Sendable",
		"func setAuthData",
		"func getAccessToken",
	} {
		if !strings.Contains(proto, want) {
			t.Fatalf("protocol missing %q:\n%s", want, proto)
		}
	}

	// Check AuthData.
	authData := readFile(t, filepath.Join(modulesRoot, "TokenProvider", "Sources", "TokenProvider.AuthData.swift"))
	for _, want := range []string{
		"public struct AuthData: Sendable",
		"accessToken: String",
		"refreshToken: String",
		"acquireDate: Date",
		"ttl: TimeInterval",
	} {
		if !strings.Contains(authData, want) {
			t.Fatalf("AuthData missing %q:\n%s", want, authData)
		}
	}

	// Check impl.
	impl := readFile(t, filepath.Join(modulesRoot, "TokenProviderImpl", "Sources", "Module", "TokenProvider.Module+Impl.swift"))
	for _, want := range []string{
		"import TokenProvider",
		"public actor Impl",
		"func setAuthData",
		"func getAccessToken",
	} {
		if !strings.Contains(impl, want) {
			t.Fatalf("impl missing %q:\n%s", want, impl)
		}
	}
}

func TestSetupUpdatesProjectSwift(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesPath := "Packages"

	setupProjectFiles(t, projectRoot, modulesPath)

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: modulesPath,
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		`"TokenProvider"`,
		`"TokenProviderImpl"`,
	} {
		if !strings.Contains(projectSwift, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectSwift)
		}
	}

	rootPkg := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	for _, want := range []string{
		`"Packages/TokenProvider"`,
		`"Packages/TokenProviderImpl"`,
	} {
		if !strings.Contains(rootPkg, want) {
			t.Fatalf("Package.swift missing %q:\n%s", want, rootPkg)
		}
	}
}

func TestSetupIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesPath := "Packages"

	setupProjectFiles(t, projectRoot, modulesPath)

	input := SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: modulesPath,
	}

	if err := Setup(input); err != nil {
		t.Fatalf("Setup() first call error = %v", err)
	}

	if err := Setup(input); err != nil {
		t.Fatalf("Setup() second call error = %v", err)
	}

	// Verify no duplicates in Project.swift.
	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	count := strings.Count(projectSwift, `"TokenProvider"`)
	// TokenProvider and TokenProviderImpl both contain "TokenProvider"
	// but the exact .external(name: "TokenProvider") should appear once.
	externalCount := strings.Count(projectSwift, `.external(name: "TokenProvider")`)
	if externalCount != 1 {
		t.Fatalf(".external(name: \"TokenProvider\") appears %d times, want 1:\n%s", externalCount, projectSwift)
	}
	_ = count
}

func TestSetupNoTemplateArtifacts(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesPath := "Packages"

	setupProjectFiles(t, projectRoot, modulesPath)

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: modulesPath,
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	modulesRoot := filepath.Join(projectRoot, modulesPath)
	swiftFiles := collectSwiftFiles(t, modulesRoot)

	for _, path := range swiftFiles {
		content := readFile(t, path)
		for _, token := range []string{"{{", "}}", "{%", "%}", "<#"} {
			if strings.Contains(content, token) {
				t.Fatalf("Swift file %q contains template artifact %q", path, token)
			}
		}
	}
}

func TestSetupPatchesExistingMatureRegistryWithoutRegeneration(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesPath := "Packages"
	setupProjectFiles(t, projectRoot, modulesPath)

	registryPath := filepath.Join(
		projectRoot,
		"Targets",
		"DemoApp",
		"Sources",
		"App",
		"DemoApp.Registry.swift",
	)
	customRegistry := `import CustomRuntime
import SwiftIoC
@_exported import Relux

extension DemoApp {
    @MainActor
    enum Registry {
        static let ioc = IoC()
        private(set) static var runtimeMode = CustomRuntimeMode.application

        static func configure(runtimeMode: CustomRuntimeMode = .current()) {
            self.runtimeMode = runtimeMode

            // MARK: - Infrastructure (scaffolding anchor: infra)
            ioc.register(Relux.self, lifecycle: .container, resolver: buildRelux)
            ioc.register(CustomRuntime.self, lifecycle: .container, resolver: buildCustomRuntime)

            // MARK: - Foundation (scaffolding anchor: foundation)
            ioc.register(CustomPersistence.self, lifecycle: .container, resolver: buildCustomPersistence)

            // MARK: - Features (scaffolding anchor: features)

            // MARK: - Network (scaffolding anchor: network)
            ioc.register(CustomAPIClient.self, lifecycle: .container, resolver: buildCustomAPIClient)

            // MARK: - Utils (scaffolding anchor: utils)
        }

        static func resolve<T>(_ type: T.Type) -> T {
            guard let value = ioc.get(by: type) else {
                preconditionFailure("custom resolve")
            }
            return value
        }
    }
}

// MARK: - Infrastructure Builders (scaffolding anchor: infra-builders)
extension DemoApp.Registry {
    private static func buildRelux() async -> Relux {
        let relux = await Relux(logger: CustomLogger(), appStore: .init(), rootSaga: .init())
        let feature = await CustomFeature(dispatcher: relux.dispatcher)
        return relux.register(feature)
    }

    private static func buildCustomRuntime() -> CustomRuntime { .init() }
    private static func buildCustomPersistence() -> CustomPersistence { .init() }
    private static func buildCustomAPIClient() -> CustomAPIClient { .init() }
}
`
	writeTestFile(t, registryPath, customRegistry)

	input := SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: modulesPath,
	}
	if err := Setup(input); err != nil {
		t.Fatalf("Setup() mature Registry error = %v", err)
	}
	first := readFile(t, registryPath)

	for _, preserved := range []string{
		"import CustomRuntime",
		"private(set) static var runtimeMode",
		"ioc.register(CustomRuntime.self",
		"ioc.register(CustomPersistence.self",
		"ioc.register(CustomAPIClient.self",
		"preconditionFailure(\"custom resolve\")",
		"let feature = await CustomFeature",
		"return relux.register(feature)",
	} {
		if !strings.Contains(first, preserved) {
			t.Fatalf("TokenProvider setup lost mature composition %q:\n%s", preserved, first)
		}
	}
	for _, integrated := range []string{
		"import TokenProvider",
		"import TokenProviderImpl",
		"TokenProvider.Module.Interface.self",
		"func buildTokenProvider()",
		"TokenProvider.Module.Impl()",
	} {
		if !strings.Contains(first, integrated) {
			t.Fatalf("TokenProvider setup missing %q:\n%s", integrated, first)
		}
	}

	if err := Setup(input); err != nil {
		t.Fatalf("second Setup() mature Registry error = %v", err)
	}
	second := readFile(t, registryPath)
	if second != first {
		t.Fatalf("second TokenProvider setup changed mature Registry:\n%s", second)
	}
}

// --- helpers ---

func setupProjectFiles(t *testing.T, projectRoot, modulesPath string) {
	t.Helper()

	mkdirs(t, filepath.Join(projectRoot, modulesPath))

	// Minimal Project.swift with dependencies section.
	projectSwift := `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            destinations: .iOS,
            product: .app,
            bundleId: "com.demo.app",
            dependencies: [
            ]
        )
    ]
)
`
	writeTestFile(t, filepath.Join(projectRoot, "Project.swift"), projectSwift)

	// Minimal root Package.swift with dependencies section.
	rootPkg := `// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "DemoAppDependencies",
    dependencies: [
    ],
    targets: []
)
`
	writeTestFile(t, filepath.Join(projectRoot, "Package.swift"), rootPkg)
}

func mkdirs(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", path, err)
		}
	}
}

func requireDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("path %q is not a directory", path)
	}
}

func requireFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("path %q is a directory, want file", path)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll for %q error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func collectSwiftFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".swift" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%q) error = %v", root, err)
	}
	return files
}
