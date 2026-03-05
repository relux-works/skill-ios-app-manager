package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIOSAppManagerSkillFrontmatterAndReferences(t *testing.T) {
	t.Parallel()

	repoRoot := skillTestRepoRoot(t)
	skillPath := filepath.Join(repoRoot, "SKILL.md")

	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}

	text := string(content)
	expected := []string{
		"---\n",
		"name: ios-app-manager\n",
		"description: iOS app project management CLI for Tuist-based projects with Relux architecture\n",
		"triggers:\n",
		"  - ios project\n",
		"  - tuist\n",
		"  - relux module\n",
		"  - ios-app-manager\n",
		"globs:\n",
		"  - ios-app-manager.json\n",
		"  - \"**/Tuist/**\"\n",
	}
	for _, want := range expected {
		if !strings.Contains(text, want) {
			t.Fatalf("SKILL.md is missing expected content %q", strings.TrimSpace(want))
		}
	}

	if !strings.HasPrefix(text, "---\nname: ios-app-manager\n") {
		t.Fatalf("SKILL.md frontmatter must start with required name field")
	}

	for _, relPath := range []string{
		filepath.Join("references", "dsl-reference.md"),
		filepath.Join("references", "cli-reference.md"),
	} {
		fullPath := filepath.Join(repoRoot, relPath)
		data, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			t.Fatalf("read %s: %v", relPath, readErr)
		}
		if strings.TrimSpace(string(data)) == "" {
			t.Fatalf("%s must not be empty", relPath)
		}
	}
}

// skillTestRepoRoot returns the root of the skill repo (two levels above tuist-starter/).
func skillTestRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}

	// file is tuist-starter/internal/cli/skill_scaffold_test.go
	// repo root is three levels up from internal/cli/
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
