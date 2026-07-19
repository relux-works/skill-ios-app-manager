package securestore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/testutil"
)

const testAccessGroup = "group.com.example.demo"

func TestSetupCreatesAllFiles(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		AccessGroup: testAccessGroup,
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	// Interface package files.
	interfaceDir := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources")
	for _, rel := range []string{
		"SecureStore.swift",
		filepath.Join("Module", "SecureStore.Module.swift"),
		filepath.Join("Module", "SecureStore.Module+Interface.swift"),
	} {
		path := filepath.Join(interfaceDir, rel)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected file %q not found: %v", rel, err)
		}
		if info.Size() == 0 {
			t.Fatalf("file %q is empty", rel)
		}
	}

	// Impl package files.
	implDir := filepath.Join(projectRoot, "Packages", "SecureStoreImpl", "Sources")
	for _, rel := range []string{
		filepath.Join("Module", "SecureStore.Module+Impl.swift"),
	} {
		path := filepath.Join(implDir, rel)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected file %q not found: %v", rel, err)
		}
		if info.Size() == 0 {
			t.Fatalf("file %q is empty", rel)
		}
	}
}

func TestSetupCreatesPackageSwift(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		AccessGroup: testAccessGroup,
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	requireFile(t, filepath.Join(projectRoot, "Packages", "SecureStore", "Package.swift"))
	requireFile(t, filepath.Join(projectRoot, "Packages", "SecureStoreImpl", "Package.swift"))
}

func TestSetupBuilderConfigUsesCanonicalAppGroupProperty(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")
	writeTestFile(t, filepath.Join(projectRoot, "ios-app-manager.json"), `{
  "app_name": "DemoApp",
  "bundle_id": "com.example.demo",
  "team_id": "ABCDE12345",
  "swift_version": "6.2",
  "min_target": "17.0",
  "marketing_version": "1.0.0",
  "project_version": "1",
  "app_groups": ["group.com.example.demo"]
}`)

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		AccessGroup: testAccessGroup,
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	builderConfig := readFile(t, filepath.Join(projectRoot, "Packages", "SecureStore", ".builder-config"))
	if !strings.Contains(builderConfig, "Configuration.AppGroups.main") {
		t.Fatalf("builder config does not use canonical app-group property:\n%s", builderConfig)
	}
	if strings.Contains(builderConfig, "GROUP_COM_EXAMPLE_DEMO") {
		t.Fatalf("builder config kept obsolete Info.plist-shaped accessor:\n%s", builderConfig)
	}
}

func TestSetupUpdatesManifests(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	modulesPath := "Packages"
	setupProjectFiles(t, projectRoot, modulesPath)

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: modulesPath,
		AccessGroup: testAccessGroup,
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		`"SecureStore"`,
		`"SecureStoreImpl"`,
	} {
		if !strings.Contains(projectSwift, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectSwift)
		}
	}

	rootPkg := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	for _, want := range []string{
		`"Packages/SecureStore"`,
		`"Packages/SecureStoreImpl"`,
	} {
		if !strings.Contains(rootPkg, want) {
			t.Fatalf("Package.swift missing %q:\n%s", want, rootPkg)
		}
	}
}

func TestSetupWithCustomModulesPath(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Modules")

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: "Modules",
		AccessGroup: testAccessGroup,
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Modules", "SecureStore", "Sources", "SecureStore.swift")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file not found at custom modules path: %v", err)
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
			input: SetupInput{AppName: "Demo", AccessGroup: testAccessGroup},
			want:  "project root is required",
		},
		{
			name:  "whitespace project root",
			input: SetupInput{ProjectRoot: "   ", AppName: "Demo", AccessGroup: testAccessGroup},
			want:  "project root is required",
		},
		{
			name:  "empty app name",
			input: SetupInput{ProjectRoot: "/tmp", AccessGroup: testAccessGroup},
			want:  "app name is required",
		},
		{
			name:  "empty access group",
			input: SetupInput{ProjectRoot: "/tmp", AppName: "Demo"},
			want:  "access group is required",
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

func TestSetupIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	input := SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}

	if err := Setup(input); err != nil {
		t.Fatalf("first Setup() error = %v", err)
	}

	if err := Setup(input); err != nil {
		t.Fatalf("second Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources", "SecureStore.swift")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing after idempotent run: %v", err)
	}

	// Verify no duplicates in Project.swift.
	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	externalCount := strings.Count(projectSwift, `.external(name: "SecureStore")`)
	if externalCount != 1 {
		t.Fatalf(".external(name: \"SecureStore\") appears %d times, want 1:\n%s", externalCount, projectSwift)
	}
}

