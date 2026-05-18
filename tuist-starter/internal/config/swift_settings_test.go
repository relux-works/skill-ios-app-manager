package config

import (
	"strings"
	"testing"
)

func TestEffectiveSwiftSettingsMapsUpcomingFeatureModes(t *testing.T) {
	t.Parallel()

	cfg := ProjectConfig{
		SwiftVersion: "6.2",
		ProjectSettings: ProjectSettings{
			Swift: SwiftProjectSettings{
				LanguageMode:       "v6",
				StrictMemorySafety: "migrate",
				Concurrency: SwiftConcurrencySettings{
					MemberImportVisibility:    "migrate",
					ExistentialAny:            "no",
					NonfrozenEnumExhaustivity: boolPtr(true),
				},
			},
		},
	}

	effective := cfg.EffectiveSwiftSettings()
	settings := buildSettingMap(effective.XcodeBuildSettings())

	if got := settings["SWIFT_UPCOMING_FEATURE_MEMBER_IMPORT_VISIBILITY"]; got != "MIGRATE" {
		t.Fatalf("SWIFT_UPCOMING_FEATURE_MEMBER_IMPORT_VISIBILITY = %q, want %q", got, "MIGRATE")
	}
	if got := settings["SWIFT_UPCOMING_FEATURE_EXISTENTIAL_ANY"]; got != "NO" {
		t.Fatalf("SWIFT_UPCOMING_FEATURE_EXISTENTIAL_ANY = %q, want %q", got, "NO")
	}
	if got := settings["SWIFT_UPCOMING_FEATURE_NONFROZEN_ENUM_EXHAUSTIVITY"]; got != "YES" {
		t.Fatalf("SWIFT_UPCOMING_FEATURE_NONFROZEN_ENUM_EXHAUSTIVITY = %q, want %q", got, "YES")
	}
	if got := settings["SWIFT_STRICT_MEMORY_SAFETY"]; got != "MIGRATE" {
		t.Fatalf("SWIFT_STRICT_MEMORY_SAFETY = %q, want %q", got, "MIGRATE")
	}
	if got := settings["SWIFT_STRICT_CONCURRENCY_DEFAULT"]; got != "complete" {
		t.Fatalf("SWIFT_STRICT_CONCURRENCY_DEFAULT = %q, want %q", got, "complete")
	}

	packageSettings := effective.PackageSwiftSettings()
	if !containsSetting(packageSettings, `.enableUpcomingFeature("StrictConcurrency")`) {
		t.Fatalf("PackageSwiftSettings() missing StrictConcurrency: %#v", packageSettings)
	}
	if !containsSetting(packageSettings, `.enableUpcomingFeature("MemberImportVisibility:migrate")`) {
		t.Fatalf("PackageSwiftSettings() missing MemberImportVisibility migrate: %#v", packageSettings)
	}
	if !containsSetting(packageSettings, `.enableUpcomingFeature("NonfrozenEnumExhaustivity")`) {
		t.Fatalf("PackageSwiftSettings() missing NonfrozenEnumExhaustivity: %#v", packageSettings)
	}
	if containsSetting(packageSettings, `.enableUpcomingFeature("ExistentialAny")`) {
		t.Fatalf("PackageSwiftSettings() unexpectedly contains ExistentialAny: %#v", packageSettings)
	}
}

func buildSettingMap(settings []SwiftBuildSetting) map[string]string {
	result := make(map[string]string, len(settings))
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}
	return result
}

func containsSetting(settings []string, want string) bool {
	for _, setting := range settings {
		if strings.TrimSpace(setting) == want {
			return true
		}
	}
	return false
}
