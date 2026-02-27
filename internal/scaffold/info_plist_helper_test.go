package scaffold

import (
	"strings"
	"testing"
)

func TestGenerateInfoPlistHelper(t *testing.T) {
	t.Parallel()

	content := GenerateInfoPlistHelper()

	checks := []string{
		"import Foundation",
		"extension Bundle",
		"static func readInfoPlistValue<T>(by key: String, from bundle: Bundle) -> T",
		"func readInfoPlistValue<T>(by key: String) -> T",
		"infoDictionary",
		"fatalError",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Fatalf("GenerateInfoPlistHelper() missing %q:\n%s", want, content)
		}
	}
}
