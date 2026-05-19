package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileBuildAnalyzesExistingLog(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)
	logPath := filepath.Join(projectRoot, "build.log")
	if err := os.WriteFile(logPath, []byte(`Build Timing Summary
CompileSwiftSources (in target 'Core' from project 'Core') 4.0 seconds
CompileSwiftSources (in target 'DemoApp' from project 'DemoApp') 2.0 seconds
`), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	output, err := executeRootCommand("profile", "build", "--config", configPath, "--log", logPath, "--skip-graph")
	if err != nil {
		t.Fatalf("executeRootCommand(profile build) error = %v", err)
	}

	for _, want := range []string{
		"build profile:",
		"timing entries: 2",
		"total target work: 6.00s",
		"Core",
		"DemoApp",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("profile build output missing %q:\n%s", want, output)
		}
	}
}

func TestProfileRuntimeScaffoldAndAnalyze(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)

	scaffoldOutput, err := executeRootCommand("profile", "runtime", "scaffold", "--config", configPath)
	if err != nil {
		t.Fatalf("executeRootCommand(profile runtime scaffold) error = %v", err)
	}
	if !strings.Contains(scaffoldOutput, "runtime profile helper written") {
		t.Fatalf("scaffold output missing success message:\n%s", scaffoldOutput)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "Targets", cfg.AppName, "Sources", "Diagnostics", "PerformanceProbe.swift")); err != nil {
		t.Fatalf("runtime probe was not written: %v", err)
	}

	logPath := filepath.Join(projectRoot, "runtime.log")
	if err := os.WriteFile(logPath, []byte(`IAM_PROFILE {"kind":"function","name":"Feed.filter","duration_ms":20,"thread":"main"}
IAM_PROFILE {"kind":"function","name":"Feed.filter","duration_ms":5,"thread":"main"}
IAM_PROFILE {"kind":"app_start","name":"AppStart","timestamp":100}
IAM_PROFILE {"kind":"first_render","name":"FeedScreen","timestamp":100.5}
`), 0o644); err != nil {
		t.Fatalf("write runtime log: %v", err)
	}

	analyzeOutput, err := executeRootCommand("profile", "runtime", "analyze", "--input", logPath, "--repeat-threshold", "2")
	if err != nil {
		t.Fatalf("executeRootCommand(profile runtime analyze) error = %v", err)
	}
	for _, want := range []string{"runtime profile:", "events: 4", "app startup to first render: 500.00ms", "Feed.filter", "called 2 times"} {
		if !strings.Contains(analyzeOutput, want) {
			t.Fatalf("runtime analyze output missing %q:\n%s", want, analyzeOutput)
		}
	}
}

func TestProfileRuntimeErrorsAnalyzesInput(t *testing.T) {
	t.Parallel()

	logPath := filepath.Join(t.TempDir(), "errors.log")
	if err := os.WriteFile(logPath, []byte(`{"logType":"error","process":"DemoApp","subsystem":"com.example.demo","category":"network","composedMessage":"Request failed with code 500"}
IAM_ERROR {"severity":"fault","message":"Uncaught exception NSInvalidArgumentException","process":"DemoApp"}
`), 0o644); err != nil {
		t.Fatalf("write runtime errors log: %v", err)
	}

	output, err := executeRootCommand("profile", "runtime", "errors", "--input", logPath)
	if err != nil {
		t.Fatalf("executeRootCommand(profile runtime errors) error = %v", err)
	}
	for _, want := range []string{"runtime errors:", "events: 2", "Request failed", "exception/fault"} {
		if !strings.Contains(output, want) {
			t.Fatalf("runtime errors output missing %q:\n%s", want, output)
		}
	}
}

func TestProfileLayoutScaffoldAndAnalyze(t *testing.T) {
	t.Parallel()

	cfg := testProjectConfig()
	configPath := writeTestConfig(t, cfg)
	projectRoot := filepath.Dir(configPath)

	scaffoldOutput, err := executeRootCommand("profile", "layout", "scaffold", "--config", configPath)
	if err != nil {
		t.Fatalf("executeRootCommand(profile layout scaffold) error = %v", err)
	}
	if !strings.Contains(scaffoldOutput, "layout hierarchy helper written") {
		t.Fatalf("layout scaffold output missing success message:\n%s", scaffoldOutput)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "Targets", cfg.AppName+"UITests", "Sources", "Diagnostics", "LayoutHierarchyProbe.swift")); err != nil {
		t.Fatalf("layout probe was not written: %v", err)
	}

	layoutPath := filepath.Join(projectRoot, "layout.xml")
	if err := os.WriteFile(layoutPath, []byte(`<layout source="XCTest" screenWidth="390" screenHeight="844">
  <element type="application" identifier="DemoApp" x="0" y="0" width="390" height="844">
    <element type="button" identifier="primary" label="Continue" x="16" y="100" width="120" height="44" hittable="true" />
    <element type="button" identifier="primary" label="Again" x="16" y="160" width="20" height="20" hittable="true" />
  </element>
</layout>`), 0o644); err != nil {
		t.Fatalf("write layout XML: %v", err)
	}

	output, err := executeRootCommand("profile", "layout", "analyze", "--input", layoutPath)
	if err != nil {
		t.Fatalf("executeRootCommand(profile layout analyze) error = %v", err)
	}
	for _, want := range []string{"layout hierarchy:", "elements: 3", "duplicate identities:", "tiny-tap-target", "primary"} {
		if !strings.Contains(output, want) {
			t.Fatalf("layout analyze output missing %q:\n%s", want, output)
		}
	}
}

func TestProfileHelpIncludesBuildAndRuntime(t *testing.T) {
	t.Parallel()

	output, err := executeRootCommand("profile", "--help")
	if err != nil {
		t.Fatalf("executeRootCommand(profile --help) error = %v", err)
	}
	for _, want := range []string{"build", "layout", "runtime"} {
		if !strings.Contains(output, want) {
			t.Fatalf("profile help missing %q:\n%s", want, output)
		}
	}
}
