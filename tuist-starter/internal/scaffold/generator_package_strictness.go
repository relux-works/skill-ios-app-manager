package scaffold

import "fmt"

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "package-strictness",
		Short:        "Generate or regenerate root and module Package.swift strictness from project config",
		Dependencies: []string{"init"},
		Run:          runGeneratePackageStrictness,
	})
}

func runGeneratePackageStrictness(input GenerateInput) (GenerateResult, error) {
	result, err := SyncPackageStrictness(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	if len(result.Updated) == 0 {
		return GenerateResult{
			Message: fmt.Sprintf(
				"package strictness manifests already up to date in %d file(s)\n",
				len(result.Scanned),
			),
		}, nil
	}

	return GenerateResult{
		Message: fmt.Sprintf(
			"regenerated package strictness manifests in %d file(s)\n",
			len(result.Updated),
		),
	}, nil
}
