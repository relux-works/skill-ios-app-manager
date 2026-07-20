package securestore

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/ioc"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	tmplutil "github.com/relux-works/ios-app-manager/internal/template"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const (
	moduleName      = "SecureStore"
	implPackageName = "SecureStoreImpl"
	defaultPlatform = "iOS(.v17)"
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// SetupInput holds parameters for the secure-store setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
	Platform    string // SwiftPM platform, e.g. "iOS(.v17)"
	AccessGroup string // App group for shared keychain access, e.g. "group.org.xflow.app"
}

// templateFile maps a template name to its output path relative to a sources dir.
type templateFile struct {
	templateName string
	outputDir    string // subdirectory under Sources/<PackageName>/
	outputFile   string
}

var interfaceFiles = []templateFile{
	{templateName: "namespace.swift.tmpl", outputDir: "", outputFile: "SecureStore.swift"},
	{templateName: "module.swift.tmpl", outputDir: "Module", outputFile: "SecureStore.Module.swift"},
	{templateName: "interface.swift.tmpl", outputDir: "Module", outputFile: "SecureStore.Module+Interface.swift"},
}

var implFiles = []templateFile{
	{templateName: "impl.swift.tmpl", outputDir: "Module", outputFile: "SecureStore.Module+Impl.swift"},
}

// Setup creates the SecureStore kit module with interface/impl split.
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
	cfg := config.ProjectConfig{}
	cfgPath := filepath.Join(input.ProjectRoot, config.DefaultConfigPath)
	if _, err := os.Stat(cfgPath); err == nil {
		cfg, err = config.LoadConfig(cfgPath)
		if err != nil {
			return fmt.Errorf("load project config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat project config: %w", err)
	}

	// 1. Create interface package: SecureStore (with Package.swift).
	interfacePkgDir := filepath.Join(modulesRoot, moduleName)
	if err := createPackageDir(interfacePkgDir, moduleName, tuistproj.PackageTypeInterface, platform, cfg); err != nil {
		return fmt.Errorf("create %s package: %w", moduleName, err)
	}

	// Write .module-type marker for IoC registry grouping.
	if err := ioc.WriteModuleType(interfacePkgDir, "kit"); err != nil {
		return fmt.Errorf("write %s: %w", ioc.ModuleTypeFile, err)
	}

	// Write .builder-config for IoC registry to pass serviceName and accessGroup.
	accessGroupKey := tmplutil.AppGroupSwiftIdentifier(cfg.BundleID, input.AccessGroup)
	builderArgs := fmt.Sprintf(
		"serviceName: Configuration.Keychain.serviceName, accessGroup: Configuration.AppGroups.%s",
		accessGroupKey,
	)
	if err := ioc.WriteBuilderConfig(interfacePkgDir, builderArgs); err != nil {
		return fmt.Errorf("write %s: %w", ioc.BuilderConfigFile, err)
	}

	// 2. Create impl package: SecureStoreImpl (with Package.swift).
	implPkgDir := filepath.Join(modulesRoot, implPackageName)
	if err := createPackageDir(implPkgDir, moduleName, tuistproj.PackageTypeImpl, platform, cfg); err != nil {
		return fmt.Errorf("create %s package: %w", implPackageName, err)
	}

	// 3. Scaffold interface Swift files.
	interfaceSourcesDir := filepath.Join(interfacePkgDir, "Sources")
	if err := scaffoldFiles(interfaceSourcesDir, interfaceFiles); err != nil {
		return fmt.Errorf("scaffold SecureStore interface: %w", err)
	}

	// 4. Scaffold impl Swift files.
	implSourcesDir := filepath.Join(implPkgDir, "Sources")
	if err := scaffoldFiles(implSourcesDir, implFiles); err != nil {
		return fmt.Errorf("scaffold SecureStoreImpl: %w", err)
	}

	// 5. Add module references to Project.swift and root Package.swift.
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

	// 6. Patch only the SecureStore slice when an existing Registry is present.
	registryPath := filepath.Join(
		input.ProjectRoot, "Targets", input.AppName, "Sources", "App",
		appTypeName+".Registry.swift",
	)
	if _, err := os.Stat(registryPath); err == nil {
		if err := patchRegistry(registryPath, appTypeName, builderArgs); err != nil {
			return fmt.Errorf("patch Registry.swift: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat Registry.swift: %w", err)
	}

	return nil
}

func patchRegistry(registryPath string, appTypeName string, builderArgs string) error {
	builderFunction := fmt.Sprintf(`private static func buildSecureStore() -> SecureStore.Module.Interface {
    SecureStore.Module.Impl(%s)
}`, builderArgs)
	return ioc.PatchFoundationRegistry(registryPath, appTypeName, ioc.RegistryFoundationPatch{
		Imports:            []string{moduleName, implPackageName},
		RegistrationMarker: "SecureStore.Module.Interface.self",
		RegistrationLine:   "ioc.register(SecureStore.Module.Interface.self, lifecycle: .container, resolver: Self.buildSecureStore)",
		BuilderMarker:      "func buildSecureStore()",
		BuilderFunction:    builderFunction,
	})
}

func validateInput(input SetupInput) error {
	if strings.TrimSpace(input.ProjectRoot) == "" {
		return fmt.Errorf("project root is required")
	}
	if strings.TrimSpace(input.AppName) == "" {
		return fmt.Errorf("app name is required")
	}
	if strings.TrimSpace(input.AccessGroup) == "" {
		return fmt.Errorf("access group is required")
	}
	return nil
}

func createPackageDir(pkgDir, modName string, pkgType tuistproj.PackageType, platform string, cfg config.ProjectConfig) error {
	if _, err := os.Stat(pkgDir); err == nil {
		// Package already exists — idempotent.
		return nil
	}

	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", pkgDir, err)
	}

	manifest, err := tuistproj.GeneratePackageSwift(tuistproj.PackageGenerationInput{
		ModuleName: modName,
		Type:       pkgType,
		Platform:   platform,
		Config:     cfg,
	})
	if err != nil {
		return fmt.Errorf("generate Package.swift: %w", err)
	}

	manifestPath := filepath.Join(pkgDir, "Package.swift")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return fmt.Errorf("write Package.swift: %w", err)
	}

	// Determine target name for Sources directory.
	targetName := modName
	if pkgType == tuistproj.PackageTypeImpl {
		targetName = modName + "Impl"
	}

	srcDir := filepath.Join(pkgDir, "Sources", targetName)
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return fmt.Errorf("mkdir Sources: %w", err)
	}

	return nil
}

