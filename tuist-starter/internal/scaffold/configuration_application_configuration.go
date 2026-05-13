package scaffold

import (
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func GenerateConfigurationApplicationConfiguration(cfg config.ProjectConfig) string {
	appName := normalizeAppName(cfg.AppName)
	sharedConfigurationModuleName := appGroupSharedConfigurationModuleName(cfg)
	sharedConfigurationTypePrefix := appGroupSharedConfigurationTypePrefix(appName)

	var b strings.Builder
	b.WriteString("import Foundation\n\n")
	b.WriteString("import " + sharedConfigurationModuleName + "\n\n")
	b.WriteString("extension Configuration {\n")
	b.WriteString("    enum ApplicationConfiguration {\n")
	b.WriteString("        static let current: " + sharedConfigurationTypePrefix + "ApplicationConfiguration = {\n")
	b.WriteString("            do {\n")
	b.WriteString("                return try " + sharedConfigurationTypePrefix + "ApplicationConfiguration.read(from: .main)\n")
	b.WriteString("            } catch {\n")
	b.WriteString("                fatalError(\"Could not read ApplicationConfiguration from Info.plist: \\(error.localizedDescription)\")\n")
	b.WriteString("            }\n")
	b.WriteString("        }()\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return b.String()
}
