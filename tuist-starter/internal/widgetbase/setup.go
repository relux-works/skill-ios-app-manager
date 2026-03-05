package widgetbase

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/extensions"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const (
	widgetExtensionPointIdentifier = "com.apple.widgetkit-extension"
	widgetExtensionBundleSuffix    = "widget"
	widgetExtensionSuffix          = "Widget"
	extensionsDirectoryName        = "Extensions"
	widgetKitDependencyName        = "WidgetKit"
	appGroupsEntitlementKey        = "com.apple.security.application-groups"
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// SetupInput holds parameters for widget-base setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
}

type widgetBundleTemplateData struct {
	AppTypeName string
}

// Setup creates the base WidgetKit extension scaffold and wires it into the host app.
func Setup(input SetupInput) error {
	if err := validateInput(input); err != nil {
		return err
	}

	cfg, err := loadProjectConfig(input.ProjectRoot)
	if err != nil {
		return err
	}

	widgetTargetName := widgetExtensionTargetName(input.AppName)
	appTypeName := scaffold.SwiftTypeName(input.AppName)

	appGroupID, err := resolveAppGroupID(cfg)
	if err != nil {
		return err
	}

	if err := extensions.MakeAppExtensionProject(extensions.ExtensionProjectInput{
		ProjectRoot:              input.ProjectRoot,
		ExtensionName:            widgetTargetName,
		BundleIDSuffix:           widgetExtensionBundleSuffix,
		ExtensionPointIdentifier: widgetExtensionPointIdentifier,
		HostBundleID:             strings.TrimSpace(cfg.BundleID),
	}); err != nil {
		return fmt.Errorf("create widget extension project: %w", err)
	}

	widgetProjectRoot := filepath.Join(input.ProjectRoot, extensionsDirectoryName, widgetTargetName)
	widgetBundlePath := filepath.Join(
		widgetProjectRoot,
		"Sources",
		appTypeName+widgetExtensionSuffix+"Bundle.swift",
	)
	if err := renderTemplate("widget_bundle.swift.tmpl", widgetBundlePath, widgetBundleTemplateData{
		AppTypeName: appTypeName,
	}); err != nil {
		return fmt.Errorf("render widget bundle: %w", err)
	}

	widgetProjectSwiftPath := filepath.Join(widgetProjectRoot, "Project.swift")
	if err := addWidgetKitDependency(widgetProjectSwiftPath); err != nil {
		return fmt.Errorf("add WidgetKit dependency: %w", err)
	}
	if err := addExtensionAppGroupEntitlement(widgetProjectSwiftPath, appGroupID); err != nil {
		return fmt.Errorf("add extension App Groups entitlement: %w", err)
	}

	if err := ensureHostAppGroupCapability(input.ProjectRoot, appGroupID); err != nil {
		return fmt.Errorf("ensure host app App Groups capability: %w", err)
	}

	projectSwiftPath := filepath.Join(input.ProjectRoot, "Project.swift")
	if err := addWidgetExtensionToHostProject(projectSwiftPath, widgetTargetName); err != nil {
		return fmt.Errorf("embed widget extension in host app: %w", err)
	}

	workspaceSwiftPath := filepath.Join(input.ProjectRoot, "Workspace.swift")
	if err := addWidgetExtensionToWorkspace(workspaceSwiftPath, widgetTargetName); err != nil {
		return fmt.Errorf("add widget extension to workspace: %w", err)
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

func addWidgetKitDependency(projectSwiftPath string) error {
	err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    widgetKitDependencyName,
		Content: `.sdk(name: "WidgetKit", type: .framework)`,
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func addExtensionAppGroupEntitlement(projectSwiftPath, appGroupID string) error {
	if strings.TrimSpace(appGroupID) == "" {
		return fmt.Errorf("app group ID is required")
	}

	payload, err := os.ReadFile(projectSwiftPath)
	if err != nil {
		return fmt.Errorf("read Project.swift: %w", err)
	}
	content := string(payload)

	if strings.Contains(content, appGroupsEntitlementKey) {
		return nil
	}

	marker := `            sources: ["Sources/**"],`
	if !strings.Contains(content, marker) {
		return fmt.Errorf("Project.swift missing %q marker", marker)
	}

	entitlementsBlock := fmt.Sprintf(
		`
            entitlements: .dictionary([
                "%s": .array([
                    .string("%s"),
                ]),
            ]),`,
		appGroupsEntitlementKey,
		appGroupID,
	)

	updated := strings.Replace(content, marker, marker+entitlementsBlock, 1)
	if err := os.WriteFile(projectSwiftPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write Project.swift: %w", err)
	}
	return nil
}

func ensureHostAppGroupCapability(projectRoot, appGroupID string) error {
	return scaffold.AddToAppCapabilities(projectRoot, "appGroups", map[string]string{
		"group": appGroupID,
	})
}

func addWidgetExtensionToHostProject(projectSwiftPath, extensionTargetName string) error {
	refPath := filepath.ToSlash(filepath.Join(extensionsDirectoryName, extensionTargetName))

	err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
		Type: tuistproj.AddDependency,
		Name: extensionTargetName,
		Content: fmt.Sprintf(
			`.project(target: "%s", path: "%s")`,
			extensionTargetName,
			refPath,
		),
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func addWidgetExtensionToWorkspace(workspaceSwiftPath, extensionTargetName string) error {
	if _, err := os.Stat(workspaceSwiftPath); os.IsNotExist(err) {
		return nil
	}

	payload, err := os.ReadFile(workspaceSwiftPath)
	if err != nil {
		return fmt.Errorf("read Workspace.swift: %w", err)
	}
	content := string(payload)

	refPath := filepath.ToSlash(filepath.Join(extensionsDirectoryName, extensionTargetName))
	refLiteral := fmt.Sprintf(`"%s"`, refPath)
	if strings.Contains(content, refLiteral) {
		return nil
	}

	marker := "projects: ["
	markerIdx := strings.Index(content, marker)
	if markerIdx < 0 {
		return fmt.Errorf("Workspace.swift missing %q marker", marker)
	}

	start := markerIdx + len(marker)
	after := content[start:]
	closeRel := strings.Index(after, "]")
	if closeRel < 0 {
		return fmt.Errorf("Workspace.swift missing closing ] for projects array")
	}
	end := start + closeRel

	projectsSection := content[start:end]
	updatedProjectsSection := appendWorkspaceProject(projectsSection, refPath)
	updatedContent := content[:start] + updatedProjectsSection + content[end:]

	if err := os.WriteFile(workspaceSwiftPath, []byte(updatedContent), 0o644); err != nil {
		return fmt.Errorf("write Workspace.swift: %w", err)
	}
	return nil
}

func appendWorkspaceProject(projectsSection, projectPath string) string {
	trimmed := strings.TrimRight(projectsSection, " \t\r\n")
	trailing := projectsSection[len(trimmed):]

	if strings.TrimSpace(trimmed) != "" && !strings.HasSuffix(trimmed, ",") {
		trimmed += ","
	}
	if trailing == "" {
		trailing = "\n    "
	}

	return trimmed + fmt.Sprintf("\n        %q,", projectPath) + trailing
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
