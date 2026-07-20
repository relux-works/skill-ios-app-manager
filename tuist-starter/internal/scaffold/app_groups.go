package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	templatepkg "github.com/relux-works/ios-app-manager/internal/template"
)

const appGroupsInfoPlistKey = "AppGroups"

var appGroupInfoPlistLinePattern = regexp.MustCompile(`^\s*"(?:APP_GROUP|GROUP)_[^"]*":\s*\.string\("group\.[^"]+"\),\s*$`)

func init() {
	RegisterAppCapability(&AppCapabilityPlugin{
		Name:         "app-groups",
		Short:        "Sync host app groups from project config",
		Dependencies: []string{"init"},
		Declarations: appGroupCapabilityDeclarations,
		Sync:         syncAppGroupsCapability,
	})
}

func appGroupCapabilityDeclarations(cfg config.ProjectConfig) []string {
	lines := make([]string, 0, len(cfg.AppGroups))
	for _, group := range normalizeAppGroups(cfg.AppGroups) {
		lines = append(lines, capabilitySwiftLine("appGroups", map[string]string{"group": group}))
	}

	return lines
}

// SyncAppGroups syncs the app-groups capability subplugin directly.
func SyncAppGroups(projectRoot string, cfg config.ProjectConfig) (AppCapabilityPluginResult, error) {
	return syncAppGroupsCapability(AppCapabilityInput{
		ProjectRoot: projectRoot,
		Config:      cfg,
	})
}

func syncAppGroupsCapability(input AppCapabilityInput) (AppCapabilityPluginResult, error) {
	root := strings.TrimSpace(input.ProjectRoot)
	if root == "" {
		return AppCapabilityPluginResult{}, fmt.Errorf("project root is required")
	}

	if err := validateAppGroupsConfig(input.Config); err != nil {
		return AppCapabilityPluginResult{}, err
	}

	normalizedCfg := input.Config
	normalizedCfg.AppGroups = normalizeAppGroups(input.Config.AppGroups)
	appName := normalizeAppName(normalizedCfg.AppName)
	sharedConfigurationModuleName := appGroupSharedConfigurationModuleName(normalizedCfg)

	result := AppCapabilityPluginResult{
		Name:    "app-groups",
		Enabled: len(normalizedCfg.AppGroups) > 0,
		Scanned: make([]string, 0, 8),
		Updated: make([]string, 0, 8),
		Message: fmt.Sprintf("%d app group(s) configured", len(normalizedCfg.AppGroups)),
	}

	appCapabilitiesPath := appCapabilitiesPath(root)
	result.Scanned = append(result.Scanned, appCapabilitiesPath)
	capabilityUpdated, err := syncAppGroupCapabilityDeclarations(root, appGroupCapabilityDeclarations(normalizedCfg))
	if err != nil {
		return result, err
	}
	if capabilityUpdated {
		result.Updated = append(result.Updated, appCapabilitiesPath)
	}

	configurationPath := configurationAppGroupsPath(root, appName)
	staleConfigurationPaths := staleConfigurationAppGroupsPaths(root, appName, configurationPath)
	result.Scanned = append(result.Scanned, configurationPath)
	result.Scanned = append(result.Scanned, staleConfigurationPaths...)
	if len(normalizedCfg.AppGroups) > 0 {
		updated, err := writeFileIfChanged(configurationPath, GenerateConfigurationAppGroups(normalizedCfg))
		if err != nil {
			return result, fmt.Errorf("sync Configuration+AppGroups.swift: %w", err)
		}
		if updated {
			result.Updated = append(result.Updated, configurationPath)
		}
	} else {
		updated, err := removeFileIfExists(configurationPath)
		if err != nil {
			return result, fmt.Errorf("remove Configuration+AppGroups.swift: %w", err)
		}
		if updated {
			result.Updated = append(result.Updated, configurationPath)
		}
	}
	for _, stalePath := range staleConfigurationPaths {
		updated, err := removeFileIfExists(stalePath)
		if err != nil {
			return result, fmt.Errorf("remove stale Configuration+AppGroups.swift: %w", err)
		}
		if updated {
			result.Updated = append(result.Updated, stalePath)
		}
	}

	packageSwiftPath := appGroupSharedConfigurationPackageSwiftPath(root, normalizedCfg)
	packageSourcePath := appGroupSharedConfigurationSourcePath(root, normalizedCfg)
	result.Scanned = append(result.Scanned, packageSwiftPath, packageSourcePath)
	if len(normalizedCfg.AppGroups) > 0 {
		updatedPaths, err := syncAppGroupSharedConfigurationPackage(root, normalizedCfg)
		if err != nil {
			return result, err
		}
		result.Updated = appendUniqueStrings(result.Updated, updatedPaths...)

		updatedPaths, err = cleanupLegacyAppGroupSharedConfigurationPackage(root, normalizedCfg)
		if err != nil {
			return result, err
		}
		result.Updated = appendUniqueStrings(result.Updated, updatedPaths...)
	}

	rootPackagePath := filepath.Join(root, "Package.swift")
	result.Scanned = append(result.Scanned, rootPackagePath)
	if len(normalizedCfg.AppGroups) > 0 {
		updated, err := syncRootPackageSharedConfigurationDependency(root, normalizedCfg)
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, rootPackagePath)
		}

		updated, err = cleanupRootPackageLegacySharedConfigurationDependency(root, normalizedCfg)
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, rootPackagePath)
		}
	}

	projectManifestPaths, err := discoverScaffoldManifestPaths(root)
	if err != nil {
		return result, err
	}
	for _, manifestPath := range projectManifestPaths {
		result.Scanned = appendUniqueStrings(result.Scanned, manifestPath)
		updated, err := syncProjectManifestAppGroups(manifestPath, normalizedCfg.BundleID, normalizedCfg.AppGroups)
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, manifestPath)
		}

		if len(normalizedCfg.AppGroups) == 0 {
			continue
		}

		updated, err = syncProjectManifestSharedConfigurationDependency(manifestPath, sharedConfigurationModuleName)
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, manifestPath)
		}

		updated, err = cleanupProjectManifestLegacySharedConfigurationDependency(manifestPath, normalizedCfg)
		if err != nil {
			return result, err
		}
		if updated {
			result.Updated = appendUniqueStrings(result.Updated, manifestPath)
		}
	}

	return result, nil
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, addition := range additions {
		if strings.TrimSpace(addition) == "" {
			continue
		}
		if _, ok := seen[addition]; ok {
			continue
		}
		values = append(values, addition)
		seen[addition] = struct{}{}
	}
	return values
}

