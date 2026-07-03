package testtargets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestSetupRequiresAtLeastOneTarget(t *testing.T) {
	t.Parallel()

	err := Setup(SetupInput{
		ProjectRoot: t.TempDir(),
		AppName:     "DemoApp",
	})
	if err == nil {
		t.Fatal("Setup() error = nil, want missing target error")
	}
	if !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("Setup() error = %q, want at least one target", err.Error())
	}
}

func TestSetupCreatesUnitAndUITestTargets(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeProjectFiles(t, projectRoot)
	writeProjectConfig(t, projectRoot)

	err := Setup(SetupInput{
		ProjectRoot:    projectRoot,
		AppName:        "DemoApp",
		UnitTargetName: "DemoAppTests",
		UITargetName:   "DemoAppUITests",
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	unitSource := readFile(t, filepath.Join(projectRoot, "Targets", "DemoAppTests", "Sources", "DemoAppTests.swift"))
	for _, want := range []string{
		"import Testing",
		"@testable import DemoApp",
		"struct DemoAppTests",
		"#expect(true)",
	} {
		if !strings.Contains(unitSource, want) {
			t.Fatalf("unit source missing %q:\n%s", want, unitSource)
		}
	}

	uiSource := readFile(t, filepath.Join(projectRoot, "Targets", "DemoAppUITests", "Sources", "DemoAppUITests.swift"))
	for _, want := range []string{
		"import XCTest",
		"final class DemoAppUITests: XCTestCase",
		"XCUIApplication()",
		"runningForeground",
	} {
		if !strings.Contains(uiSource, want) {
			t.Fatalf("UI source missing %q:\n%s", want, uiSource)
		}
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, want := range []string{
		`name: "DemoAppTests"`,
		`product: .unitTests`,
		`bundleId: "\(bundleID).tests"`,
		`name: "DemoAppUITests"`,
		`product: .uiTests`,
		`bundleId: "\(bundleID).ui-tests"`,
		`.target(name: appName)`,
		`"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget)`,
		`"DEVELOPMENT_TEAM": .string(developmentTeam)`,
		`"SWIFT_VERSION": "6.0"`,
	} {
		if !strings.Contains(projectSwift, want) {
			t.Fatalf("Project.swift missing %q:\n%s", want, projectSwift)
		}
	}
}

func TestSetupCanCreateOnlyUnitTarget(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeProjectFiles(t, projectRoot)
	writeProjectConfig(t, projectRoot)

	err := Setup(SetupInput{
		ProjectRoot:    projectRoot,
		AppName:        "DemoApp",
		UnitTargetName: "DemoAppTests",
	})
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	if !strings.Contains(projectSwift, `name: "DemoAppTests"`) {
		t.Fatalf("Project.swift missing unit test target:\n%s", projectSwift)
	}
	if strings.Contains(projectSwift, ".uiTests") {
		t.Fatalf("Project.swift unexpectedly contains UI test target:\n%s", projectSwift)
	}
}

func TestSetupIsIdempotentAndPreservesExistingStarterFiles(t *testing.T) {
	t.Parallel()

	projectRoot := t.TempDir()
	writeProjectFiles(t, projectRoot)
	writeProjectConfig(t, projectRoot)

	input := SetupInput{
		ProjectRoot:    projectRoot,
		AppName:        "DemoApp",
		UnitTargetName: "DemoAppTests",
		UITargetName:   "DemoAppUITests",
	}

	if err := Setup(input); err != nil {
		t.Fatalf("first Setup() error = %v", err)
	}

	unitSourcePath := filepath.Join(projectRoot, "Targets", "DemoAppTests", "Sources", "DemoAppTests.swift")
	customUnitSource := "// handwritten test source\n"
	if err := os.WriteFile(unitSourcePath, []byte(customUnitSource), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", unitSourcePath, err)
	}

	if err := Setup(input); err != nil {
		t.Fatalf("second Setup() error = %v", err)
	}

	projectSwift := readFile(t, filepath.Join(projectRoot, "Project.swift"))
	for _, targetName := range []string{`name: "DemoAppTests"`, `name: "DemoAppUITests"`} {
		if got := strings.Count(projectSwift, targetName); got != 1 {
			t.Fatalf("%s appears %d times, want 1:\n%s", targetName, got, projectSwift)
		}
	}

	unitSource := readFile(t, unitSourcePath)
	if unitSource != customUnitSource {
		t.Fatalf("existing unit source was overwritten:\n%s", unitSource)
	}
}

func TestNewPlanRejectsInvalidTargetNames(t *testing.T) {
	t.Parallel()

	_, err := NewPlan(SetupInput{
		ProjectRoot:    t.TempDir(),
		AppName:        "DemoApp",
		UnitTargetName: "demo-app-tests",
	})
	if err == nil {
		t.Fatal("NewPlan() error = nil, want invalid target name")
	}
	if !strings.Contains(err.Error(), "must match") {
		t.Fatalf("NewPlan() error = %q, want name validation", err.Error())
	}
}

func writeProjectFiles(t *testing.T, projectRoot string) {
	t.Helper()

	projectSwift := `import ProjectDescription
import ProjectDescriptionHelpers

let appName = "DemoApp"
let bundleID = "com.demo.app"
let developmentTeam = "TEAM123456"
let marketingVersion = "1.0.0"
let currentProjectVersion = "1"
let minTarget = "17.0"

let project = Project(
    name: appName,
    targets: [
        .target(
            name: appName,
            destinations: .iOS,
            product: .app,
            bundleId: bundleID,
            deploymentTargets: .iOS(minTarget),
            sources: ["Targets/DemoApp/Sources/**"],
            dependencies: [
            ]
        )
    ]
)
`
	writeFile(t, filepath.Join(projectRoot, "Project.swift"), projectSwift)
}

func writeProjectConfig(t *testing.T, projectRoot string) {
	t.Helper()

	cfg := config.ProjectConfig{
		AppName:          "DemoApp",
		BundleID:         "com.demo.app",
		TeamID:           "TEAM123456",
		SwiftVersion:     "6.0",
		MinTarget:        "17.0",
		MarketingVersion: "1.0.0",
		ProjectVersion:   "1",
	}

	payload, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal(config) error = %v", err)
	}
	writeFile(t, filepath.Join(projectRoot, config.DefaultConfigPath), string(payload))
}

func writeFile(t *testing.T, path string, content string) {
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

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(payload)
}
