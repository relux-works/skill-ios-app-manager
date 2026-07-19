package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	bundleIDPattern     = regexp.MustCompile(`^[A-Za-z0-9]+(?:\.[A-Za-z0-9][A-Za-z0-9-]*)+$`)
	versionPattern      = regexp.MustCompile(`^\d+\.\d+$`)
	languageModePattern = regexp.MustCompile(`^v\d+(?:_\d+)?$`)
	swiftModulePattern  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// ValidationErrors aggregates all config validation issues in one error.
type ValidationErrors struct {
	Issues []string
}

func (e *ValidationErrors) Error() string {
	return fmt.Sprintf("validation failed: %s", strings.Join(e.Issues, "; "))
}

// Validate checks required fields and format constraints.
func (c ProjectConfig) Validate() error {
	issues := make([]string, 0, 8)

	requiredString(c.AppName, "AppName", &issues)
	requiredString(c.BundleID, "BundleID", &issues)
	requiredString(c.TeamID, "TeamID", &issues)
	requiredString(c.SwiftVersion, "SwiftVersion", &issues)
	requiredString(c.MinTarget, "MinTarget", &issues)
	requiredString(c.MarketingVersion, "MarketingVersion", &issues)
	requiredString(c.ProjectVersion, "ProjectVersion", &issues)

	if value := strings.TrimSpace(c.BundleID); value != "" && !bundleIDPattern.MatchString(value) {
		issues = append(issues, "BundleID must use reverse-domain format (e.g. com.example.app)")
	}
	if value := strings.TrimSpace(c.SwiftVersion); value != "" && !versionPattern.MatchString(value) {
		issues = append(issues, "SwiftVersion must use major.minor format (e.g. 6.2)")
	}
	if value := strings.TrimSpace(c.MinTarget); value != "" && !versionPattern.MatchString(value) {
		issues = append(issues, "MinTarget must use major.minor format (e.g. 17.0)")
	}
	switch value := strings.ToLower(strings.TrimSpace(c.Theme)); value {
	case "", ThemeAutomatic, ThemeLight, ThemeDark:
	default:
		issues = append(issues, "Theme must be automatic, light, or dark")
	}
	switch value := strings.ToLower(strings.TrimSpace(c.Orientation)); value {
	case "", OrientationAutomatic, OrientationPortrait, OrientationLandscape:
	default:
		issues = append(issues, "Orientation must be automatic, portrait, or landscape")
	}
	if c.Platforms != nil {
		validatePlatformDestination(c.Platforms.IOS, "Platforms.IOS", &issues)
		validatePlatformDestination(c.Platforms.IPad, "Platforms.IPad", &issues)
		if !c.IOSTargetEnabled() && !c.IPadTargetEnabled() {
			issues = append(issues, "Platforms must enable at least one destination")
		}
		if !c.IOSTargetEnabled() && c.Platforms.IOS.MacWithIPadDesign != nil && *c.Platforms.IOS.MacWithIPadDesign {
			issues = append(issues, "Platforms.IOS.MacWithIPadDesign requires Platforms.IOS.Enabled")
		}
	}
	if value := strings.TrimSpace(c.ProjectSettings.Swift.LanguageMode); value != "" && !languageModePattern.MatchString(value) {
		issues = append(issues, "ProjectSettings.Swift.LanguageMode must use SwiftPM format (e.g. v6)")
	}
	if !isValidUpcomingFeatureMode(c.ProjectSettings.Swift.StrictMemorySafety) {
		issues = append(issues, "ProjectSettings.Swift.StrictMemorySafety must be yes, migrate, or no")
	}
	if value := strings.TrimSpace(c.SharedConfig.ModuleName); value != "" && !swiftModulePattern.MatchString(value) {
		issues = append(issues, "SharedConfig.ModuleName must be a valid Swift module identifier (e.g. SharedConfig)")
	}

	switch value := strings.TrimSpace(c.ProjectSettings.Swift.Concurrency.DefaultActorIsolation); value {
	case "", defaultSwiftDefaultActorIsolation, swiftDefaultActorIsolationMain:
	default:
		issues = append(issues, "ProjectSettings.Swift.Concurrency.DefaultActorIsolation must be nonisolated or MainActor")
	}

	switch value := strings.TrimSpace(c.ProjectSettings.Swift.Concurrency.StrictChecking); value {
	case "", "minimal", "targeted", defaultSwiftStrictChecking:
	default:
		issues = append(issues, "ProjectSettings.Swift.Concurrency.StrictChecking must be minimal, targeted, or complete")
	}
	if !isValidUpcomingFeatureMode(c.ProjectSettings.Swift.Concurrency.MemberImportVisibility) {
		issues = append(issues, "ProjectSettings.Swift.Concurrency.MemberImportVisibility must be yes, migrate, or no")
	}
	if !isValidUpcomingFeatureMode(c.ProjectSettings.Swift.Concurrency.ExistentialAny) {
		issues = append(issues, "ProjectSettings.Swift.Concurrency.ExistentialAny must be yes, migrate, or no")
	}

	for i, appGroup := range c.AppGroups {
		if strings.TrimSpace(appGroup) == "" {
			issues = append(issues, fmt.Sprintf("AppGroups[%d] must not be empty", i))
		}
	}

	for i, mode := range c.BackgroundModes {
		if !isAllowedBackgroundMode(mode) {
			issues = append(issues, fmt.Sprintf(
				"BackgroundModes[%d] %q is not a UIBackgroundModes value (allowed: %s)",
				i, mode, strings.Join(AllowedBackgroundModes, ", "),
			))
		}
	}

	for i, cfg := range c.Configurations {
		if strings.TrimSpace(cfg) == "" {
			issues = append(issues, fmt.Sprintf("Configurations[%d] must not be empty", i))
		}
	}
	for i, script := range c.Scripts.PreTuistGenerate {
		validateScriptConfig(script, fmt.Sprintf("Scripts.PreTuistGenerate[%d]", i), &issues)
	}
	validateRuntimeProfiles(c, &issues)

	if len(issues) == 0 {
		return nil
	}

	return &ValidationErrors{Issues: issues}
}

