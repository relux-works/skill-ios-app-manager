package appintents

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const (
	extensionsDirectoryName  = "Extensions"
	widgetExtensionSuffix    = "Widget"
	extensionCoreSuffix      = "Core"
	appIntentsDependencyName = "AppIntents"
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// SetupInput holds parameters for app-intents setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
}

type templateData struct {
	AppTypeName string
	AppGroupID  string
}

// Setup scaffolds App Intents files in the widget extension.
func Setup(input SetupInput) error {
	if err := validateInput(input); err != nil {
		return err
	}

	appName := strings.TrimSpace(input.AppName)
	appTypeName := scaffold.SwiftTypeName(appName)
	widgetTargetName := widgetExtensionTargetName(appName)

	cfg, err := loadProjectConfig(input.ProjectRoot)
	if err != nil {
		return err
	}

	appGroupID, err := resolveAppGroupID(cfg)
	if err != nil {
		return err
	}

	widgetCoreSourcesDir := filepath.Join(
		input.ProjectRoot,
		extensionsDirectoryName,
		widgetTargetName,
		widgetTargetName+extensionCoreSuffix,
		"Sources",
	)
	if err := os.MkdirAll(widgetCoreSourcesDir, 0o755); err != nil {
		return fmt.Errorf("create widget Core sources directory: %w", err)
	}

	data := templateData{
		AppTypeName: appTypeName,
		AppGroupID:  appGroupID,
	}

	intentPath := filepath.Join(widgetCoreSourcesDir, appTypeName+"WidgetToggleIntent.swift")
	if err := renderTemplate("widget_toggle_intent.swift.tmpl", intentPath, data); err != nil {
		return fmt.Errorf("render widget toggle intent: %w", err)
	}

	widgetProjectPath := filepath.Join(input.ProjectRoot, extensionsDirectoryName, widgetTargetName, "Project.swift")
	if err := addAppIntentsDependency(widgetProjectPath); err != nil {
		return fmt.Errorf("add AppIntents dependency: %w", err)
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

func widgetExtensionTargetName(appName string) string {
	return scaffold.SwiftTypeName(strings.TrimSpace(appName) + widgetExtensionSuffix)
}

func loadProjectConfig(projectRoot string) (config.ProjectConfig, error) {
	cfgPath := filepath.Join(projectRoot, config.DefaultConfigPath)
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return config.ProjectConfig{}, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

func resolveAppGroupID(cfg config.ProjectConfig) (string, error) {
	for _, appGroup := range cfg.AppGroups {
		trimmed := strings.TrimSpace(appGroup)
		if trimmed != "" {
			return trimmed, nil
		}
	}

	bundleID := strings.TrimSpace(cfg.BundleID)
	if bundleID == "" {
		return "", fmt.Errorf("bundle ID is required")
	}
	return "group." + bundleID, nil
}

func addAppIntentsDependency(projectSwiftPath string) error {
	err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    appIntentsDependencyName,
		Content: `.sdk(name: "AppIntents", type: .framework)`,
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
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
