package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestSyncBundleIDUpdatesHostAndExtensionManifests(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	hostManifestPath := filepath.Join(projectRoot, "Project.swift")
	extensionManifestPath := filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift")

	writeVersionManifest(t, hostManifestPath, `import ProjectDescription

let appName = "DemoApp"
let bundleID = "com.old.demo"

let project = Project(
    name: appName,
    targets: [
        .target(
            name: appName,
            bundleId: bundleID
        )
    ]
)
`)
	writeVersionManifest(t, extensionManifestPath, `import ProjectDescription

let hostBundleId = "com.old.demo"

let project = Project(
    name: "DemoWidget",
    targets: [
        .target(
            name: "DemoWidget",
            product: .appExtension,
            bundleId: "\(hostBundleId).widget"
        )
    ]
)
`)

	result, err := SyncBundleID(projectRoot, config.ProjectConfig{BundleID: "com.example.demo"})
	if err != nil {
		t.Fatalf("SyncBundleID() error = %v", err)
	}

	if len(result.Scanned) != 2 {
		t.Fatalf("scanned len = %d, want 2", len(result.Scanned))
	}
	if len(result.Updated) != 2 {
		t.Fatalf("updated len = %d, want 2", len(result.Updated))
	}

	hostManifest := readVersionManifest(t, hostManifestPath)
	if !strings.Contains(hostManifest, `let bundleID = "com.example.demo"`) {
		t.Fatalf("host manifest missing synced bundleID:\n%s", hostManifest)
	}
	if strings.Contains(hostManifest, "com.old.demo") {
		t.Fatalf("host manifest kept stale bundle id:\n%s", hostManifest)
	}

	extensionManifest := readVersionManifest(t, extensionManifestPath)
	if !strings.Contains(extensionManifest, `let hostBundleId = "com.example.demo"`) {
		t.Fatalf("extension manifest missing synced hostBundleId:\n%s", extensionManifest)
	}
	if !strings.Contains(extensionManifest, `bundleId: "\(hostBundleId).widget"`) {
		t.Fatalf("extension manifest did not preserve suffix interpolation:\n%s", extensionManifest)
	}
	if strings.Contains(extensionManifest, "com.old.demo") {
		t.Fatalf("extension manifest kept stale host bundle id:\n%s", extensionManifest)
	}
}

func TestSyncBundleIDUpdatesDirectLiteralExtensionBundleID(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	extensionManifestPath := filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift")

	writeVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: "com.old.demo"
        )
    ]
)
`)
	writeVersionManifest(t, extensionManifestPath, `import ProjectDescription

let project = Project(
    name: "DemoWidget",
    targets: [
        .target(
            name: "DemoWidget",
            product: .appExtension,
            bundleId: "com.old.demo.widget"
        )
    ]
)
`)

	result, err := SyncBundleID(projectRoot, config.ProjectConfig{BundleID: "com.example.demo"})
	if err != nil {
		t.Fatalf("SyncBundleID() error = %v", err)
	}
	if len(result.Updated) != 2 {
		t.Fatalf("updated len = %d, want 2", len(result.Updated))
	}

	extensionManifest := readVersionManifest(t, extensionManifestPath)
	if !strings.Contains(extensionManifest, `bundleId: "com.example.demo.widget"`) {
		t.Fatalf("extension manifest missing synced direct bundle id:\n%s", extensionManifest)
	}
}

func TestSyncBundleIDReportsUpToDateManifests(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let bundleID = "com.example.demo"

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            bundleId: bundleID
        )
    ]
)
`)
	writeVersionManifest(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"), `import ProjectDescription

let hostBundleId = "com.example.demo"

let project = Project(
    name: "DemoWidget",
    targets: [
        .target(
            name: "DemoWidget",
            product: .appExtension,
            bundleId: "\(hostBundleId).widget"
        )
    ]
)
`)

	result, err := SyncBundleID(projectRoot, config.ProjectConfig{BundleID: "com.example.demo"})
	if err != nil {
		t.Fatalf("SyncBundleID() error = %v", err)
	}
	if len(result.Updated) != 0 {
		t.Fatalf("updated len = %d, want 0", len(result.Updated))
	}
}
