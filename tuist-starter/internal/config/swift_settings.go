package config

import (
	"fmt"
	"strings"
)

const (
	defaultSwiftToolsVersion          = "6.2"
	defaultSwiftLanguageMode          = "v6"
	defaultSwiftDefaultActorIsolation = "nonisolated"
	defaultSwiftStrictChecking        = "complete"
	defaultSwiftStrictMemorySafety    = swiftUpcomingFeatureModeYes
	defaultSwiftUpcomingFeatureMode   = swiftUpcomingFeatureModeYes
	swiftDefaultActorIsolationMain    = "MainActor"
	swiftLanguageModeV5               = "v5"
	swiftLanguageModeV6               = "v6"
	swiftUpcomingFeatureModeYes       = "yes"
	swiftUpcomingFeatureModeMigrate   = "migrate"
	swiftUpcomingFeatureModeNo        = "no"
)

// ProjectSettings contains config-driven project generation settings.
type ProjectSettings struct {
	Swift SwiftProjectSettings `json:"swift,omitempty"`
}

// SwiftProjectSettings defines the source of truth for Swift-related generation.
type SwiftProjectSettings struct {
	LanguageMode       string                   `json:"language_mode,omitempty"`        // e.g. "v6"
	StrictMemorySafety string                   `json:"strict_memory_safety,omitempty"` // yes | migrate | no
	Concurrency        SwiftConcurrencySettings `json:"concurrency,omitempty"`
}

// SwiftConcurrencySettings stores explicit concurrency-related config knobs.
type SwiftConcurrencySettings struct {
	Approachable                      *bool  `json:"approachable,omitempty"`
	DefaultActorIsolation             string `json:"default_actor_isolation,omitempty"` // nonisolated | MainActor
	StrictChecking                    string `json:"strict_checking,omitempty"`         // minimal | targeted | complete
	ConciseMagicFile                  *bool  `json:"concise_magic_file,omitempty"`
	DisableOutwardActorIsolation      *bool  `json:"disable_outward_actor_isolation,omitempty"`
	GlobalActorIsolatedTypesUsability *bool  `json:"global_actor_isolated_types_usability,omitempty"`
	InferIsolatedConformances         *bool  `json:"infer_isolated_conformances,omitempty"`
	InferSendableFromCaptures         *bool  `json:"infer_sendable_from_captures,omitempty"`
	GlobalConcurrency                 *bool  `json:"global_concurrency,omitempty"`
	MemberImportVisibility            string `json:"member_import_visibility,omitempty"` // yes | migrate | no
	NonfrozenEnumExhaustivity         *bool  `json:"nonfrozen_enum_exhaustivity,omitempty"`
	RegionBasedIsolation              *bool  `json:"region_based_isolation,omitempty"`
	ExistentialAny                    string `json:"existential_any,omitempty"` // yes | migrate | no
	NonisolatedNonsendingByDefault    *bool  `json:"nonisolated_nonsending_by_default,omitempty"`
}

// EffectiveSwiftSettings is the normalized Swift settings view used by generators and sync plugins.
type EffectiveSwiftSettings struct {
	ToolsVersion         string
	LanguageMode         string
	XcodeLanguageVersion string
	StrictMemorySafety   string
	Concurrency          EffectiveSwiftConcurrencySettings
}

// EffectiveSwiftConcurrencySettings is the normalized concurrency settings view.
type EffectiveSwiftConcurrencySettings struct {
	Approachable                      bool
	DefaultActorIsolation             string
	StrictChecking                    string
	ConciseMagicFile                  bool
	DisableOutwardActorIsolation      bool
	GlobalActorIsolatedTypesUsability bool
	InferIsolatedConformances         bool
	InferSendableFromCaptures         bool
	GlobalConcurrency                 bool
	MemberImportVisibility            string
	NonfrozenEnumExhaustivity         bool
	RegionBasedIsolation              bool
	ExistentialAny                    string
	NonisolatedNonsendingByDefault    bool
}

// SwiftBuildSetting represents one Xcode/Tuist Swift build setting assignment.
type SwiftBuildSetting struct {
	Key   string
	Value string
}

