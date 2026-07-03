package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// SyncBundleID updates scaffolded app and extension manifests to use the configured bundle identifier.
func SyncBundleID(projectRoot string, cfg config.ProjectConfig) (ManifestSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return ManifestSyncResult{}, fmt.Errorf("project root is required")
	}

	bundleID := strings.TrimSuffix(strings.TrimSpace(cfg.BundleID), ".")
	if bundleID == "" {
		return ManifestSyncResult{}, fmt.Errorf("bundle id is required")
	}

	manifestPaths, err := discoverScaffoldManifestPaths(root)
	if err != nil {
		return ManifestSyncResult{}, err
	}
	if len(manifestPaths) == 0 {
		return ManifestSyncResult{}, fmt.Errorf("no scaffold Project.swift manifests found in %q; run init first", root)
	}

	result := ManifestSyncResult{
		Scanned: append([]string(nil), manifestPaths...),
		Updated: make([]string, 0, len(manifestPaths)),
	}

	for _, manifestPath := range manifestPaths {
		payload, err := os.ReadFile(manifestPath)
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("read manifest %q: %w", manifestPath, err)
		}

		updated, changed, err := syncBundleIDManifest(
			string(payload),
			bundleID,
			isExtensionManifestPath(root, manifestPath),
		)
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("sync bundle id in %q: %w", manifestPath, err)
		}
		if !changed {
			continue
		}

		if err := os.WriteFile(manifestPath, []byte(updated), 0o644); err != nil {
			return ManifestSyncResult{}, fmt.Errorf("write manifest %q: %w", manifestPath, err)
		}

		result.Updated = append(result.Updated, manifestPath)
	}

	return result, nil
}

func isExtensionManifestPath(projectRoot, manifestPath string) bool {
	rel, err := filepath.Rel(projectRoot, manifestPath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return strings.HasPrefix(rel, "Extensions/")
}

func syncBundleIDManifest(content string, bundleID string, isExtension bool) (string, bool, error) {
	if isExtension {
		return syncExtensionBundleIDManifest(content, bundleID)
	}
	return syncHostBundleIDManifest(content, bundleID)
}

func syncHostBundleIDManifest(content string, bundleID string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "let bundleID = ") {
			continue
		}

		replacement := leadingIndent(line) + fmt.Sprintf("let bundleID = %q", bundleID)
		if replacement == line {
			return content, false, nil
		}
		lines[index] = replacement
		return joinSyncLines(lines, hasTrailingNewline), true, nil
	}

	for index, line := range lines {
		value, ok := parseBundleIDLineLiteral(line)
		if !ok {
			continue
		}

		replacement := replaceBundleIDLineLiteral(line, value, bundleID)
		if replacement == line {
			return content, false, nil
		}
		lines[index] = replacement
		return joinSyncLines(lines, hasTrailingNewline), true, nil
	}

	return "", false, fmt.Errorf("bundle id marker not found")
}

func syncExtensionBundleIDManifest(content string, hostBundleID string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "let hostBundleId = ") {
			continue
		}

		replacement := leadingIndent(line) + fmt.Sprintf("let hostBundleId = %q", hostBundleID)
		if replacement == line {
			return content, false, nil
		}
		lines[index] = replacement
		return joinSyncLines(lines, hasTrailingNewline), true, nil
	}

	for index, line := range lines {
		value, ok := parseBundleIDLineLiteral(line)
		if !ok {
			continue
		}

		nextBundleID, err := deriveExtensionBundleID(hostBundleID, value)
		if err != nil {
			return "", false, err
		}
		replacement := replaceBundleIDLineLiteral(line, value, nextBundleID)
		if replacement == line {
			return content, false, nil
		}
		lines[index] = replacement
		return joinSyncLines(lines, hasTrailingNewline), true, nil
	}

	return "", false, fmt.Errorf("extension bundle id marker not found")
}

func parseBundleIDLineLiteral(line string) (string, bool) {
	tokenIndex := strings.Index(line, "bundleId:")
	if tokenIndex < 0 {
		return "", false
	}

	valueStart := tokenIndex + len("bundleId:")
	for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
		valueStart++
	}
	if valueStart >= len(line) || line[valueStart] != '"' {
		return "", false
	}

	valueEnd := valueStart + 1
	for valueEnd < len(line) && line[valueEnd] != '"' {
		valueEnd++
	}
	if valueEnd >= len(line) {
		return "", false
	}

	return line[valueStart+1 : valueEnd], true
}

func replaceBundleIDLineLiteral(line string, oldValue string, newValue string) string {
	return strings.Replace(line, fmt.Sprintf("%q", oldValue), fmt.Sprintf("%q", newValue), 1)
}

func deriveExtensionBundleID(hostBundleID string, currentBundleID string) (string, error) {
	current := strings.TrimSuffix(strings.TrimSpace(currentBundleID), ".")
	if current == "" {
		return "", fmt.Errorf("extension bundle id is empty")
	}
	if strings.HasPrefix(current, hostBundleID+".") {
		return current, nil
	}

	lastDot := strings.LastIndex(current, ".")
	if lastDot < 0 || lastDot == len(current)-1 {
		return "", fmt.Errorf("extension bundle id %q does not contain a suffix", currentBundleID)
	}

	return hostBundleID + "." + current[lastDot+1:], nil
}
