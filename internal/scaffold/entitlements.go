package scaffold

import (
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// GenerateEntitlements returns a plist-formatted entitlements file.
func GenerateEntitlements(cfg config.ProjectConfig) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n")
	b.WriteString(`<dict>` + "\n")
	b.WriteString(`	<key>aps-environment</key>` + "\n")
	b.WriteString(`	<string>development</string>` + "\n")

	appGroups := compactStrings(cfg.AppGroups)
	if len(appGroups) > 0 {
		b.WriteString(`	<key>com.apple.security.application-groups</key>` + "\n")
		b.WriteString(`	<array>` + "\n")
		for _, appGroup := range appGroups {
			b.WriteString(`		<string>` + xmlEscape(appGroup) + `</string>` + "\n")
		}
		b.WriteString(`	</array>` + "\n")
	}

	b.WriteString(`</dict>` + "\n")
	b.WriteString(`</plist>` + "\n")

	return b.String()
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}
