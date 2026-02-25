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

	repoRoot := cliTestRepoRoot(t)
	skillPath := filepath.Join(repoRoot, "agents", "skills", "ios-app-manager", "SKILL.md")

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
		fullPath := filepath.Join(repoRoot, "agents", "skills", "ios-app-manager", relPath)
		data, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			t.Fatalf("read %s: %v", relPath, readErr)
		}
		if strings.TrimSpace(string(data)) == "" {
			t.Fatalf("%s must not be empty", relPath)
		}
	}
}

func TestIOSAppManagerSkillSymlink(t *testing.T) {
	t.Parallel()

	repoRoot := cliTestRepoRoot(t)
	linkPath := filepath.Join(repoRoot, ".claude", "skills", "ios-app-manager")

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink", linkPath)
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != "../../agents/skills/ios-app-manager" {
		t.Fatalf("symlink target = %q, want %q", target, "../../agents/skills/ios-app-manager")
	}

	resolved := filepath.Clean(filepath.Join(filepath.Dir(linkPath), target))
	expected := filepath.Join(repoRoot, "agents", "skills", "ios-app-manager")
	if resolved != expected {
		t.Fatalf("resolved target = %q, want %q", resolved, expected)
	}
}

func cliTestRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
