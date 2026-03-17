package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func init() {
	RegisterGenerator(&GeneratorPlugin{
		Name:  "makefile",
		Short: "Generate or regenerate Makefile from project config",
		Run:   runGenerateMakefile,
	})
}

func runGenerateMakefile(input GenerateInput) (GenerateResult, error) {
	makefilePath := filepath.Clean(filepath.Join(input.ProjectRoot, "Makefile"))
	existingContent, err := os.ReadFile(makefilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return GenerateResult{}, fmt.Errorf("read existing makefile %q: %w", makefilePath, err)
	}

	existing := ""
	hadExisting := err == nil
	if hadExisting {
		existing = string(existingContent)
	}

	regenerated := GenerateMakefilePreservingCustom(input.Config, existing)
	if err := os.WriteFile(makefilePath, []byte(regenerated), 0o644); err != nil {
		return GenerateResult{}, fmt.Errorf("write makefile %q: %w", makefilePath, err)
	}

	if !hadExisting {
		return GenerateResult{
			Message: fmt.Sprintf(
				"created %s (generated section + default custom section)\n",
				makefilePath,
			),
		}, nil
	}
	if regenerated == existing {
		return GenerateResult{
			Message: fmt.Sprintf(
				"makefile already up to date at %s (generated section verified, custom section preserved)\n",
				makefilePath,
			),
		}, nil
	}

	return GenerateResult{
		Message: fmt.Sprintf(
			"regenerated %s (updated generated section, preserved custom section)\n",
			makefilePath,
		),
	}, nil
}
