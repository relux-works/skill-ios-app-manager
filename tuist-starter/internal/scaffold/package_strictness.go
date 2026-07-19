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
	rootPackageTargets, err := discoverRootPackageSwiftTargetNames(root, cfg.ModulesPath)
	if err != nil {
		return ManifestSyncResult{}, err
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

		var (
			updated string
			changed bool
		)
		if manifestPath == rootPackagePath {
			updated, changed, err = syncRootPackageStrictnessManifest(string(payload), cfg, effectiveSwift, rootPackageTargets)
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

func discoverRootPackageSwiftTargetNames(projectRoot, modulesPath string) ([]string, error) {
	manifestPaths, err := discoverLocalPackageManifestGraph(projectRoot, modulesPath)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	targets := make([]string, 0)
	for _, manifestPath := range manifestPaths {
		payload, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("read local package manifest %q: %w", manifestPath, err)
		}

		toolsVersion, err := parseSwiftToolsVersion(string(payload))
		if err != nil {
			return nil, fmt.Errorf("parse swift tools version in %q: %w", manifestPath, err)
		}
		if !strings.HasPrefix(toolsVersion, "6.") {
			continue
		}

		for _, targetName := range parseRegularPackageTargetNames(string(payload)) {
			if _, exists := seen[targetName]; exists {
				continue
			}
			seen[targetName] = struct{}{}
			targets = append(targets, targetName)
		}
	}

	sort.Strings(targets)
	return targets, nil
}

func discoverLocalPackageManifestGraph(projectRoot, modulesPath string) ([]string, error) {
	rootPackagePath := filepath.Join(projectRoot, "Package.swift")
	queue := make([]string, 0)
	visited := make(map[string]struct{})

	moduleManifests, err := discoverPackageManifestPaths(projectRoot, modulesPath)
	if err != nil {
		return nil, err
	}
	queue = append(queue, moduleManifests...)

	if _, err := os.Stat(rootPackagePath); err == nil {
		queue = append(queue, rootPackagePath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat root package manifest %q: %w", rootPackagePath, err)
	}

	result := make([]string, 0, len(queue))
	for len(queue) > 0 {
		manifestPath := filepath.Clean(queue[0])
		queue = queue[1:]
		if _, exists := visited[manifestPath]; exists {
			continue
		}
		visited[manifestPath] = struct{}{}

		payload, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("read package manifest %q: %w", manifestPath, err)
		}

		if manifestPath != rootPackagePath {
			result = append(result, manifestPath)
		}

		for _, dependencyPath := range parseLocalPackageDependencyPaths(string(payload)) {
			dependencyManifest := filepath.Join(filepath.Dir(manifestPath), dependencyPath, "Package.swift")
			dependencyManifest = filepath.Clean(dependencyManifest)
			if _, exists := visited[dependencyManifest]; exists {
				continue
			}
			if _, err := os.Stat(dependencyManifest); err == nil {
				queue = append(queue, dependencyManifest)
			} else if !os.IsNotExist(err) {
				return nil, fmt.Errorf("stat local package dependency %q: %w", dependencyManifest, err)
			}
		}
	}

	sort.Strings(result)
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

func syncRootPackageStrictnessManifest(content string, cfg config.ProjectConfig, swift config.EffectiveSwiftSettings, packageTargetNames []string) (string, bool, error) {
	updated, changed, err := ensureSwiftToolsVersionLine(content, swift.ToolsVersion)
	if err != nil {
		return "", false, err
	}

	next, stripped, err := removeGeneratedRootPackageStrictnessBlock(updated)
	if err != nil {
		return "", false, err
	}

	next, targetSettingsChanged, err := ensureRootPackageTargetSettings(next, packageTargetNames, swift.XcodeBuildSettings())
	if err != nil {
		return "", false, err
	}

	runtimeUpdated, err := syncRuntimeProfilePackageManifestContent(next, cfg, cfg.HasRuntimeProfiles())
	if err != nil {
		return "", false, err
	}

	return runtimeUpdated, changed || stripped || targetSettingsChanged || runtimeUpdated != next, nil
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

	if !strings.Contains(content, strictSettingsAnchor) {
		return content, false, nil
	}

	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	strictStart := findLineContaining(lines, strictSettingsAnchor)
	strictEnd, err := findDelimitedBlockEnd(lines, strictStart, "[", "]")
	if err != nil {
		return "", false, fmt.Errorf("generated root strictness settings are unterminated")
	}
	lines = append(append([]string(nil), lines[:strictStart]...), lines[strictEnd+1:]...)

	baseSettingsLine := -1
	for index, line := range lines {
		if strings.Contains(line, "baseSettings:") && strings.Contains(line, "strictPackageBaseSettings") {
			baseSettingsLine = index
			break
		}
	}
	if baseSettingsLine >= 0 {
		baseSettingsEnd := baseSettingsLine
		if strings.Count(lines[baseSettingsLine], "(") > strings.Count(lines[baseSettingsLine], ")") {
			baseSettingsEnd, err = findDelimitedBlockEnd(lines, baseSettingsLine, "(", ")")
			if err != nil {
				return "", false, fmt.Errorf("generated root baseSettings are unterminated")
			}
		}
		lines = append(append([]string(nil), lines[:baseSettingsLine]...), lines[baseSettingsEnd+1:]...)
	}

	return joinSyncLines(lines, hasTrailingNewline), true, nil
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

func ensureRootPackageTargetSettings(content string, targetNames []string, settings []config.SwiftBuildSetting) (string, bool, error) {
	normalizedTargets := normalizeSwiftPackageTargetNames(targetNames)
	if len(normalizedTargets) == 0 || len(settings) == 0 {
		return content, false, nil
	}

	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	changed := false

	tuistStart, tuistEnd := findTuistBlockLineRange(lines)
	if tuistStart < 0 {
		blockLines := renderRootPackageTargetSettingsTuistBlock(normalizedTargets, settings)
		if strings.TrimSpace(content) == "" {
			return joinSyncLines(blockLines, true), true, nil
		}
		lines = append(lines, blockLines...)
		return joinSyncLines(lines, true), true, nil
	}
	if tuistEnd < 0 {
		return "", false, fmt.Errorf("unterminated #if TUIST block")
	}

	if !tuistBlockHasProjectDescriptionImport(lines, tuistStart, tuistEnd) {
		insertIndex := tuistStart + 1
		lines = insertSyncLine(lines, insertIndex, "import ProjectDescription")
		changed = true
		tuistEnd++
	}

	packageSettingsIndex := findLineContainingInRange(lines, tuistStart, tuistEnd, "let packageSettings = PackageSettings(")
	if packageSettingsIndex < 0 {
		insertIndex := tuistEnd
		blockLines := renderRootPackageSettingsDeclaration(normalizedTargets, settings)
		lines = insertSyncLines(lines, insertIndex, blockLines)
		changed = true
		return joinSyncLines(lines, hasTrailingNewline), true, nil
	}

	nextLines, helperChanged, err := ensureRootPackageTargetSettingsHelper(lines, tuistStart, packageSettingsIndex, settings)
	if err != nil {
		return "", false, err
	}
	if helperChanged {
		tuistEnd += len(nextLines) - len(lines)
		packageSettingsIndex += len(nextLines) - len(lines)
		lines = nextLines
		changed = true
	}

	packageSettingsEnd, err := findDelimitedBlockEnd(lines, packageSettingsIndex, "(", ")")
	if err != nil {
		return "", false, fmt.Errorf("unterminated PackageSettings block")
	}

	nextLines, targetSettingsChanged, err := ensureRootPackageSettingsTargetEntries(lines, packageSettingsIndex, packageSettingsEnd, normalizedTargets)
	if err != nil {
		return "", false, err
	}
	if targetSettingsChanged {
		lines = nextLines
		changed = true
	}

	return joinSyncLines(lines, hasTrailingNewline), changed, nil
}

func findTuistBlockLineRange(lines []string) (int, int) {
	for index, line := range lines {
		if strings.TrimSpace(line) != "#if TUIST" {
			continue
		}
		for end := index + 1; end < len(lines); end++ {
			if strings.TrimSpace(lines[end]) == "#endif" {
				return index, end
			}
		}
		return index, -1
	}
	return -1, -1
}

func tuistBlockHasProjectDescriptionImport(lines []string, startIndex, endIndex int) bool {
	for index := startIndex + 1; index < endIndex; index++ {
		if strings.TrimSpace(lines[index]) == "import ProjectDescription" {
			return true
		}
	}
	return false
}

func findLineContainingInRange(lines []string, startIndex, endIndex int, value string) int {
	for index := startIndex; index <= endIndex && index < len(lines); index++ {
		if strings.Contains(lines[index], value) {
			return index
		}
	}
	return -1
}

func ensureRootPackageTargetSettingsHelper(lines []string, tuistStart, packageSettingsIndex int, settings []config.SwiftBuildSetting) ([]string, bool, error) {
	helperStart := findLineContainingInRange(lines, tuistStart, packageSettingsIndex, "let swiftPackageTargetSettings: Settings = .settings(base: [")
	helperLines := renderRootPackageTargetSettingsHelper(settings)
	if helperStart < 0 {
		insertIndex := packageSettingsIndex
		if insertIndex > tuistStart+1 && strings.TrimSpace(lines[insertIndex-1]) == "" {
			insertIndex--
		}
		withHelper := insertSyncLines(lines, insertIndex, append(helperLines, ""))
		return withHelper, true, nil
	}

	helperEnd, err := findDelimitedBlockEnd(lines, helperStart, "[", "]")
	if err != nil {
		return nil, false, fmt.Errorf("unterminated swiftPackageTargetSettings block")
	}
	if helperEnd+1 < len(lines) && strings.TrimSpace(lines[helperEnd+1]) == ")" {
		helperEnd++
	}

	updated := append([]string{}, lines...)
	updated = append(updated[:helperStart], append(helperLines, updated[helperEnd+1:]...)...)

	if sameStringSlice(updated, lines) {
		return lines, false, nil
	}
	return updated, true, nil
}

func ensureRootPackageSettingsTargetEntries(lines []string, packageSettingsIndex, packageSettingsEnd int, targetNames []string) ([]string, bool, error) {
	targetSettingsIndex := -1
	for index := packageSettingsIndex + 1; index < packageSettingsEnd; index++ {
		if strings.Contains(lines[index], "targetSettings: [") {
			targetSettingsIndex = index
			break
		}
	}

	entries := renderRootPackageTargetSettingsEntries(targetNames)
	if targetSettingsIndex < 0 {
		insertIndex := packageSettingsEnd
		if projectOptionsIndex := findLineContainingInRange(lines, packageSettingsIndex+1, packageSettingsEnd, "projectOptions:"); projectOptionsIndex >= 0 {
			insertIndex = projectOptionsIndex
		}
		blockLines := append([]string{"    targetSettings: ["}, entries...)
		blockLines = append(blockLines, "    ],")
		updated := insertSyncLines(lines, insertIndex, blockLines)
		return updated, true, nil
	}

	targetSettingsEnd, err := findDelimitedBlockEnd(lines, targetSettingsIndex, "[", "]")
	if err != nil {
		return nil, false, fmt.Errorf("unterminated targetSettings block")
	}

	updated := append([]string{}, lines...)
	for index := targetSettingsEnd - 1; index > targetSettingsIndex; index-- {
		targetName, ok := parseTargetSettingsEntryName(updated[index])
		if !ok || !containsString(targetNames, targetName) {
			continue
		}

		entryEnd := index
		if strings.Contains(updated[index], "[") {
			var err error
			entryEnd, err = findDelimitedBlockEnd(updated, index, "[", "]")
			if err != nil {
				return nil, false, fmt.Errorf("unterminated targetSettings entry for %s", targetName)
			}
			if entryEnd+1 < len(updated) && strings.TrimSpace(updated[entryEnd+1]) == ")," {
				entryEnd++
			}
		}

		updated = append(updated[:index], updated[entryEnd+1:]...)
		targetSettingsEnd -= entryEnd - index + 1
	}

	insertIndex := targetSettingsEnd
	updated = insertSyncLines(updated, insertIndex, entries)
	if sameStringSlice(updated, lines) {
		return lines, false, nil
	}
	return updated, true, nil
}

func renderRootPackageTargetSettingsTuistBlock(targetNames []string, settings []config.SwiftBuildSetting) []string {
	lines := []string{
		"",
		"#if TUIST",
		"import ProjectDescription",
		"",
	}
	lines = append(lines, renderRootPackageSettingsDeclaration(targetNames, settings)...)
	lines = append(lines, "#endif")
	return lines
}

func renderRootPackageSettingsDeclaration(targetNames []string, settings []config.SwiftBuildSetting) []string {
	lines := renderRootPackageTargetSettingsHelper(settings)
	lines = append(lines,
		"",
		"let packageSettings = PackageSettings(",
		"    targetSettings: [",
	)
	lines = append(lines, renderRootPackageTargetSettingsEntries(targetNames)...)
	lines = append(lines,
		"    ]",
		")",
	)
	return lines
}

func renderRootPackageTargetSettingsHelper(settings []config.SwiftBuildSetting) []string {
	lines := []string{
		"let swiftPackageTargetSettings: Settings = .settings(base: [",
	}
	for _, setting := range settings {
		lines = append(lines, fmt.Sprintf("    %q: %q,", setting.Key, setting.Value))
	}
	lines = append(lines, "])")
	return lines
}

func renderRootPackageTargetSettingsEntries(targetNames []string) []string {
	lines := make([]string, 0, len(targetNames))
	for _, targetName := range targetNames {
		lines = append(lines, fmt.Sprintf("        %q: swiftPackageTargetSettings,", targetName))
	}
	return lines
}

func parseTargetSettingsEntryName(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, `"`) {
		return "", false
	}
	end := strings.Index(trimmed[1:], `"`)
	if end < 0 {
		return "", false
	}
	name := trimmed[1 : 1+end]
	afterName := strings.TrimSpace(trimmed[1+end+1:])
	if !strings.HasPrefix(afterName, ":") {
		return "", false
	}
	return name, true
}

func normalizeSwiftPackageTargetNames(targetNames []string) []string {
	seen := make(map[string]struct{}, len(targetNames))
	normalized := make([]string, 0, len(targetNames))
	for _, targetName := range targetNames {
		trimmed := strings.TrimSpace(targetName)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func parseLocalPackageDependencyPaths(content string) []string {
	lines := strings.Split(content, "\n")
	paths := make([]string, 0)
	seen := map[string]struct{}{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || !strings.Contains(trimmed, ".package(") || !strings.Contains(trimmed, "path:") {
			continue
		}
		pathIndex := strings.Index(trimmed, "path:")
		quoteStart := strings.Index(trimmed[pathIndex:], `"`)
		if quoteStart < 0 {
			continue
		}
		quoteStart += pathIndex + 1
		quoteEnd := strings.Index(trimmed[quoteStart:], `"`)
		if quoteEnd < 0 {
			continue
		}
		value := strings.TrimSpace(trimmed[quoteStart : quoteStart+quoteEnd])
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		paths = append(paths, value)
	}
	return paths
}

func parseRegularPackageTargetNames(content string) []string {
	lines := strings.Split(content, "\n")
	names := make([]string, 0)
	seen := map[string]struct{}{}
	for index := 0; index < len(lines); index++ {
		if !isModulePackageTargetLine(lines[index]) {
			continue
		}
		endIndex, err := findDelimitedBlockEnd(lines, index, "(", ")")
		if err != nil {
			continue
		}
		block := strings.Join(lines[index:endIndex+1], "\n")
		name, ok := parsePackageTargetName(block)
		if ok {
			if _, exists := seen[name]; !exists {
				seen[name] = struct{}{}
				names = append(names, name)
			}
		}
		index = endIndex
	}
	sort.Strings(names)
	return names
}

func parsePackageTargetName(block string) (string, bool) {
	nameIndex := strings.Index(block, "name:")
	if nameIndex < 0 {
		return "", false
	}
	quoteStart := strings.Index(block[nameIndex:], `"`)
	if quoteStart < 0 {
		return "", false
	}
	quoteStart += nameIndex + 1
	quoteEnd := strings.Index(block[quoteStart:], `"`)
	if quoteEnd < 0 {
		return "", false
	}
	name := strings.TrimSpace(block[quoteStart : quoteStart+quoteEnd])
	return name, name != ""
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
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
	swiftSettingsRanges := make([]lineRange, 0, 1)
	insertIndex := endIndex

	for index := startIndex + 1; index < endIndex; index++ {
		trimmed := strings.TrimSpace(lines[index])
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, ":") {
			propertyIndent = leadingIndent(lines[index])
		}
		if strings.HasPrefix(trimmed, "swiftSettings:") {
			blockEnd := index
			if strings.Contains(trimmed, "[") {
				var err error
				blockEnd, err = findDelimitedBlockEnd(lines, index, "[", "]")
				if err != nil {
					return nil, false, fmt.Errorf("unterminated swiftSettings block")
				}
			}
			swiftSettingsRanges = append(swiftSettingsRanges, lineRange{start: index, end: blockEnd})
			propertyIndent = leadingIndent(lines[index])
			index = blockEnd
			continue
		}
		if isPostSwiftSettingsTargetProperty(trimmed) && insertIndex == endIndex {
			insertIndex = index
		}
	}

	lines = append([]string{}, lines...)
	for index := len(swiftSettingsRanges) - 1; index >= 0; index-- {
		removal := swiftSettingsRanges[index]
		lines = append(lines[:removal.start], lines[removal.end+1:]...)
		endIndex -= removal.end - removal.start + 1
		insertIndex = adjustInsertionIndexAfterRemoval(insertIndex, removal.start, removal.end)
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

type lineRange struct {
	start int
	end   int
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
