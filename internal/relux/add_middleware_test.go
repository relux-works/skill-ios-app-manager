package relux

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddMiddlewareCommandRunGeneratesNamedMiddleware(t *testing.T) {
	modulePath := scaffoldLegacyReluxModuleForTest(t, "Notes")

	engine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("NewTemplateEngine() error = %v", err)
	}

	command, err := NewAddMiddlewareCommand(engine)
	if err != nil {
		t.Fatalf("NewAddMiddlewareCommand() error = %v", err)
	}

	generatedPath, err := command.Run(context.Background(), AddMiddlewareInput{
		ModuleName:     "Notes",
		ModulePath:     modulePath,
		MiddlewareName: "Analytics",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	expectedPath := filepath.Join(modulePath+"Impl", "Sources", "NotesImpl", "analytics_middleware.swift")
	if generatedPath != expectedPath {
		t.Fatalf("Run() path = %q, want %q", generatedPath, expectedPath)
	}

	content, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", generatedPath, err)
	}

	asString := string(content)
	if !strings.Contains(asString, "protocol NotesAnalyticsMiddlewareProtocol") {
		t.Fatalf("generated middleware missing protocol rename:\n%s", asString)
	}
	if !strings.Contains(asString, "actor NotesAnalyticsMiddleware: NotesAnalyticsMiddlewareProtocol") {
		t.Fatalf("generated middleware missing actor rename:\n%s", asString)
	}
}
