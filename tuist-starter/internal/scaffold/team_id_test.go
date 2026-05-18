package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestSyncTeamIDUpdatesAppAndExtensionManifests(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeVersionManifest(t, filepath.Join(projectRoot, "Project.swift"), `import ProjectDescription

let appName = "DemoApp"
let bundleID = "com.demo.app"
let developmentTeam = "OLDTEAM123"
let marketingVersion = "1.0.0"

let project = Project(
    name: appName,
    settings: .settings(
        base: [
            "SWIFT_VERSION": "6.0",
        ]
    ),
    targets: [
        .target(
            name: appName,
            bundleId: bundleID,
            settings: .settings(
                base: [
                    "DEVELOPMENT_TEAM": .string("OLDTEAM123"),
                    "MARKETING_VERSION": .string(marketingVersion),
                ]
            )
        )
    ]
)
`)
	writeVersionManifest(t, filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"), `import ProjectDescription

let hostBundleId = "com.demo.app"
let marketingVersion = "1.0.0"

let project = Project(
    name: "DemoWidget",
    targets: [
        .target(
            name: "DemoWidget",
            product: .appExtension,
            bundleId: "\(hostBundleId).widget",
            settings: .settings(
                base: [
                    "MARKETING_VERSION": .string(marketingVersion),
                ]
            )
        )
    ]
)
`)

	result, err := SyncTeamID(projectRoot, config.ProjectConfig{TeamID: "NEWTEAM999"})
	if err != nil {
		t.Fatalf("SyncTeamID() error = %v", err)
	}

	if len(result.Scanned) != 2 {
		t.Fatalf("scanned len = %d, want 2", len(result.Scanned))
	}
	if len(result.Updated) != 2 {
		t.Fatalf("updated len = %d, want 2", len(result.Updated))
	}

	for _, manifestPath := range []string{
		filepath.Join(projectRoot, "Project.swift"),
		filepath.Join(projectRoot, "Extensions", "DemoWidget", "Project.swift"),
	} {
		content := readVersionManifest(t, manifestPath)
		if !strings.Contains(content, `let developmentTeam = "NEWTEAM999"`) {
			t.Fatalf("%s missing synced developmentTeam constant:\n%s", manifestPath, content)
		}
		if !strings.Contains(content, `"DEVELOPMENT_TEAM": .string(developmentTeam)`) {
			t.Fatalf("%s missing synced DEVELOPMENT_TEAM setting:\n%s", manifestPath, content)
		}
		if strings.Contains(content, "OLDTEAM123") {
			t.Fatalf("%s kept stale team id:\n%s", manifestPath, content)
		}
	}
}

func TestSyncTeamIDReportsUpToDateManifest(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	projectSwiftPath := filepath.Join(projectRoot, "Project.swift")
	writeVersionManifest(t, projectSwiftPath, `import ProjectDescription

let appName = "DemoApp"
let developmentTeam = "ABCDE12345"

let project = Project(
    name: appName,
    targets: [
        .target(
            name: appName,
            settings: .settings(
                base: [
                    "DEVELOPMENT_TEAM": .string(developmentTeam),
                ]
            )
        )
    ]
)
`)

	result, err := SyncTeamID(projectRoot, config.ProjectConfig{TeamID: "ABCDE12345"})
	if err != nil {
		t.Fatalf("SyncTeamID() error = %v", err)
	}
	if len(result.Updated) != 0 {
		t.Fatalf("updated len = %d, want 0", len(result.Updated))
	}
}
