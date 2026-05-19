package profile

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestAnalyzeLayoutXMLFromProbeOutput(t *testing.T) {
	t.Parallel()

	raw := `<?xml version="1.0" encoding="UTF-8"?>
<layout source="XCTest" platform="iOS" screenWidth="390" screenHeight="844">
  <element type="application" identifier="DemoApp" x="0" y="0" width="390" height="844" enabled="true" hittable="false">
    <element type="window" x="0" y="0" width="390" height="844">
      <element type="button" identifier="primary" label="Continue" x="16" y="120" width="120" height="44" enabled="true" hittable="true" />
      <element type="button" identifier="primary" label="Duplicate" x="16" y="180" width="32" height="32" enabled="true" hittable="true" />
      <element type="button" x="16" y="240" width="80" height="44" enabled="true" hittable="true" />
      <element type="staticText" label="Title" x="16" y="-200" width="80" height="20" />
    </element>
  </element>
</layout>`

	report := AnalyzeLayoutXML(raw, LayoutAnalyzeOptions{MinTapSize: 44, MaxElements: 20})
	if report.ElementCount != 6 {
		t.Fatalf("ElementCount = %d, want 6", report.ElementCount)
	}
	if report.MaxDepth != 2 {
		t.Fatalf("MaxDepth = %d, want 2", report.MaxDepth)
	}
	if report.Screen.Width != 390 || report.Screen.Height != 844 {
		t.Fatalf("Screen = %#v, want 390x844", report.Screen)
	}
	if len(report.DuplicateIdentities) != 1 || report.DuplicateIdentities[0].Identity != "primary" {
		t.Fatalf("DuplicateIdentities = %#v, want duplicate primary", report.DuplicateIdentities)
	}

	assertLayoutIssue(t, report, "missing-accessibility-identity")
	assertLayoutIssue(t, report, "tiny-tap-target")
	assertLayoutIssue(t, report, "offscreen")
}

func TestAnalyzeLayoutXMLFromIAMMarkedLog(t *testing.T) {
	t.Parallel()

	raw := `noise before
IAM_LAYOUT_XML_START Feed
<layout screenWidth="100" screenHeight="100">
  <element type="application" x="0" y="0" width="100" height="100" />
</layout>
IAM_LAYOUT_XML_END Feed
noise after`

	report := AnalyzeLayoutXML(raw, LayoutAnalyzeOptions{})
	if len(report.ParseErrors) != 0 {
		t.Fatalf("ParseErrors = %#v, want none", report.ParseErrors)
	}
	if report.ElementCount != 1 {
		t.Fatalf("ElementCount = %d, want 1", report.ElementCount)
	}
}

func TestAnalyzeLayoutXMLFromAppiumSource(t *testing.T) {
	t.Parallel()

	raw := `<AppiumAUT>
  <XCUIElementTypeApplication type="XCUIElementTypeApplication" name="DemoApp" x="0" y="0" width="390" height="844">
    <XCUIElementTypeWindow type="XCUIElementTypeWindow" x="0" y="0" width="390" height="844">
      <XCUIElementTypeButton type="XCUIElementTypeButton" name="loginButton" label="Log in" visible="true" enabled="true" x="20" y="40" width="88" height="44" />
    </XCUIElementTypeWindow>
  </XCUIElementTypeApplication>
</AppiumAUT>`

	report := AnalyzeLayoutXML(raw, LayoutAnalyzeOptions{})
	if report.Source != "Appium/WDA" {
		t.Fatalf("Source = %q, want Appium/WDA", report.Source)
	}
	if report.ElementCount != 3 {
		t.Fatalf("ElementCount = %d, want 3", report.ElementCount)
	}
	if got := report.Elements[2].Type; got != "button" {
		t.Fatalf("button type = %q, want button", got)
	}
	if got := report.Elements[2].Name; got != "loginButton" {
		t.Fatalf("button name = %q, want loginButton", got)
	}
}

func TestScaffoldLayoutProbeWritesXCTestHelper(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	result, err := ScaffoldLayoutProbe(LayoutScaffoldOptions{
		ProjectRoot: root,
		Config: config.ProjectConfig{
			AppName:  "DemoApp",
			BundleID: "com.example.demo",
		},
	})
	if err != nil {
		t.Fatalf("ScaffoldLayoutProbe() error = %v", err)
	}

	wantPath := filepath.Join(root, "Targets", "DemoAppUITests", "Sources", "Diagnostics", "LayoutHierarchyProbe.swift")
	if result.Path != wantPath {
		t.Fatalf("path = %q, want %q", result.Path, wantPath)
	}

	content, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("read generated probe: %v", err)
	}
	for _, marker := range []string{"import XCTest", "LayoutHierarchyProbe", "IAM_LAYOUT_XML_START", "attachLayoutHierarchyXML", "children(matching: .any)"} {
		if !strings.Contains(string(content), marker) {
			t.Fatalf("generated probe missing %q:\n%s", marker, content)
		}
	}
}

func TestGeneratedLayoutHierarchyProbeTypechecks(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "darwin" {
		t.Skip("Swift/XCTest typecheck requires Darwin")
	}
	if _, err := os.Stat("/usr/bin/xcrun"); err != nil {
		t.Skip("xcrun is unavailable")
	}

	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "LayoutHierarchyProbe.swift")
	if err := os.WriteFile(sourcePath, []byte(GenerateLayoutHierarchyProbeSwift()), 0o644); err != nil {
		t.Fatalf("write generated swift: %v", err)
	}

	sdkPath, err := ExecRunner{}.Run(context.Background(), dir, "xcrun", "--sdk", "iphonesimulator", "--show-sdk-path")
	if err != nil {
		t.Skipf("iphonesimulator SDK unavailable: %v", err)
	}
	frameworkPath, err := ExecRunner{}.Run(context.Background(), dir, "xcode-select", "-p")
	if err != nil {
		t.Skipf("xcode-select unavailable: %v", err)
	}
	frameworkDir := filepath.Join(strings.TrimSpace(string(frameworkPath)), "Platforms", "iPhoneSimulator.platform", "Developer", "Library", "Frameworks")
	_, err = ExecRunner{}.Run(
		context.Background(),
		dir,
		"xcrun",
		"swiftc",
		"-typecheck",
		"-parse-as-library",
		"-target",
		"arm64-apple-ios17.0-simulator",
		"-sdk",
		strings.TrimSpace(string(sdkPath)),
		"-F",
		frameworkDir,
		sourcePath,
	)
	if err != nil {
		t.Fatalf("generated LayoutHierarchyProbe.swift failed typecheck: %v", err)
	}
}

func assertLayoutIssue(t *testing.T, report LayoutReport, kind string) {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Kind == kind {
			return
		}
	}
	t.Fatalf("issue %q not found in %#v", kind, report.Issues)
}
