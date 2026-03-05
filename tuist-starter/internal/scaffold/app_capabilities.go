package scaffold

import (
	"fmt"
	"os"
	"strings"
)

// GenerateAppCapabilities returns the initial AppCapabilities.swift content.
func GenerateAppCapabilities() string {
	return `import ProjectDescription

/// Shared capability sets used across targets.
///
/// Add new capabilities here when running module setup commands.
public enum AppCapabilities {
    /// Capabilities for the main app target.
    public static let app: [Capability] = [
        // capabilities are added by module setup commands
    ]
}
`
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
		return fmt.Sprintf(`        .appGroups(group: .custom(id: "%s")),`, group)
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

	// Idempotent: skip if line already present.
	trimmed := strings.TrimSpace(line)
	if strings.Contains(content, trimmed) {
		return nil
	}

	// Find the closing bracket of the app array.
	marker := "static let app: [Capability] = ["
	idx := strings.Index(content, marker)
	if idx < 0 {
		return fmt.Errorf("AppCapabilities.swift missing %q marker", marker)
	}

	// Find the closing ] after the marker.
	afterMarker := content[idx+len(marker):]
	closingIdx := strings.Index(afterMarker, "]")
	if closingIdx < 0 {
		return fmt.Errorf("AppCapabilities.swift missing closing ] for app array")
	}

	insertPos := idx + len(marker) + closingIdx
	updated := content[:insertPos] + "\n" + line + "\n    " + content[insertPos:]

	return os.WriteFile(path, []byte(updated), 0o644)
}

func appCapabilitiesPath(projectRoot string) string {
	return fmt.Sprintf("%s/Tuist/ProjectDescriptionHelpers/AppCapabilities.swift", projectRoot)
}
