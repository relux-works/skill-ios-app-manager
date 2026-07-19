package appconfig

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Resolve the AppConfig manager via IoC:

    let configManager: IApiConfigManager = IoC.resolve(IApiConfigManager.self)

  Get current API configuration:

    let config = configManager.apiConfiguration
    let baseURL = config.baseURL

  Switch environment:

    configManager.resolver()  // returns current ApiConfiguration

  AppConfig depends on SecureStore for persisting environment selection.
  Environment presets are defined in AppConfig.Env+Configuration+Presets.swift.`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.AppConfig,
		Name:         "AppConfig",
		Description:  "AppConfig manager with environment switching",
		Category:     registry.Foundation,
		Dependencies: []registry.ModuleID{registry.IoC, registry.SecureStore},

		Plan:       Plan,
		Setup:      SetupFromRegistry,
		UsageGuide: usageGuide,

		CLIUse:     "app-config",
		CLIShort:   "Manage AppConfig setup",
		SetupShort: "Create AppConfig manager and ApiConfigurator",
	})
}

// SetupFromRegistry adapts the registry.SetupInput to the local Setup().
func SetupFromRegistry(input registry.SetupInput) error {
	cfg := config.ProjectConfig{AppName: input.AppName}
	if input.ConfigPath != "" {
		loaded, err := config.LoadConfig(input.ConfigPath)
		if err != nil {
			return fmt.Errorf("load app-config runtime profiles: %w", err)
		}
		cfg = loaded
	}
	return Setup(SetupInput{
		ProjectRoot: input.ProjectRoot,
		AppName:     input.AppName,
		Config:      cfg,
	})
}

// Plan returns what will be created/patched.
func Plan(input registry.SetupInput) (string, error) {
	plan := fmt.Sprintf(`## AppConfig Setup Plan

  Create (in Targets/%s/Sources/AppConfig/):
    AppConfig.swift                           — namespace enum
    AppConfig.Env.swift                       — environment enumeration
    AppConfig.Env+Configuration.swift         — API endpoint config per env
    AppConfig.Env+Configuration+Presets.swift  — environment presets
    AppConfig.Manager+Protocols.swift          — IApiConfigManager protocol
    AppConfig.Manager.swift                    — manager implementation
    AppConfig.ApiConfigurator.swift             — API configuration builder
    AppConfig.UrlComponents.swift              — URL component utilities

  Patch:
    Registry.swift — add IApiConfigManager registration + builder
                     (uses SecureStoring dependency from IoC)`, input.AppName)
	return plan, nil
}
