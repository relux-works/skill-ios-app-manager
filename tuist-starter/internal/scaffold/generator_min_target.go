package scaffold

import "fmt"

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "min-target",
		Short:        "Synchronize app, root package, and local package deployment targets from project config",
		Dependencies: []string{"init"},
		Run:          runGenerateMinTarget,
	})
}

func runGenerateMinTarget(input GenerateInput) (GenerateResult, error) {
	result, err := SyncMinTarget(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	if len(result.Updated) == 0 {
		return GenerateResult{
			Message: fmt.Sprintf(
				"min target manifests already up to date in %d file(s)\n",
				len(result.Scanned),
			),
		}, nil
	}

	return GenerateResult{
		Message: fmt.Sprintf(
			"regenerated min target manifests in %d file(s)\n",
			len(result.Updated),
		),
	}, nil
}
