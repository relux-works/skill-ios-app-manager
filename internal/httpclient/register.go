package httpclient

import (
	"fmt"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

const usageGuide = `## Usage

  Resolve the HTTP client via IoC:

    let client: IRpcAsyncClient = IoC.resolve(IRpcAsyncClient.self)

  Make API requests:

    let response = try await client.send(request)

  Timeouts are configured via Configuration.HttpClient:

    Configuration.HttpClient.timeoutForResponse       // default: 10s
    Configuration.HttpClient.timeoutResourceInterval   // default: 120s

  The client is registered with container lifecycle (singleton).`

func init() {
	registry.Register(&registry.Module{
		ID:           registry.HttpClient,
		Name:         "HttpClient",
		Description:  "HTTP client IoC registration with swift-httpclient",
		Category:     registry.Network,
		Dependencies: []registry.ModuleID{registry.IoC},
		ExternalDeps: []registry.ExternalDep{
			{
				URL:     "https://github.com/relux-works/swift-httpclient.git",
				Version: "6.0.0",
				Product: "HttpClient",
				Package: "HttpClient",
			},
		},

		Plan:       Plan,
		Setup:      SetupFromRegistry,
		UsageGuide: usageGuide,

		CLIUse:     "http-client",
		CLIShort:   "Manage HttpClient IoC registration",
		SetupShort: "Add HttpClient IoC registration",
	})
}

// SetupFromRegistry adapts registry.SetupInput to the local Setup().
func SetupFromRegistry(input registry.SetupInput) error {
	return Setup(SetupInput{
		ProjectRoot: input.ProjectRoot,
		AppName:     input.AppName,
		ModulesPath: input.ModulesPath,
	})
}

// Plan returns what will be created/patched.
func Plan(input registry.SetupInput) (string, error) {
	plan := fmt.Sprintf(`## HttpClient Setup Plan

  Create:
    Targets/%s/Sources/Configuration/Configuration+HttpClient.swift
      — timeout constants for HTTP requests

  Patch:
    Package.swift   — add swift-httpclient external dependency
    Project.swift   — add .external(name: "HttpClient")
    Registry.swift  — add IRpcAsyncClient registration + buildHttpClient builder
                      (in Network section)`, input.AppName)
	return plan, nil
}
