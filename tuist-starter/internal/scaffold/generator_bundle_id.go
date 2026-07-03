package scaffold

import "fmt"

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "bundle-id",
		Short:        "Generate or regenerate app and extension bundle identifiers from project config",
		Dependencies: []string{"init"},
		Run:          runGenerateBundleID,
	})
}

func runGenerateBundleID(input GenerateInput) (GenerateResult, error) {
	result, err := SyncBundleID(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	if len(result.Updated) == 0 {
		return GenerateResult{
			Message: fmt.Sprintf(
				"bundle id manifests already up to date in %d file(s)\n",
				len(result.Scanned),
			),
		}, nil
	}

	return GenerateResult{
		Message: fmt.Sprintf(
			"regenerated bundle id manifests in %d file(s)\n",
			len(result.Updated),
		),
	}, nil
}
