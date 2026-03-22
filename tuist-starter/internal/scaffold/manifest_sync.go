package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ManifestSyncResult describes which manifests were scanned and updated.
type ManifestSyncResult struct {
	Scanned []string
	Updated []string
}

// VersionSyncResult preserves the public result type used by version sync.
type VersionSyncResult = ManifestSyncResult

func discoverScaffoldManifestPaths(projectRoot string) ([]string, error) {
	paths := make([]string, 0, 4)

	rootProjectPath := filepath.Join(projectRoot, "Project.swift")
	if _, err := os.Stat(rootProjectPath); err == nil {
		paths = append(paths, rootProjectPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat manifest %q: %w", rootProjectPath, err)
	}

	extensionsRoot := filepath.Join(projectRoot, "Extensions")
	if _, err := os.Stat(extensionsRoot); err != nil {
		if os.IsNotExist(err) {
			return paths, nil
		}
		return nil, fmt.Errorf("stat extensions directory %q: %w", extensionsRoot, err)
	}

	err := filepath.WalkDir(extensionsRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "Project.swift" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover extension manifests: %w", err)
	}

	sort.Strings(paths)
	return paths, nil
}

func replaceStringLiteralArgument(line, value string) (string, bool) {
	stringCallIndex := strings.Index(line, ".string(")
	if stringCallIndex < 0 {
		return "", false
	}

	argStart := stringCallIndex + len(".string(")
	argEnd := strings.Index(line[argStart:], ")")
	if argEnd < 0 {
		return "", false
	}
	argEnd += argStart

	if argStart >= len(line) || line[argStart] != '"' {
		return "", false
	}
	if argEnd == 0 || line[argEnd-1] != '"' {
		return "", false
	}

	return line[:argStart] + fmt.Sprintf("%q", value) + line[argEnd:], true
}

func leadingIndent(line string) string {
	end := 0
	for end < len(line) && (line[end] == ' ' || line[end] == '\t') {
		end++
	}
	return line[:end]
}