func scaffoldFiles(sourcesDir string, files []templateFile) error {
	for _, tf := range files {
		outputDir := sourcesDir
		if tf.outputDir != "" {
			outputDir = filepath.Join(sourcesDir, tf.outputDir)
		}

		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("create directory %q: %w", outputDir, err)
		}

		outputPath := filepath.Join(outputDir, tf.outputFile)
		if err := renderTemplate(tf.templateName, outputPath); err != nil {
			return fmt.Errorf("render %s: %w", tf.outputFile, err)
		}
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
	for _, name := range []string{moduleName, implPackageName} {
		err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
			Type:    tuistproj.AddDependency,
			Name:    name,
			Content: fmt.Sprintf(`.external(name: "%s")`, name),
		})
		if err != nil && strings.Contains(err.Error(), "already contains") {
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func addModuleToRootPackageSwift(rootPackageSwiftPath, modulesRelPath string) error {
	for _, name := range []string{moduleName, implPackageName} {
		refPath := filepath.ToSlash(filepath.Join(modulesRelPath, name))
		err := tuistproj.ApplyManifestEditsToFile(rootPackageSwiftPath, tuistproj.ManifestEdit{
			Type:    tuistproj.AddDependency,
			Name:    name,
			Content: fmt.Sprintf(`.package(path: "%s")`, refPath),
		})
		if err != nil && strings.Contains(err.Error(), "already contains") {
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func addModuleToWorkspaceSwift(workspaceSwiftPath, modulesRelPath string) error {
	if _, err := os.Stat(workspaceSwiftPath); os.IsNotExist(err) {
		return nil
	}
	for _, name := range []string{moduleName, implPackageName} {
		refPath := filepath.ToSlash(filepath.Join(modulesRelPath, name))
		err := tuistproj.ApplyManifestEditsToFile(workspaceSwiftPath, tuistproj.ManifestEdit{
			Type:    tuistproj.AddDependency,
			Name:    name,
			Content: fmt.Sprintf(`.package(path: "%s")`, refPath),
		})
		if err != nil && isIgnorableManifestError(err) {
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func isIgnorableManifestError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "already contains") ||
		strings.Contains(msg, "not found")
}
