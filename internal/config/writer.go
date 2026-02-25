package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteProjectConfig validates and writes a project config to disk.
func WriteProjectConfig(path string, cfg ProjectConfig) error {
	if strings.TrimSpace(path) == "" {
		path = DefaultConfigPath
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config for %q: %w", path, err)
	}

	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config for %q: %w", path, err)
	}

	payload = append(payload, '\n')

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create config directory %q: %w", dir, err)
		}
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write config file %q: %w", path, err)
	}

	return nil
}
