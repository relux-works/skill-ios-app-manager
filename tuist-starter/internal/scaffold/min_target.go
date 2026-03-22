package scaffold

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

var packageIOSMinTargetPattern = regexp.MustCompile(`\.iOS\(\.v(\d+)(?:_(\d+))?\)`)

// SyncMinTarget updates scaffolded app and extension manifests to use the configured minimum target.
func SyncMinTarget(projectRoot string, cfg config.ProjectConfig) (ManifestSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return ManifestSyncResult{}, fmt.Errorf("project root is required")
	}

	minTarget := strings.TrimSpace(cfg.MinTarget)
	if minTarget == "" {
		return ManifestSyncResult{}, fmt.Errorf("min target is required")
	}

	effectiveMinTarget, err := resolveEffectiveMinTarget(root, cfg, minTarget)
	if err != nil {
		return ManifestSyncResult{}, err
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

		updated, changed, err := syncMinTargetManifest(string(payload), effectiveMinTarget)
		if err != nil {
			return ManifestSyncResult{}, fmt.Errorf("sync minTarget in %q: %w", manifestPath, err)
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

func resolveEffectiveMinTarget(projectRoot string, cfg config.ProjectConfig, configuredMinTarget string) (string, error) {
	manifestPaths, err := discoverPackageManifestPaths(projectRoot, cfg.ModulesPath)
	if err != nil {
		return "", err
	}

	effective := configuredMinTarget
	for _, manifestPath := range manifestPaths {
		payload, err := os.ReadFile(manifestPath)
		if err != nil {
			return "", fmt.Errorf("read package manifest %q: %w", manifestPath, err)
		}

		packageMinTarget, ok, err := detectPackageIOSMinTarget(string(payload))
		if err != nil {
			return "", fmt.Errorf("detect iOS min target in %q: %w", manifestPath, err)
		}
		if !ok {
			continue
		}

		greater, err := isVersionGreater(packageMinTarget, effective)
		if err != nil {
			return "", err
		}
		if greater {
			effective = packageMinTarget
		}
	}

	return effective, nil
}

func detectPackageIOSMinTarget(content string) (string, bool, error) {
	matches := packageIOSMinTargetPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return "", false, nil
	}

	maxVersion := ""
	for _, match := range matches {
		version := match[1] + ".0"
		if len(match) > 2 && strings.TrimSpace(match[2]) != "" {
			version = match[1] + "." + match[2]
		}

		if maxVersion == "" {
			maxVersion = version
			continue
		}

		greater, err := isVersionGreater(version, maxVersion)
		if err != nil {
			return "", false, err
		}
		if greater {
			maxVersion = version
		}
	}

	return maxVersion, true, nil
}

func isVersionGreater(lhs, rhs string) (bool, error) {
	leftMajor, leftMinor, err := parseMajorMinorVersion(lhs)
	if err != nil {
		return false, err
	}
	rightMajor, rightMinor, err := parseMajorMinorVersion(rhs)
	if err != nil {
		return false, err
	}

	if leftMajor != rightMajor {
		return leftMajor > rightMajor, nil
	}
	return leftMinor > rightMinor, nil
}

func parseMajorMinorVersion(value string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("version %q must use major.minor format", value)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse major version from %q: %w", value, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse minor version from %q: %w", value, err)
	}

	return major, minor, nil
}

func syncMinTargetManifest(content, minTarget string) (string, bool, error) {
	updated := content
	changed := false

	next, constantChanged, err := ensureMinTargetConstant(updated, minTarget)
	if err != nil {
		return "", false, err
	}
	updated = next
	changed = changed || constantChanged

	next, deploymentChanged, err := ensureDeploymentTargetsMarker(updated)
	if err != nil {
		return "", false, err
	}
	updated = next
	changed = changed || deploymentChanged

	next, buildSettingChanged, err := ensureMinTargetBuildSetting(updated)
	if err != nil {
		return "", false, err
	}
	updated = next
	changed = changed || buildSettingChanged

	return updated, changed, nil
}

func ensureMinTargetConstant(content, minTarget string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	insertAfter := -1
	insertBeforeProject := -1
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "let minTarget = ") {
			replacement := leadingIndent(line) + fmt.Sprintf(`let minTarget = %q`, minTarget)
			if replacement == line {
				return content, false, nil
			}
			lines[index] = replacement
			return joinSyncLines(lines, hasTrailingNewline), true, nil
		}

		if strings.HasPrefix(trimmed, "let currentProjectVersion = ") {
			insertAfter = index
			continue
		}
		if insertAfter < 0 && strings.HasPrefix(trimmed, "let marketingVersion = ") {
			insertAfter = index
			continue
		}
		if insertBeforeProject < 0 && strings.HasPrefix(trimmed, "let project = Project(") {
			insertBeforeProject = index
		}
	}

	insertLine := fmt.Sprintf(`let minTarget = %q`, minTarget)
	switch {
	case insertAfter >= 0:
		lines = insertSyncLine(lines, insertAfter+1, insertLine)
	case insertBeforeProject >= 0:
		lines = insertSyncLine(lines, insertBeforeProject, insertLine)
	default:
		return "", false, fmt.Errorf("min target insertion anchor not found")
	}

	return joinSyncLines(lines, hasTrailingNewline), true, nil
}

func ensureDeploymentTargetsMarker(content string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "deploymentTargets:") {
			replacement := leadingIndent(line) + "deploymentTargets: .iOS(minTarget),"
			if replacement == line {
				return content, false, nil
			}
			lines[index] = replacement
			return joinSyncLines(lines, hasTrailingNewline), true, nil
		}
	}

	anchorIndex := -1
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "bundleId:") {
			anchorIndex = index
			break
		}
		if anchorIndex < 0 && strings.HasPrefix(trimmed, "product:") {
			anchorIndex = index
		}
	}
	if anchorIndex < 0 {
		return "", false, fmt.Errorf("deploymentTargets insertion anchor not found")
	}

	insertLine := leadingIndent(lines[anchorIndex]) + "deploymentTargets: .iOS(minTarget),"
	lines = insertSyncLine(lines, anchorIndex+1, insertLine)
	return joinSyncLines(lines, hasTrailingNewline), true, nil
}

func ensureMinTargetBuildSetting(content string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	for index, line := range lines {
		if strings.Contains(line, `"IPHONEOS_DEPLOYMENT_TARGET": .string(`) {
			replacement := leadingIndent(line) + `"IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),`
			if replacement == line {
				return content, false, nil
			}
			lines[index] = replacement
			return joinSyncLines(lines, hasTrailingNewline), true, nil
		}
	}

	baseIndex := -1
	for index, line := range lines {
		if strings.Contains(line, "base: [") {
			baseIndex = index
			break
		}
	}
	if baseIndex < 0 {
		return "", false, fmt.Errorf("settings base insertion anchor not found")
	}

	insertLine := leadingIndent(lines[baseIndex]) + `    "IPHONEOS_DEPLOYMENT_TARGET": .string(minTarget),`
	lines = insertSyncLine(lines, baseIndex+1, insertLine)
	return joinSyncLines(lines, hasTrailingNewline), true, nil
}

func insertSyncLine(lines []string, index int, line string) []string {
	if index < 0 {
		index = 0
	}
	if index > len(lines) {
		index = len(lines)
	}

	updated := make([]string, 0, len(lines)+1)
	updated = append(updated, lines[:index]...)
	updated = append(updated, line)
	updated = append(updated, lines[index:]...)
	return updated
}

func joinSyncLines(lines []string, hasTrailingNewline bool) string {
	updated := strings.Join(lines, "\n")
	if hasTrailingNewline && !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	return updated
}