func TestNamespaceContent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources", "SecureStore.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "public enum SecureStore") {
		t.Fatalf("SecureStore.swift missing namespace:\n%s", s)
	}
}

func TestInterfaceContent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources", "Module", "SecureStore.Module+Interface.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	s := string(content)

	for _, expected := range []string{
		"protocol Interface: Sendable",
		"func save(key: String, data: Data) throws",
		"func load(key: String) throws -> Data?",
		"func delete(key: String) throws",
		"func clear() throws",
		"func save<T: Codable>(key: String, value: T) throws",
		"func load<T: Codable>(key: String) throws -> T?",
		"SecureStoring",
	} {
		if !strings.Contains(s, expected) {
			t.Fatalf("Interface missing %q:\n%s", expected, s)
		}
	}
}

func TestImplContent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "SecureStoreImpl", "Sources", "Module", "SecureStore.Module+Impl.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	s := string(content)

	for _, expected := range []string{
		"import SecureStore",
		"import Security",
		"public actor Impl: SecureStoring",
		"kSecClassGenericPassword",
		"SecItemAdd",
		"SecItemCopyMatching",
		"SecItemDelete",
		"SecItemUpdate",
		"serviceName",
		"KeychainError",
		"JSONEncoder",
		"JSONDecoder",
		"accessGroup",
		"kSecAttrAccessGroup",
		"init(serviceName: String, accessGroup: String)",
	} {
		if !strings.Contains(s, expected) {
			t.Fatalf("Impl missing %q:\n%s", expected, s)
		}
	}
}

func TestModuleContent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources", "Module", "SecureStore.Module.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	s := string(content)

	for _, expected := range []string{
		"extension SecureStore",
		"public enum Module",
	} {
		if !strings.Contains(s, expected) {
			t.Fatalf("Module definition missing %q:\n%s", expected, s)
		}
	}
}

func TestDirectoryStructure(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	expectedDirs := []string{
		filepath.Join("Packages", "SecureStore", "Sources"),
		filepath.Join("Packages", "SecureStore", "Sources", "Module"),
		filepath.Join("Packages", "SecureStoreImpl", "Sources"),
		filepath.Join("Packages", "SecureStoreImpl", "Sources", "Module"),
	}

	for _, rel := range expectedDirs {
		dirPath := filepath.Join(projectRoot, rel)
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Fatalf("expected directory %q not found: %v", rel, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", rel)
		}
	}
}

func TestGoldenNamespace(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources", "SecureStore.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	testutil.AssertGoldenFile(t, "securestore/namespace", string(content))
}

func TestGoldenModule(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources", "Module", "SecureStore.Module.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	testutil.AssertGoldenFile(t, "securestore/module", string(content))
}

func TestGoldenInterface(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources", "Module", "SecureStore.Module+Interface.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	testutil.AssertGoldenFile(t, "securestore/interface", string(content))
}

func TestGoldenImpl(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "SecureStoreImpl", "Sources", "Module", "SecureStore.Module+Impl.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	testutil.AssertGoldenFile(t, "securestore/impl", string(content))
}

func TestSetupIdempotentContentUnchanged(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")
	input := SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp", AccessGroup: testAccessGroup}

	if err := Setup(input); err != nil {
		t.Fatalf("first Setup() error = %v", err)
	}

	interfacePath := filepath.Join(projectRoot, "Packages", "SecureStore", "Sources", "Module", "SecureStore.Module+Interface.swift")
	firstContent, err := os.ReadFile(interfacePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if err := Setup(input); err != nil {
		t.Fatalf("second Setup() error = %v", err)
	}

	secondContent, err := os.ReadFile(interfacePath)
	if err != nil {
		t.Fatalf("ReadFile() after second run error = %v", err)
	}

	if string(firstContent) != string(secondContent) {
		t.Fatalf("content changed after idempotent run:\nfirst:\n%s\nsecond:\n%s", firstContent, secondContent)
	}
}

// --- helpers ---

func setupProjectFiles(t *testing.T, projectRoot, modulesPath string) {
	t.Helper()

	mkdirs(t, filepath.Join(projectRoot, modulesPath))

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
