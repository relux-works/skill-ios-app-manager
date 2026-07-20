package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

func TestGenerateAppCapabilities(t *testing.T) {
	t.Parallel()

	content := GenerateAppCapabilities()
	checks := []string{
		"import ProjectDescription",
		"public enum AppCapabilities",
		"public static let app: [Capability] = [",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Errorf("GenerateAppCapabilities() missing %q", want)
		}
	}
}

func TestGenerateAppCapabilitiesForConfigIncludesAppGroups(t *testing.T) {
	t.Parallel()

	content := GenerateAppCapabilitiesForConfig(config.ProjectConfig{
		AppGroups: []string{
			"group.com.example.app",
			"group.com.example.shared",
		},
	})

	for _, want := range []string{
		`.appGroups(group: .custom(id: "group.com.example.app"))`,
		`.appGroups(group: .custom(id: "group.com.example.shared"))`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("GenerateAppCapabilitiesForConfig() missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, `.appGroups(group: .custom(id: "group.com.example.shared")),`) {
		t.Fatalf("GenerateAppCapabilitiesForConfig() leaves a trailing collection comma:\n%s", content)
	}
}

func TestAddToAppCapabilities_KeychainSharing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	helpersDir := filepath.Join(dir, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "AppCapabilities.swift"), []byte(GenerateAppCapabilities()), 0o644); err != nil {
		t.Fatal(err)
	}

	err := AddToAppCapabilities(dir, "keychainSharing", nil)
	if err != nil {
		t.Fatalf("AddToAppCapabilities() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(helpersDir, "AppCapabilities.swift"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, ".keychainSharing()") {
		t.Errorf("expected .keychainSharing() in content:\n%s", content)
	}
}

func TestAddToAppCapabilities_AppGroups(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	helpersDir := filepath.Join(dir, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "AppCapabilities.swift"), []byte(GenerateAppCapabilities()), 0o644); err != nil {
		t.Fatal(err)
	}

	args := map[string]string{"group": "group.com.example.app"}
	err := AddToAppCapabilities(dir, "appGroups", args)
	if err != nil {
		t.Fatalf("AddToAppCapabilities() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(helpersDir, "AppCapabilities.swift"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `group.com.example.app`) {
		t.Errorf("expected appGroups line in content:\n%s", content)
	}
}

func TestSyncAppCapabilityDeclarationsAddsConfiguredLines(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	helpersDir := filepath.Join(dir, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "AppCapabilities.swift"), []byte(GenerateAppCapabilities()), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.ProjectConfig{
		AppGroups: []string{
			"group.com.example.app",
			"group.com.example.app",
			"group.com.example.shared",
		},
	}
	updated, err := SyncAppCapabilityDeclarations(dir, appGroupCapabilityDeclarations(cfg))
	if err != nil {
		t.Fatalf("SyncAppCapabilityDeclarations() error = %v", err)
	}
	if !updated {
		t.Fatal("SyncAppCapabilityDeclarations() updated = false, want true")
	}

	data, err := os.ReadFile(filepath.Join(helpersDir, "AppCapabilities.swift"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		`group.com.example.app`,
		`group.com.example.shared`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("expected app group %q in content:\n%s", want, content)
		}
	}
	for lineNumber, line := range strings.Split(content, "\n") {
		if strings.TrimRight(line, " \t") != line {
			t.Fatalf("AppCapabilities.swift line %d has trailing whitespace: %q", lineNumber+1, line)
		}
	}
	if strings.Contains(content, ".appGroups(group: .custom(id: \"group.com.example.shared\")),\n    ]") {
		t.Fatalf("AppCapabilities.swift leaves a trailing collection comma:\n%s", content)
	}

	secondUpdated, err := SyncAppCapabilityDeclarations(dir, appGroupCapabilityDeclarations(cfg))
	if err != nil {
		t.Fatalf("second SyncAppCapabilityDeclarations() error = %v", err)
	}
	if secondUpdated {
		t.Fatal("second SyncAppCapabilityDeclarations() updated = true, want false")
	}
	afterSecond, err := os.ReadFile(filepath.Join(helpersDir, "AppCapabilities.swift"))
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(afterSecond), ".appGroups("); count != 2 {
		t.Fatalf("app group capability count = %d, want 2:\n%s", count, string(afterSecond))
	}
}

