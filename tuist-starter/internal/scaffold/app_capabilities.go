package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// AppCapabilityInput is the normalized context passed to app capability subplugins.
type AppCapabilityInput struct {
	ProjectRoot string
	Config      config.ProjectConfig
}

// AppCapabilityPluginResult describes one app capability subplugin run.
type AppCapabilityPluginResult struct {
	Name    string
	Enabled bool
	Scanned []string
	Updated []string
	Message string
}

// AppCapabilityPlugin describes one pluggable host app capability sync slice.
type AppCapabilityPlugin struct {
	Name         string
	Short        string
	Dependencies []string
	Declarations func(config.ProjectConfig) []string
	Sync         func(AppCapabilityInput) (AppCapabilityPluginResult, error)
}

var appCapabilityPluginRegistry = make(map[string]*AppCapabilityPlugin)

// RegisterAppCapability registers a pluggable app capability sync slice.
func RegisterAppCapability(plugin *AppCapabilityPlugin) {
	if plugin == nil {
		panic("app capability plugin is nil")
	}

	name := strings.TrimSpace(plugin.Name)
	if name == "" {
		panic("app capability plugin name is required")
	}
	if strings.TrimSpace(plugin.Short) == "" {
		panic(fmt.Sprintf("app capability plugin %q short description is required", name))
	}
	if plugin.Declarations == nil {
		panic(fmt.Sprintf("app capability plugin %q declarations func is required", name))
	}
	if plugin.Sync == nil {
		panic(fmt.Sprintf("app capability plugin %q sync func is required", name))
	}
	if _, exists := appCapabilityPluginRegistry[name]; exists {
		panic(fmt.Sprintf("duplicate app capability plugin %q", name))
	}

	plugin.Name = name
	appCapabilityPluginRegistry[name] = plugin
}

// AllAppCapabilityPlugins returns registered app capability subplugins sorted by name.
func AllAppCapabilityPlugins() []*AppCapabilityPlugin {
	plugins := make([]*AppCapabilityPlugin, 0, len(appCapabilityPluginRegistry))
	for _, plugin := range appCapabilityPluginRegistry {
		plugins = append(plugins, plugin)
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})

	return plugins
}

// GenerateAppCapabilities returns the initial AppCapabilities.swift content.
func GenerateAppCapabilities() string {
	return generateAppCapabilities(nil)
}

// GenerateAppCapabilitiesForConfig returns AppCapabilities.swift content with
// host app capabilities derived from project config.
func GenerateAppCapabilitiesForConfig(cfg config.ProjectConfig) string {
	lines := make([]string, 0)
	for _, plugin := range AllAppCapabilityPlugins() {
		lines = append(lines, plugin.Declarations(cfg)...)
	}

	return generateAppCapabilities(lines)
}

func generateAppCapabilities(lines []string) string {
	var appLines strings.Builder
	if len(lines) == 0 {
		appLines.WriteString("        // capabilities are added by module setup commands\n")
	} else {
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			appLines.WriteString(line)
			appLines.WriteString("\n")
		}
	}

	return `import ProjectDescription

/// Shared capability sets used across targets.
///
/// Add new capabilities here when running module setup commands.
public enum AppCapabilities {
    /// Capabilities for the main app target.
    public static let app: [Capability] = [
` + appLines.String() + `    ]
}
`
}

// AppCapabilitiesSyncResult describes changes made by app capability subplugins.
type AppCapabilitiesSyncResult struct {
	Plugins []AppCapabilityPluginResult
	Scanned []string
	Updated []string
}

