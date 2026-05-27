package scaffold

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const exportComplianceUsesNonExemptEncryptionInfoPlistKey = "ITSAppUsesNonExemptEncryption"

type ExportComplianceConfigSyncResult struct {
	Scanned []string
	Updated []string
}

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "export-compliance-config",
		Short:        "Sync host app export compliance Info.plist configuration",
		Dependencies: []string{"init"},
		Run:          runGenerateExportComplianceConfig,
	})
}

func runGenerateExportComplianceConfig(input GenerateInput) (GenerateResult, error) {
	result, err := SyncExportComplianceConfig(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	if len(result.Updated) > 0 {
		return GenerateResult{
			Message: fmt.Sprintf("regenerated export compliance config in %d file(s)\n", len(result.Updated)),
		}, nil
	}

	return GenerateResult{
		Message: "export compliance config already up to date\n",
	}, nil
}

func SyncExportComplianceConfig(projectRoot string, cfg config.ProjectConfig) (ExportComplianceConfigSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return ExportComplianceConfigSyncResult{}, fmt.Errorf("project root is required")
	}

	result := ExportComplianceConfigSyncResult{
		Scanned: make([]string, 0, 4),
		Updated: make([]string, 0, 4),
	}

	projectManifestPaths, err := discoverScaffoldManifestPaths(root)
	if err != nil {
		return result, err
	}

	for _, manifestPath := range projectManifestPaths {
		result.Scanned = appendUniqueStrings(result.Scanned, manifestPath)
		updated, err := syncProjectManifestExportComplianceConfig(manifestPath, cfg)
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, manifestPath)
		}
	}

	return result, nil
}

func syncProjectManifestExportComplianceConfig(path string, cfg config.ProjectConfig) (bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read Project.swift: %w", err)
	}

	updated, changed, err := syncProjectManifestExportComplianceConfigContent(string(payload), cfg)
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

func syncProjectManifestExportComplianceConfigContent(content string, cfg config.ProjectConfig) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	filtered, err := removeProjectManifestExportComplianceConfigEntries(lines)
	if err != nil {
		return "", false, err
	}

	if cfg.UsesNonExemptEncryption == nil {
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
		if !targetReceivesExportComplianceConfig(targetLines) {
			continue
		}

		nextTargetLines, _, err := syncTargetExportComplianceInfoPlistLines(
			targetLines,
			func(indent string) []string {
				return renderExportComplianceInfoPlistLines(indent, cfg)
			},
		)
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

func removeProjectManifestExportComplianceConfigEntries(lines []string) ([]string, error) {
	keys := map[string]struct{}{
		exportComplianceUsesNonExemptEncryptionInfoPlistKey: {},
	}

	filtered := make([]string, 0, len(lines))
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		if !isInfoPlistRootEntryLine(line, keys) {
			filtered = append(filtered, line)
			continue
		}

		if strings.Contains(line, ".array(") || strings.Contains(line, ".dictionary(") {
			closeLine, ok := findArrayCloseLine(lines, index)
			if !ok {
				return nil, fmt.Errorf("Info.plist entry opened on line %d has no closing bracket", index+1)
			}
			index = closeLine
		}
	}

	return filtered, nil
}

func syncTargetExportComplianceInfoPlistLines(
	lines []string,
	render infoPlistDictionaryRenderer,
) ([]string, bool, error) {
	infoPlistLine := findLineContaining(lines, "infoPlist:")
	if infoPlistLine < 0 {
		return insertInfoPlistDictionaryLines(lines, render)
	}

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
		return nil, false, fmt.Errorf("infoPlist declaration does not use .extendingDefault(with: [...])")
	}

	closeLine, ok := findArrayCloseLine(lines, withLine)
	if !ok {
		return nil, false, fmt.Errorf("Info.plist dictionary opened on line %d has no closing bracket", withLine+1)
	}

	insertIndex := closeLine
	insertIndent := leadingIndent(lines[closeLine]) + "    "
	for index := withLine + 1; index < closeLine; index++ {
		if strings.Contains(lines[index], `"ApplicationConfiguration":`) ||
			strings.Contains(lines[index], `"AppGroups":`) ||
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

func targetReceivesExportComplianceConfig(lines []string) bool {
	for _, line := range lines {
		if projectManifestAppProductPattern.MatchString(line) {
			return true
		}
	}
	return false
}

func renderExportComplianceInfoPlistLines(indent string, cfg config.ProjectConfig) []string {
	if cfg.UsesNonExemptEncryption == nil {
		return nil
	}

	return []string{
		indent + strconv.Quote(exportComplianceUsesNonExemptEncryptionInfoPlistKey) + ": .boolean(" + strconv.FormatBool(*cfg.UsesNonExemptEncryption) + "),",
	}
}
