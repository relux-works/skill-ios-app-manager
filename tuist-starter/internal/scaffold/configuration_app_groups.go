package scaffold

import (
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	templatepkg "github.com/relux-works/ios-app-manager/internal/template"
)

// GenerateConfigurationAppGroups returns the Configuration+AppGroups.swift extension.
// Each app group gets a static property keyed by its Info.plist key name.
// Only call when cfg.AppGroups is non-empty.
func GenerateConfigurationAppGroups(cfg config.ProjectConfig) string {
	bundleID := strings.TrimSpace(cfg.BundleID)

	var b strings.Builder
	b.WriteString("extension Configuration {\n")
	b.WriteString("    enum AppGroups {\n")
	b.WriteString(`        static let serviceName: String = "` + bundleID + "\"\n")

	for _, group := range cfg.AppGroups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		key := templatepkg.InfoPlistKey(group)
		b.WriteString("\n")
		b.WriteString("        static let " + key + ": String = {\n")
		b.WriteString(`            Bundle.main.readInfoPlistValue(by: "` + key + "\")\n")
		b.WriteString("        }()\n")
	}

	b.WriteString("    }\n")
	b.WriteString("}\n")

	return b.String()
}
