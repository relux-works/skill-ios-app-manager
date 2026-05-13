package scaffold

import (
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	templatepkg "github.com/relux-works/ios-app-manager/internal/template"
)

// GenerateConfigurationAppGroups returns the Configuration+AppGroups.swift extension.
// Each app group gets a stable Swift property backed by an AppGroups dictionary key.
// Only call when cfg.AppGroups is non-empty.
func GenerateConfigurationAppGroups(cfg config.ProjectConfig) string {
	bundleID := strings.TrimSpace(cfg.BundleID)
	appName := normalizeAppName(cfg.AppName)
	sharedConfigurationModuleName := appGroupSharedConfigurationModuleName(cfg)
	sharedConfigurationTypePrefix := appGroupSharedConfigurationTypePrefix(appName)
	appGroups := normalizeAppGroups(cfg.AppGroups)

	var b strings.Builder
	b.WriteString("import Foundation\n\n")
	b.WriteString("import " + sharedConfigurationModuleName + "\n\n")
	b.WriteString("extension Configuration {\n")
	b.WriteString("    enum AppGroups {\n")
	b.WriteString("        static let serviceName: String = ApplicationConfiguration.current.applicationBundleIdentifier\n")
	b.WriteString("\n")
	b.WriteString("        private static let resolved: " + sharedConfigurationTypePrefix + "AppGroups = {\n")
	b.WriteString("            do {\n")
	b.WriteString("                return try " + sharedConfigurationTypePrefix + "AppGroups.read(from: .main)\n")
	b.WriteString("            } catch {\n")
	b.WriteString("                fatalError(\"Could not read app groups from Info.plist: \\(error.localizedDescription)\")\n")
	b.WriteString("            }\n")
	b.WriteString("        }()\n")

	for _, group := range appGroups {
		propertyName := templatepkg.AppGroupSwiftIdentifier(bundleID, group)
		b.WriteString("\n")
		b.WriteString("        static let " + propertyName + ": String = resolved." + propertyName + "\n")
	}

	b.WriteString("    }\n")
	b.WriteString("}\n")

	return b.String()
}
