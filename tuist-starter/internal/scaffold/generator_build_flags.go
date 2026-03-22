package scaffold

import "fmt"

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "build-flags",
		Short:        "Generate or regenerate strict Swift build flags for app and extensions",
		Dependencies: []string{"init"},
		Run:          runGenerateBuildFlags,
	})
}

func runGenerateBuildFlags(input GenerateInput) (GenerateResult, error) {
	result, err := SyncBuildFlags(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	if len(result.Updated) == 0 {
		return GenerateResult{
			Message: fmt.Sprintf(
				"build flag manifests already up to date in %d file(s)\n",
				len(result.Scanned),
			),
		}, nil
	}

	return GenerateResult{
		Message: fmt.Sprintf(
			"regenerated build flag manifests in %d file(s)\n",
			len(result.Updated),
		),
	}, nil
}
