package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
