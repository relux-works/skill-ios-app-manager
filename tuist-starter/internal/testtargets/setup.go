package testtargets

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const (
	defaultTargetRootDirectory = "Targets"
)

var targetNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// SetupInput holds parameters for test target setup.
type SetupInput struct {
	ProjectRoot    string
	AppName        string
	UnitTargetName string
	UITargetName   string
}

// Plan is the normalized test target setup plan.
type Plan struct {
	ProjectRoot    string
	AppName        string
	UnitTargetName string
	UITargetName   string
	UnitTypeName   string
	UITestTypeName string
}

type targetKind string

const (
	targetKindUnit targetKind = "unit"
	targetKindUI   targetKind = "ui"
)

type subplugin interface {
	Spec(Plan) (targetSpec, bool)
}

type unitTestSubplugin struct{}
type uiTestSubplugin struct{}

type targetSpec struct {
	Kind       targetKind
	Name       string
	TypeName   string
	Product    string
	BundleStem string
	Source     string
}

// NewPlan validates and normalizes setup input.
func NewPlan(input SetupInput) (Plan, error) {
	if strings.TrimSpace(input.ProjectRoot) == "" {
		return Plan{}, fmt.Errorf("project root is required")
	}
	appName := strings.TrimSpace(input.AppName)
	if appName == "" {
		return Plan{}, fmt.Errorf("app name is required")
	}

	unitTargetName := strings.TrimSpace(input.UnitTargetName)
	uiTargetName := strings.TrimSpace(input.UITargetName)
	if unitTargetName == "" && uiTargetName == "" {
		return Plan{}, fmt.Errorf("at least one of --unit-target or --ui-target is required")
	}
	if unitTargetName != "" {
		if err := validateTargetName(unitTargetName, "unit target"); err != nil {
			return Plan{}, err
		}
	}
	if uiTargetName != "" {
		if err := validateTargetName(uiTargetName, "UI target"); err != nil {
			return Plan{}, err
		}
	}
	if unitTargetName != "" && uiTargetName != "" && unitTargetName == uiTargetName {
		return Plan{}, fmt.Errorf("unit and UI target names must differ")
	}

	return Plan{
		ProjectRoot:    strings.TrimSpace(input.ProjectRoot),
		AppName:        appName,
		UnitTargetName: unitTargetName,
		UITargetName:   uiTargetName,
		UnitTypeName:   scaffold.SwiftTypeName(unitTargetName),
		UITestTypeName: scaffold.SwiftTypeName(uiTargetName),
	}, nil
}

// Setup creates requested test targets and starter sources.
func Setup(input SetupInput) error {
	plan, err := NewPlan(input)
	if err != nil {
		return err
	}

	cfg, err := loadConfig(plan.ProjectRoot)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	projectSwiftPath := filepath.Join(plan.ProjectRoot, "Project.swift")
	for _, spec := range targetSpecs(plan) {
		if err := ensureTestSource(plan.ProjectRoot, spec); err != nil {
			return err
		}
		if err := ensureProjectTarget(projectSwiftPath, plan.AppName, cfg, spec); err != nil {
			return err
		}
	}

	return nil
}

