package appconfig

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

const (
	foundationAnchor         = "// MARK: - Foundation (scaffolding anchor: foundation)"
	foundationBuildersAnchor = "// MARK: - Foundation Builders (scaffolding anchor: foundation-builders)"

	registrationLine = `            ioc.register(IApiConfigManager.self, lifecycle: .container, resolver: Self.buildAppConfigManager)`
	builderFunc      = `
    private static func buildAppConfigManager() -> IApiConfigManager {
        AppConfig.Business.Manager(secureStore: resolve(SecureStoring.self))
    }`
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// templateFile maps a template name to its output filename.
type templateFile struct {
	templateName string
	outputFile   string
}

var appConfigFiles = []templateFile{
	{templateName: "namespace.swift.tmpl", outputFile: "AppConfig.swift"},
	{templateName: "env.swift.tmpl", outputFile: "AppConfig.Env.swift"},
	{templateName: "configuration.swift.tmpl", outputFile: "AppConfig.Env+Configuration.swift"},
	{templateName: "presets.swift.tmpl", outputFile: "AppConfig.Env+Configuration+Presets.swift"},
	{templateName: "protocols.swift.tmpl", outputFile: "AppConfig.Manager+Protocols.swift"},
	{templateName: "manager.swift.tmpl", outputFile: "AppConfig.Manager.swift"},
	{templateName: "api_configurator.swift.tmpl", outputFile: "AppConfig.ApiConfigurator.swift"},
	{templateName: "url_components.swift.tmpl", outputFile: "AppConfig.UrlComponents.swift"},
}

// SetupInput holds parameters for the app-config setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
}

// Setup scaffolds AppConfig files and patches Registry.swift with IoC registration.
func Setup(input SetupInput) error {
	if err := validateInput(input); err != nil {
		return err
	}

	appTypeName := scaffold.SwiftTypeName(input.AppName)

	// 1. Check Registry.swift exists (ioc setup must run first).
	registryPath := filepath.Join(
		input.ProjectRoot, "Targets", input.AppName, "Sources", "App",
		appTypeName+".Registry.swift",
	)
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		return fmt.Errorf("Registry.swift not found at %s — run 'ioc setup' first", registryPath)
	}

	// 2. Check SecureStore is registered (secure-store setup must run first).
	registryData, err := os.ReadFile(registryPath)
	if err != nil {
		return fmt.Errorf("read Registry.swift: %w", err)
	}
	if !strings.Contains(string(registryData), "SecureStore.Module.Interface.self") {
		return fmt.Errorf("SecureStore not found in Registry.swift — run 'secure-store setup' first")
	}

	// 3. Scaffold 8 Swift files into Targets/<AppName>/Sources/AppConfig/.
	appConfigDir := filepath.Join(input.ProjectRoot, "Targets", input.AppName, "Sources", "AppConfig")
	if err := scaffoldFiles(appConfigDir); err != nil {
		return fmt.Errorf("scaffold AppConfig files: %w", err)
	}

	// 4. Patch Registry.swift with registration + builder.
	if err := patchRegistry(registryPath); err != nil {
		return fmt.Errorf("patch Registry.swift: %w", err)
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

func scaffoldFiles(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create directory %q: %w", outputDir, err)
	}

	for _, tf := range appConfigFiles {
		outputPath := filepath.Join(outputDir, tf.outputFile)

		// Idempotent: skip files that already exist.
		if _, err := os.Stat(outputPath); err == nil {
			continue
		}

		content, err := setupTemplatesFS.ReadFile("setup_templates/" + tf.templateName)
		if err != nil {
			return fmt.Errorf("read template %q: %w", tf.templateName, err)
		}

		if err := os.WriteFile(outputPath, content, 0o644); err != nil {
			return fmt.Errorf("write %q: %w", tf.outputFile, err)
		}
	}

	return nil
}

func patchRegistry(registryPath string) error {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return fmt.Errorf("read Registry.swift: %w", err)
	}

	content := string(data)

	// Idempotent: skip if already patched.
	if strings.Contains(content, "IApiConfigManager.self") {
		return nil
	}

	// Insert registration after foundation anchor.
	if !strings.Contains(content, foundationAnchor) {
		return fmt.Errorf("foundation anchor not found in Registry.swift")
	}
	content = strings.Replace(
		content,
		foundationAnchor,
		foundationAnchor+"\n"+registrationLine,
		1,
	)

	// Insert builder into Foundation Builders extension.
	anchorIdx := strings.Index(content, foundationBuildersAnchor)
	if anchorIdx < 0 {
		return fmt.Errorf("foundation-builders anchor not found in Registry.swift")
	}

	rest := content[anchorIdx:]
	extStart := strings.Index(rest, "{")
	if extStart < 0 {
		return fmt.Errorf("extension opening brace not found after foundation-builders anchor")
	}

	closingIdx := findMatchingBrace(rest, extStart)
	if closingIdx < 0 {
		return fmt.Errorf("extension closing brace not found after foundation-builders anchor")
	}

	insertPos := anchorIdx + closingIdx
	content = content[:insertPos] + builderFunc + "\n" + content[insertPos:]

	return os.WriteFile(registryPath, []byte(content), 0o644)
}

// findMatchingBrace finds the index of the closing brace matching the opening brace at pos.
func findMatchingBrace(s string, pos int) int {
	depth := 0
	for i := pos; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
