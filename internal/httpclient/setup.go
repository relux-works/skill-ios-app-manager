package httpclient

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/scaffold"
)

const (
	registrationLine = `            ioc.register(IRpcAsyncClient.self, lifecycle: .container, resolver: Self.buildHttpClient)`
	builderFunc      = `
    private static func buildHttpClient() -> IRpcAsyncClient {
        RpcClient(
            sessionConfig: ApiSessionConfigBuilder.buildConfig(
                timeoutForResponse: Configuration.HttpClient.timeoutForResponse,
                timeoutResourceInterval: Configuration.HttpClient.timeoutResourceInterval
            )
        )
    }`

	networkAnchor         = "// MARK: - Network (scaffolding anchor: network)"
	networkBuildersAnchor = "// MARK: - Network Builders (scaffolding anchor: network-builders)"
)

//go:embed setup_templates/*.tmpl
var setupTemplatesFS embed.FS

// SetupInput holds parameters for the http-client setup command.
type SetupInput struct {
	ProjectRoot string
	AppName     string
	ModulesPath string
}

// Setup adds HttpClient IoC registration, swift-httpclient dependency,
// and Configuration+HttpClient constants.
func Setup(input SetupInput) error {
	if err := validateInput(input); err != nil {
		return err
	}

	appTypeName := scaffold.SwiftTypeName(input.AppName)

	// 1. Create Configuration+HttpClient.swift.
	configDir := filepath.Join(input.ProjectRoot, "Targets", input.AppName, "Sources", "Configuration")
	if err := scaffoldConfigurationExtension(configDir); err != nil {
		return fmt.Errorf("scaffold Configuration+HttpClient: %w", err)
	}

	// 2. Patch Registry.swift with registration + builder + import.
	registryPath := filepath.Join(
		input.ProjectRoot, "Targets", input.AppName, "Sources", "App",
		appTypeName+".Registry.swift",
	)
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

func scaffoldConfigurationExtension(configDir string) error {
	outputPath := filepath.Join(configDir, "Configuration+HttpClient.swift")

	// Idempotent: skip if already exists.
	if _, err := os.Stat(outputPath); err == nil {
		return nil
	}

	content, err := setupTemplatesFS.ReadFile("setup_templates/configuration_http_client.swift.tmpl")
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	return os.WriteFile(outputPath, content, 0o644)
}

func patchRegistry(registryPath string) error {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return fmt.Errorf("read Registry.swift: %w", err)
	}

	content := string(data)

	// Idempotent: skip if already patched.
	if strings.Contains(content, "IRpcAsyncClient.self") {
		return nil
	}

	// Add import HttpClient after SwiftIoC import.
	if !strings.Contains(content, "import HttpClient") {
		content = strings.Replace(content, "import SwiftIoC", "import SwiftIoC\nimport HttpClient", 1)
	}

	// Insert registration after network anchor.
	content = strings.Replace(
		content,
		networkAnchor,
		networkAnchor+"\n"+registrationLine,
		1,
	)

	// Insert builder into Network Builders extension.
	// Find the extension block after the network-builders anchor and insert inside.
	anchorIdx := strings.Index(content, networkBuildersAnchor)
	if anchorIdx < 0 {
		return fmt.Errorf("network-builders anchor not found in Registry.swift")
	}

	// Find the closing brace of the extension block.
	rest := content[anchorIdx:]
	extStart := strings.Index(rest, "{")
	if extStart < 0 {
		return fmt.Errorf("extension opening brace not found after network-builders anchor")
	}

	// Find matching closing brace — the extension is simple (no nested types).
	closingIdx := findMatchingBrace(rest, extStart)
	if closingIdx < 0 {
		return fmt.Errorf("extension closing brace not found after network-builders anchor")
	}

	// Insert builder before the closing brace.
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