func targetSpecs(plan Plan) []targetSpec {
	plugins := []subplugin{
		unitTestSubplugin{},
		uiTestSubplugin{},
	}

	specs := make([]targetSpec, 0, len(plugins))
	for _, plugin := range plugins {
		spec, ok := plugin.Spec(plan)
		if !ok {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}

func (unitTestSubplugin) Spec(plan Plan) (targetSpec, bool) {
	if plan.UnitTargetName == "" {
		return targetSpec{}, false
	}
	return targetSpec{
		Kind:       targetKindUnit,
		Name:       plan.UnitTargetName,
		TypeName:   plan.UnitTypeName,
		Product:    ".unitTests",
		BundleStem: "tests",
		Source:     unitTestSource(plan.AppName, plan.UnitTypeName),
	}, true
}

func (uiTestSubplugin) Spec(plan Plan) (targetSpec, bool) {
	if plan.UITargetName == "" {
		return targetSpec{}, false
	}
	return targetSpec{
		Kind:       targetKindUI,
		Name:       plan.UITargetName,
		TypeName:   plan.UITestTypeName,
		Product:    ".uiTests",
		BundleStem: "ui-tests",
		Source:     uiTestSource(plan.UITestTypeName),
	}, true
}

func validateTargetName(name, label string) error {
	if !targetNamePattern.MatchString(name) {
		return fmt.Errorf("%s name %q must match %s", label, name, targetNamePattern.String())
	}
	return nil
}

func loadConfig(projectRoot string) (config.ProjectConfig, error) {
	cfgPath := filepath.Join(projectRoot, config.DefaultConfigPath)
	return config.LoadConfig(cfgPath)
}

func ensureTestSource(projectRoot string, spec targetSpec) error {
	sourceDir := filepath.Join(projectRoot, defaultTargetRootDirectory, spec.Name, "Sources")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return fmt.Errorf("create %s source directory: %w", spec.Name, err)
	}

	sourcePath := filepath.Join(sourceDir, spec.TypeName+".swift")
	if _, err := os.Stat(sourcePath); err == nil {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat %s source file: %w", spec.Name, err)
	}

	if err := os.WriteFile(sourcePath, []byte(spec.Source), 0o644); err != nil {
		return fmt.Errorf("write %s source file: %w", spec.Name, err)
	}
	return nil
}

func ensureProjectTarget(projectSwiftPath, appName string, cfg config.ProjectConfig, spec targetSpec) error {
	content := projectTargetContent(appName, cfg, spec)
	err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddTarget,
		Name:    spec.Name,
		Content: content,
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	if err != nil {
		return fmt.Errorf("add %s target to Project.swift: %w", spec.Name, err)
	}
	return nil
}

func projectTargetContent(appName string, cfg config.ProjectConfig, spec targetSpec) string {
	lines := []string{
		".target(",
		fmt.Sprintf("    name: %s,", strconv.Quote(spec.Name)),
		"    destinations: .iOS,",
		fmt.Sprintf("    product: %s,", spec.Product),
		fmt.Sprintf("    bundleId: \"\\(bundleID).%s\",", spec.BundleStem),
		"    deploymentTargets: .iOS(minTarget),",
		"    infoPlist: .default,",
		fmt.Sprintf("    sources: [%s],", strconv.Quote(filepath.ToSlash(filepath.Join(defaultTargetRootDirectory, spec.Name, "Sources/**")))),
		"    dependencies: [",
		"        .target(name: appName),",
		"    ],",
		"    settings: .settings(",
		"        base: [",
	}

	for _, setting := range sortedSwiftBuildSettings(cfg.EffectiveSwiftSettings().XcodeBuildSettings()) {
		lines = append(lines, fmt.Sprintf("            %s: %s,", strconv.Quote(setting.Key), strconv.Quote(setting.Value)))
	}

	lines = append(lines,
		"            \"IPHONEOS_DEPLOYMENT_TARGET\": .string(minTarget),",
		"            \"DEVELOPMENT_TEAM\": .string(developmentTeam),",
		"        ]",
		"    )",
		")",
	)

	return strings.Join(lines, "\n")
}

func sortedSwiftBuildSettings(settings []config.SwiftBuildSetting) []config.SwiftBuildSetting {
	out := append([]config.SwiftBuildSetting(nil), settings...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out
}

func unitTestSource(appName, typeName string) string {
	return fmt.Sprintf(`import Testing
@testable import %s

@Suite
struct %s {
    @Test
    func scaffoldBuilds() {
        #expect(true)
    }
}
`, scaffold.SwiftTypeName(appName), typeName)
}

func uiTestSource(typeName string) string {
	return fmt.Sprintf(`import XCTest

final class %s: XCTestCase {
    func testAppLaunches() throws {
        let app = XCUIApplication()
        app.launch()

        XCTAssertTrue(app.wait(for: .runningForeground, timeout: 5))
    }
}
`, typeName)
}
