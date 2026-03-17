package scaffold

import "fmt"

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "versions",
		Short:        "Generate or regenerate app and extension versions from project config",
		Dependencies: []string{"init"},
		Run:          runGenerateVersions,
	})
}

func runGenerateVersions(input GenerateInput) (GenerateResult, error) {
	result, err := SyncVersions(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	if len(result.Updated) == 0 {
		return GenerateResult{
			Message: fmt.Sprintf(
				"version manifests already up to date in %d file(s)\n",
				len(result.Scanned),
			),
		}, nil
	}

	return GenerateResult{
		Message: fmt.Sprintf(
			"regenerated version manifests in %d file(s)\n",
			len(result.Updated),
		),
	}, nil
}
