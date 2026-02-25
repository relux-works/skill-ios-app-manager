package scaffold

import (
	"strings"
	"testing"
)

func TestGenerateGitignoreContainsRequiredPatterns(t *testing.T) {
	t.Parallel()

	gitignore := GenerateGitignore()

	requiredPatterns := []string{
		".DS_Store",
		"*.xcodeproj",
		"*.xcworkspace",
		"xcuserdata/",
		"DerivedData/",
		"build/",
		"Derived/",
	}
	for _, pattern := range requiredPatterns {
		if !strings.Contains(gitignore, pattern) {
			t.Fatalf("GenerateGitignore() missing %q:\n%s", pattern, gitignore)
		}
	}
}
