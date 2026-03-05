package tokenprovider

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Resolve the TokenProvider protocol via IoC:

    let provider: TokenProviding = IoC.resolve(TokenProviding.self)

  Store auth credentials after login:

    await provider.setAuthData(authData)

  Retrieve current access token (for API requests):

    let token: String? = await provider.getAccessToken()

  After setup, re-run ioc setup to register TokenProvider in Registry:

    ios-app-manager ioc setup`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.TokenProvider,
		Name:         "TokenProvider",
		Description:  "Token storage and refresh module",
		Category:     registry.Foundation,
		Dependencies: []registry.ModuleID{},

		Plan:       Plan,
		Setup:      SetupFromRegistry,
		UsageGuide: usageGuide,

		CLIUse:     "token-provider",
		CLIShort:   "Manage TokenProvider module",
		SetupShort: "Create TokenProvider module",
	})
}

// SetupFromRegistry adapts the registry.SetupInput to the local Setup().
func SetupFromRegistry(input registry.SetupInput) error {
	return Setup(SetupInput{
		ProjectRoot: input.ProjectRoot,
		AppName:     input.AppName,
		ModulesPath: input.ModulesPath,
	})
}

// Plan returns what will be created/patched.
func Plan(input registry.SetupInput) (string, error) {
	plan := fmt.Sprintf(`## TokenProvider Setup Plan

  Create:
    Packages/TokenProvider/Sources/TokenProvider/
      TokenProvider.swift                          — namespace enum
      TokenProvider.AuthData.swift                 — auth data struct
      Module/TokenProvider.Module.swift            — module declaration
      Module/TokenProvider.Module+Interface.swift  — TokenProviding protocol
    Packages/TokenProviderImpl/Sources/TokenProviderImpl/
      Module/TokenProvider.Module+Impl.swift       — implementation

  Patch:
    Package.swift   — add TokenProvider + TokenProviderImpl paths
    Project.swift   — add dependencies
    Registry.swift  — will be updated on next ioc setup

  App: %s`, input.AppName)
	return plan, nil
}
