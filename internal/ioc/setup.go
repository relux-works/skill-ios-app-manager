package ioc

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/relux-works/ios-app-manager/internal/deps"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const (
	swiftIoCURL     = "https://github.com/relux-works/swift-ioc.git"
	swiftIoCVersion = "from: 1.0.1"
	swiftIoCPackage = "SwiftIoC"
	implSuffix      = "Impl"

	// ModuleTypeFile is the marker file written in each module's interface package root.
	ModuleTypeFile = ".module-type"

	// BuilderConfigFile is an optional marker file that specifies custom builder args
	// for the IoC registry builder function. Contents are injected verbatim into the
	// Impl() constructor call in Registry.swift.
	BuilderConfigFile = ".builder-config"
)

// ModuleCategory represents a semantic grouping for registry sections.
type ModuleCategory string

const (
	CategoryFoundation ModuleCategory = "foundation"
	CategoryFeature    ModuleCategory = "feature"
	CategoryNetwork    ModuleCategory = "network"
	CategoryUtils      ModuleCategory = "utils"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// SetupInput holds parameters for the IoC setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
}

// DiscoveredModule represents a module with Interface/Impl split.
type DiscoveredModule struct {
	Name             string
	InterfacePackage string
	ImplPackage      string
	IsAsync          bool           // true for relux modules (Impl has async init)
	Category         ModuleCategory // semantic category read from .module-type
	BuilderArgs      string         // optional args injected into Impl() call in Registry
}

// GroupedModules holds modules grouped by semantic category.
type GroupedModules struct {
	Foundation []DiscoveredModule
	Features   []DiscoveredModule
	Network    []DiscoveredModule
	Utils      []DiscoveredModule
}

// RegistryTemplateData holds parameters for the Registry.swift template.
type RegistryTemplateData struct {
	AppTypeName string
	Imports     []string
	Modules     []DiscoveredModule
	HasRelux    bool
	Groups      GroupedModules
}

// Setup integrates SwiftIoC into a Tuist project.
func Setup(input SetupInput) error {
	if err := validateInput(input); err != nil {
		return err
	}

	modulesRoot := ResolveModulesPath(input.ProjectRoot, input.ModulesPath)

	if err := addSwiftIoCToPackageSwift(modulesRoot); err != nil {
		return fmt.Errorf("add SwiftIoC to Package.swift: %w", err)
	}

	projectSwiftPath := filepath.Join(input.ProjectRoot, "Project.swift")
	if err := addSwiftIoCToProjectSwift(projectSwiftPath); err != nil {
		return fmt.Errorf("add SwiftIoC to Project.swift: %w", err)
	}

	modules, err := DiscoverModules(modulesRoot)
	if err != nil {
		return fmt.Errorf("discover modules: %w", err)
	}

	appTypeName := scaffold.SwiftTypeName(input.AppName)

	registryPath := filepath.Join(input.ProjectRoot, "Targets", input.AppName, "Sources", "App", appTypeName+".Registry.swift")
	hasRelux := detectReluxSetup(filepath.Join(input.ProjectRoot, "Targets", input.AppName, "Sources", "App.swift"))
	if err := scaffoldRegistryFull(registryPath, appTypeName, modules, hasRelux); err != nil {
		return fmt.Errorf("scaffold Registry.swift: %w", err)
	}

	appSwiftPath := filepath.Join(input.ProjectRoot, "Targets", input.AppName, "Sources", "App.swift")
	if err := updateAppSwift(appSwiftPath, modules); err != nil {
		return fmt.Errorf("update App.swift: %w", err)
	}

	return nil
}

// DiscoverModules scans the modules directory for Interface/Impl split modules.
func DiscoverModules(modulesRoot string) ([]DiscoveredModule, error) {
	entries, err := os.ReadDir(modulesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan modules directory %q: %w", modulesRoot, err)
	}

	dirs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		dirs[name] = struct{}{}
	}

	var modules []DiscoveredModule
	for name := range dirs {
		if strings.HasSuffix(name, implSuffix) {
			continue
		}
		implName := name + implSuffix
		if _, ok := dirs[implName]; !ok {
			continue
		}
		modules = append(modules, DiscoveredModule{
			Name:             name,
			InterfacePackage: name,
			ImplPackage:      implName,
			IsAsync:          hasReluxImport(filepath.Join(modulesRoot, name)),
			Category:         readModuleCategory(filepath.Join(modulesRoot, name)),
			BuilderArgs:      readBuilderConfig(filepath.Join(modulesRoot, name)),
		})
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})

	return modules, nil
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

