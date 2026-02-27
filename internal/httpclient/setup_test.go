package httpclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchRegistryInsertsAtAnchors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	registryPath := filepath.Join(dir, "Registry.swift")

	content := `import SwiftIoC

extension TestApp {
    @MainActor
    enum Registry {
        static let ioc = IoC()

        static func configure() {

            // MARK: - Network (scaffolding anchor: network)

            // MARK: - Utils (scaffolding anchor: utils)
        }
    }
}

// MARK: - Network Builders (scaffolding anchor: network-builders)
extension TestApp.Registry {}

// MARK: - Utils Builders (scaffolding anchor: utils-builders)
extension TestApp.Registry {}
`
	if err := os.WriteFile(registryPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	if err := patchRegistry(registryPath); err != nil {
		t.Fatalf("patchRegistry error = %v", err)
	}

	result, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	patched := string(result)

	for _, want := range []string{
		"import HttpClient",
		"IRpcAsyncClient.self",
		"buildHttpClient",
		"RpcClient(",
		"Configuration.HttpClient.timeoutForResponse",
	} {
		if !strings.Contains(patched, want) {
			t.Fatalf("patched Registry missing %q:\n%s", want, patched)
		}
	}
}

func TestPatchRegistryIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	registryPath := filepath.Join(dir, "Registry.swift")

	content := `import SwiftIoC

extension TestApp {
    @MainActor
    enum Registry {
        static let ioc = IoC()

        static func configure() {

            // MARK: - Network (scaffolding anchor: network)

            // MARK: - Utils (scaffolding anchor: utils)
        }
    }
}

// MARK: - Network Builders (scaffolding anchor: network-builders)
extension TestApp.Registry {}

// MARK: - Utils Builders (scaffolding anchor: utils-builders)
extension TestApp.Registry {}
`
	if err := os.WriteFile(registryPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	if err := patchRegistry(registryPath); err != nil {
		t.Fatalf("first patchRegistry error = %v", err)
	}

	if err := patchRegistry(registryPath); err != nil {
		t.Fatalf("second patchRegistry error = %v", err)
	}

	result, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	count := strings.Count(string(result), "IRpcAsyncClient.self")
	if count != 1 {
		t.Fatalf("IRpcAsyncClient.self appears %d times, want 1:\n%s", count, string(result))
	}
}

func TestFindMatchingBrace(t *testing.T) {
	cases := []struct {
		name string
		s    string
		pos  int
		want int
	}{
		{"simple", "{}", 0, 1},
		{"nested", "{ { } }", 0, 6},
		{"deep", "{ { { } } }", 0, 10},
		{"no match", "{", 0, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findMatchingBrace(tc.s, tc.pos)
			if got != tc.want {
				t.Fatalf("findMatchingBrace(%q, %d) = %d, want %d", tc.s, tc.pos, got, tc.want)
			}
		})
	}
}

func TestScaffoldConfigurationExtensionCreatesFile(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "Configuration")

	if err := scaffoldConfigurationExtension(dir); err != nil {
		t.Fatalf("scaffoldConfigurationExtension error = %v", err)
	}

	path := filepath.Join(dir, "Configuration+HttpClient.swift")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	for _, want := range []string{
		"Configuration",
		"HttpClient",
		"timeoutForResponse",
		"timeoutResourceInterval",
	} {
		if !strings.Contains(string(content), want) {
			t.Fatalf("Configuration+HttpClient.swift missing %q:\n%s", want, string(content))
		}
	}
}

func TestScaffoldConfigurationExtensionIdempotent(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "Configuration")

	if err := scaffoldConfigurationExtension(dir); err != nil {
		t.Fatalf("first scaffoldConfigurationExtension error = %v", err)
	}

	// Write custom content to verify it's not overwritten.
	path := filepath.Join(dir, "Configuration+HttpClient.swift")
	if err := os.WriteFile(path, []byte("custom"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	if err := scaffoldConfigurationExtension(dir); err != nil {
		t.Fatalf("second scaffoldConfigurationExtension error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	if string(content) != "custom" {
		t.Fatalf("file was overwritten, got %q, want %q", string(content), "custom")
	}
}

func TestValidateInput(t *testing.T) {
	cases := []struct {
		name    string
		input   SetupInput
		wantErr bool
	}{
		{"valid", SetupInput{ProjectRoot: "/tmp", AppName: "App"}, false},
		{"no root", SetupInput{AppName: "App"}, true},
		{"no app name", SetupInput{ProjectRoot: "/tmp"}, true},
		{"empty", SetupInput{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInput(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateInput() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