func configurationAppGroupsPath(root string, appName string) string {
	return configurationFilePath(root, appName, "Configuration+AppGroups.swift")
}

func staleConfigurationAppGroupsPaths(root string, appName string, selectedPath string) []string {
	return staleConfigurationFilePaths(root, appName, "Configuration+AppGroups.swift", selectedPath)
}

func defaultConfigurationAppGroupsPath(root string, appName string) string {
	return defaultConfigurationFilePath(root, appName, "Configuration+AppGroups.swift")
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func syncAppGroupCapabilityDeclarations(projectRoot string, declarations []string) (bool, error) {
	path := appCapabilitiesPath(projectRoot)
	payload, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read AppCapabilities.swift: %w", err)
	}

	lines := strings.Split(string(payload), "\n")
	hasTrailingNewline := strings.HasSuffix(string(payload), "\n")
	filtered := make([]string, 0, len(lines))
	insertIndex := -1

	for _, line := range lines {
		if strings.Contains(line, ".appGroups(group:") {
			continue
		}
		if insertIndex < 0 && strings.TrimSpace(line) == "]" {
			insertIndex = len(filtered)
		}
		filtered = append(filtered, line)
	}

	if insertIndex < 0 {
		return false, fmt.Errorf("AppCapabilities.swift missing app array closing bracket")
	}

	nextLines := make([]string, 0, len(filtered)+len(declarations))
	nextLines = append(nextLines, filtered[:insertIndex]...)
	for _, declaration := range declarations {
		if strings.TrimSpace(declaration) == "" {
			continue
		}
		nextLines = append(nextLines, declaration)
	}
	nextLines = append(nextLines, filtered[insertIndex:]...)

	updated := joinSyncLines(nextLines, hasTrailingNewline)
	updated, err = normalizeAppCapabilitiesContent(updated)
	if err != nil {
		return false, err
	}
	if updated == string(payload) {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write AppCapabilities.swift: %w", err)
	}

	return true, nil
}