// ResolveModulesPath resolves the modules root path (default: Packages).
func ResolveModulesPath(projectRoot, modulesPath string) string {
	mp := strings.TrimSpace(modulesPath)
	if mp == "" {
		mp = "Packages"
	}
	if filepath.IsAbs(mp) {
		return filepath.Clean(mp)
	}
	return filepath.Clean(filepath.Join(projectRoot, mp))
}

func addSwiftIoCToPackageSwift(modulesRoot string) error {
	err := deps.AddExternalDep(swiftIoCURL, swiftIoCVersion, swiftIoCPackage, "", modulesRoot)
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func addSwiftIoCToProjectSwift(projectSwiftPath string) error {
	err := tuistproj.ApplyManifestEditsToFile(projectSwiftPath, tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    swiftIoCPackage,
		Content: `.external(name: "SwiftIoC")`,
	})
	if err != nil && strings.Contains(err.Error(), "already contains") {
		return nil
	}
	return err
}

func scaffoldRegistry(registryPath, appTypeName string, modules []DiscoveredModule) error {
	return scaffoldRegistryFull(registryPath, appTypeName, modules, false)
}

func scaffoldRegistryFull(registryPath, appTypeName string, modules []DiscoveredModule, hasRelux bool) error {
	content, err := RenderRegistryWithData(RegistryTemplateData{
		AppTypeName: appTypeName,
		Imports:     BuildModuleImports(modules),
		Modules:     modules,
		HasRelux:    hasRelux,
	})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(registryPath), 0o755); err != nil {
		return fmt.Errorf("create directory for Registry.swift: %w", err)
	}

	return os.WriteFile(registryPath, []byte(content), 0o644)
}

// detectReluxSetup checks App.swift for SwiftUIRelux import (set by relux setup).
func detectReluxSetup(appSwiftPath string) bool {
	data, err := os.ReadFile(appSwiftPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "import SwiftUIRelux")
}

// ScaffoldRegistryWithData writes Registry.swift using full template data.
func ScaffoldRegistryWithData(registryPath string, data RegistryTemplateData) error {
	content, err := RenderRegistryWithData(data)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(registryPath), 0o755); err != nil {
		return fmt.Errorf("create directory for Registry.swift: %w", err)
	}

	return os.WriteFile(registryPath, []byte(content), 0o644)
}

// RenderRegistry renders the Registry.swift content from the template.
func RenderRegistry(appTypeName string, modules []DiscoveredModule) (string, error) {
	return RenderRegistryWithData(RegistryTemplateData{
		AppTypeName: appTypeName,
		Imports:     BuildModuleImports(modules),
		Modules:     modules,
	})
}

// RenderRegistryWithData renders the Registry.swift content from the template using full data.
func RenderRegistryWithData(data RegistryTemplateData) (string, error) {
	// Always compute groups from modules.
	data.Groups = GroupModulesByCategory(data.Modules)

	tmplContent, err := templatesFS.ReadFile("templates/registry.swift.tmpl")
	if err != nil {
		return "", fmt.Errorf("read registry template: %w", err)
	}

	tmpl, err := template.New("registry.swift.tmpl").Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("parse registry template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render registry template: %w", err)
	}

	return buf.String(), nil
}

// BuildModuleImports builds sorted import list from discovered modules.
func BuildModuleImports(modules []DiscoveredModule) []string {
	seen := make(map[string]struct{})
	var imports []string
	for _, m := range modules {
		if _, ok := seen[m.InterfacePackage]; !ok {
			seen[m.InterfacePackage] = struct{}{}
			imports = append(imports, m.InterfacePackage)
		}
		if _, ok := seen[m.ImplPackage]; !ok {
			seen[m.ImplPackage] = struct{}{}
			imports = append(imports, m.ImplPackage)
		}
	}
	sort.Strings(imports)
	return imports
}

var appStructPattern = regexp.MustCompile(`(?m)^([ \t]*)struct\s+\w+\s*:\s*App\s*\{`)

func updateAppSwift(appSwiftPath string, modules []DiscoveredModule) error {
	content, err := os.ReadFile(appSwiftPath)
	if err != nil {
		return fmt.Errorf("read App.swift: %w", err)
	}

	updated := EditAppSwift(string(content), modules)

	return os.WriteFile(appSwiftPath, []byte(updated), 0o644)
}