func TestSyncAppCapabilityDeclarationsRepairsConvergedLegacyFormatting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	helpersDir := filepath.Join(dir, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := `import ProjectDescription

public enum AppCapabilities {
    public static let app: [Capability] = [
        // capabilities are added by module setup commands
` + "    \n" + `        .keychainSharing(),
        .appGroups(group: .custom(id: "group.com.example.app")),
    ]
}
`
	path := filepath.Join(helpersDir, "AppCapabilities.swift")
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	declarations := []string{
		"        .keychainSharing(),",
		`        .appGroups(group: .custom(id: "group.com.example.app")),`,
	}
	updated, err := SyncAppCapabilityDeclarations(dir, declarations)
	if err != nil {
		t.Fatalf("SyncAppCapabilityDeclarations() error = %v", err)
	}
	if !updated {
		t.Fatal("SyncAppCapabilityDeclarations() updated = false, want legacy formatting repair")
	}

	content := string(mustReadAppCapabilities(t, path))
	for lineNumber, line := range strings.Split(content, "\n") {
		if strings.TrimRight(line, " \t") != line {
			t.Fatalf("AppCapabilities.swift line %d has trailing whitespace: %q", lineNumber+1, line)
		}
	}
	if strings.Contains(content, ".appGroups(group: .custom(id: \"group.com.example.app\")),\n    ]") {
		t.Fatalf("AppCapabilities.swift leaves a trailing collection comma:\n%s", content)
	}

	secondUpdated, err := SyncAppCapabilityDeclarations(dir, declarations)
	if err != nil {
		t.Fatalf("second SyncAppCapabilityDeclarations() error = %v", err)
	}
	if secondUpdated {
		t.Fatal("second SyncAppCapabilityDeclarations() updated = true, want converged output")
	}
}

func mustReadAppCapabilities(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAddToAppCapabilities_PushNotifications(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	helpersDir := filepath.Join(dir, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "AppCapabilities.swift"), []byte(GenerateAppCapabilities()), 0o644); err != nil {
		t.Fatal(err)
	}

	err := AddToAppCapabilities(dir, "pushNotifications", nil)
	if err != nil {
		t.Fatalf("AddToAppCapabilities() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(helpersDir, "AppCapabilities.swift"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, ".pushNotifications(environment: .production)") {
		t.Errorf("expected pushNotifications line in content:\n%s", content)
	}
}

func TestAddToAppCapabilities_Idempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	helpersDir := filepath.Join(dir, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "AppCapabilities.swift"), []byte(GenerateAppCapabilities()), 0o644); err != nil {
		t.Fatal(err)
	}

	// Add twice.
	if err := AddToAppCapabilities(dir, "keychainSharing", nil); err != nil {
		t.Fatalf("first AddToAppCapabilities() error = %v", err)
	}
	if err := AddToAppCapabilities(dir, "keychainSharing", nil); err != nil {
		t.Fatalf("second AddToAppCapabilities() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(helpersDir, "AppCapabilities.swift"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	count := strings.Count(content, ".keychainSharing()")
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of .keychainSharing(), got %d:\n%s", count, content)
	}
}

func TestAddToAppCapabilities_UnknownType(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	helpersDir := filepath.Join(dir, "Tuist", "ProjectDescriptionHelpers")
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "AppCapabilities.swift"), []byte(GenerateAppCapabilities()), 0o644); err != nil {
		t.Fatal(err)
	}

	err := AddToAppCapabilities(dir, "unknownCapability", nil)
	if err == nil {
		t.Fatal("expected error for unknown capability type")
	}
	if !strings.Contains(err.Error(), "unknown capability type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCapabilitySwiftLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		capType string
		args    map[string]string
		want    string
	}{
		{
			name:    "keychainSharing",
			capType: "keychainSharing",
			want:    ".keychainSharing(),",
		},
		{
			name:    "appGroups",
			capType: "appGroups",
			args:    map[string]string{"group": "group.com.example"},
			want:    `group.com.example`,
		},
		{
			name:    "pushNotifications",
			capType: "pushNotifications",
			want:    ".pushNotifications(environment: .production),",
		},
		{
			name:    "unknown returns empty",
			capType: "foobar",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := capabilitySwiftLine(tt.capType, tt.args)
			if tt.want == "" && got != "" {
				t.Errorf("capabilitySwiftLine(%q) = %q, want empty", tt.capType, got)
			}
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Errorf("capabilitySwiftLine(%q) = %q, want to contain %q", tt.capType, got, tt.want)
			}
		})
	}
}
