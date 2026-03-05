package scaffold

import (
	"strings"
	"unicode"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// GenerateAppStub returns a minimal SwiftUI app entry point.
func GenerateAppStub(cfg config.ProjectConfig) string {
	appTypeName := SwiftTypeName(cfg.AppName)

	return `import SwiftUI

@main
struct ` + appTypeName + `: App {
    var body: some Scene {
        WindowGroup {
            Text("Hello, World!")
        }
    }
}
`
}

// SwiftTypeName converts a raw app name into a valid Swift type name.
func SwiftTypeName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "AppMain"
	}

	var b strings.Builder
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}

	name := b.String()
	if name == "" {
		return "AppMain"
	}

	first := rune(name[0])
	if unicode.IsDigit(first) {
		return "_" + name
	}

	return name
}