func (c *ProjectConfig) applySwiftDefaults() {
	if c == nil {
		return
	}

	if strings.TrimSpace(c.ProjectSettings.Swift.LanguageMode) == "" {
		c.ProjectSettings.Swift.LanguageMode = deriveSwiftLanguageMode(c.SwiftVersion)
	}
	if strings.TrimSpace(c.ProjectSettings.Swift.StrictMemorySafety) == "" {
		c.ProjectSettings.Swift.StrictMemorySafety = defaultSwiftStrictMemorySafety
	}

	c.ProjectSettings.Swift.Concurrency.applyDefaults()
}

func (c *SwiftConcurrencySettings) applyDefaults() {
	if c == nil {
		return
	}

	if c.Approachable == nil {
		c.Approachable = boolPtr(false)
	}
	if strings.TrimSpace(c.DefaultActorIsolation) == "" {
		c.DefaultActorIsolation = defaultSwiftDefaultActorIsolation
	}
	if strings.TrimSpace(c.StrictChecking) == "" {
		c.StrictChecking = defaultSwiftStrictChecking
	}
	if c.ConciseMagicFile == nil {
		c.ConciseMagicFile = boolPtr(true)
	}
	if c.DisableOutwardActorIsolation == nil {
		c.DisableOutwardActorIsolation = boolPtr(true)
	}
	if c.GlobalActorIsolatedTypesUsability == nil {
		c.GlobalActorIsolatedTypesUsability = boolPtr(true)
	}
	if c.InferIsolatedConformances == nil {
		c.InferIsolatedConformances = boolPtr(true)
	}
	if c.InferSendableFromCaptures == nil {
		c.InferSendableFromCaptures = boolPtr(true)
	}
	if c.GlobalConcurrency == nil {
		c.GlobalConcurrency = boolPtr(true)
	}
	if strings.TrimSpace(c.MemberImportVisibility) == "" {
		c.MemberImportVisibility = defaultSwiftUpcomingFeatureMode
	}
	if c.NonfrozenEnumExhaustivity == nil {
		c.NonfrozenEnumExhaustivity = boolPtr(true)
	}
	if c.RegionBasedIsolation == nil {
		c.RegionBasedIsolation = boolPtr(true)
	}
	if strings.TrimSpace(c.ExistentialAny) == "" {
		c.ExistentialAny = defaultSwiftUpcomingFeatureMode
	}
	if c.NonisolatedNonsendingByDefault == nil {
		c.NonisolatedNonsendingByDefault = boolPtr(true)
	}
}

// EffectiveSwiftSettings returns the normalized Swift settings used by generators and sync plugins.
func (c ProjectConfig) EffectiveSwiftSettings() EffectiveSwiftSettings {
	toolsVersion := strings.TrimSpace(c.SwiftVersion)
	if toolsVersion == "" {
		toolsVersion = defaultSwiftToolsVersion
	}

	languageMode := strings.TrimSpace(c.ProjectSettings.Swift.LanguageMode)
	if languageMode == "" {
		languageMode = deriveSwiftLanguageMode(toolsVersion)
	}
	strictMemorySafety := normalizeUpcomingFeatureMode(
		c.ProjectSettings.Swift.StrictMemorySafety,
		defaultSwiftStrictMemorySafety,
	)

	concurrency := c.ProjectSettings.Swift.Concurrency
	concurrency.applyDefaults()

	return EffectiveSwiftSettings{
		ToolsVersion:         toolsVersion,
		LanguageMode:         languageMode,
		XcodeLanguageVersion: languageModeToXcodeSwiftVersion(languageMode),
		StrictMemorySafety:   strictMemorySafety,
		Concurrency:          concurrency.effective(),
	}
}

// XcodeBuildSettings returns Swift-related target build settings for Project.swift and PackageSettings.
func (s EffectiveSwiftSettings) XcodeBuildSettings() []SwiftBuildSetting {
	settings := make([]SwiftBuildSetting, 0, 16)
	settings = append(settings, SwiftBuildSetting{
		Key:   "SWIFT_VERSION",
		Value: s.XcodeLanguageVersion,
	})
	settings = append(settings, SwiftBuildSetting{
		Key:   "SWIFT_STRICT_MEMORY_SAFETY",
		Value: upcomingFeatureModeBuildValue(s.StrictMemorySafety),
	})
	settings = append(settings, s.Concurrency.BuildSettings()...)
	return settings
}

