package scaffold

import "fmt"

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "background-modes",
		Short:        "Generate or regenerate the app target UIBackgroundModes entry from project config",
		Dependencies: []string{"init"},
		Run:          runGenerateBackgroundModes,
	})
}

func runGenerateBackgroundModes(input GenerateInput) (GenerateResult, error) {
	result, err := SyncBackgroundModes(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	if len(result.Updated) == 0 {
		return GenerateResult{
			Message: fmt.Sprintf(
				"background modes already up to date in %d file(s)\n",
				len(result.Scanned),
			),
		}, nil
	}

	return GenerateResult{
		Message: fmt.Sprintf(
			"regenerated background modes in %d file(s)\n",
			len(result.Updated),
		),
	}, nil
}
