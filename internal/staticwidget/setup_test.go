package staticwidget

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

func TestSetupCreatesStaticWidgetFilesAndRegistersWidget(t *testing.T) {
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
	widgetPath := filepath.Join(widgetSourcesDir, "DemoAppWidget.swift")
	providerPath := filepath.Join(widgetSourcesDir, "DemoAppTimelineProvider.swift")
	entryPath := filepath.Join(widgetSourcesDir, "DemoAppTimelineEntry.swift")
	viewPath := filepath.Join(widgetSourcesDir, "DemoAppWidgetView.swift")

	requireFile(t, widgetPath)
	requireFile(t, providerPath)
	requireFile(t, entryPath)
	requireFile(t, viewPath)

	widget := readFile(t, widgetPath)
	for _, want := range []string{
		"struct DemoAppWidget: Widget",
		`let kind: String = "DemoAppWidget"`,
		"StaticConfiguration(kind: kind, provider: DemoAppTimelineProvider())",
		"DemoAppWidgetView(entry: entry)",
		`.configurationDisplayName("DemoApp")`,
		`.description("DemoApp widget")`,
	} {
		if !strings.Contains(widget, want) {
			t.Fatalf("DemoAppWidget.swift missing %q:\n%s", want, widget)
		}
	}

	provider := readFile(t, providerPath)
	for _, want := range []string{
		"struct DemoAppTimelineProvider: TimelineProvider",
		`UserDefaults(suiteName: "group.com.demo.shared")`,
		"func placeholder(in context: Context) -> DemoAppTimelineEntry",
		"func getSnapshot(in context: Context, completion: @escaping (DemoAppTimelineEntry) -> Void)",
		"func getTimeline(in context: Context, completion: @escaping (Timeline<DemoAppTimelineEntry>) -> Void)",
	} {
		if !strings.Contains(provider, want) {
			t.Fatalf("DemoAppTimelineProvider.swift missing %q:\n%s", want, provider)
		}
	}

	entry := readFile(t, entryPath)
	for _, want := range []string{
		"struct DemoAppTimelineEntry: TimelineEntry",
		"let date: Date",
		"let title: String",
		"let isToggled: Bool",
	} {
		if !strings.Contains(entry, want) {
			t.Fatalf("DemoAppTimelineEntry.swift missing %q:\n%s", want, entry)
		}
	}

	view := readFile(t, viewPath)
	for _, want := range []string{
		"import AppIntents",
		"struct DemoAppWidgetView: View",
		"let entry: DemoAppTimelineEntry",
		"Text(entry.title)",
		"Text(entry.date, style: .time)",
		"Button(intent: DemoAppWidgetToggleIntent())",
		"entry.isToggled",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("DemoAppWidgetView.swift missing %q:\n%s", want, view)
		}
	}

	widgetBundle := readFile(t, filepath.Join(widgetSourcesDir, "DemoAppWidgetBundle.swift"))
	if !strings.Contains(widgetBundle, "DemoAppWidget()") {
		t.Fatalf("WidgetBundle missing static widget registration:\n%s", widgetBundle)
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

	widgetBundle := readFile(
		t,
		filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppWidgetBundle.swift"),
	)
	if got := strings.Count(widgetBundle, "DemoAppWidget()"); got != 1 {
		t.Fatalf("WidgetBundle registration appears %d times, want 1:\n%s", got, widgetBundle)
	}
}

func TestGoldenStaticWidgetTemplate(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "DemoApp", "com.demo.app", []string{"group.com.demo.shared"})

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	widget := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppWidget.swift"))
	testutil.AssertGoldenFile(t, "staticwidget/static_widget", widget)
}

func TestGoldenTimelineProviderTemplate(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "DemoApp", "com.demo.app", []string{"group.com.demo.shared"})

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	provider := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppTimelineProvider.swift"))
	testutil.AssertGoldenFile(t, "staticwidget/timeline_provider", provider)
}

func TestGoldenTimelineEntryTemplate(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "DemoApp", "com.demo.app", []string{"group.com.demo.shared"})

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	entry := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppTimelineEntry.swift"))
	testutil.AssertGoldenFile(t, "staticwidget/timeline_entry", entry)
}

func TestGoldenWidgetViewTemplate(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	setupProjectFiles(t, projectRoot, "DemoApp", "com.demo.app", []string{"group.com.demo.shared"})

	if err := Setup(SetupInput{
		ProjectRoot: projectRoot,
		AppName:     "DemoApp",
	}); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	view := readFile(t, filepath.Join(projectRoot, "Extensions", "DemoAppWidget", "Sources", "DemoAppWidgetView.swift"))
	testutil.AssertGoldenFile(t, "staticwidget/widget_view", view)
}

func setupProjectFiles(t *testing.T, projectRoot, appName, bundleID string, appGroups []string) {
	t.Helper()

	widgetSourcesDir := filepath.Join(projectRoot, "Extensions", appName+"Widget", "Sources")
	if err := os.MkdirAll(widgetSourcesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", widgetSourcesDir, err)
	}

	widgetBundle := `import WidgetKit
import SwiftUI

@main
struct ` + appName + `WidgetBundle: WidgetBundle {
    var body: some Widget {
        // Widget plugins register here
    }
}
`
	writeTestFile(t, filepath.Join(widgetSourcesDir, appName+"WidgetBundle.swift"), widgetBundle)

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
