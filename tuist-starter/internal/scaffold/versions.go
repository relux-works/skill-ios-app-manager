package scaffold

import (
	"fmt"
	"os"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

type versionField struct {
	ConstantName    string
	InfoPlistKey    string
	BuildSettingKey string
	Value           string
}

// SyncVersions updates version markers in the scaffolded host app and extension manifests.
func SyncVersions(projectRoot string, cfg config.ProjectConfig) (VersionSyncResult, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return VersionSyncResult{}, fmt.Errorf("project root is required")
	}

	marketingVersion := strings.TrimSpace(cfg.MarketingVersion)
	if marketingVersion == "" {
		return VersionSyncResult{}, fmt.Errorf("marketing version is required")
	}
	projectVersion := strings.TrimSpace(cfg.ProjectVersion)
	if projectVersion == "" {
		return VersionSyncResult{}, fmt.Errorf("project version is required")
	}

	manifestPaths, err := discoverScaffoldManifestPaths(root)
	if err != nil {
		return VersionSyncResult{}, err
	}
	if len(manifestPaths) == 0 {
		return VersionSyncResult{}, fmt.Errorf("no scaffold Project.swift manifests found in %q; run init first", root)
	}

	result := VersionSyncResult{
		Scanned: append([]string(nil), manifestPaths...),
		Updated: make([]string, 0, len(manifestPaths)),
	}

	fields := []versionField{
		{
			ConstantName:    "marketingVersion",
			InfoPlistKey:    "CFBundleShortVersionString",
			BuildSettingKey: "MARKETING_VERSION",
			Value:           marketingVersion,
		},
		{
			ConstantName:    "currentProjectVersion",
			InfoPlistKey:    "CFBundleVersion",
			BuildSettingKey: "CURRENT_PROJECT_VERSION",
			Value:           projectVersion,
		},
	}

	for _, manifestPath := range manifestPaths {
		payload, err := os.ReadFile(manifestPath)
		if err != nil {
			return VersionSyncResult{}, fmt.Errorf("read manifest %q: %w", manifestPath, err)
		}

		updated := string(payload)
		changed := false
		for _, field := range fields {
			next, fieldChanged, err := syncVersionField(updated, field)
			if err != nil {
				return VersionSyncResult{}, fmt.Errorf("sync %s in %q: %w", field.ConstantName, manifestPath, err)
			}
			updated = next
			changed = changed || fieldChanged
		}

		if !changed {
			continue
		}

		if err := os.WriteFile(manifestPath, []byte(updated), 0o644); err != nil {
			return VersionSyncResult{}, fmt.Errorf("write manifest %q: %w", manifestPath, err)
		}

		result.Updated = append(result.Updated, manifestPath)
	}

	return result, nil
}
func syncVersionField(content string, field versionField) (string, bool, error) {
	lines := strings.Split(content, "\n")
	hasTrailingNewline := strings.HasSuffix(content, "\n")

	foundMarker := false
	changed := false

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "let "+field.ConstantName+" = ") {
			foundMarker = true
			replacement := leadingIndent(line) + fmt.Sprintf(`let %s = %q`, field.ConstantName, field.Value)
			if replacement != line {
				lines[index] = replacement
				changed = true
			}
			continue
		}

		infoLineToken := fmt.Sprintf(`"%s": .string(`, field.InfoPlistKey)
		if strings.Contains(line, infoLineToken) {
			foundMarker = true
			if strings.Contains(line, field.ConstantName) {
				continue
			}
			replacement, replaced := replaceStringLiteralArgument(line, field.Value)
			if !replaced {
				return "", false, fmt.Errorf("unable to rewrite %s infoPlist entry", field.InfoPlistKey)
			}
			if replacement != line {
				lines[index] = replacement
				changed = true
			}
			continue
		}

		buildSettingToken := fmt.Sprintf(`"%s": .string(`, field.BuildSettingKey)
		if strings.Contains(line, buildSettingToken) {
			foundMarker = true
			if strings.Contains(line, field.ConstantName) {
				continue
			}
			replacement, replaced := replaceStringLiteralArgument(line, field.Value)
			if !replaced {
				return "", false, fmt.Errorf("unable to rewrite %s build setting", field.BuildSettingKey)
			}
			if replacement != line {
				lines[index] = replacement
				changed = true
			}
		}
	}

	if !foundMarker {
		return "", false, fmt.Errorf("version markers not found")
	}

	updated := strings.Join(lines, "\n")
	if hasTrailingNewline && !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}

	return updated, changed, nil
}
