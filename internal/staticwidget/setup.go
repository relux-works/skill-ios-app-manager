package staticwidget

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

const (
	extensionsDirectoryName = "Extensions"
	widgetExtensionSuffix   = "Widget"
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// SetupInput holds parameters for static-widget setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
}

type templateData struct {
	AppName     string
	AppTypeName string
}

type widgetTemplateFile struct {
	templateName string
	fileSuffix   string
}

var widgetTemplateFiles = []widgetTemplateFile{
	{templateName: "static_widget.swift.tmpl", fileSuffix: "Widget.swift"},
	{templateName: "timeline_provider.swift.tmpl", fileSuffix: "TimelineProvider.swift"},
	{templateName: "timeline_entry.swift.tmpl", fileSuffix: "TimelineEntry.swift"},
	{templateName: "widget_view.swift.tmpl", fileSuffix: "WidgetView.swift"},
}

// Setup creates static WidgetKit files in the widget extension target and registers it in WidgetBundle.
func Setup(input SetupInput) error {
	if err := validateInput(input); err != nil {
		return err
	}

	appName := strings.TrimSpace(input.AppName)
	appTypeName := scaffold.SwiftTypeName(appName)
	widgetTargetName := widgetExtensionTargetName(appName)

	widgetSourcesDir := filepath.Join(input.ProjectRoot, extensionsDirectoryName, widgetTargetName, "Sources")
	if err := os.MkdirAll(widgetSourcesDir, 0o755); err != nil {
		return fmt.Errorf("create widget sources directory: %w", err)
	}

	data := templateData{
		AppName:     appName,
		AppTypeName: appTypeName,
	}

	for _, tf := range widgetTemplateFiles {
		outputPath := filepath.Join(widgetSourcesDir, appTypeName+tf.fileSuffix)
		if err := renderTemplate(tf.templateName, outputPath, data); err != nil {
			return fmt.Errorf("render %s: %w", outputPath, err)
		}
	}

	widgetBundlePath := filepath.Join(widgetSourcesDir, widgetTargetName+"Bundle.swift")
	if err := appendWidgetBundleEntry(widgetBundlePath, appTypeName+"Widget"); err != nil {
		return fmt.Errorf("register static widget in WidgetBundle: %w", err)
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
