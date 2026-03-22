package config

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	bundleIDPattern     = regexp.MustCompile(`^[A-Za-z0-9]+(?:\.[A-Za-z0-9][A-Za-z0-9-]*)+$`)
	versionPattern      = regexp.MustCompile(`^\d+\.\d+$`)
	languageModePattern = regexp.MustCompile(`^v\d+(?:_\d+)?$`)
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
	if value := strings.TrimSpace(c.ProjectSettings.Swift.LanguageMode); value != "" && !languageModePattern.MatchString(value) {
		issues = append(issues, "ProjectSettings.Swift.LanguageMode must use SwiftPM format (e.g. v6)")
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

	for i, cfg := range c.Configurations {
		if strings.TrimSpace(cfg) == "" {
			issues = append(issues, fmt.Sprintf("Configurations[%d] must not be empty", i))
		}
	}

	if len(issues) == 0 {
		return nil
	}

	return &ValidationErrors{Issues: issues}
}

func requiredString(value string, fieldName string, issues *[]string) {
	if strings.TrimSpace(value) == "" {
		*issues = append(*issues, fmt.Sprintf("%s is required", fieldName))
	}
}
