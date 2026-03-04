package liveactivity

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/relux-works/ios-app-manager/internal/ioc"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const (
	sharedKitPackageName                    = "SharedKit"
	extensionsDirectoryName                 = "Extensions"
	widgetExtensionSuffix                   = "Widget"
	liveActivityInfoPlistKey                = "NSSupportsLiveActivities"
	liveActivityFrequentUpdatesInfoPlistKey = "NSSupportsLiveActivitiesFrequentUpdates"
	activityKitDependencyName               = "ActivityKit"
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// SetupInput holds parameters for live-activity setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
}

type templateData struct {
	AppTypeName string
}

// Setup scaffolds Live Activity shared models, widget configuration, and app manager files.
func Setup(input SetupInput) error {
	if err := validateInput(input); err != nil {
		return err
	}

	appName := strings.TrimSpace(input.AppName)
	appTypeName := scaffold.SwiftTypeName(appName)
	data := templateData{
		AppTypeName: appTypeName,
	}

	modulesRoot := ioc.ResolveModulesPath(input.ProjectRoot, input.ModulesPath)
	sharedKitSourcesDir := filepath.Join(modulesRoot, sharedKitPackageName, "Sources")
	if err := os.MkdirAll(sharedKitSourcesDir, 0o755); err != nil {
		return fmt.Errorf("create SharedKit sources directory: %w", err)
	}

	attributesPath := filepath.Join(sharedKitSourcesDir, appTypeName+"ActivityAttributes.swift")
	if err := renderTemplate("activity_attributes.swift.tmpl", attributesPath, data); err != nil {
		return fmt.Errorf("render ActivityAttributes: %w", err)
	}

	widgetTargetName := appTypeName + widgetExtensionSuffix
	widgetRoot := filepath.Join(input.ProjectRoot, extensionsDirectoryName, widgetTargetName)
	widgetSourcesDir := filepath.Join(widgetRoot, "Sources")
	if err := os.MkdirAll(widgetSourcesDir, 0o755); err != nil {
		return fmt.Errorf("create widget sources directory: %w", err)
	}

	liveActivityWidgetPath := filepath.Join(widgetSourcesDir, appTypeName+"LiveActivityWidget.swift")
	if err := renderTemplate("activity_configuration.swift.tmpl", liveActivityWidgetPath, data); err != nil {
		return fmt.Errorf("render live activity widget: %w", err)
	}

	widgetProjectPath := filepath.Join(widgetRoot, "Project.swift")
	if err := addActivityKitDependency(widgetProjectPath); err != nil {
		return fmt.Errorf("add ActivityKit dependency to widget extension: %w", err)
	}
	if err := addSharedKitDependency(widgetProjectPath); err != nil {
		return fmt.Errorf("add SharedKit dependency to widget extension: %w", err)
	}

	widgetBundlePath := filepath.Join(widgetSourcesDir, widgetTargetName+"Bundle.swift")
	if err := appendWidgetBundleEntry(widgetBundlePath, appTypeName+"LiveActivityWidget"); err != nil {
		return fmt.Errorf("register live activity widget in WidgetBundle: %w", err)
	}

	appSourcesDir := filepath.Join(input.ProjectRoot, "Targets", appName, "Sources")
	if err := os.MkdirAll(appSourcesDir, 0o755); err != nil {
		return fmt.Errorf("create app sources directory: %w", err)
	}

	managerPath := filepath.Join(appSourcesDir, "LiveActivityManager.swift")
	if err := renderTemplate("live_activity_manager.swift.tmpl", managerPath, data); err != nil {
		return fmt.Errorf("render LiveActivityManager: %w", err)
	}

	projectSwiftPath := filepath.Join(input.ProjectRoot, "Project.swift")
	if err := addActivityKitDependency(projectSwiftPath); err != nil {
		return fmt.Errorf("add ActivityKit dependency to host app: %w", err)
	}
	if err := addLiveActivityInfoPlistKeys(projectSwiftPath); err != nil {
		return fmt.Errorf("patch host app Info.plist: %w", err)
	}

	return nil
}