// PackageSwiftSettings returns SwiftPM target swiftSettings expressions.
func (s EffectiveSwiftSettings) PackageSwiftSettings() []string {
	settings := []string{
		fmt.Sprintf(".swiftLanguageMode(.%s)", s.LanguageMode),
	}

	if strings.EqualFold(s.Concurrency.DefaultActorIsolation, swiftDefaultActorIsolationMain) {
		settings = append(settings, ".defaultIsolation(MainActor.self)")
	}
	if strings.EqualFold(s.Concurrency.StrictChecking, defaultSwiftStrictChecking) {
		settings = append(settings, `.enableUpcomingFeature("StrictConcurrency")`)
	}

	for _, feature := range s.Concurrency.PackageUpcomingFeatures() {
		settings = append(settings, fmt.Sprintf(`.enableUpcomingFeature("%s")`, feature))
	}

	return settings
}

// BuildSettings returns concurrency-related target build settings.
func (s EffectiveSwiftConcurrencySettings) BuildSettings() []SwiftBuildSetting {
	return []SwiftBuildSetting{
		{Key: "SWIFT_APPROACHABLE_CONCURRENCY", Value: yesNo(s.Approachable)},
		{Key: "SWIFT_DEFAULT_ACTOR_ISOLATION", Value: s.DefaultActorIsolation},
		{Key: "SWIFT_STRICT_CONCURRENCY_DEFAULT", Value: s.StrictChecking},
		{Key: "SWIFT_STRICT_CONCURRENCY", Value: s.StrictChecking},
		{Key: "SWIFT_UPCOMING_FEATURE_CONCISE_MAGIC_FILE", Value: yesNo(s.ConciseMagicFile)},
		{Key: "SWIFT_UPCOMING_FEATURE_DISABLE_OUTWARD_ACTOR_ISOLATION", Value: yesNo(s.DisableOutwardActorIsolation)},
		{Key: "SWIFT_UPCOMING_FEATURE_GLOBAL_ACTOR_ISOLATED_TYPES_USABILITY", Value: yesNo(s.GlobalActorIsolatedTypesUsability)},
		{Key: "SWIFT_UPCOMING_FEATURE_INFER_ISOLATED_CONFORMANCES", Value: yesNo(s.InferIsolatedConformances)},
		{Key: "SWIFT_UPCOMING_FEATURE_INFER_SENDABLE_FROM_CAPTURES", Value: yesNo(s.InferSendableFromCaptures)},
		{Key: "SWIFT_UPCOMING_FEATURE_GLOBAL_CONCURRENCY", Value: yesNo(s.GlobalConcurrency)},
		{Key: "SWIFT_UPCOMING_FEATURE_MEMBER_IMPORT_VISIBILITY", Value: upcomingFeatureModeBuildValue(s.MemberImportVisibility)},
		{Key: "SWIFT_UPCOMING_FEATURE_NONFROZEN_ENUM_EXHAUSTIVITY", Value: yesNo(s.NonfrozenEnumExhaustivity)},
		{Key: "SWIFT_UPCOMING_FEATURE_REGION_BASED_ISOLATION", Value: yesNo(s.RegionBasedIsolation)},
		{Key: "SWIFT_UPCOMING_FEATURE_EXISTENTIAL_ANY", Value: upcomingFeatureModeBuildValue(s.ExistentialAny)},
		{Key: "SWIFT_UPCOMING_FEATURE_NONISOLATED_NONSENDING_BY_DEFAULT", Value: yesNo(s.NonisolatedNonsendingByDefault)},
	}
}

// PackageUpcomingFeatures returns native SwiftPM upcoming feature names enabled by config.
func (s EffectiveSwiftConcurrencySettings) PackageUpcomingFeatures() []string {
	features := make([]string, 0, 11)
	if s.ConciseMagicFile {
		features = append(features, "ConciseMagicFile")
	}
	if s.DisableOutwardActorIsolation {
		features = append(features, "DisableOutwardActorInference")
	}
	if s.GlobalActorIsolatedTypesUsability {
		features = append(features, "GlobalActorIsolatedTypesUsability")
	}
	if s.InferIsolatedConformances {
		features = append(features, "InferIsolatedConformances")
	}
	if s.InferSendableFromCaptures {
		features = append(features, "InferSendableFromCaptures")
	}
	if s.GlobalConcurrency {
		features = append(features, "GlobalConcurrency")
	}
	if feature := packageUpcomingFeature("MemberImportVisibility", s.MemberImportVisibility); feature != "" {
		features = append(features, feature)
	}
	if s.NonfrozenEnumExhaustivity {
		features = append(features, "NonfrozenEnumExhaustivity")
	}
	if s.RegionBasedIsolation {
		features = append(features, "RegionBasedIsolation")
	}
	if feature := packageUpcomingFeature("ExistentialAny", s.ExistentialAny); feature != "" {
		features = append(features, feature)
	}
	if s.NonisolatedNonsendingByDefault {
		features = append(features, "NonisolatedNonsendingByDefault")
	}
	return features
}

