package tokenprovider

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/ioc"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const (
	moduleName       = "TokenProvider"
	implPackageName  = "TokenProviderImpl"
	defaultPlatform  = "iOS(.v17)"
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// SetupInput holds parameters for the token-provider setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
	Platform    string // SwiftPM platform, e.g. "iOS(.v17)"
}

// Setup scaffolds TokenProvider and TokenProviderImpl module packages and
// re-generates the IoC Registry to include them.
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

	// 1. Create interface package: TokenProvider
	interfacePkgDir := filepath.Join(modulesRoot, moduleName)
	if err := createPackageDir(interfacePkgDir, moduleName, tuistproj.PackageTypeInterface, platform); err != nil {
		return fmt.Errorf("create %s package: %w", moduleName, err)
	}

	// Write .module-type marker for IoC registry grouping.
	if err := ioc.WriteModuleType(interfacePkgDir, "kit"); err != nil {
		return fmt.Errorf("write %s: %w", ioc.ModuleTypeFile, err)
	}

	// 2. Create impl package: TokenProviderImpl
	implPkgDir := filepath.Join(modulesRoot, implPackageName)
	if err := createPackageDir(implPkgDir, moduleName, tuistproj.PackageTypeImpl, platform); err != nil {
		return fmt.Errorf("create %s package: %w", implPackageName, err)
	}

	// 3. Render TokenProvider Swift files into interface package.
	interfaceSrcDir := filepath.Join(interfacePkgDir, "Sources")

	protocolPath := filepath.Join(interfaceSrcDir, "Module", moduleName+".Module+Interface.swift")
	if err := renderTemplate("setup_templates/token_provider_protocol.swift.tmpl", protocolPath); err != nil {
		return fmt.Errorf("scaffold protocol: %w", err)
	}

	authDataPath := filepath.Join(interfaceSrcDir, moduleName+".AuthData.swift")
	if err := renderTemplate("setup_templates/token_provider_auth_data.swift.tmpl", authDataPath); err != nil {
		return fmt.Errorf("scaffold AuthData: %w", err)
	}

	namespacePath := filepath.Join(interfaceSrcDir, moduleName+".swift")
	if err := writeNamespace(namespacePath); err != nil {
		return fmt.Errorf("scaffold namespace: %w", err)
	}

	moduleDeclPath := filepath.Join(interfaceSrcDir, "Module", moduleName+".Module.swift")
	if err := writeModuleDecl(moduleDeclPath); err != nil {
		return fmt.Errorf("scaffold module decl: %w", err)
	}

	// 4. Render TokenProviderImpl Swift file into impl package.
	implSrcDir := filepath.Join(implPkgDir, "Sources")

	implPath := filepath.Join(implSrcDir, "Module", moduleName+".Module+Impl.swift")
	if err := renderTemplate("setup_templates/token_provider_impl.swift.tmpl", implPath); err != nil {
		return fmt.Errorf("scaffold impl: %w", err)
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

	// 6. Re-scaffold Registry.swift if IoC is set up.
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

func createPackageDir(pkgDir, modName string, pkgType tuistproj.PackageType, platform string) error {
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

func renderTemplate(templatePath, outputPath string) error {
	content, err := setupTemplatesFS.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template %q: %w", templatePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", outputPath, err)
	}

	return os.WriteFile(outputPath, content, 0o644)
}

func writeNamespace(path string) error {
	content := "public enum TokenProvider {}\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func writeModuleDecl(path string) error {
	content := `extension TokenProvider {
    public enum Module {}
}
`
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
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

// hasReluxInRegistry checks if the existing Registry.swift contains Relux imports.
func hasReluxInRegistry(registryPath string) bool {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "import Relux") ||
		strings.Contains(string(data), "@_exported import Relux")
}
