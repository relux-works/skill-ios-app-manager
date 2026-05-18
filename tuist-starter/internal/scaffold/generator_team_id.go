package scaffold

import "fmt"

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "team-id",
		Short:        "Generate or regenerate app and extension signing team IDs from project config",
		Dependencies: []string{"init"},
		Run:          runGenerateTeamID,
	})
}

func runGenerateTeamID(input GenerateInput) (GenerateResult, error) {
	result, err := SyncTeamID(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	if len(result.Updated) == 0 {
		return GenerateResult{
			Message: fmt.Sprintf(
				"team id manifests already up to date in %d file(s)\n",
				len(result.Scanned),
			),
		}, nil
	}

	return GenerateResult{
		Message: fmt.Sprintf(
			"regenerated team id manifests in %d file(s)\n",
			len(result.Updated),
		),
	}, nil
}
