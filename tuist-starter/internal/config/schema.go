package config

import "strings"

const (
	DefaultConfigPath             = "ios-app-manager.json"
	DefaultSharedConfigModuleName = "SharedConfig"
	DefaultTheme                  = "automatic"
	DefaultOrientation            = "automatic"
	defaultModulesPath            = "Packages"
)

const (
	ThemeAutomatic = "automatic"
	ThemeLight     = "light"
	ThemeDark      = "dark"
)

const (
	OrientationAutomatic = "automatic"
	OrientationPortrait  = "portrait"
	OrientationLandscape = "landscape"
)

const (
	BackgroundModeAudio              = "audio"
	BackgroundModeRemoteNotification = "remote-notification"
	BackgroundModeVoIP               = "voip"
)

// AllowedBackgroundModes lists Apple's documented UIBackgroundModes values.
var AllowedBackgroundModes = []string{
	BackgroundModeAudio,
	"bluetooth-central",
	"bluetooth-peripheral",
	"external-accessory",
	"fetch",
	"location",
	"nearby-interaction",
	"processing",
	"push-to-talk",
	BackgroundModeRemoteNotification,
	BackgroundModeVoIP,
}

// ProjectConfig defines project-init schema for ios-app-manager.
type ProjectConfig struct {
	// Identity
	AppName  string `json:"app_name"`
	BundleID string `json:"bundle_id"`
	TeamID   string `json:"team_id"`
	OrgName  string `json:"organization_name"`

	// Versioning
	MarketingVersion string `json:"marketing_version"` // e.g. "1.0.0"
	ProjectVersion   string `json:"project_version"`   // build number, e.g. "1"
	SwiftVersion     string `json:"swift_version"`     // Swift tools / Tuist version, e.g. "6.2"
	MinTarget        string `json:"min_target"`        // e.g. "17.0"

	// URLs and schemes
	URLScheme string   `json:"url_scheme,omitempty"`
	AppGroups []string `json:"app_groups,omitempty"`

	// Presentation
	Theme       string                      `json:"theme,omitempty"`       // automatic, light, dark
	Orientation string                      `json:"orientation,omitempty"` // automatic, portrait, landscape
	Platforms   *PlatformDestinationsConfig `json:"platforms,omitempty"`

	// Export compliance
	UsesNonExemptEncryption *bool `json:"uses_non_exempt_encryption,omitempty"`

	// Privacy usage descriptions
	PrivacyUsageDescriptions PrivacyUsageDescriptionsConfig `json:"privacy_usage_descriptions,omitempty"`

	// Background execution (UIBackgroundModes Info.plist values, e.g. "audio", "voip", "push-to-talk")
	BackgroundModes []string `json:"background_modes,omitempty"`

	// Build
	ProductName     string          `json:"product_name,omitempty"`   // defaults to AppName
	Configurations  []string        `json:"configurations,omitempty"` // e.g. ["Debug", "Release"]
	ProjectSettings ProjectSettings `json:"project_settings,omitempty"`
	Scripts         ScriptsConfig   `json:"scripts,omitempty"`

	// Modules
	ModulesPath  string             `json:"modules_path,omitempty"` // default: "Packages"
	SharedConfig SharedConfigConfig `json:"shared_config,omitempty"`

	// Push (optional)
	PushKeyPath   string `json:"push_key_path,omitempty"`
	PushKeyID     string `json:"push_key_id,omitempty"`
	PushTokenPath string `json:"push_token_path,omitempty"`
}

type PlatformDestinationsConfig struct {
	IOS  PlatformDestinationConfig `json:"ios,omitempty"`
	IPad PlatformDestinationConfig `json:"ipad,omitempty"`
}

type PlatformDestinationConfig struct {
	Enabled           *bool  `json:"enabled,omitempty"`
	Orientation       string `json:"orientation,omitempty"` // automatic, portrait, landscape
	MacWithIPadDesign *bool  `json:"mac_with_ipad_design,omitempty"`
}