func validateInput(input SetupInput) error {
	if strings.TrimSpace(input.ProjectRoot) == "" {
		return fmt.Errorf("project root is required")
	}
	if strings.TrimSpace(input.AppName) == "" {
		return fmt.Errorf("app name is required")
	}
	return nil
}

func renderTemplate(templateName, outputPath string, data any) error {
	tmplPath := "setup_templates/" + templateName
	content, err := setupTemplatesFS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("read template %q: %w", tmplPath, err)
	}

	tmpl, err := template.New(templateName).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template %q: %w", templateName, err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create template output directory: %w", err)
	}

	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, data); err != nil {
		return fmt.Errorf("execute template %q: %w", templateName, err)
	}

	if err := os.WriteFile(outputPath, []byte(rendered.String()), 0o644); err != nil {
		return fmt.Errorf("write rendered template: %w", err)
	}

	return nil
}

func addSharedKitDependency(projectSwiftPath string) error {
	err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    sharedKitPackageName,
		Content: fmt.Sprintf(`.external(name: "%s")`, sharedKitPackageName),
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func addActivityKitDependency(projectSwiftPath string) error {
	err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    activityKitDependencyName,
		Content: `.sdk(name: "ActivityKit", type: .framework)`,
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func addLiveActivityInfoPlistKeys(projectSwiftPath string) error {
	payload, err := os.ReadFile(projectSwiftPath)
	if err != nil {
		return fmt.Errorf("read Project.swift: %w", err)
	}
	content := string(payload)

	keyLine := fmt.Sprintf(`"%s": .boolean(true),`, liveActivityInfoPlistKey)
	frequentKeyLine := fmt.Sprintf(`"%s": .boolean(true),`, liveActivityFrequentUpdatesInfoPlistKey)
	hasPrimary := strings.Contains(content, keyLine)
	hasFrequent := strings.Contains(content, frequentKeyLine)
	if hasPrimary && hasFrequent {
		return nil
	}

	marker := `                    "UILaunchScreen": .dictionary([:]),`
	if !strings.Contains(content, marker) {
		return fmt.Errorf("Project.swift missing %q marker", marker)
	}

	var insert strings.Builder
	if !hasPrimary {
		insert.WriteString(fmt.Sprintf("                    %s\n", keyLine))
	}
	if !hasFrequent {
		insert.WriteString(fmt.Sprintf("                    %s\n", frequentKeyLine))
	}

	updated := strings.Replace(content, marker, insert.String()+marker, 1)
	if err := os.WriteFile(projectSwiftPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write Project.swift: %w", err)
	}

	return nil
}

func appendWidgetBundleEntry(widgetBundlePath, widgetTypeName string) error {
	payload, err := os.ReadFile(widgetBundlePath)
	if err != nil {
		return fmt.Errorf("read WidgetBundle file: %w", err)
	}
	content := string(payload)

	registrationLine := fmt.Sprintf("%s()", widgetTypeName)
	if strings.Contains(content, registrationLine) {
		return nil
	}

	anchor := "// Widget plugins register here"
	if strings.Contains(content, anchor) {
		replacement := fmt.Sprintf("        %s\n        %s", registrationLine, anchor)
		updated := strings.Replace(content, anchor, replacement, 1)
		if err := os.WriteFile(widgetBundlePath, []byte(updated), 0o644); err != nil {
			return fmt.Errorf("write WidgetBundle file: %w", err)
		}
		return nil
	}

	bodyMarker := "var body: some Widget {"
	bodyIdx := strings.Index(content, bodyMarker)
	if bodyIdx < 0 {
		return fmt.Errorf("WidgetBundle missing %q marker", bodyMarker)
	}

	lineEnd := strings.Index(content[bodyIdx:], "\n")
	if lineEnd < 0 {
		return fmt.Errorf("WidgetBundle body marker line has no newline")
	}

	insertPos := bodyIdx + lineEnd + 1
	updated := content[:insertPos] + fmt.Sprintf("        %s\n", registrationLine) + content[insertPos:]
	if err := os.WriteFile(widgetBundlePath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write WidgetBundle file: %w", err)
	}

	return nil
}
