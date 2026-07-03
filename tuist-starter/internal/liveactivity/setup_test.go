package liveactivity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestSetupCreatesLiveActivityFilesAndPatches(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: "Packages",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	attributesPath := filepath.Join(projectRoot, "Packages", "SharedKit", "Sources", "DemoAppActivityAttributes.swift")
	widgetPath := filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "DemoAppWidgetCore", "Sources", "DemoAppLiveActivityWidget.swift")
	managerPath := filepath.Join(projectRoot, "Targets", "DemoApp", "Sources", "LiveActivityManager.swift")

	requireFile(t, attributesPath)
	requireFile(t, widgetPath)
	requireFile(t, managerPath)

	// ActivityAttributes must be generated in SharedKit, not in widget extension.
	if _, err := os.Stat(filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppActivityAttributes.swift")); err == nil {
		t.Fatal("ActivityAttributes unexpectedly generated in widget extension Sources/")
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppLiveActivityWidget.swift")); err == nil {
		t.Fatal("Live Activity widget unexpectedly generated in widget extension Sources/")
	}

	attributes := readFile(t, attributesPath)
	for _, want := range []string{
		"import ActivityKit",
		"struct DemoAppActivityAttributes: ActivityAttributes",
		"struct ContentState: Codable, Hashable",
	} {
		if !strings.Contains(attributes, want) {
			t.Fatalf("ActivityAttributes file missing %q:\n%s", want, attributes)
		}
	}

	widget := readFile(t, widgetPath)
	for _, want := range []string{
		"import SharedKit",
		"public struct DemoAppLiveActivityWidget: Widget",
		"public init() {}",
		"public var body: some WidgetConfiguration",
		"ActivityConfiguration(for: DemoAppActivityAttributes.self)",
		"DynamicIsland",
	} {
		if !strings.Contains(widget, want) {
			t.Fatalf("Live Activity widget missing %q:\n%s", want, widget)
		}
	}

	manager := readFile(t, managerPath)
	for _, want := range []string{
		"import SharedKit",
		"Activity.request(",
		"await activity.update(content)",
		"await activity.end(nil, dismissalPolicy: dismissalPolicy)",
		"activity.pushTokenUpdates",
	} {
		if !strings.Contains(manager, want) {
			t.Fatalf("LiveActivityManager missing %q:\n%s", want, manager)
		}
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		`"NSSupportsLiveActivities": .boolean(true),`,
		`"NSSupportsLiveActivitiesFrequentUpdates": .boolean(true),`,
		`.sdk(name: "ActivityKit", type: .framework)`,
	} {
		if !strings.Contains(projectSwift, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectSwift)
		}
	}

	widgetProject := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Project.swift"))
	if !strings.Contains(widgetProject, `.sdk(name: "ActivityKit", type: .framework)`) {
		t.Fatalf("widget extension Project.swift missing ActivityKit dependency:\n%s", widgetProject)
	}

	widgetBundle := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppWidgetBundle.swift"))
	for _, want := range []string{
		"import DemoAppWidgetCore",
		"DemoAppLiveActivityWidget()",
	} {
		if !strings.Contains(widgetBundle, want) {
			t.Fatalf("WidgetBundle missing %q:\n%s", want, widgetBundle)
		}
	}
}

