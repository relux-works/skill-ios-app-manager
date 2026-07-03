package scaffold

import (
	"fmt"
	"os"
	"strings"
)

type infoPlistDictionaryRenderer func(indent string) []string

func syncProjectManifestInfoPlistDictionary(
	path string,
	key string,
	enabled bool,
	render infoPlistDictionaryRenderer,
) (bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read Project.swift: %w", err)
	}

	updated, changed, err := syncProjectManifestInfoPlistDictionaryContent(
		string(payload),
		key,
		enabled,
		render,
	)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write Project.swift: %w", err)
	}

	return true, nil
}

func syncProjectManifestInfoPlistDictionaryContent(
	content string,
	key string,
	enabled bool,
	render infoPlistDictionaryRenderer,
) (string, bool, error) {
	if strings.TrimSpace(key) == "" {
		return "", false, fmt.Errorf("Info.plist dictionary key is required")
	}
	if enabled && render == nil {
		return "", false, fmt.Errorf("Info.plist dictionary renderer is required")
	}

	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	filtered, err := removeProjectManifestInfoPlistDictionaryEntries(lines, key)
	if err != nil {
		return "", false, err
	}

	if !enabled {
		updated := joinSyncLines(filtered, hasTrailingNewline)
		return updated, updated != content, nil
	}

	targets := findProjectTargetBlocks(filtered)
	if len(targets) == 0 {
		return "", false, fmt.Errorf("Project.swift target declarations not found")
	}

	updatedLines := filtered
	for index := len(targets) - 1; index >= 0; index-- {
		target := targets[index]
		targetLines := append([]string(nil), updatedLines[target.start:target.end+1]...)
		nextTargetLines, _, err := syncTargetInfoPlistDictionaryLines(targetLines, render)
		if err != nil {
			return "", false, err
		}

		nextLines := make([]string, 0, len(updatedLines)-len(targetLines)+len(nextTargetLines))
		nextLines = append(nextLines, updatedLines[:target.start]...)
		nextLines = append(nextLines, nextTargetLines...)
		nextLines = append(nextLines, updatedLines[target.end+1:]...)
		updatedLines = nextLines
	}

	updated := joinSyncLines(updatedLines, hasTrailingNewline)
	return updated, updated != content, nil
}

func removeProjectManifestInfoPlistDictionaryEntries(lines []string, key string) ([]string, error) {
	filtered := make([]string, 0, len(lines))
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		if isInfoPlistDictionaryLine(line, key) {
			closeLine, ok := findArrayCloseLine(lines, index)
			if !ok {
				return nil, fmt.Errorf("%s Info.plist dictionary opened on line %d has no closing bracket", key, index+1)
			}
			index = closeLine
			continue
		}
		filtered = append(filtered, line)
	}

	return filtered, nil
}

func isInfoPlistDictionaryLine(line string, key string) bool {
	return strings.Contains(line, fmt.Sprintf("%q:", key)) &&
		strings.Contains(line, ".dictionary(")
}

func syncTargetInfoPlistDictionaryLines(
	lines []string,
	render infoPlistDictionaryRenderer,
) ([]string, bool, error) {
	infoPlistLine := findLineContaining(lines, "infoPlist:")
	if infoPlistLine >= 0 {
		return syncExistingInfoPlistDictionaryLines(lines, infoPlistLine, render)
	}

	return insertInfoPlistDictionaryLines(lines, render)
}

func syncExistingInfoPlistDictionaryLines(
	lines []string,
	infoPlistLine int,
	render infoPlistDictionaryRenderer,
) ([]string, bool, error) {
	withLine := -1
	for index := infoPlistLine; index < len(lines); index++ {
		if strings.Contains(lines[index], "with:") && strings.Contains(lines[index], "[") {
			withLine = index
			break
		}
		if strings.Contains(lines[index], "sources:") || strings.Contains(lines[index], "dependencies:") {
			break
		}
	}
	if withLine < 0 {
		if targetInfoPlistUsesDefault(lines, infoPlistLine) {
			return lines, false, nil
		}
		return nil, false, fmt.Errorf("infoPlist declaration does not use .extendingDefault(with: [...])")
	}

	closeLine, ok := findArrayCloseLine(lines, withLine)
	if !ok {
		return nil, false, fmt.Errorf("Info.plist dictionary opened on line %d has no closing bracket", withLine+1)
	}

	insertIndex := closeLine
	insertIndent := leadingIndent(lines[closeLine]) + "    "
	for index := withLine + 1; index < closeLine; index++ {
		if strings.Contains(lines[index], `"AppGroups":`) ||
			strings.Contains(lines[index], `"UILaunchScreen":`) {
			insertIndex = index
			insertIndent = leadingIndent(lines[index])
			break
		}
	}

	rendered := render(insertIndent)
	updated := make([]string, 0, len(lines)+len(rendered))
	updated = append(updated, lines[:insertIndex]...)
	updated = append(updated, rendered...)
	updated = append(updated, lines[insertIndex:]...)

	return updated, true, nil
}

func targetInfoPlistUsesDefault(lines []string, infoPlistLine int) bool {
	for index := infoPlistLine; index < len(lines); index++ {
		line := lines[index]
		if strings.Contains(line, ".default") {
			return true
		}
		if strings.Contains(line, "sources:") || strings.Contains(line, "dependencies:") {
			break
		}
	}

	return false
}

func insertInfoPlistDictionaryLines(
	lines []string,
	render infoPlistDictionaryRenderer,
) ([]string, bool, error) {
	insertIndex := findFirstLineContainingAny(lines, []string{
		"sources:",
		"resources:",
		"entitlements:",
		"dependencies:",
		"settings:",
	})
	insertIndent := ""
	if insertIndex >= 0 {
		insertIndent = leadingIndent(lines[insertIndex])
	} else {
		insertIndex = len(lines) - 1
		if insertIndex < 0 {
			return nil, false, fmt.Errorf("empty target declaration")
		}
		insertIndent = leadingIndent(lines[insertIndex]) + "    "
	}

	rendered := []string{
		insertIndent + "infoPlist: .extendingDefault(",
		insertIndent + "    with: [",
	}
	rendered = append(rendered, render(insertIndent+"        ")...)
	rendered = append(rendered,
		insertIndent+"    ]",
		insertIndent+"),",
	)

	updated := make([]string, 0, len(lines)+len(rendered))
	updated = append(updated, lines[:insertIndex]...)
	updated = append(updated, rendered...)
	updated = append(updated, lines[insertIndex:]...)

	return updated, true, nil
}