// EditAppSwift injects Registry.configure() init into App.swift content.
// Module imports are NOT added here — Registry.swift handles all module imports.
func EditAppSwift(content string, modules []DiscoveredModule) string {
	result := content

	// Only add SwiftIoC import — module imports live in Registry.swift.
	result = EnsureImport(result, "SwiftIoC")

	// Add init() { Registry.configure() } if not already present.
	if strings.Contains(result, "Registry.configure()") {
		return result
	}

	match := appStructPattern.FindStringIndex(result)
	if match == nil {
		return result
	}

	openBrace := match[1]

	// Detect indentation from the struct line.
	structLine := result[match[0]:match[1]]
	baseIndent := ""
	for _, ch := range structLine {
		if ch == ' ' || ch == '\t' {
			baseIndent += string(ch)
		} else {
			break
		}
	}
	memberIndent := baseIndent + "    "

	initBlock := "\n" + memberIndent + "init() {\n" + memberIndent + "    Registry.configure()\n" + memberIndent + "}\n"

	result = result[:openBrace] + initBlock + result[openBrace:]
	return result
}

// hasReluxImport checks if any Swift source in the package directory imports Relux.
func hasReluxImport(packageDir string) bool {
	sourcesDir := filepath.Join(packageDir, "Sources")
	found := false
	_ = filepath.WalkDir(sourcesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".swift") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(data), "import Relux") {
			found = true
		}
		return nil
	})
	return found
}

// MapModuleTypeToCategory maps a module type string to a registry category.
func MapModuleTypeToCategory(moduleType string) ModuleCategory {
	switch strings.ToLower(strings.TrimSpace(moduleType)) {
	case "kit", "shared":
		return CategoryFoundation
	case "utility":
		return CategoryUtils
	case "feature", "relux-feature", "ui":
		return CategoryFeature
	default:
		return CategoryFeature
	}
}

// GroupModulesByCategory splits modules into semantic groups.
func GroupModulesByCategory(modules []DiscoveredModule) GroupedModules {
	var g GroupedModules
	for _, m := range modules {
		switch m.Category {
		case CategoryFoundation:
			g.Foundation = append(g.Foundation, m)
		case CategoryNetwork:
			g.Network = append(g.Network, m)
		case CategoryUtils:
			g.Utils = append(g.Utils, m)
		default:
			g.Features = append(g.Features, m)
		}
	}
	return g
}

// readModuleCategory reads the .module-type marker file from a module directory.
func readModuleCategory(moduleDir string) ModuleCategory {
	data, err := os.ReadFile(filepath.Join(moduleDir, ModuleTypeFile))
	if err != nil {
		return CategoryFeature
	}
	return MapModuleTypeToCategory(string(data))
}

// WriteModuleType writes the .module-type marker file for a module.
func WriteModuleType(moduleDir, moduleType string) error {
	return os.WriteFile(
		filepath.Join(moduleDir, ModuleTypeFile),
		[]byte(moduleType+"\n"),
		0o644,
	)
}

// WriteBuilderConfig writes the .builder-config marker file for a module.
// The content is injected verbatim into Impl() in Registry.swift.
func WriteBuilderConfig(moduleDir, args string) error {
	return os.WriteFile(
		filepath.Join(moduleDir, BuilderConfigFile),
		[]byte(args+"\n"),
		0o644,
	)
}

// readBuilderConfig reads the .builder-config marker file from a module directory.
func readBuilderConfig(moduleDir string) string {
	data, err := os.ReadFile(filepath.Join(moduleDir, BuilderConfigFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// EnsureImport adds an import statement if not already present.
func EnsureImport(content, moduleName string) string {
	importLine := "import " + moduleName
	importPattern := regexp.MustCompile(`(?m)^import\s+` + regexp.QuoteMeta(moduleName) + `\s*$`)
	if importPattern.MatchString(content) {
		return content
	}

	// Find the last import line and add after it.
	lastImportPattern := regexp.MustCompile(`(?m)^import\s+\S+[^\n]*\n`)
	matches := lastImportPattern.FindAllStringIndex(content, -1)
	if len(matches) > 0 {
		lastMatch := matches[len(matches)-1]
		insertAt := lastMatch[1]
		return content[:insertAt] + importLine + "\n" + content[insertAt:]
	}

	// No imports found, add at the beginning.
	return importLine + "\n" + content
}
