package scaffold

import (
	"fmt"
	"strings"
)

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "project-config",
		Short:        "Sync project manifest config from ios-app-manager.json",
		Dependencies: []string{"init"},
		Run:          runGenerateProjectConfig,
	})
}

func runGenerateProjectConfig(input GenerateInput) (GenerateResult, error) {
	if input.Config.HasRuntimeProfiles() {
		if err := input.Config.Validate(); err != nil {
			return GenerateResult{}, fmt.Errorf("runtime-profiles preflight: %w", err)
		}
		if err := ValidateFirebaseClientConfigurationInputs(input.ProjectRoot, input.Config); err != nil {
			return GenerateResult{}, fmt.Errorf("runtime-profiles preflight: %w", err)
		}
	}
	type leafRun struct {
		name string
		run  func(GenerateInput) (GenerateResult, error)
	}

	leafRuns := []leafRun{
		{name: "bundle-id", run: runGenerateBundleID},
		{name: "versions", run: runGenerateVersions},
		{name: "min-target", run: runGenerateMinTarget},
		{name: "team-id", run: runGenerateTeamID},
		{name: "platform-destinations", run: runGeneratePlatformDestinations},
		{name: "background-modes-config", run: runGenerateBackgroundModesConfig},
		{name: "presentation-config", run: runGeneratePresentationConfig},
		{name: "export-compliance-config", run: runGenerateExportComplianceConfig},
		{name: "privacy-usage-descriptions-config", run: runGeneratePrivacyUsageDescriptionsConfig},
		{name: "application-configuration", run: runGenerateApplicationConfiguration},
		{name: "runtime-profiles", run: runGenerateRuntimeProfiles},
		{name: "app-capabilities", run: runGenerateAppCapabilities},
		{name: "build-flags", run: runGenerateBuildFlags},
		{name: "package-strictness", run: runGeneratePackageStrictness},
	}

	lines := make([]string, 0, len(leafRuns)+1)
	lines = append(lines, "project config sync summary:")

	for _, leaf := range leafRuns {
		result, err := leaf.run(input)
		if err != nil {
			return GenerateResult{}, fmt.Errorf("%s: %w", leaf.name, err)
		}

		message := strings.TrimSpace(result.Message)
		if message == "" {
			message = "no output"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", leaf.name, message))
	}

	return GenerateResult{
		Message: strings.Join(lines, "\n") + "\n",
	}, nil
}
