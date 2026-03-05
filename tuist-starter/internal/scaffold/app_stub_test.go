package scaffold

import (
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestGenerateAppStubUsesAppName(t *testing.T) {
	t.Parallel()

	appStub := GenerateAppStub(config.ProjectConfig{
		AppName: "DemoApp",
	})

	requiredSnippets := []string{
		"import SwiftUI",
		"@main",
		"struct DemoApp: App",
		`Text("Hello, World!")`,
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(appStub, snippet) {
			t.Fatalf("GenerateAppStub() missing %q:\n%s", snippet, appStub)
		}
	}
}

func TestGenerateAppStubSanitizesInvalidTypeName(t *testing.T) {
	t.Parallel()

	appStub := GenerateAppStub(config.ProjectConfig{
		AppName: "123 Demo App",
	})

	if !strings.Contains(appStub, "struct _123DemoApp: App") {
		t.Fatalf("GenerateAppStub() should sanitize type name:\n%s", appStub)
	}
}
