package scaffold

import (
	"fmt"
	"os"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// SyncBuildFlags updates scaffolded app, nested app, and extension manifests to use config-driven Swift build flags.
func SyncBuildFlags(projectRoot string, cfg config.ProjectConfig) (ManifestSyncResult, error) {
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

	settings := effectiveBuildFlagSettings(cfg)

	for _, manifestPath := range manifestPaths {
		payload, err := os.ReadFile(manifestPath)
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("read manifest %q: %w", manifestPath, err)
		}

		updated, changed, err := syncBuildFlagsManifest(string(payload), settings)
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("sync build flags in %q: %w", manifestPath, err)
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

func syncBuildFlagsManifest(content string, settings []config.SwiftBuildSetting) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	changed := false
	sawBaseBlock := false

	for index := 0; index < len(lines); index++ {
		if !isBuildSettingsDictionaryBlockLine(lines[index]) {
			continue
		}

		sawBaseBlock = true

		endIndex, err := findDelimitedBlockEnd(lines, index, "[", "]")
		if err != nil {
			return "", false, err
		}

		nextLines, blockChanged, err := ensureBuildFlagSettingsBlock(lines, index, endIndex, settings)
		if err != nil {
			return "", false, err
		}
		if blockChanged {
			changed = true
			endIndex += len(nextLines) - len(lines)
			lines = nextLines
		}

		index = endIndex
	}

	if !sawBaseBlock {
		return "", false, fmt.Errorf("build flag insertion anchor not found")
	}

	return joinSyncLines(lines, hasTrailingNewline), changed, nil
}

func isBuildSettingsDictionaryBlockLine(line string) bool {
	return strings.Contains(line, "base: [") ||
		(strings.Contains(line, "SettingsDictionary") && strings.Contains(line, "["))
}

func ensureBuildFlagSettingsBlock(lines []string, startIndex, endIndex int, settings []config.SwiftBuildSetting) ([]string, bool, error) {
	if endIndex <= startIndex {
		return nil, false, fmt.Errorf("invalid build settings block")
	}

	blockIndent := leadingIndent(lines[startIndex]) + "    "
	updated := append([]string{}, lines...)
	changed := false

	for _, setting := range settings {
		token := fmt.Sprintf(`"%s":`, setting.Key)
		found := false

		for index := startIndex + 1; index < endIndex; index++ {
			if !strings.Contains(updated[index], token) {
				continue
			}

			found = true
			replacement := renderBuildFlagSettingLine(blockIndent, setting)
			if updated[index] != replacement {
				updated[index] = replacement
				changed = true
			}
			break
		}

		if found {
			continue
		}

		insertIndex := endIndex
		lastPropertyIndex := endIndex - 1
		for lastPropertyIndex > startIndex && strings.TrimSpace(updated[lastPropertyIndex]) == "" {
			lastPropertyIndex--
		}
		if lastPropertyIndex > startIndex {
			trimmed := strings.TrimSpace(updated[lastPropertyIndex])
			if trimmed != "" && !strings.HasSuffix(trimmed, ",") {
				updated[lastPropertyIndex] = updated[lastPropertyIndex] + ","
				changed = true
			}
		}

		updated = insertSyncLine(updated, insertIndex, renderBuildFlagSettingLine(blockIndent, setting))
		endIndex++
		changed = true
	}

	return updated, changed, nil
}

func renderBuildFlagSettingLine(indent string, setting config.SwiftBuildSetting) string {
	return indent + fmt.Sprintf(`"%s": %q,`, setting.Key, setting.Value)
}

func effectiveBuildFlagSettings(cfg config.ProjectConfig) []config.SwiftBuildSetting {
	return cfg.EffectiveSwiftSettings().XcodeBuildSettings()
}
