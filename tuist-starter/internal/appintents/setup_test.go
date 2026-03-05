package appintents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/testutil"
)

func TestSetupValidatesInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input SetupInput
		want  string
	}{
		{
			name:  "empty project root",
			input: SetupInput{AppName: "DemoApp"},
			want:  "project root is required",
		},
		{
			name:  "empty app name",
			input: SetupInput{ProjectRoot: "/tmp"},
			want:  "app name is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := Setup(tc.input)
			if err == nil {
				t.Fatal("Setup() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Setup() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestSetupCreatesIntentFileAndAddsDependency(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "DemoApp", "com.demo.app", []string{"group.com.demo.shared"})

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	widgetSourcesDir := filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources")

	intentPath := filepath.Join(widgetSourcesDir, "DemoAppWidgetToggleIntent.swift")
	requireFile(t, intentPath)

	intent := readFile(t, intentPath)
	for _, want := range []string{
		"import AppIntents",
		"import WidgetKit",
		"struct DemoAppWidgetToggleIntent: AppIntent",
		`UserDefaults(suiteName: "group.com.demo.shared")`,
		"WidgetCenter.shared.reloadAllTimelines()",
		"func perform() async throws -> some IntentResult",
	} {
		if !strings.Contains(intent, want) {
			t.Fatalf("intent file missing %q:\n%s", want, intent)
		}
	}

	extensionProject := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Project.swift"))
	if !strings.Contains(extensionProject, `.sdk(name: "AppIntents", type: .framework)`) {
		t.Fatalf("extension Project.swift missing AppIntents dependency:\n%s", extensionProject)
	}
}

func TestSetupDerivesAppGroupFromBundleID(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "DemoApp", "com.demo.app", nil)

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	intent := readFile(t, filepath.Join(
		projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppWidgetToggleIntent.swift",
	))
	if !strings.Contains(intent, `UserDefaults(suiteName: "group.com.demo.app")`) {
		t.Fatalf("intent missing derived app group:\n%s", intent)
	}
}

func TestSetupIsIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "DemoApp", "com.demo.app", []string{"group.com.demo.shared"})

	input := SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}

	if err := Setup(input); err != nil {
		t.Fatalf("first Setup() error = %v", err)
	}
	if err := Setup(input); err != nil {
		t.Fatalf("second Setup() error = %v", err)
	}

	extensionProject := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Project.swift"))
	if got := strings.Count(extensionProject, `.sdk(name: "AppIntents", type: .framework)`); got != 1 {
		t.Fatalf("AppIntents dependency appears %d times, want 1:\n%s", got, extensionProject)
	}
}

func TestGoldenWidgetToggleIntentTemplate(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "DemoApp", "com.demo.app", []string{"group.com.demo.shared"})

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	intent := readFile(t, filepath.Join(
		projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppWidgetToggleIntent.swift",
	))
	testutil.AssertGoldenFile(t, "appintents/widget_toggle_intent", intent)
}

func setupProjectFiles(t *testing.T, projectRoot, appName, bundleID string, appGroups []string) {
	t.Helper()

	widgetExtDir := filepath.Join(projectRoot, "Extensions", appName+"Widget")
	widgetSourcesDir := filepath.Join(widgetExtDir, "Sources")
	if err := os.MkdirAll(widgetSourcesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", widgetSourcesDir, err)
	}

	extensionProject := `import ProjectDescription

let hostBundleId = "` + bundleID + `"

let project = Project(
    name: "` + appName + `Widget",
    targets: [
        .target(
            name: "` + appName + `Widget",
            destinations: .iOS,
            product: .appExtension,
            bundleId: "\(hostBundleId).widget",
            infoPlist: .extendingDefault(with: [
                "NSExtension": .dictionary([
                    "NSExtensionPointIdentifier": .string("com.apple.widgetkit-extension"),
                ]),
            ]),
            sources: ["Sources/**"],
            dependencies: [
                .sdk(name: "WidgetKit", type: .framework),
            ]
        )
    ]
)
`
	writeTestFile(t, filepath.Join(widgetExtDir, "Project.swift"), extensionProject)

	cfg := config.ProjectConfig{
		AppName:          appName,
		BundleID:         bundleID,
		TeamID:           "TEAM123456",
		SwiftVersion:     "6.0",
		MinTarget:        "17.0",
		MarketingVersion: "1.0.0",
		AppGroups:        appGroups,
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal(config) error = %v", err)
	}
	writeTestFile(t, filepath.Join(projectRoot, config.DefaultConfigPath), string(raw))
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
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
