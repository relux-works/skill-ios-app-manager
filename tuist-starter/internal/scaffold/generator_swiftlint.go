package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:  "swiftlint",
		Short: "Generate or regenerate .swiftlint.yml from project config",
		Run:   runGenerateSwiftLint,
	})
}

func runGenerateSwiftLint(input GenerateInput) (GenerateResult, error) {
	swiftlintPath := filepath.Clean(filepath.Join(input.ProjectRoot, ".swiftlint.yml"))
	existingContent, err := os.ReadFile(swiftlintPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return GenerateResult{}, fmt.Errorf("read existing swiftlint config %q: %w", swiftlintPath, err)
	}

	existing := ""
	hadExisting := err == nil
	if hadExisting {
		existing = string(existingContent)
	}

	regenerated := GenerateSwiftLintConfig(input.Config)
	if err := os.WriteFile(swiftlintPath, []byte(regenerated), 0o644); err != nil {
		return GenerateResult{}, fmt.Errorf("write swiftlint config %q: %w", swiftlintPath, err)
	}

	if !hadExisting {
		return GenerateResult{
			Message: fmt.Sprintf(
				"created %s (generated from project config)\n",
				swiftlintPath,
			),
		}, nil
	}
	if regenerated == existing {
		return GenerateResult{
			Message: fmt.Sprintf(
				"swiftlint config already up to date at %s\n",
				swiftlintPath,
			),
		}, nil
	}

	return GenerateResult{
		Message: fmt.Sprintf(
			"regenerated %s (updated from project config)\n",
			swiftlintPath,
		),
	}, nil
}
