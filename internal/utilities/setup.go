package utilities

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
	moduleName      = "Utilities"
	defaultPlatform = "iOS(.v17)"
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// SetupInput holds parameters for the utilities setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
	Platform    string // SwiftPM platform, e.g. "iOS(.v17)"
}

// templateFile maps a template name to its output path relative to the module sources dir.
type templateFile struct {
	templateName string
	outputFile   string
}

var httpClientUtilsFiles = []templateFile{
	{templateName: "header_maps.swift.tmpl", outputFile: "HeaderMaps.swift"},
	{templateName: "base_encoder.swift.tmpl", outputFile: "BaseEncoder.swift"},
	{templateName: "base_decoder.swift.tmpl", outputFile: "BaseDecoder.swift"},
}

// Setup creates the Utilities module with HttpClientUtils files.
func Setup(input SetupInput) error {
	if err := validateInput(input); err != nil {
		return err
	}

	platform := strings.TrimSpace(input.Platform)
	if platform == "" {
		platform = defaultPlatform
	}

	modulesRoot := ioc.ResolveModulesPath(input.ProjectRoot, input.ModulesPath)
	appTypeName := scaffold.SwiftTypeName(input.AppName)

	// 1. Create package directory with Package.swift (utility = single package, interface type).
	pkgDir := filepath.Join(modulesRoot, moduleName)
	if err := createPackageDir(pkgDir, moduleName, platform); err != nil {
		return fmt.Errorf("create %s package: %w", moduleName, err)
	}

	// Write .module-type marker for IoC registry grouping.
	if err := ioc.WriteModuleType(pkgDir, "utility"); err != nil {
		return fmt.Errorf("write %s: %w", ioc.ModuleTypeFile, err)
	}

	// 2. Scaffold HttpClientUtils Swift files.
	sourcesDir := filepath.Join(pkgDir, "Sources", "HttpClientUtils")
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		return fmt.Errorf("create HttpClientUtils directory: %w", err)
	}

	for _, tf := range httpClientUtilsFiles {
		outputPath := filepath.Join(sourcesDir, tf.outputFile)
		if err := renderTemplate(tf.templateName, outputPath); err != nil {
			return fmt.Errorf("render %s: %w", tf.outputFile, err)
		}
	}

	// 3. Add module references to Project.swift and root Package.swift.
	projectSwiftPath := filepath.Join(input.ProjectRoot, "Project.swift")
	if err := addModuleToProjectSwift(projectSwiftPath); err != nil {
		return fmt.Errorf("add to Project.swift: %w", err)
	}

	modulesRelPath := strings.TrimSpace(input.ModulesPath)
	if modulesRelPath == "" {
		modulesRelPath = "Packages"
	}
	rootPackageSwiftPath := filepath.Join(input.ProjectRoot, "Package.swift")
	if err := addModuleToRootPackageSwift(rootPackageSwiftPath, modulesRelPath); err != nil {
		return fmt.Errorf("add to root Package.swift: %w", err)
	}

	workspaceSwiftPath := filepath.Join(input.ProjectRoot, "Workspace.swift")
	if err := addModuleToWorkspaceSwift(workspaceSwiftPath, modulesRelPath); err != nil {
		return fmt.Errorf("add to Workspace.swift: %w", err)
	}

	// 4. Re-scaffold Registry.swift if IoC is set up.
	registryPath := filepath.Join(
		input.ProjectRoot, "Targets", input.AppName, "Sources", "App",
		appTypeName+".Registry.swift",
	)
	if _, err := os.Stat(registryPath); err == nil {
		modules, err := ioc.DiscoverModules(modulesRoot)
		if err != nil {
			return fmt.Errorf("discover modules: %w", err)
		}

		hasRelux := hasReluxInRegistry(registryPath)

		if err := ioc.ScaffoldRegistryWithData(registryPath, ioc.RegistryTemplateData{
			AppTypeName: appTypeName,
			Imports:     ioc.BuildModuleImports(modules),
			Modules:     modules,
			HasRelux:    hasRelux,
		}); err != nil {
			return fmt.Errorf("regenerate Registry.swift: %w", err)
		}
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

func createPackageDir(pkgDir, modName, platform string) error {
	if _, err := os.Stat(pkgDir); err == nil {
		// Package already exists — idempotent.
		return nil
	}

	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", pkgDir, err)
	}

	manifest, err := tuistproj.GeneratePackageSwift(tuistproj.PackageGenerationInput{
		ModuleName: modName,
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

	srcDir := filepath.Join(pkgDir, "Sources", modName)
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return fmt.Errorf("mkdir Sources: %w", err)
	}

	return nil
}

func renderTemplate(templateName, outputPath string) error {
	tmplPath := "setup_templates/" + templateName
	tmplContent, err := setupTemplatesFS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("read template %q: %w", tmplPath, err)
	}

	tmpl, err := template.New(templateName).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("parse template %q: %w", templateName, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, nil); err != nil {
		return fmt.Errorf("render template %q: %w", templateName, err)
	}

	return os.WriteFile(outputPath, []byte(buf.String()), 0o644)
}

func addModuleToProjectSwift(projectSwiftPath string) error {
	name := moduleName
	err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    name,
		Content: fmt.Sprintf(`.external(name: "%s")`, name),
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func addModuleToRootPackageSwift(rootPackageSwiftPath, modulesRelPath string) error {
	name := moduleName
	refPath := filepath.ToSlash(filepath.Join(modulesRelPath, name))
	err := tuistproj.ApplyManifestEditsToFile(rootPackageSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    name,
		Content: fmt.Sprintf(`.package(path: "%s")`, refPath),
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func addModuleToWorkspaceSwift(workspaceSwiftPath, modulesRelPath string) error {
	if _, err := os.Stat(workspaceSwiftPath); os.IsNotExist(err) {
		return nil
	}
	name := moduleName
	refPath := filepath.ToSlash(filepath.Join(modulesRelPath, name))
	err := tuistproj.ApplyManifestEditsToFile(workspaceSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    name,
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

// hasReluxInRegistry checks if the existing Registry.swift contains Relux imports.
func hasReluxInRegistry(registryPath string) bool {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "import Relux") ||
		strings.Contains(string(data), "@_exported import Relux")
}
