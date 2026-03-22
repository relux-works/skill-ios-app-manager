package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// SyncPackageStrictness updates module Package.swift manifests to use
// config-driven Swift strictness settings and keeps the root Package.swift
// limited to the swift-tools-version line.
func SyncPackageStrictness(projectRoot string, cfg config.ProjectConfig) (ManifestSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return ManifestSyncResult{}, fmt.Errorf("project root is required")
	}

	rootPackagePath := filepath.Join(root, "Package.swift")
	manifestPaths, err := discoverPackageManifestPaths(root, cfg.ModulesPath)
	if err != nil {
		return ManifestSyncResult{}, err
	}

	if _, err := os.Stat(rootPackagePath); err == nil {
		manifestPaths = append([]string{rootPackagePath}, manifestPaths...)
	} else if !os.IsNotExist(err) {
		return ManifestSyncResult{}, fmt.Errorf("stat root package manifest %q: %w", rootPackagePath, err)
	}

	if len(manifestPaths) == 0 {
		return ManifestSyncResult{}, fmt.Errorf("no Package.swift manifests found in %q; run init first", root)
	}

	effectiveSwift := cfg.EffectiveSwiftSettings()
	result := ManifestSyncResult{
		Scanned: append([]string(nil), manifestPaths...),
		Updated: make([]string, 0, len(manifestPaths)),
	}

	for _, manifestPath := range manifestPaths {
		payload, err := os.ReadFile(manifestPath)
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("read manifest %q: %w", manifestPath, err)
		}

		var (
			updated string
			changed bool
		)
		if manifestPath == rootPackagePath {
			updated, changed, err = syncRootPackageStrictnessManifest(string(payload), effectiveSwift)
		} else {
			updated, changed, err = syncModulePackageStrictnessManifest(string(payload), effectiveSwift)
		}
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("sync package strictness in %q: %w", manifestPath, err)
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

func discoverPackageManifestPaths(projectRoot, modulesPath string) ([]string, error) {
	path := strings.TrimSpace(modulesPath)
	if path == "" {
		path = "Packages"
	}

	scanPath := path
	if !filepath.IsAbs(scanPath) {
		scanPath = filepath.Join(projectRoot, scanPath)
	}

	entries, err := os.ReadDir(scanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read modules directory %q: %w", scanPath, err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(scanPath, entry.Name(), "Package.swift")
		if _, err := os.Stat(manifestPath); err == nil {
			paths = append(paths, manifestPath)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat package manifest %q: %w", manifestPath, err)
		}
	}

	sort.Strings(paths)
	return paths, nil
}

func syncRootPackageStrictnessManifest(content string, swift config.EffectiveSwiftSettings) (string, bool, error) {
	updated, changed, err := ensureSwiftToolsVersionLine(content, swift.ToolsVersion)
	if err != nil {
		return "", false, err
	}

	next, stripped, err := removeGeneratedRootPackageStrictnessBlock(updated)
	if err != nil {
		return "", false, err
	}

	return next, changed || stripped, nil
}

func syncModulePackageStrictnessManifest(content string, swift config.EffectiveSwiftSettings) (string, bool, error) {
	currentToolsVersion, err := parseSwiftToolsVersion(content)
	if err != nil {
		return "", false, err
	}
	if !strings.HasPrefix(currentToolsVersion, "6.") {
		return content, false, nil
	}

	updated, changed, err := ensureSwiftToolsVersionLine(content, swift.ToolsVersion)
	if err != nil {
		return "", false, err
	}

	next, settingsChanged, err := ensureModulePackageSwiftSettings(updated, swift.PackageSwiftSettings())
	if err != nil {
		return "", false, err
	}

	return next, changed || settingsChanged, nil
}

func ensureSwiftToolsVersionLine(content, toolsVersion string) (string, bool, error) {
	current, err := parseSwiftToolsVersion(content)
	if err != nil {
		return "", false, err
	}

	replacement := "// swift-tools-version: " + toolsVersion
	if strings.HasPrefix(content, replacement) {
		return content, false, nil
	}

	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return "", false, fmt.Errorf("manifest is empty")
	}

	lines[0] = strings.Replace(lines[0], current, toolsVersion, 1)
	return strings.Join(lines, "\n"), true, nil
}

