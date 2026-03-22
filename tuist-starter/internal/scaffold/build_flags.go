package scaffold

import (
	"fmt"
	"os"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// SyncBuildFlags updates scaffolded app and extension manifests to use the strict Swift build flag baseline.
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
	updated := content
	changed := false

	for _, setting := range settings {
		next, settingChanged, err := ensureBuildFlagSetting(updated, setting, settings)
		if err != nil {
			return "", false, err
		}
		updated = next
		changed = changed || settingChanged
	}

	return updated, changed, nil
}

func ensureBuildFlagSetting(content string, setting config.SwiftBuildSetting, settings []config.SwiftBuildSetting) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	token := fmt.Sprintf(`"%s":`, setting.Key)
	for index, line := range lines {
		if !strings.Contains(line, token) {
			continue
		}

		replacement := leadingIndent(line) + fmt.Sprintf(`"%s": %q,`, setting.Key, setting.Value)
		if replacement == line {
			return content, false, nil
		}

		lines[index] = replacement
		return joinSyncLines(lines, hasTrailingNewline), true, nil
	}

	insertIndex, insertIndent := buildFlagInsertionPoint(lines, settings)
	if insertIndex < 0 {
		return "", false, fmt.Errorf("build flag insertion anchor not found")
	}

	insertLine := insertIndent + fmt.Sprintf(`    "%s": %q,`, setting.Key, setting.Value)
	lines = insertSyncLine(lines, insertIndex, insertLine)
	return joinSyncLines(lines, hasTrailingNewline), true, nil
}

func buildFlagInsertionPoint(lines []string, settings []config.SwiftBuildSetting) (int, string) {
	baseIndex := -1
	baseIndent := ""
	lastAnchorIndex := -1

	anchorTokens := []string{
		`"SWIFT_VERSION":`,
		`"IPHONEOS_DEPLOYMENT_TARGET":`,
	}
	for _, setting := range settings {
		anchorTokens = append(anchorTokens, fmt.Sprintf(`"%s":`, setting.Key))
	}

	for index, line := range lines {
		if baseIndex < 0 && strings.Contains(line, "base: [") {
			baseIndex = index
			baseIndent = leadingIndent(line)
		}

		for _, token := range anchorTokens {
			if strings.Contains(line, token) {
				lastAnchorIndex = index
				break
			}
		}
	}

	if lastAnchorIndex >= 0 {
		return lastAnchorIndex + 1, baseIndent
	}
	if baseIndex >= 0 {
		return baseIndex + 1, baseIndent
	}
	return -1, ""
}

func effectiveBuildFlagSettings(cfg config.ProjectConfig) []config.SwiftBuildSetting {
	return cfg.EffectiveSwiftSettings().XcodeBuildSettings()
}
