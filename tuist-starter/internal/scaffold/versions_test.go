package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestSyncVersionsUpdatesManifestConstantsAcrossAppAndExtensions(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            infoPlist: .extendingDefault(with: [
                "CFBundleShortVersionString": .string(marketingVersion),
                "CFBundleVersion": .string(currentProjectVersion),
            ]),
            settings: .settings(
                base: [
                    "MARKETING_VERSION": .string(marketingVersion),
                    "CURRENT_PROJECT_VERSION": .string(currentProjectVersion),
                ]
            )
        )
    ]
)
`)
	writeVersionManifest(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"), `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "1"

let project = Project(
    name: "DemoWidget",
    targets: [
        .target(
            name: "DemoWidget",
            infoPlist: .extendingDefault(with: [
                "CFBundleShortVersionString": .string(marketingVersion),
                "CFBundleVersion": .string(currentProjectVersion),
            ]),
            settings: .settings(
                base: [
                    "MARKETING_VERSION": .string(marketingVersion),
                    "CURRENT_PROJECT_VERSION": .string(currentProjectVersion),
                ]
            )
        )
    ]
)
`)

	result, err := SyncVersions(projectRoot, config.ProjectConfig{
		MarketingVersion: "2.3.4",
		ProjectVersion:   "42",
	})
	if err != nil {
		t.Fatalf("SyncVersions() error = %v", err)
	}

	if len(result.Scanned) != 2 {
		t.Fatalf("scanned len = %d, want 2", len(result.Scanned))
	}
	if len(result.Updated) != 2 {
		t.Fatalf("updated len = %d, want 2", len(result.Updated))
	}

	for _, manifestPath := range result.Updated {
		content := readVersionManifest(t, manifestPath)
		for _, want := range []string{
			`let marketingVersion = "2.3.4"`,
			`let currentProjectVersion = "42"`,
			`.string(marketingVersion)`,
			`.string(currentProjectVersion)`,
		} {
			if !strings.Contains(content, want) {
				t.Fatalf("%s missing %q:\n%s", manifestPath, want, content)
			}
		}
	}
}

func TestSyncVersionsRewritesDirectLiteralFallback(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectSwiftPath := filepath.Join(projectRoot, "Project.swift")
	writeVersionManifest(t, projectSwiftPath, `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            infoPlist: .extendingDefault(with: [
                "CFBundleShortVersionString": .string("1.0.0"),
                "CFBundleVersion": .string("1"),
            ]),
            settings: .settings(
                base: [
                    "MARKETING_VERSION": .string("1.0.0"),
                    "CURRENT_PROJECT_VERSION": .string("1"),
                ]
            )
        )
    ]
)
`)

	result, err := SyncVersions(projectRoot, config.ProjectConfig{
		MarketingVersion: "3.0.0",
		ProjectVersion:   "7",
	})
	if err != nil {
		t.Fatalf("SyncVersions() error = %v", err)
	}
	if len(result.Updated) != 1 {
		t.Fatalf("updated len = %d, want 1", len(result.Updated))
	}

	content := readVersionManifest(t, projectSwiftPath)
	for _, want := range []string{
		`"CFBundleShortVersionString": .string("3.0.0")`,
		`"CFBundleVersion": .string("7")`,
		`"MARKETING_VERSION": .string("3.0.0")`,
		`"CURRENT_PROJECT_VERSION": .string("7")`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, content)
		}
	}
}

func TestSyncVersionsRequiresProjectVersion(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectSwiftPath := filepath.Join(projectRoot, "Project.swift")
	writeVersionManifest(t, projectSwiftPath, `import ProjectDescription

let marketingVersion = "1.0.0"
let currentProjectVersion = "63172"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            infoPlist: .extendingDefault(with: [
                "CFBundleShortVersionString": .string(marketingVersion),
                "CFBundleVersion": .string(currentProjectVersion),
            ]),
            settings: .settings(
                base: [
                    "MARKETING_VERSION": .string(marketingVersion),
                    "CURRENT_PROJECT_VERSION": .string(currentProjectVersion),
                ]
            )
        )
    ]
)
`)

	_, err := SyncVersions(projectRoot, config.ProjectConfig{
		MarketingVersion: "1.0.1",
	})
	if err != nil {
		if !strings.Contains(err.Error(), "project version is required") {
			t.Fatalf("SyncVersions() error = %q, want project version requirement", err.Error())
		}
		return
	}

	t.Fatal("SyncVersions() error = nil, want project version requirement")
}

func writeVersionManifest(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readVersionManifest(t *testing.T, path string) string {
	t.Helper()

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(payload)
}
