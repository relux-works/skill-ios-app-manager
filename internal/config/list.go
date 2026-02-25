package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListConfigs scans a directory for JSON configs that match ProjectConfig schema.
func ListConfigs(dir string) ([]string, error) {
	if strings.TrimSpace(dir) == "" {
		dir = "."
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if _, err := LoadConfig(path); err != nil {
			continue
		}

		paths = append(paths, path)
	}

	sort.Strings(paths)
	return paths, nil
}
