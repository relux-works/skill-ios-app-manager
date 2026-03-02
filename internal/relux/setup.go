package relux

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/relux-works/ios-app-manager/internal/ioc"
	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// SetupInput holds parameters for the relux setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
}

type setupTemplateData struct {
	AppTypeName string
}

// Setup integrates Relux into a Tuist project that already has IoC configured.
func Setup(input SetupInput) error {
	if err := validateSetupInput(input); err != nil {
		return err
	}

	appTypeName := scaffold.SwiftTypeName(input.AppName)
	modulesRoot := ioc.ResolveModulesPath(input.ProjectRoot, input.ModulesPath)
	sourcesDir := filepath.Join(input.ProjectRoot, "Targets", input.AppName, "Sources")
	appDir := filepath.Join(sourcesDir, "App")

	registryPath := filepath.Join(appDir, appTypeName+".Registry.swift")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		return fmt.Errorf("Registry not found at %s — run 'ioc setup' first", registryPath)
	}

	modules, err := ioc.DiscoverModules(modulesRoot)
	if err != nil {
		return fmt.Errorf("discover modules: %w", err)
	}

	if err := ioc.ScaffoldRegistryWithData(registryPath, ioc.RegistryTemplateData{
		AppTypeName: appTypeName,
		Imports:     ioc.BuildModuleImports(modules),
		Modules:     modules,
		HasRelux:    true,
	}); err != nil {
		return fmt.Errorf("regenerate Registry.swift: %w", err)
	}

	appSwiftPath := filepath.Join(sourcesDir, "App.swift")
	if err := updateAppSwiftForRelux(appSwiftPath, appTypeName); err != nil {
		return fmt.Errorf("update App.swift: %w", err)
	}

	tmplData := setupTemplateData{AppTypeName: appTypeName}

	splashPath := filepath.Join(appDir, appTypeName+".Splash.swift")
	if err := renderSetupTemplate("setup_templates/splash.swift.tmpl", splashPath, tmplData); err != nil {
		return fmt.Errorf("scaffold Splash.swift: %w", err)
	}

	contentPath := filepath.Join(appDir, appTypeName+".Content.swift")
	if err := renderSetupTemplate("setup_templates/content.swift.tmpl", contentPath, tmplData); err != nil {
		return fmt.Errorf("scaffold Content.swift: %w", err)
	}

	loggerPath := filepath.Join(appDir, appTypeName+".ReluxLogger.swift")
	if err := renderSetupTemplate("setup_templates/relux_logger.swift.tmpl", loggerPath, tmplData); err != nil {
		return fmt.Errorf("scaffold ReluxLogger.swift: %w", err)
	}

	packageSwiftPath := filepath.Join(modulesRoot, "..", "Package.swift")
	if err := patchPackageSwiftForRelux(packageSwiftPath); err != nil {
		return fmt.Errorf("patch Package.swift: %w", err)
	}

	return nil
}

func validateSetupInput(input SetupInput) error {
	if strings.TrimSpace(input.ProjectRoot) == "" {
		return fmt.Errorf("project root is required")
	}
	if strings.TrimSpace(input.AppName) == "" {
		return fmt.Errorf("app name is required")
	}
	return nil
}

func updateAppSwiftForRelux(appSwiftPath, appTypeName string) error {
	content, err := os.ReadFile(appSwiftPath)
	if err != nil {
		return fmt.Errorf("read App.swift: %w", err)
	}

	updated := EditAppSwiftForRelux(string(content), appTypeName)

	return os.WriteFile(appSwiftPath, []byte(updated), 0o644)
}

// EditAppSwiftForRelux transforms App.swift to use the Relux.Resolver pattern.
func EditAppSwiftForRelux(content, appTypeName string) string {
	result := content

	// Add imports.
	result = ioc.EnsureImport(result, "SwiftUIRelux")
	result = ensureExportedImport(result, "Relux")

	// Replace WindowGroup body with Relux.Resolver.
	result = replaceWindowGroupBody(result, appTypeName)

	return result
}

// ensureExportedImport adds @_exported import if not already present.
func ensureExportedImport(content, moduleName string) string {
	exportedLine := "@_exported import " + moduleName
	exportedPattern := regexp.MustCompile(`(?m)^@_exported\s+import\s+` + regexp.QuoteMeta(moduleName) + `\s*$`)
	if exportedPattern.MatchString(content) {
		return content
	}

	// Remove plain import if exists (we'll replace with @_exported).
	plainPattern := regexp.MustCompile(`(?m)^import\s+` + regexp.QuoteMeta(moduleName) + `\s*\n`)
	content = plainPattern.ReplaceAllString(content, "")

	// Insert @_exported import at the very top.
	return exportedLine + "\n" + content
}

var windowGroupPattern = regexp.MustCompile(`(?s)(WindowGroup\s*\{)\s*\n[^}]*?\n(\s*\})`)

// replaceWindowGroupBody replaces the WindowGroup { ... } body with Relux.Resolver.
func replaceWindowGroupBody(content, appTypeName string) string {
	if strings.Contains(content, "Relux.Resolver(") {
		return content
	}

	// Find WindowGroup { ... } and replace its body.
	// We need to find the matching brace for WindowGroup {.
	idx := strings.Index(content, "WindowGroup {")
	if idx == -1 {
		return content
	}

	openBrace := idx + len("WindowGroup {") - 1
	closeBrace, err := findMatchingBrace(content, openBrace)
	if err != nil {
		return content
	}

	// Detect indentation of the WindowGroup line.
	lineStart := strings.LastIndex(content[:idx], "\n") + 1
	windowGroupLine := content[lineStart:idx]
	baseIndent := ""
	for _, ch := range windowGroupLine {
		if ch == ' ' || ch == '\t' {
			baseIndent += string(ch)
		} else {
			break
		}
	}
	innerIndent := baseIndent + "    "

	resolverBody := fmt.Sprintf(`
%sRelux.Resolver(
%s    splash: { %s.Splash() },
%s    content: { relux in %s.Content() },
%s    resolver: { await Registry.resolveAsync(Relux.self) }
%s)
%s`, innerIndent, innerIndent, appTypeName, innerIndent, appTypeName, innerIndent, innerIndent, baseIndent)

	result := content[:openBrace+1] + resolverBody + content[closeBrace:]
	return result
}

const tuistPackageSettingsBlock = `
#if TUIST
import ProjectDescription

let packageSettings = PackageSettings(
    productTypes: [
        "Relux": .framework,
    ]
)
#endif
`

// patchPackageSwiftForRelux appends #if TUIST PackageSettings block to Package.swift.
func patchPackageSwiftForRelux(packageSwiftPath string) error {
	content, err := os.ReadFile(packageSwiftPath)
	if err != nil {
		return fmt.Errorf("read Package.swift: %w", err)
	}

	s := string(content)

	if strings.Contains(s, "PackageSettings") {
		return nil
	}

	s = strings.TrimRight(s, "\n") + "\n" + tuistPackageSettingsBlock

	return os.WriteFile(packageSwiftPath, []byte(s), 0o644)
}

func renderSetupTemplate(templatePath, outputPath string, data setupTemplateData) error {
	tmplContent, err := setupTemplatesFS.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template %q: %w", templatePath, err)
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("parse template %q: %w", templatePath, err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("render template %q: %w", templatePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", outputPath, err)
	}

	return os.WriteFile(outputPath, []byte(buf.String()), 0o644)
}
