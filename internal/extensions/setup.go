package extensions

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
	sharedKitModuleName     = "SharedKit"
	sharedKitModuleType     = "utility"
	defaultPlatform         = "iOS(.v17)"
	defaultModulesRelPath   = "Packages"
	extensionsDirectoryName = "Extensions"
	defaultHostBundleID     = "com.example.app"
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// SetupInput holds parameters for app-extensions setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
	Platform    string
}

// ExtensionProjectInput defines one extension target scaffold operation.
type ExtensionProjectInput struct {
	ProjectRoot              string
	ExtensionName            string
	BundleIDSuffix           string
	ExtensionPointIdentifier string
	HostBundleID             string
}

type extensionProjectTemplateData struct {
	ProjectName              string
	TargetName               string
	HostBundleID             string
	BundleIDSuffix           string
	ExtensionPointIdentifier string
}

// Setup creates shared extension infrastructure and patches host manifests.
func Setup(input SetupInput) error {
	if err := validateInput(input); err != nil {
		return err
	}

	platform := strings.TrimSpace(input.Platform)
	if platform == "" {
		platform = defaultPlatform
	}

	modulesRoot := ioc.ResolveModulesPath(input.ProjectRoot, input.ModulesPath)
	if err := scaffoldSharedKit(modulesRoot, platform); err != nil {
		return fmt.Errorf("scaffold %s: %w", sharedKitModuleName, err)
	}

	extensionsRoot := filepath.Join(input.ProjectRoot, extensionsDirectoryName)
	if err := os.MkdirAll(extensionsRoot, 0o755); err != nil {
		return fmt.Errorf("create %s directory: %w", extensionsDirectoryName, err)
	}

	projectSwiftPath := filepath.Join(input.ProjectRoot, "Project.swift")
	if err := addSharedKitToProjectSwift(projectSwiftPath); err != nil {
		return fmt.Errorf("add %s to Project.swift: %w", sharedKitModuleName, err)
	}

	modulesRelPath := normalizeModulesRelPath(input.ModulesPath)
	rootPackageSwiftPath := filepath.Join(input.ProjectRoot, "Package.swift")
	if err := addSharedKitToRootPackageSwift(rootPackageSwiftPath, modulesRelPath); err != nil {
		return fmt.Errorf("add %s to Package.swift: %w", sharedKitModuleName, err)
	}

	workspaceSwiftPath := filepath.Join(input.ProjectRoot, "Workspace.swift")
	if err := addSharedKitToWorkspaceSwift(workspaceSwiftPath, modulesRelPath); err != nil {
		return fmt.Errorf("add %s to Workspace.swift: %w", sharedKitModuleName, err)
	}

	return nil
}

// MakeAppExtensionProject creates an extension target scaffold under Extensions/.
func MakeAppExtensionProject(input ExtensionProjectInput) error {
	return makeAppExtensionProject(input)
}