// SyncAppCapabilities runs all registered host app capability subplugins.
func SyncAppCapabilities(projectRoot string, cfg config.ProjectConfig) (AppCapabilitiesSyncResult, error) {
	result := AppCapabilitiesSyncResult{
		Plugins: make([]AppCapabilityPluginResult, 0, len(AllAppCapabilityPlugins())),
		Scanned: make([]string, 0),
		Updated: make([]string, 0),
	}

	for _, plugin := range AllAppCapabilityPlugins() {
		pluginResult, err := plugin.Sync(AppCapabilityInput{
			ProjectRoot: projectRoot,
			Config:      cfg,
		})
		if err != nil {
			return result, fmt.Errorf("%s: %w", plugin.Name, err)
		}
		if pluginResult.Name == "" {
			pluginResult.Name = plugin.Name
		}

		result.Plugins = append(result.Plugins, pluginResult)
		result.Scanned = append(result.Scanned, pluginResult.Scanned...)
		result.Updated = append(result.Updated, pluginResult.Updated...)
	}

	return result, nil
}

// SyncAppCapabilityDeclarations inserts host app capability declarations into
// Tuist/ProjectDescriptionHelpers/AppCapabilities.swift. It is additive and
// idempotent so declarations from other subplugins are preserved.
func SyncAppCapabilityDeclarations(projectRoot string, declarations []string) (bool, error) {
	path := appCapabilitiesPath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read AppCapabilities.swift: %w", err)
	}

	content := string(data)
	updated := content
	changed := false
	for _, line := range declarations {
		if strings.TrimSpace(line) == "" || containsCapabilityLine(updated, line) {
			continue
		}

		next, err := insertCapabilityLine(updated, line)
		if err != nil {
			return false, err
		}
		updated = next
		changed = true
	}

	if !changed {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write AppCapabilities.swift: %w", err)
	}

	return true, nil
}

func normalizeAppGroups(appGroups []string) []string {
	normalized := make([]string, 0, len(appGroups))
	seen := make(map[string]struct{}, len(appGroups))

	for _, raw := range appGroups {
		group := strings.TrimSpace(raw)
		if group == "" {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		normalized = append(normalized, group)
	}

	return normalized
}

func swiftStringLiteralValue(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(value)
}

func containsCapabilityLine(content string, line string) bool {
	return strings.Contains(content, strings.TrimSpace(line))
}

func insertCapabilityLine(content string, line string) (string, error) {
	// Find the closing bracket of the app array.
	marker := "static let app: [Capability] = ["
	idx := strings.Index(content, marker)
	if idx < 0 {
		return "", fmt.Errorf("AppCapabilities.swift missing %q marker", marker)
	}

	// Find the closing ] after the marker.
	afterMarker := content[idx+len(marker):]
	closingIdx := strings.Index(afterMarker, "]")
	if closingIdx < 0 {
		return "", fmt.Errorf("AppCapabilities.swift missing closing ] for app array")
	}

	insertPos := idx + len(marker) + closingIdx
	updated := content[:insertPos] + "\n" + line + "\n    " + content[insertPos:]

	return updated, nil
}

// capabilitySwiftLine maps a Capability type + args to a Swift DSL expression.
func capabilitySwiftLine(capType string, args map[string]string) string {
	switch capType {
	case "keychainSharing":
		return "        .keychainSharing(),"
	case "appGroups":
		group := args["group"]
		if group == "" {
			return ""
		}
		return fmt.Sprintf(`        .appGroups(group: .custom(id: "%s")),`, swiftStringLiteralValue(group))
	case "pushNotifications":
		return "        .pushNotifications(environment: .production),"
	default:
		return ""
	}
}

// AddToAppCapabilities reads AppCapabilities.swift, inserts a capability line
// if not already present, and writes the file back. It is idempotent.
func AddToAppCapabilities(projectRoot string, capType string, args map[string]string) error {
	path := appCapabilitiesPath(projectRoot)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read AppCapabilities.swift: %w", err)
	}
	content := string(data)

	line := capabilitySwiftLine(capType, args)
	if line == "" {
		return fmt.Errorf("unknown capability type: %s", capType)
	}

	if containsCapabilityLine(content, line) {
		return nil
	}

	updated, err := insertCapabilityLine(content, line)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(updated), 0o644)
}

func appCapabilitiesPath(projectRoot string) string {
	return filepath.Join(projectRoot, "Tuist", "ProjectDescriptionHelpers", "AppCapabilities.swift")
}