func isAllowedBackgroundMode(mode string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(mode))
	if trimmed == "" {
		return false
	}
	for _, allowed := range AllowedBackgroundModes {
		if trimmed == allowed {
			return true
		}
	}
	return false
}

func validatePlatformDestination(destination PlatformDestinationConfig, fieldName string, issues *[]string) {
	switch value := strings.ToLower(strings.TrimSpace(destination.Orientation)); value {
	case "", OrientationAutomatic, OrientationPortrait, OrientationLandscape:
	default:
		*issues = append(*issues, fmt.Sprintf("%s.Orientation must be automatic, portrait, or landscape", fieldName))
	}
}

func requiredString(value string, fieldName string, issues *[]string) {
	if strings.TrimSpace(value) == "" {
		*issues = append(*issues, fmt.Sprintf("%s is required", fieldName))
	}
}

func validateScriptConfig(script ScriptConfig, fieldName string, issues *[]string) {
	path := strings.TrimSpace(script.Path)
	if path == "" {
		*issues = append(*issues, fmt.Sprintf("%s.Path is required", fieldName))
	} else {
		if filepath.IsAbs(path) {
			*issues = append(*issues, fmt.Sprintf("%s.Path must be relative to the project root", fieldName))
		}
		clean := filepath.Clean(path)
		if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
			*issues = append(*issues, fmt.Sprintf("%s.Path must not escape the project root", fieldName))
		}
		if strings.Contains(path, "\x00") || strings.ContainsAny(path, "\r\n") {
			*issues = append(*issues, fmt.Sprintf("%s.Path must be a single-line path", fieldName))
		}
	}

	switch strings.ToLower(strings.TrimSpace(script.Language)) {
	case "bash", "swift", "go", "executable":
	default:
		*issues = append(*issues, fmt.Sprintf("%s.Language must be bash, swift, go, or executable", fieldName))
	}
}