func makeAppExtensionProject(input ExtensionProjectInput) error {
	if err := validateExtensionProjectInput(input); err != nil {
		return err
	}

	targetName := scaffold.SwiftTypeName(input.ExtensionName)
	hostBundleID := strings.TrimSuffix(strings.TrimSpace(input.HostBundleID), ".")
	if hostBundleID == "" {
		hostBundleID = defaultHostBundleID
	}

	bundleIDSuffix := strings.Trim(strings.TrimSpace(input.BundleIDSuffix), ".")
	projectDir := filepath.Join(input.ProjectRoot, extensionsDirectoryName, targetName)
	sourcesDir := filepath.Join(projectDir, "Sources")

	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		return fmt.Errorf("create extension sources directory: %w", err)
	}

	projectData := extensionProjectTemplateData{
		ProjectName:              targetName,
		TargetName:               targetName,
		HostBundleID:             hostBundleID,
		BundleIDSuffix:           bundleIDSuffix,
		ExtensionPointIdentifier: strings.TrimSpace(input.ExtensionPointIdentifier),
	}

	projectSwiftPath := filepath.Join(projectDir, "Project.swift")
	if err := renderTemplate("extension_project.swift.tmpl", projectSwiftPath, projectData); err != nil {
		return fmt.Errorf("render extension Project.swift: %w", err)
	}

	sourceFilePath := filepath.Join(sourcesDir, targetName+".swift")
	sourceFileContent := fmt.Sprintf(
		"import Foundation\n\npublic enum %sEntryPoint {}\n",
		targetName,
	)
	if err := os.WriteFile(sourceFilePath, []byte(sourceFileContent), 0o644); err != nil {
		return fmt.Errorf("write extension source file: %w", err)
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

func validateExtensionProjectInput(input ExtensionProjectInput) error {
	if strings.TrimSpace(input.ProjectRoot) == "" {
		return fmt.Errorf("project root is required")
	}
	if strings.TrimSpace(input.ExtensionName) == "" {
		return fmt.Errorf("extension name is required")
	}
	if strings.TrimSpace(input.BundleIDSuffix) == "" {
		return fmt.Errorf("bundle ID suffix is required")
	}
	if strings.TrimSpace(input.ExtensionPointIdentifier) == "" {
		return fmt.Errorf("extension point identifier is required")
	}
	return nil
}

func normalizeModulesRelPath(raw string) string {
	modulesRelPath := strings.TrimSpace(raw)
	if modulesRelPath == "" {
		return defaultModulesRelPath
	}
	return modulesRelPath
}

func scaffoldSharedKit(modulesRoot, platform string) error {
	pkgDir := filepath.Join(modulesRoot, sharedKitModuleName)
	if err := createPackageDir(pkgDir, sharedKitModuleName, platform); err != nil {
		return err
	}

	if err := ioc.WriteModuleType(pkgDir, sharedKitModuleType); err != nil {
		return fmt.Errorf("write %s: %w", ioc.ModuleTypeFile, err)
	}

	sourcesDir := filepath.Join(pkgDir, "Sources")
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		return fmt.Errorf("create Sources directory: %w", err)
	}

	sharedKitFilePath := filepath.Join(sourcesDir, "SharedKit.swift")
	if err := renderTemplate("shared_kit.swift.tmpl", sharedKitFilePath, nil); err != nil {
		return fmt.Errorf("render SharedKit.swift: %w", err)
	}

	return nil
}

func createPackageDir(pkgDir, moduleName, platform string) error {
	if _, err := os.Stat(pkgDir); err == nil {
		return nil
	}

	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", pkgDir, err)
	}

	manifest, err := tuistproj.GeneratePackageSwift(tuistproj.PackageGenerationInput{
		ModuleName: moduleName,
		Type:       tuistproj.PackageTypeInterface,
		Platform:   platform,
	})
	if err != nil {
		return fmt.Errorf("generate Package.swift: %w", err)
	}

	manifestPath := filepath.Join(pkgDir, "Package.swift")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return fmt.Errorf("write Package.swift: %w", err)
	}

	sourcesDir := filepath.Join(pkgDir, "Sources")
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		return fmt.Errorf("mkdir Sources: %w", err)
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

func addSharedKitToProjectSwift(projectSwiftPath string) error {
	err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    sharedKitModuleName,
		Content: fmt.Sprintf(`.external(name: "%s")`, sharedKitModuleName),
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func addSharedKitToRootPackageSwift(rootPackageSwiftPath, modulesRelPath string) error {
	refPath := filepath.ToSlash(filepath.Join(modulesRelPath, sharedKitModuleName))
	err := tuistproj.ApplyManifestEditsToFile(rootPackageSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    sharedKitModuleName,
		Content: fmt.Sprintf(`.package(path: "%s")`, refPath),
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func addSharedKitToWorkspaceSwift(workspaceSwiftPath, modulesRelPath string) error {
	if _, err := os.Stat(workspaceSwiftPath); os.IsNotExist(err) {
		return nil
	}

	refPath := filepath.ToSlash(filepath.Join(modulesRelPath, sharedKitModuleName))
	err := tuistproj.ApplyManifestEditsToFile(workspaceSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    sharedKitModuleName,
		Content: fmt.Sprintf(`.package(path: "%s")`, refPath),
	})
	if err != nil && isIgnorableManifestError(err) {
		return nil
	}

	return err
}

func isIgnorableManifestError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "already contains") ||
		strings.Contains(msg, "not found")
}