func removeGeneratedRootPackageStrictnessBlock(content string) (string, bool, error) {
	const strictSettingsAnchor = "let strictPackageBaseSettings: SettingsDictionary = ["

	strictIndex := strings.Index(content, strictSettingsAnchor)
	if strictIndex < 0 {
		return content, false, nil
	}

	start := strings.LastIndex(content[:strictIndex], "#if TUIST")
	if start < 0 {
		return "", false, fmt.Errorf("generated root strictness block start not found")
	}

	endRelative := strings.Index(content[strictIndex:], "#endif")
	if endRelative < 0 {
		return "", false, fmt.Errorf("generated root strictness block end not found")
	}
	end := strictIndex + endRelative + len("#endif")

	prefix := strings.TrimRight(content[:start], "\n")
	suffix := strings.TrimLeft(content[end:], "\n")

	switch {
	case prefix == "" && suffix == "":
		return "", true, nil
	case prefix == "":
		return suffix, true, nil
	case suffix == "":
		return prefix + "\n", true, nil
	default:
		return prefix + "\n\n" + suffix, true, nil
	}
}

func ensureRootPackageSettingsBlock(content string, settings []config.SwiftBuildSetting) (string, bool, error) {
	const importAnchor = "import PackageDescription\n"

	anchorIndex := strings.Index(content, importAnchor)
	if anchorIndex < 0 {
		return "", false, fmt.Errorf("PackageDescription import not found")
	}

	block := renderRootPackageSettingsBlock(settings)
	insertIndex := anchorIndex + len(importAnchor)
	suffix := content[insertIndex:]

	if start := strings.Index(suffix, "#if TUIST"); start >= 0 {
		endRel := strings.Index(suffix[start:], "#endif")
		if endRel < 0 {
			return "", false, fmt.Errorf("unterminated #if TUIST block")
		}
		end := start + endRel + len("#endif")
		suffix = suffix[:start] + suffix[end:]
	}

	suffix = strings.TrimLeft(suffix, "\n")
	updated := content[:insertIndex] + block + "\n\n" + suffix
	if updated == content {
		return content, false, nil
	}

	return updated, true, nil
}

func renderRootPackageSettingsBlock(settings []config.SwiftBuildSetting) string {
	lines := []string{
		"#if TUIST",
		"import ProjectDescription",
		"",
		"let strictPackageBaseSettings: SettingsDictionary = [",
	}
	for _, setting := range settings {
		lines = append(lines, fmt.Sprintf("    %q: %q,", setting.Key, setting.Value))
	}
	lines = append(lines,
		"]",
		"",
		"let packageSettings = PackageSettings(",
		"    baseSettings: .settings(base: strictPackageBaseSettings)",
		")",
		"#endif",
	)
	return strings.Join(lines, "\n")
}

func ensureModulePackageSwiftSettings(content string, settings []string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	changed := false
	sawTarget := false

	for index := 0; index < len(lines); index++ {
		if !isModulePackageTargetLine(lines[index]) {
			continue
		}

		sawTarget = true

		endIndex, err := findDelimitedBlockEnd(lines, index, "(", ")")
		if err != nil {
			return "", false, err
		}

		nextLines, targetChanged, err := ensureModuleTargetSwiftSettings(lines, index, endIndex, settings)
		if err != nil {
			return "", false, err
		}
		if targetChanged {
			changed = true
			endIndex += len(nextLines) - len(lines)
			lines = nextLines
		}

		index = endIndex
	}
	if !sawTarget {
		return "", false, fmt.Errorf("no .target(...) blocks found")
	}

	return joinSyncLines(lines, hasTrailingNewline), changed, nil
}

