package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestAnalyzeRuntimeProfileLog(t *testing.T) {
	t.Parallel()

	raw := `
noise
IAM_PROFILE {"kind":"app_start","name":"AppStart","timestamp":100}
IAM_PROFILE {"kind":"first_render","name":"FeedScreen","timestamp":101.25}
IAM_PROFILE {"kind":"function","name":"Feed.filter","duration_ms":20,"thread":"main"}
IAM_PROFILE {"kind":"function","name":"Feed.filter","duration_ms":5,"thread":"main"}
IAM_PROFILE {"kind":"view_body","name":"FeedScreen","duration_ms":1,"thread":"main"}
IAM_PROFILE {"kind":"function","name":"Background.sync","duration_ms":50,"thread":"background"}
`

	report := AnalyzeRuntimeProfileLog(raw, RuntimeAnalyzeOptions{SlowThresholdMS: 16, RepeatThreshold: 2})
	if report.EventCount != 6 {
		t.Fatalf("event count = %d, want 6", report.EventCount)
	}
	if report.Startup == nil || report.Startup.DurationMS != 1250 || report.Startup.FirstRenderName != "FeedScreen" {
		t.Fatalf("startup = %#v, want FeedScreen 1250ms", report.Startup)
	}
	if len(report.Groups) != 5 {
		t.Fatalf("group count = %d, want 5: %#v", len(report.Groups), report.Groups)
	}

	var feed RuntimeGroup
	for _, group := range report.Groups {
		if group.Name == "Feed.filter" {
			feed = group
		}
	}
	if feed.Count != 2 || feed.TotalDurationMS != 25 || feed.SlowCount != 1 {
		t.Fatalf("Feed.filter group = %#v", feed)
	}
	if len(report.Warnings) != 2 {
		t.Fatalf("warnings = %#v, want repeated + slow warnings", report.Warnings)
	}
}

func TestScaffoldRuntimeProbeWritesDebugHelper(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	result, err := ScaffoldRuntimeProbe(RuntimeScaffoldOptions{
		ProjectRoot: root,
		Config: config.ProjectConfig{
			AppName:  "DemoApp",
			BundleID: "com.example.demo",
		},
	})
	if err != nil {
		t.Fatalf("ScaffoldRuntimeProbe() error = %v", err)
	}

	wantPath := filepath.Join(root, "Targets", "DemoApp", "Sources", "Diagnostics", "PerformanceProbe.swift")
	if result.Path != wantPath {
		t.Fatalf("path = %q, want %q", result.Path, wantPath)
	}

	content, err := osReadFile(result.Path)
	if err != nil {
		t.Fatalf("read generated probe: %v", err)
	}
	for _, marker := range []string{"#if DEBUG", "public static func measure", "markAppStart", "firstRenderProfiled", "IAM_ERROR", "IAM_PROFILE", "func profiled"} {
		if !strings.Contains(content, marker) {
			t.Fatalf("generated probe missing %q:\n%s", marker, content)
		}
	}
}

func osReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
