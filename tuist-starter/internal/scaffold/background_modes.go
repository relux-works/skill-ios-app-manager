package scaffold

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

var backgroundModesLinePattern = regexp.MustCompile(`^\s*"UIBackgroundModes":\s*\.array\(\[.*\]\),\s*$`)

// SyncBackgroundModes updates the scaffolded app manifest so the app target's
// Info.plist `UIBackgroundModes` entry matches the configured background modes.
//
// Behavior mirrors the other manifest sync generators: replace the existing
// single-line entry in place, insert one before the `UILaunchScreen` anchor
// when missing, and remove the entry when the configured list is empty.
func SyncBackgroundModes(projectRoot string, cfg config.ProjectConfig) (ManifestSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return ManifestSyncResult{}, fmt.Errorf("project root is required")
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

		updated, changed, err := syncBackgroundModesManifest(string(payload), cfg.BackgroundModes)
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("sync background modes in %q: %w", manifestPath, err)
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

func syncBackgroundModesManifest(content string, backgroundModes []string) (string, bool, error) {
	modes := normalizeBackgroundModes(backgroundModes)

	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	existingIndex := -1
	launchScreenIndex := -1
	for index, line := range lines {
		if existingIndex < 0 && backgroundModesLinePattern.MatchString(line) {
			existingIndex = index
		}
		if launchScreenIndex < 0 && strings.HasPrefix(strings.TrimSpace(line), `"UILaunchScreen":`) {
			launchScreenIndex = index
		}
	}

	if len(modes) == 0 {
		if existingIndex < 0 {
			return content, false, nil
		}
		lines = append(lines[:existingIndex], lines[existingIndex+1:]...)
		return joinSyncLines(lines, hasTrailingNewline), true, nil
	}

	if existingIndex >= 0 {
		replacement := leadingIndent(lines[existingIndex]) + backgroundModesEntry(modes)
		if replacement == lines[existingIndex] {
			return content, false, nil
		}
		lines[existingIndex] = replacement
		return joinSyncLines(lines, hasTrailingNewline), true, nil
	}

	if launchScreenIndex < 0 {
		// No UILaunchScreen anchor means this is not the host app manifest
		// (extension manifests have no app Info.plist block): leave it alone.
		return content, false, nil
	}

	insertLine := leadingIndent(lines[launchScreenIndex]) + backgroundModesEntry(modes)
	lines = insertSyncLine(lines, launchScreenIndex, insertLine)
	return joinSyncLines(lines, hasTrailingNewline), true, nil
}

func backgroundModesEntry(modes []string) string {
	values := make([]string, 0, len(modes))
	for _, mode := range modes {
		values = append(values, fmt.Sprintf(".string(%q)", mode))
	}
	return fmt.Sprintf(`"UIBackgroundModes": .array([%s]),`, strings.Join(values, ", "))
}

func normalizeBackgroundModes(modes []string) []string {
	normalized := make([]string, 0, len(modes))
	for _, mode := range modes {
		trimmed := strings.TrimSpace(mode)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
