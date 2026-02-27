package utilities

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupCreatesAllFiles(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	httpUtilsDir := filepath.Join(projectRoot, "Packages", "Utilities", "Sources", "Utilities", "HttpClientUtils")

	expectedFiles := []string{
		"HeaderMaps.swift",
		"BaseEncoder.swift",
		"BaseDecoder.swift",
	}

	for _, name := range expectedFiles {
		path := filepath.Join(httpUtilsDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected file %q not found: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("file %q is empty", name)
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
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	requireFile(t, filepath.Join(projectRoot, "Packages", "Utilities", "Package.swift"))
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
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	if !strings.Contains(projectSwift, `"Utilities"`) {
		t.Fatalf("Project.swift missing Utilities:\n%s", projectSwift)
	}

	rootPkg := readFile(t, filepath.Join(projectRoot, "Package.swift"))
	if !strings.Contains(rootPkg, `"Packages/Utilities"`) {
		t.Fatalf("Package.swift missing Utilities:\n%s", rootPkg)
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
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	httpUtilsDir := filepath.Join(projectRoot, "Modules", "Utilities", "Sources", "Utilities", "HttpClientUtils")

	path := filepath.Join(httpUtilsDir, "HeaderMaps.swift")
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
			input: SetupInput{AppName: "Demo"},
			want:  "project root is required",
		},
		{
			name:  "whitespace project root",
			input: SetupInput{ProjectRoot: "   ", AppName: "Demo"},
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

func TestSetupIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	input := SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp"}

	if err := Setup(input); err != nil {
		t.Fatalf("first Setup() error = %v", err)
	}

	if err := Setup(input); err != nil {
		t.Fatalf("second Setup() error = %v", err)
	}

	httpUtilsDir := filepath.Join(projectRoot, "Packages", "Utilities", "Sources", "Utilities", "HttpClientUtils")
	path := filepath.Join(httpUtilsDir, "HeaderMaps.swift")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing after idempotent run: %v", err)
	}

	// Verify no duplicates in Project.swift.
	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	externalCount := strings.Count(projectSwift, `.external(name: "Utilities")`)
	if externalCount != 1 {
		t.Fatalf(".external(name: \"Utilities\") appears %d times, want 1:\n%s", externalCount, projectSwift)
	}
}

func TestHeaderMapsContent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp"}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "Utilities", "Sources", "Utilities", "HttpClientUtils", "HeaderMaps.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	s := string(content)

	for _, expected := range []string{
		"jsonHeaders",
		"application/json",
		"formHeaders",
		"application/x-www-form-urlencoded",
		"func authHeader(token: String)",
		"Bearer",
	} {
		if !strings.Contains(s, expected) {
			t.Fatalf("HeaderMaps.swift missing %q:\n%s", expected, s)
		}
	}
}

func TestBaseEncoderContent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp"}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "Utilities", "Sources", "Utilities", "HttpClientUtils", "BaseEncoder.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	s := string(content)

	for _, expected := range []string{
		"JSONEncoder",
		"convertToSnakeCase",
		"iso8601",
	} {
		if !strings.Contains(s, expected) {
			t.Fatalf("BaseEncoder.swift missing %q:\n%s", expected, s)
		}
	}
}

func TestBaseDecoderContent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{ProjectRoot: projectRoot, AppName: "DemoApp"}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	path := filepath.Join(projectRoot, "Packages", "Utilities", "Sources", "Utilities", "HttpClientUtils", "BaseDecoder.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	s := string(content)

	for _, expected := range []string{
		"JSONDecoder",
		"convertFromSnakeCase",
		"iso8601",
	} {
		if !strings.Contains(s, expected) {
			t.Fatalf("BaseDecoder.swift missing %q:\n%s", expected, s)
		}
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
