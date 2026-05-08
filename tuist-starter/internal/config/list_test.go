package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListConfigs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	alphaPath := filepath.Join(dir, "config-alpha.json")
	betaPath := filepath.Join(dir, "config-beta.json")
	invalidPath := filepath.Join(dir, "config-invalid.json")
	nonJSONPath := filepath.Join(dir, "README.txt")

	alphaCfg := validProjectConfig()
	alphaCfg.BundleID = "com.example.alpha"
	alphaCfg.AppName = "Alpha"
	if err := WriteProjectConfig(alphaPath, alphaCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", alphaPath, err)
	}

	betaCfg := validProjectConfig()
	betaCfg.BundleID = "com.example.beta"
	betaCfg.AppName = "Beta"
	if err := WriteProjectConfig(betaPath, betaCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", betaPath, err)
	}

	if err := os.WriteFile(invalidPath, []byte(`{"app_name":"oops"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", invalidPath, err)
	}

	if err := os.WriteFile(nonJSONPath, []byte("not json"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", nonJSONPath, err)
	}

	got, err := ListConfigs(dir)
	if err != nil {
		t.Fatalf("ListConfigs(%q) error = %v", dir, err)
	}

	want := []string{alphaPath, betaPath}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListConfigs(%q) = %#v, want %#v", dir, got, want)
	}
}

func TestListConfigsEmptyDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	got, err := ListConfigs(dir)
	if err != nil {
		t.Fatalf("ListConfigs(%q) error = %v", dir, err)
	}

	if len(got) != 0 {
		t.Fatalf("ListConfigs(%q) = %#v, want empty slice", dir, got)
	}
}