func ensureModuleTargetSwiftSettings(lines []string, startIndex, endIndex int, settings []string) ([]string, bool, error) {
	originalLines := lines
	originalEndIndex := endIndex
	propertyIndent := leadingIndent(lines[startIndex]) + "    "
	swiftSettingsStart := -1
	swiftSettingsEnd := -1
	insertIndex := endIndex

	for index := startIndex + 1; index < endIndex; index++ {
		trimmed := strings.TrimSpace(lines[index])
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, ":") {
			propertyIndent = leadingIndent(lines[index])
		}
		if strings.HasPrefix(trimmed, "swiftSettings: [") {
			blockEnd, err := findDelimitedBlockEnd(lines, index, "[", "]")
			if err != nil {
				return nil, false, fmt.Errorf("unterminated swiftSettings block")
			}
			swiftSettingsStart = index
			swiftSettingsEnd = blockEnd
			propertyIndent = leadingIndent(lines[index])
			index = blockEnd
			continue
		}
		if isPostSwiftSettingsTargetProperty(trimmed) && insertIndex == endIndex {
			insertIndex = index
		}
	}

	lines = append([]string{}, lines...)
	if swiftSettingsStart >= 0 {
		lines = append(lines[:swiftSettingsStart], lines[swiftSettingsEnd+1:]...)
		endIndex -= swiftSettingsEnd - swiftSettingsStart + 1
		insertIndex = adjustInsertionIndexAfterRemoval(insertIndex, swiftSettingsStart, swiftSettingsEnd)
	}

	if insertIndex > endIndex {
		insertIndex = endIndex
	}

	previousPropertyIndex := insertIndex - 1
	for previousPropertyIndex > startIndex && strings.TrimSpace(lines[previousPropertyIndex]) == "" {
		previousPropertyIndex--
	}
	if previousPropertyIndex <= startIndex {
		previousPropertyIndex = findLastTargetPropertyLine(lines, startIndex, endIndex)
		if previousPropertyIndex < 0 {
			return nil, false, fmt.Errorf("target property insertion anchor not found")
		}
		insertIndex = endIndex
	}

	if trimmed := strings.TrimSpace(lines[previousPropertyIndex]); trimmed != "" && !strings.HasSuffix(trimmed, ",") {
		lines[previousPropertyIndex] = lines[previousPropertyIndex] + ","
	}

	trailingComma := hasNonEmptyLines(lines, insertIndex, endIndex)
	blockLines := renderModuleSwiftSettingsLines(propertyIndent, settings, trailingComma)
	lines = insertSyncLines(lines, insertIndex, blockLines)

	updatedEndIndex := originalEndIndex + len(lines) - len(originalLines)
	currentBlock := lines[startIndex : updatedEndIndex+1]
	originalBlock := originalLines[startIndex : originalEndIndex+1]
	if sameStringSlice(currentBlock, originalBlock) {
		return originalLines, false, nil
	}

	return lines, true, nil
}

func renderModuleSwiftSettingsLines(indent string, settings []string, trailingComma bool) []string {
	lines := make([]string, 0, len(settings)+2)
	lines = append(lines, indent+`swiftSettings: [`)
	for _, setting := range settings {
		lines = append(lines, indent+"    "+setting+",")
	}
	closing := indent + `]`
	if trailingComma {
		closing += ","
	}
	lines = append(lines, closing)
	return lines
}

func parseSwiftToolsVersion(content string) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("manifest is empty")
	}

	const prefix = "// swift-tools-version: "
	if !strings.HasPrefix(lines[0], prefix) {
		return "", fmt.Errorf("swift tools version header not found")
	}

	value := strings.TrimSpace(strings.TrimPrefix(lines[0], prefix))
	if value == "" {
		return "", fmt.Errorf("swift tools version header is empty")
	}

	return value, nil
}

func isModulePackageTargetLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), ".target(")
}

func findDelimitedBlockEnd(lines []string, startIndex int, open, close string) (int, error) {
	depth := 0
	for index := startIndex; index < len(lines); index++ {
		line := lines[index]
		depth += strings.Count(line, open)
		depth -= strings.Count(line, close)
		if depth == 0 {
			return index, nil
		}
	}
	return -1, fmt.Errorf("unterminated block starting at line %d", startIndex+1)
}

func hasNonEmptyLines(lines []string, startIndex, endIndex int) bool {
	for index := startIndex; index < endIndex; index++ {
		if strings.TrimSpace(lines[index]) != "" {
			return true
		}
	}
	return false
}

func findLastTargetPropertyLine(lines []string, startIndex, endIndex int) int {
	for index := endIndex - 1; index > startIndex; index-- {
		trimmed := strings.TrimSpace(lines[index])
		if trimmed == "" {
			continue
		}
		return index
	}
	return -1
}

func isPostSwiftSettingsTargetProperty(trimmed string) bool {
	return strings.HasPrefix(trimmed, "linkerSettings:") || strings.HasPrefix(trimmed, "plugins:")
}

func adjustInsertionIndexAfterRemoval(insertIndex, removedStart, removedEnd int) int {
	if insertIndex <= removedStart {
		return insertIndex
	}
	if insertIndex > removedEnd {
		return insertIndex - (removedEnd - removedStart + 1)
	}
	return removedStart
}

func sameStringSlice(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func insertSyncLines(lines []string, index int, inserts []string) []string {
	result := lines
	for offset, line := range inserts {
		result = insertSyncLine(result, index+offset, line)
	}
	return result
}