type SharedConfigConfig struct {
	ModuleName string `json:"module_name,omitempty"` // default: SharedConfig
}

type PrivacyUsageDescriptionsConfig struct {
	BluetoothAlways     string `json:"bluetooth_always,omitempty"`
	BluetoothPeripheral string `json:"bluetooth_peripheral,omitempty"`
	Camera              string `json:"camera,omitempty"`
	Microphone          string `json:"microphone,omitempty"`
	LocalNetwork        string `json:"local_network,omitempty"`
}

type ScriptsConfig struct {
	PreTuistGenerate []ScriptConfig `json:"pre_tuist_generate,omitempty"`
}

type ScriptConfig struct {
	Path        string `json:"path"`
	Language    string `json:"language"`
	Description string `json:"description,omitempty"`
}

func (c *ProjectConfig) applyDefaults() {
	if strings.TrimSpace(c.ProductName) == "" {
		c.ProductName = strings.TrimSpace(c.AppName)
	}

	c.Theme = normalizeEnumDefault(c.Theme, DefaultTheme)
	c.Orientation = normalizeEnumDefault(c.Orientation, DefaultOrientation)
	c.applyPlatformDefaults()
	c.BackgroundModes = normalizeBackgroundModes(c.BackgroundModes)

	if strings.TrimSpace(c.ModulesPath) == "" {
		c.ModulesPath = defaultModulesPath
	}

	if strings.TrimSpace(c.SharedConfig.ModuleName) == "" {
		c.SharedConfig.ModuleName = DefaultSharedConfigModuleName
	} else {
		c.SharedConfig.ModuleName = strings.TrimSpace(c.SharedConfig.ModuleName)
	}

	c.applySwiftDefaults()
}

func (c *ProjectConfig) applyPlatformDefaults() {
	if c.Platforms == nil {
		return
	}

	c.Platforms.IOS.Orientation = normalizeEnumDefault(c.Platforms.IOS.Orientation, c.Orientation)
	c.Platforms.IPad.Orientation = normalizeEnumDefault(c.Platforms.IPad.Orientation, DefaultOrientation)
}

func (c ProjectConfig) UsesExplicitPlatformDestinations() bool {
	return c.Platforms != nil
}

func (c ProjectConfig) IOSTargetEnabled() bool {
	if c.Platforms == nil {
		return true
	}
	return boolDefault(c.Platforms.IOS.Enabled, true)
}

func (c ProjectConfig) IPadTargetEnabled() bool {
	if c.Platforms == nil {
		return true
	}
	return boolDefault(c.Platforms.IPad.Enabled, false)
}

func (c ProjectConfig) MacWithIPadDesignTargetEnabled() bool {
	if c.Platforms == nil || !c.IOSTargetEnabled() {
		return false
	}
	return boolDefault(c.Platforms.IOS.MacWithIPadDesign, false)
}

func (c ProjectConfig) IOSTargetOrientation() string {
	if c.Platforms == nil {
		return normalizeEnumDefault(c.Orientation, DefaultOrientation)
	}
	return normalizeEnumDefault(c.Platforms.IOS.Orientation, normalizeEnumDefault(c.Orientation, DefaultOrientation))
}

func (c ProjectConfig) IPadTargetOrientation() string {
	if c.Platforms == nil {
		return normalizeEnumDefault(c.Orientation, DefaultOrientation)
	}
	return normalizeEnumDefault(c.Platforms.IPad.Orientation, DefaultOrientation)
}

func (c ProjectConfig) NormalizedBackgroundModes() []string {
	return normalizeBackgroundModes(c.BackgroundModes)
}

func normalizeEnumDefault(value string, defaultValue string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultValue
	}
	return strings.ToLower(trimmed)
}

func boolDefault(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func normalizeBackgroundModes(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	return normalized
}
