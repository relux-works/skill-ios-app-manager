package scaffold

import (
	"fmt"
)

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:         "runtime-profiles",
		Short:        "Sync typed distribution profiles and backend environments",
		Dependencies: []string{"init", "application-configuration"},
		Run:          runGenerateRuntimeProfiles,
	})
}

func runGenerateRuntimeProfiles(input GenerateInput) (GenerateResult, error) {
	if input.Config.HasRuntimeProfiles() {
		if err := input.Config.Validate(); err != nil {
			return GenerateResult{}, fmt.Errorf("runtime profile config: %w", err)
		}
		if err := ValidateFirebaseClientConfigurationInputs(input.ProjectRoot, input.Config); err != nil {
			return GenerateResult{}, fmt.Errorf("Firebase client configuration inputs: %w", err)
		}
	}
	applicationResult, err := SyncApplicationConfiguration(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("application-configuration: %w", err)
	}
	runtimeResult, err := SyncRuntimeProfiles(input.ProjectRoot, input.Config)
	if err != nil {
		return GenerateResult{}, err
	}

	updated := appendUniqueStrings(nil, applicationResult.Updated...)
	updated = appendUniqueStrings(updated, runtimeResult.Updated...)
	if len(updated) == 0 {
		return GenerateResult{Message: "runtime profiles already up to date\n"}, nil
	}
	return GenerateResult{
		Message: fmt.Sprintf("regenerated runtime profiles in %d file(s)\n", len(updated)),
	}, nil
}
