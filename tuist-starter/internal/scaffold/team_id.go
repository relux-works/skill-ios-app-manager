package scaffold

import (
	"fmt"
	"os"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// SyncTeamID updates scaffolded app, nested app, and extension manifests to use the configured signing team.
func SyncTeamID(projectRoot string, cfg config.ProjectConfig) (ManifestSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return ManifestSyncResult{}, fmt.Errorf("project root is required")
	}

	teamID := strings.TrimSpace(cfg.TeamID)
	if teamID == "" {
		return ManifestSyncResult{}, fmt.Errorf("team id is required")
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

		updated, changed, err := syncTeamIDManifest(string(payload), teamID)
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("sync team id in %q: %w", manifestPath, err)
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

func syncTeamIDManifest(content string, teamID string) (string, bool, error) {
	updated, constantChanged, err := ensureDevelopmentTeamConstant(content, teamID)
	if err != nil {
		return "", false, err
	}

	next, settingsChanged, err := ensureDevelopmentTeamBuildSettings(updated)
	if err != nil {
		return "", false, err
	}

	return next, constantChanged || settingsChanged, nil
}

func ensureDevelopmentTeamConstant(content string, teamID string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "let developmentTeam = ") {
			continue
		}

		replacement := leadingIndent(line) + fmt.Sprintf("let developmentTeam = %q", teamID)
		if replacement == line {
			return content, false, nil
		}
		lines[index] = replacement
		return joinSyncLines(lines, hasTrailingNewline), true, nil
	}

	insertAfter := -1
	insertBeforeProject := -1
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "let bundleID = "):
			insertAfter = index
		case strings.HasPrefix(trimmed, "let hostBundleId = "):
			insertAfter = index
		case insertAfter < 0 && strings.HasPrefix(trimmed, "let appName = "):
			insertAfter = index
		case insertBeforeProject < 0 && strings.HasPrefix(trimmed, "let project = Project("):
			insertBeforeProject = index
		}
	}

	insertLine := fmt.Sprintf("let developmentTeam = %q", teamID)
	switch {
	case insertAfter >= 0:
		lines = insertSyncLine(lines, insertAfter+1, insertLine)
	case insertBeforeProject >= 0:
		if insertBeforeProject > 0 && strings.TrimSpace(lines[insertBeforeProject-1]) == "" {
			insertBeforeProject--
		}
		lines = insertSyncLine(lines, insertBeforeProject, insertLine)
	default:
		return "", false, fmt.Errorf("developmentTeam insertion anchor not found")
	}

	return joinSyncLines(lines, hasTrailingNewline), true, nil
}

func ensureDevelopmentTeamBuildSettings(content string) (string, bool, error) {
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

		nextLines, blockChanged, err := ensureDevelopmentTeamBuildSettingInBlock(lines, index, endIndex)
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
		return "", false, fmt.Errorf("settings base insertion anchor not found")
	}

	return joinSyncLines(lines, hasTrailingNewline), changed, nil
}

func ensureDevelopmentTeamBuildSettingInBlock(lines []string, startIndex, endIndex int) ([]string, bool, error) {
	if endIndex <= startIndex {
		return nil, false, fmt.Errorf("invalid build settings block")
	}

	blockIndent := leadingIndent(lines[startIndex]) + "    "
	replacement := blockIndent + `"DEVELOPMENT_TEAM": .string(developmentTeam),`
	updated := append([]string{}, lines...)

	for index := startIndex + 1; index < endIndex; index++ {
		if !strings.Contains(updated[index], `"DEVELOPMENT_TEAM":`) {
			continue
		}
		if updated[index] == replacement {
			return updated, false, nil
		}
		updated[index] = replacement
		return updated, true, nil
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
		}
	}

	updated = insertSyncLine(updated, insertIndex, replacement)
	return updated, true, nil
}