func (c SwiftConcurrencySettings) effective() EffectiveSwiftConcurrencySettings {
	return EffectiveSwiftConcurrencySettings{
		Approachable:                      boolValue(c.Approachable, false),
		DefaultActorIsolation:             strings.TrimSpace(c.DefaultActorIsolation),
		StrictChecking:                    strings.TrimSpace(c.StrictChecking),
		ConciseMagicFile:                  boolValue(c.ConciseMagicFile, true),
		DisableOutwardActorIsolation:      boolValue(c.DisableOutwardActorIsolation, true),
		GlobalActorIsolatedTypesUsability: boolValue(c.GlobalActorIsolatedTypesUsability, true),
		InferIsolatedConformances:         boolValue(c.InferIsolatedConformances, true),
		InferSendableFromCaptures:         boolValue(c.InferSendableFromCaptures, true),
		GlobalConcurrency:                 boolValue(c.GlobalConcurrency, true),
		MemberImportVisibility:            normalizeUpcomingFeatureMode(c.MemberImportVisibility, defaultSwiftUpcomingFeatureMode),
		NonfrozenEnumExhaustivity:         boolValue(c.NonfrozenEnumExhaustivity, true),
		RegionBasedIsolation:              boolValue(c.RegionBasedIsolation, true),
		ExistentialAny:                    normalizeUpcomingFeatureMode(c.ExistentialAny, defaultSwiftUpcomingFeatureMode),
		NonisolatedNonsendingByDefault:    boolValue(c.NonisolatedNonsendingByDefault, true),
	}
}

func deriveSwiftLanguageMode(swiftVersion string) string {
	value := strings.TrimSpace(swiftVersion)
	switch {
	case strings.HasPrefix(value, "5."):
		return swiftLanguageModeV5
	case strings.HasPrefix(value, "6."):
		return swiftLanguageModeV6
	default:
		return defaultSwiftLanguageMode
	}
}

func languageModeToXcodeSwiftVersion(mode string) string {
	value := strings.TrimPrefix(strings.TrimSpace(mode), "v")
	value = strings.ReplaceAll(value, "_", ".")
	if value == "" {
		value = "6"
	}
	if !strings.Contains(value, ".") {
		value += ".0"
	}
	return value
}

func yesNo(enabled bool) string {
	if enabled {
		return "YES"
	}
	return "NO"
}

func boolPtr(value bool) *bool {
	return &value
}

func boolValue(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func normalizeUpcomingFeatureMode(value string, defaultValue string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return defaultValue
	}
	return normalized
}

func upcomingFeatureModeBuildValue(mode string) string {
	switch normalizeUpcomingFeatureMode(mode, defaultSwiftUpcomingFeatureMode) {
	case swiftUpcomingFeatureModeYes:
		return "YES"
	case swiftUpcomingFeatureModeMigrate:
		return "MIGRATE"
	default:
		return "NO"
	}
}

func packageUpcomingFeature(name string, mode string) string {
	switch normalizeUpcomingFeatureMode(mode, defaultSwiftUpcomingFeatureMode) {
	case swiftUpcomingFeatureModeYes:
		return name
	case swiftUpcomingFeatureModeMigrate:
		return name + ":migrate"
	default:
		return ""
	}
}

func isValidUpcomingFeatureMode(value string) bool {
	switch normalizeUpcomingFeatureMode(value, "") {
	case "", swiftUpcomingFeatureModeYes, swiftUpcomingFeatureModeMigrate, swiftUpcomingFeatureModeNo:
		return true
	default:
		return false
	}
}
