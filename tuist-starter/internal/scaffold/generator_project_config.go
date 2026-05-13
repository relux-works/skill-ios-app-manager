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
	type leafRun struct {
		name string
		run  func(GenerateInput) (GenerateResult, error)
	}

	leafRuns := []leafRun{
		{name: "versions", run: runGenerateVersions},
		{name: "min-target", run: runGenerateMinTarget},
		{name: "application-configuration", run: runGenerateApplicationConfiguration},
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
