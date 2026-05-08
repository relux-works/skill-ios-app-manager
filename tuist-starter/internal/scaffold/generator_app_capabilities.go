package scaffold

import "fmt"

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "app-capabilities",
		Short:        "Generate or regenerate host app capabilities from project config",
		Dependencies: []string{"init"},
		Run:          runGenerateAppCapabilities,
	})
}

func runGenerateAppCapabilities(input GenerateInput) (GenerateResult, error) {
	result, err := SyncAppCapabilities(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	enabledCount := 0
	for _, plugin := range result.Plugins {
		if plugin.Enabled {
			enabledCount++
		}
	}

	if len(result.Updated) > 0 {
		return GenerateResult{
			Message: fmt.Sprintf(
				"regenerated app capabilities via %d enabled subplugin(s), updated %d file(s)\n",
				enabledCount,
				len(result.Updated),
			),
		}, nil
	}

	if enabledCount == 0 {
		return GenerateResult{
			Message: "app capabilities already up to date (no enabled subplugins)\n",
		}, nil
	}

	return GenerateResult{
		Message: fmt.Sprintf("app capabilities already up to date via %d enabled subplugin(s)\n", enabledCount),
	}, nil
}
