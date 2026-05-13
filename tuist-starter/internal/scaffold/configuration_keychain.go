package scaffold

import (
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// GenerateConfigurationKeychain returns the Configuration+Keychain.swift extension.
// Runtime values are resolved from the generated ApplicationConfiguration plist facade.
func GenerateConfigurationKeychain(cfg config.ProjectConfig) string {
	sharedConfigurationModuleName := appGroupSharedConfigurationModuleName(cfg)

	var b strings.Builder
	b.WriteString("import " + sharedConfigurationModuleName + "\n\n")
	b.WriteString(`extension Configuration {
    enum Keychain {
        private static let applicationConfiguration = ApplicationConfiguration.current

        static let serviceName = applicationConfiguration.applicationBundleIdentifier
        static let accessGroup = "\(applicationConfiguration.developmentTeamID).\(applicationConfiguration.applicationBundleIdentifier).shared"
    }
}
`)

	return b.String()
}
