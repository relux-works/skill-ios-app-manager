package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   SetupInput
		wantErr string
	}{
		{
			name:    "empty project root",
			input:   SetupInput{ProjectRoot: "", AppName: "App"},
			wantErr: "project root is required",
		},
		{
			name:    "empty app name",
			input:   SetupInput{ProjectRoot: "/tmp", AppName: ""},
			wantErr: "app name is required",
		},
		{
			name:  "valid input",
			input: SetupInput{ProjectRoot: "/tmp", AppName: "App"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInput(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateInput() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateInput() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateInput() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestScaffoldFiles(t *testing.T) {
	t.Parallel()

	outputDir := filepath.Join(t.TempDir(), "AppConfig")

	if err := scaffoldFiles(outputDir); err != nil {
		t.Fatalf("scaffoldFiles() error = %v", err)
	}

	expectedFiles := []string{
		"AppConfig.swift",
		"AppConfig.Env.swift",
		"AppConfig.Env+Configuration.swift",
		"AppConfig.Env+Configuration+Presets.swift",
		"AppConfig.Manager+Protocols.swift",
		"AppConfig.Manager.swift",
		"AppConfig.ApiConfigurator.swift",
		"AppConfig.UrlComponents.swift",
	}

	for _, name := range expectedFiles {
		path := filepath.Join(outputDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("file %q not found: %v", name, err)
		}
		if info.IsDir() {
			t.Fatalf("%q is a directory, want file", name)
		}
		if info.Size() == 0 {
			t.Fatalf("%q is empty", name)
		}
	}
}

func TestScaffoldFilesIdempotent(t *testing.T) {
	t.Parallel()

	outputDir := filepath.Join(t.TempDir(), "AppConfig")

	// First run.
	if err := scaffoldFiles(outputDir); err != nil {
		t.Fatalf("first scaffoldFiles() error = %v", err)
	}

	// Overwrite one file with custom content.
	customPath := filepath.Join(outputDir, "AppConfig.swift")
	customContent := "// custom content\n"
	if err := os.WriteFile(customPath, []byte(customContent), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	// Second run should skip existing files.
	if err := scaffoldFiles(outputDir); err != nil {
		t.Fatalf("second scaffoldFiles() error = %v", err)
	}

	// Custom content should be preserved.
	data, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	if string(data) != customContent {
		t.Fatalf("file content = %q, want %q (idempotency broken)", string(data), customContent)
	}
}

func TestPatchRegistry(t *testing.T) {
	t.Parallel()

	registryPath := filepath.Join(t.TempDir(), "Registry.swift")
	registryContent := sampleRegistry()
	if err := os.WriteFile(registryPath, []byte(registryContent), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	if err := patchRegistry(registryPath); err != nil {
		t.Fatalf("patchRegistry() error = %v", err)
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	content := string(data)

	// Registration should be in the Foundation section.
	if !strings.Contains(content, "IApiConfigManager.self") {
		t.Fatal("Registry missing IApiConfigManager.self registration")
	}

	// Builder should be present.
	if !strings.Contains(content, "buildAppConfigManager") {
		t.Fatal("Registry missing buildAppConfigManager builder")
	}

	// Builder should reference SecureStoring.
	if !strings.Contains(content, "resolve(SecureStoring.self)") {
		t.Fatal("Registry builder missing SecureStoring resolution")
	}

	// Registration should appear after foundation anchor.
	foundationIdx := strings.Index(content, foundationAnchor)
	regIdx := strings.Index(content, "IApiConfigManager.self")
	featuresIdx := strings.Index(content, "// MARK: - Features")
	if regIdx < foundationIdx || regIdx > featuresIdx {
		t.Fatalf("IApiConfigManager registration not in Foundation section (foundation=%d, reg=%d, features=%d)",
			foundationIdx, regIdx, featuresIdx)
	}

	// Builder should appear in Foundation Builders section.
	foundationBuildersIdx := strings.Index(content, foundationBuildersAnchor)
	featureBuildersIdx := strings.Index(content, "// MARK: - Feature Builders")
	builderIdx := strings.Index(content, "func buildAppConfigManager()")
	if builderIdx < foundationBuildersIdx || builderIdx > featureBuildersIdx {
		t.Fatalf("buildAppConfigManager not in Foundation Builders section")
	}
}

func TestPatchRegistryIdempotent(t *testing.T) {
	t.Parallel()

	registryPath := filepath.Join(t.TempDir(), "Registry.swift")
	registryContent := sampleRegistry()
	if err := os.WriteFile(registryPath, []byte(registryContent), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	// Patch twice.
	if err := patchRegistry(registryPath); err != nil {
		t.Fatalf("first patchRegistry() error = %v", err)
	}
	if err := patchRegistry(registryPath); err != nil {
		t.Fatalf("second patchRegistry() error = %v", err)
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	content := string(data)

	count := strings.Count(content, "IApiConfigManager.self")
	if count != 1 {
		t.Fatalf("IApiConfigManager.self appears %d times, want 1:\n%s", count, content)
	}

	builderCount := strings.Count(content, "buildAppConfigManager")
	if builderCount != 2 {
		// 1 in registration line + 1 in builder func declaration
		t.Fatalf("buildAppConfigManager appears %d times, want 2:\n%s", builderCount, content)
	}
}

// sampleRegistry returns a minimal Registry.swift with all required anchors.
func sampleRegistry() string {
	return `import SwiftIoC
import SecureStore
import SecureStoreImpl

extension DemoApp {
    @MainActor
    enum Registry {
        static let ioc = IoC()

        static func configure() {

            // MARK: - Foundation (scaffolding anchor: foundation)
            ioc.register(SecureStore.Module.Interface.self, lifecycle: .container, resolver: Self.buildSecureStore)

            // MARK: - Features (scaffolding anchor: features)

            // MARK: - Network (scaffolding anchor: network)

            // MARK: - Utils (scaffolding anchor: utils)
        }

        static func resolve<T>(_ type: T.Type) -> T {
            ioc.get(by: type)!
        }

        static func resolveAsync<T>(_ type: T.Type) async -> T {
            await ioc.getAsync(by: type)!
        }
    }
}

// MARK: - Foundation Builders (scaffolding anchor: foundation-builders)
extension DemoApp.Registry {
    private static func buildSecureStore() -> SecureStore.Module.Interface {
        SecureStore.Module.Impl(serviceName: Configuration.Keychain.serviceName, accessGroup: Configuration.AppGroups.GROUP_COM_EXAMPLE_DEMO)
    }
}

// MARK: - Feature Builders (scaffolding anchor: feature-builders)
extension DemoApp.Registry {
}

// MARK: - Network Builders (scaffolding anchor: network-builders)
extension DemoApp.Registry {
}

// MARK: - Utils Builders (scaffolding anchor: utils-builders)
extension DemoApp.Registry {
}
`
}