func syncProjectManifestAppGroups(path string, bundleID string, appGroups []string) (bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read Project.swift: %w", err)
	}

	updated, changed, err := syncProjectManifestAppGroupContent(string(payload), bundleID, appGroups)
	if err != nil {
		return false, fmt.Errorf("sync Project.swift app groups: %w", err)
	}
	if !changed {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write Project.swift: %w", err)
	}

	return true, nil
}

func syncProjectManifestAppGroupContent(content string, bundleID string, appGroups []string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if appGroupInfoPlistLinePattern.MatchString(line) {
			continue
		}
		filtered = append(filtered, line)
	}

	withoutLegacyRootKeys := joinSyncLines(filtered, hasTrailingNewline)
	return syncProjectManifestInfoPlistDictionaryContent(
		withoutLegacyRootKeys,
		appGroupsInfoPlistKey,
		len(appGroups) > 0,
		func(indent string) []string {
			return renderAppGroupInfoPlistLines(indent, bundleID, appGroups)
		},
	)
}

type projectTargetBlock struct {
	start int
	end   int
}

func findProjectTargetBlocks(lines []string) []projectTargetBlock {
	blocks := make([]projectTargetBlock, 0)

	for index := 0; index < len(lines); index++ {
		line := lines[index]
		targetCallIndex := strings.Index(line, ".target(")
		if targetCallIndex < 0 {
			continue
		}

		depth := 0
		for end := index; end < len(lines); end++ {
			startColumn := 0
			if end == index {
				startColumn = targetCallIndex
			}
			depth += parenDeltaOutsideStrings(lines[end], startColumn)
			if depth <= 0 {
				candidate := projectTargetBlock{start: index, end: end}
				if projectTargetBlockLooksLikeDeclaration(lines[candidate.start : candidate.end+1]) {
					blocks = append(blocks, candidate)
				}
				index = end
				break
			}
		}
	}

	return blocks
}

func projectTargetBlockLooksLikeDeclaration(lines []string) bool {
	for _, line := range lines {
		if strings.Contains(line, "name:") || strings.Contains(line, "product:") {
			return true
		}
	}
	return false
}

func parenDeltaOutsideStrings(line string, start int) int {
	delta := 0
	inString := false
	escaped := false
	for index := start; index < len(line); index++ {
		ch := line[index]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '/' && index+1 < len(line) && line[index+1] == '/' {
			break
		}
		if ch == '"' {
			inString = true
			continue
		}
		switch ch {
		case '(':
			delta++
		case ')':
			delta--
		}
	}

	return delta
}

func findLineContaining(lines []string, pattern string) int {
	for index, line := range lines {
		if strings.Contains(line, pattern) {
			return index
		}
	}
	return -1
}

func findFirstLineContainingAny(lines []string, patterns []string) int {
	for index, line := range lines {
		for _, pattern := range patterns {
			if strings.Contains(line, pattern) {
				return index
			}
		}
	}
	return -1
}

func renderAppGroupInfoPlistLines(indent string, bundleID string, appGroups []string) []string {
	lines := make([]string, 0, len(appGroups)+2)
	lines = append(lines, indent+strconv.Quote(appGroupsInfoPlistKey)+": .dictionary([")
	for _, group := range appGroups {
		dictionaryKey := templatepkg.AppGroupSwiftIdentifier(bundleID, group)
		lines = append(lines, indent+"    "+fmt.Sprintf(`%s: .string(%s),`, strconv.Quote(dictionaryKey), strconv.Quote(group)))
	}
	lines = append(lines, indent+"]),")
	return lines
}

func writeFileIfChanged(path string, content string) (bool, error) {
	current, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("read %q: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return false, fmt.Errorf("create parent directory %q: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return false, fmt.Errorf("write %q: %w", path, err)
		}
		return true, nil
	}

	if string(current) == content {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return false, fmt.Errorf("write %q: %w", path, err)
	}

	return true, nil
}

func removeFileIfExists(path string) (bool, error) {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("remove %q: %w", path, err)
	}
	return true, nil
}
