package securestore

import (
	"fmt"
	"path/filepath"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Resolve the SecureStore protocol via IoC:

    let store: SecureStoring = IoC.resolve(SecureStoring.self)

  Store and retrieve data:

    try store.save(key: "token", data: tokenData)
    let data = try store.load(key: "token")
    try store.delete(key: "token")
    try store.clear()

  Generic Codable convenience:

    try store.save(key: "user", object: userModel)
    let user: User? = try store.load(key: "user")

  When Registry.swift already exists, setup adds only the SecureStore imports,
  registration, and builder. Existing custom composition is preserved.`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.SecureStore,
		Name:         "SecureStore",
		Description:  "Keychain wrapper with interface/impl split",
		Category:     registry.Foundation,
		Dependencies: []registry.ModuleID{},

		Plan:       Plan,
		Setup:      SetupFromRegistry,
		UsageGuide: usageGuide,

		CLIUse:     "secure-store",
		CLIShort:   "Manage SecureStore module",
		SetupShort: "Create SecureStore kit module",
		ExtraFlags: []registry.ExtraFlag{
			{Name: "access-group", Usage: "app group for shared keychain access", Required: false, ArgKey: "access-group"},
		},
		Capabilities: []registry.Capability{
			{Type: "keychainSharing"},
		},
	})
}

// SetupFromRegistry adapts registry.SetupInput to the local Setup().
func SetupFromRegistry(input registry.SetupInput) error {
	return Setup(SetupInput{
		ProjectRoot: input.ProjectRoot,
		AppName:     input.AppName,
		ModulesPath: input.ModulesPath,
		AccessGroup: input.ExtraArgs["access-group"],
	})
}

// Plan returns what SecureStore setup will create/patch.
func Plan(input registry.SetupInput) (string, error) {
	// Load config to validate access-group against app_groups.
	configPath := filepath.Join(input.ProjectRoot, config.DefaultConfigPath)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	accessGroup := input.ExtraArgs["access-group"]
	if err := validateAccessGroup(accessGroup, cfg.AppGroups); err != nil {
		return "", err
	}

	plan := fmt.Sprintf(`## SecureStore Setup Plan

  Create:
    Packages/SecureStore/Sources/SecureStore/
      SecureStore.swift                        — namespace enum
      Module/SecureStore.Module.swift          — module declaration
      Module/SecureStore.Module+Interface.swift — SecureStoring protocol
    Packages/SecureStoreImpl/Sources/SecureStoreImpl/
      Module/SecureStore.Module+Impl.swift     — Keychain-backed implementation

  Patch:
    Package.swift   — add SecureStore + SecureStoreImpl paths
    Project.swift   — add dependencies
    Registry.swift  — add focused imports, registration, and builder in place

  Access group: %s`, accessGroup)
	return plan, nil
}

func validateAccessGroup(group string, configGroups []string) error {
	if group == "" {
		if len(configGroups) == 0 {
			return fmt.Errorf("--access-group is required but no app_groups defined in config\nadd groups via \"app_groups\" field in ios-app-manager.json, e.g.:\n  \"app_groups\": [\"group.com.example.app\"]")
		}
		return fmt.Errorf("--access-group is required\navailable groups in config: %v", configGroups)
	}

	for _, g := range configGroups {
		if g == group {
			return nil
		}
	}

	return fmt.Errorf("access group %q not found in config\navailable groups: %v", group, configGroups)
}
