package config

import "strings"

const (
	DefaultConfigPath             = "ios-app-manager.json"
	DefaultSharedConfigModuleName = "SharedConfig"
	defaultModulesPath            = "Packages"
)

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

	// Build
	ProductName     string          `json:"product_name,omitempty"`   // defaults to AppName
	Configurations  []string        `json:"configurations,omitempty"` // e.g. ["Debug", "Release"]
	ProjectSettings ProjectSettings `json:"project_settings,omitempty"`

	// Modules
	ModulesPath  string             `json:"modules_path,omitempty"` // default: "Packages"
	SharedConfig SharedConfigConfig `json:"shared_config,omitempty"`

	// Push (optional)
	PushKeyPath   string `json:"push_key_path,omitempty"`
	PushKeyID     string `json:"push_key_id,omitempty"`
	PushTokenPath string `json:"push_token_path,omitempty"`
}

type SharedConfigConfig struct {
	ModuleName string `json:"module_name,omitempty"` // default: SharedConfig
}

func (c *ProjectConfig) applyDefaults() {
	if strings.TrimSpace(c.ProductName) == "" {
		c.ProductName = strings.TrimSpace(c.AppName)
	}

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