func TestSetupIdempotent(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	input := SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: "Packages",
	}

	if err := Setup(input); err != nil {
		t.Fatalf("first Setup() error = %v", err)
	}
	if err := Setup(input); err != nil {
		t.Fatalf("second Setup() error = %v", err)
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	if got := strings.Count(projectSwift, `"NSSupportsLiveActivities": .boolean(true),`); got != 1 {
		t.Fatalf("NSSupportsLiveActivities appears %d times, want 1:\n%s", got, projectSwift)
	}
	if got := strings.Count(projectSwift, `"NSSupportsLiveActivitiesFrequentUpdates": .boolean(true),`); got != 1 {
		t.Fatalf("NSSupportsLiveActivitiesFrequentUpdates appears %d times, want 1:\n%s", got, projectSwift)
	}
	if got := strings.Count(projectSwift, `.sdk(name: "ActivityKit", type: .framework)`); got != 1 {
		t.Fatalf("host Project.swift ActivityKit dependency appears %d times, want 1:\n%s", got, projectSwift)
	}

	widgetProject := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Project.swift"))
	if got := strings.Count(widgetProject, `.sdk(name: "ActivityKit", type: .framework)`); got != 1 {
		t.Fatalf("widget Project.swift ActivityKit dependency appears %d times, want 1:\n%s", got, widgetProject)
	}

	widgetBundle := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppWidgetBundle.swift"))
	if got := strings.Count(widgetBundle, "DemoAppLiveActivityWidget()"); got != 1 {
		t.Fatalf("WidgetBundle registration appears %d times, want 1:\n%s", got, widgetBundle)
	}
	if got := strings.Count(widgetBundle, "import DemoAppWidgetCore"); got != 1 {
		t.Fatalf("WidgetBundle Core import appears %d times, want 1:\n%s", got, widgetBundle)
	}
}

func TestSetupWithCustomModulesPath(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Modules")

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: "Modules",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	requireFile(t, filepath.Join(projectRoot, "Modules", "SharedKit", "Sources", "DemoAppActivityAttributes.swift"))
}

func TestGoldenActivityAttributesTemplate(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: "Packages",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	attributes := readFile(t, filepath.Join(projectRoot, "Packages", "SharedKit", "Sources", "DemoAppActivityAttributes.swift"))
	testutil.AssertGoldenFile(t, "liveactivity/activity_attributes", attributes)
}

func TestGoldenActivityConfigurationTemplate(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: "Packages",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	widget := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "DemoAppWidgetCore", "Sources", "DemoAppLiveActivityWidget.swift"))
	testutil.AssertGoldenFile(t, "liveactivity/activity_configuration", widget)
}

func TestGoldenLiveActivityManagerTemplate(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "Packages")

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
		ModulesPath: "Packages",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	manager := readFile(t, filepath.Join(projectRoot, "Targets", "DemoApp", "Sources", "LiveActivityManager.swift"))
	testutil.AssertGoldenFile(t, "liveactivity/live_activity_manager", manager)
}

func setupProjectFiles(t *testing.T, projectRoot, modulesPath string) {
	t.Helper()

	mkdirs(
		t,
		filepath.Join(projectRoot, modulesPath, "SharedKit", "Sources"),
		filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources"),
		filepath.Join(projectRoot, "Targets", "DemoApp", "Sources"),
	)

	projectSwift := `import ProjectDescription

let project = Project(
    name: "DemoApp",
    targets: [
        .target(
            name: "DemoApp",
            destinations: .iOS,
            product: .app,
            bundleId: "com.demo.app",
            infoPlist: .extendingDefault(
                with: [
                    "CFBundleDisplayName": .string("DemoApp"),
                    "UILaunchScreen": .dictionary([:]),
                ]
            ),
            dependencies: [
            ]
        )
    ]
)
`
	writeTestFile(t, filepath.Join(projectRoot, "Project.swift"), projectSwift)

	widgetProject := `import ProjectDescription

let project = Project(
    name: "DemoAppWidget",
    targets: [
        .target(
            name: "DemoAppWidget",
            destinations: .iOS,
            product: .appExtension,
            bundleId: "com.demo.app.widget",
            dependencies: [
                .sdk(name: "WidgetKit", type: .framework),
            ]
        )
    ]
)
`
	writeTestFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Project.swift"), widgetProject)

	widgetBundle := `import WidgetKit
import SwiftUI

@main
struct DemoAppWidgetBundle: WidgetBundle {
    var body: some Widget {
        // Widget plugins register here
    }
}
`
	writeTestFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppWidgetBundle.swift"), widgetBundle)
	writeTestFile(t, filepath.Join(projectRoot, modulesPath, "SharedKit", "Sources", "SharedKit.swift"), "public enum SharedKit {}\n")
}

func mkdirs(t *testing.T, paths ...string) {
	t.Helper()

	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", path, err)
		}
	}
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
