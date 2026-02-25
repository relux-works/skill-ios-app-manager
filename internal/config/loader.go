package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// LoadConfig reads and validates a project config file from disk.
func LoadConfig(path string) (ProjectConfig, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultConfigPath
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProjectConfig{}, fmt.Errorf("config file %q does not exist", path)
		}

		return ProjectConfig{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return ProjectConfig{}, fmt.Errorf("parse config file %q: %w", path, err)
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return ProjectConfig{}, fmt.Errorf("validate config file %q: %w", path, err)
	}

	return cfg, nil
}

// LoadProjectConfig keeps compatibility with older callsites.
func LoadProjectConfig(path string) (ProjectConfig, error) {
	return LoadConfig(path)
}
