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

	mtsPath := filepath.Join(dir, "config-mts.json")
	fcPath := filepath.Join(dir, "config-fc.json")
	invalidPath := filepath.Join(dir, "config-invalid.json")
	nonJSONPath := filepath.Join(dir, "README.txt")

	mtsCfg := validProjectConfig()
	mtsCfg.BundleID = "com.example.mts"
	mtsCfg.AppName = "MTS"
	if err := WriteProjectConfig(mtsPath, mtsCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", mtsPath, err)
	}

	fcCfg := validProjectConfig()
	fcCfg.BundleID = "com.example.fc"
	fcCfg.AppName = "FC"
	if err := WriteProjectConfig(fcPath, fcCfg); err != nil {
		t.Fatalf("WriteProjectConfig(%q) error = %v", fcPath, err)
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

	want := []string{fcPath, mtsPath}
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
