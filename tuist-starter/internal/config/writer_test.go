package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWriteProjectConfigPrettyPrint(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.json")

	cfg := validProjectConfig()
	if err := WriteProjectConfig(path, cfg); err != nil {
		t.Fatalf("WriteProjectConfig() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "\n  \"app_name\":") {
		t.Fatalf("written config = %q, want snake_case indented JSON", text)
	}

	if !strings.HasSuffix(text, "\n") {
		t.Fatalf("written config should end with newline: %q", text)
	}

	got, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if !reflect.DeepEqual(got, cfg) {
		t.Fatalf("LoadConfig() = %#v, want %#v", got, cfg)
	}
}

func TestWriteProjectConfigDefaultPath(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	if err := WriteProjectConfig("", validProjectConfig()); err != nil {
		t.Fatalf("WriteProjectConfig(\"\") error = %v", err)
	}

	if _, err := os.Stat(DefaultConfigPath); err != nil {
		t.Fatalf("os.Stat(%q) error = %v", DefaultConfigPath, err)
	}
}

func TestWriteProjectConfigValidationError(t *testing.T) {
	t.Parallel()

	cfg := validProjectConfig()
	cfg.TeamID = ""

	err := WriteProjectConfig("config.json", cfg)
	if err == nil {
		t.Fatal("WriteProjectConfig() error = nil, want validation error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "invalid config") || !strings.Contains(msg, "TeamID is required") {
		t.Fatalf("WriteProjectConfig() error = %q, want invalid config/TeamID is required", msg)
	}
}

func TestWriteProjectConfigAppliesDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config-defaults.json")

	cfg := validProjectConfig()
	cfg.ProductName = ""
	cfg.Theme = ""
	cfg.Orientation = ""
	cfg.ModulesPath = ""
	cfg.SharedConfig.ModuleName = ""

	if err := WriteProjectConfig(path, cfg); err != nil {
		t.Fatalf("WriteProjectConfig() error = %v", err)
	}

	got, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got.ProductName != cfg.AppName {
		t.Fatalf("ProductName = %q, want %q", got.ProductName, cfg.AppName)
	}
	if got.ModulesPath != "Packages" {
		t.Fatalf("ModulesPath = %q, want %q", got.ModulesPath, "Packages")
	}
	if got.SharedConfig.ModuleName != "SharedConfig" {
		t.Fatalf("SharedConfig.ModuleName = %q, want %q", got.SharedConfig.ModuleName, "SharedConfig")
	}
	if got.Theme != ThemeAutomatic {
		t.Fatalf("Theme = %q, want %q", got.Theme, ThemeAutomatic)
	}
	if got.Orientation != OrientationAutomatic {
		t.Fatalf("Orientation = %q, want %q", got.Orientation, OrientationAutomatic)
	}
}
