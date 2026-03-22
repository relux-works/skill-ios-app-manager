package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// SyncPackageStrictness updates root and module Package.swift manifests to use config-driven Swift strictness settings.
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

	next, blockChanged, err := ensureRootPackageSettingsBlock(updated, swift.XcodeBuildSettings())
	if err != nil {
		return "", false, err
	}

	return next, changed || blockChanged, nil
}

func syncModulePackageStrictnessManifest(content string, swift config.EffectiveSwiftSettings) (string, bool, error) {
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
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return "", false, fmt.Errorf("manifest is empty")
	}

	const prefix = "// swift-tools-version: "
	if !strings.HasPrefix(lines[0], prefix) {
		return "", false, fmt.Errorf("swift tools version header not found")
	}

	replacement := prefix + toolsVersion
	if lines[0] == replacement {
		return content, false, nil
	}

	lines[0] = replacement
	return strings.Join(lines, "\n"), true, nil
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

	pathIndex := -1
	pathIndent := ""
	for index, line := range lines {
		if strings.Contains(line, `path: "Sources"`) {
			pathIndex = index
			pathIndent = leadingIndent(line)
			break
		}
	}
	if pathIndex < 0 {
		return "", false, fmt.Errorf(`target path "Sources" anchor not found`)
	}

	changed := false
	canonicalPathLine := pathIndent + `path: "Sources",`
	if lines[pathIndex] != canonicalPathLine {
		lines[pathIndex] = canonicalPathLine
		changed = true
	}

	blockLines := renderModuleSwiftSettingsLines(pathIndent, settings)
	start := -1
	end := -1
	for index, line := range lines {
		if strings.Contains(line, "swiftSettings: [") {
			start = index
			for next := index + 1; next < len(lines); next++ {
				if strings.TrimSpace(lines[next]) == "]" || strings.TrimSpace(lines[next]) == "]," {
					end = next
					break
				}
			}
			if end < 0 {
				return "", false, fmt.Errorf("unterminated swiftSettings block")
			}
			break
		}
	}

	if start >= 0 {
		currentBlock := lines[start : end+1]
		if sameStringSlice(currentBlock, blockLines) {
			return joinSyncLines(lines, hasTrailingNewline), changed, nil
		}

		updated := append([]string{}, lines[:start]...)
		updated = append(updated, blockLines...)
		updated = append(updated, lines[end+1:]...)
		return joinSyncLines(updated, hasTrailingNewline), true, nil
	}

	lines = insertSyncLines(lines, pathIndex+1, blockLines)
	return joinSyncLines(lines, hasTrailingNewline), true, nil
}

func renderModuleSwiftSettingsLines(indent string, settings []string) []string {
	lines := make([]string, 0, len(settings)+2)
	lines = append(lines, indent+`swiftSettings: [`)
	for _, setting := range settings {
		lines = append(lines, indent+"    "+setting+",")
	}
	lines = append(lines, indent+`]`)
	return lines
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
