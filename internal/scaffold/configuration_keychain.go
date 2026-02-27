package scaffold

import (
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// GenerateConfigurationKeychain returns the Configuration+Keychain.swift extension.
// The accessGroup uses the team ID prefix (e.g. "ABCDE12345.com.example.app.shared"),
// matching the entitlements plist value where $(AppIdentifierPrefix) resolves to "<teamId>.".
func GenerateConfigurationKeychain(cfg config.ProjectConfig) string {
	bundleID := strings.TrimSpace(cfg.BundleID)
	teamID := strings.TrimSpace(cfg.TeamID)
	accessGroup := teamID + "." + bundleID + ".shared"

	return `extension Configuration {
    enum Keychain {
        static let serviceName = "` + bundleID + `"
        static let accessGroup = "` + accessGroup + `"
    }
}
`
}
