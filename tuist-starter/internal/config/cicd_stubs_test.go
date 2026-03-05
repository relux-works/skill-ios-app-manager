package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWorkflowStubsContainRequiredHooks(t *testing.T) {
	root := repoRoot(t)

	testCases := []struct {
		path     string
		contains []string
	}{
		{
			path: ".github/workflows/build.yml",
			contains: []string{
				"# STUB — not yet wired to real project",
				"runs-on: macos-latest",
				"make setup",
				"make build",
				"push:",
				"pull_request:",
				"branches: [main]",
			},
		},
		{
			path: ".github/workflows/test.yml",
			contains: []string{
				"# STUB — not yet wired to real project",
				"runs-on: macos-latest",
				"make setup",
				"make test",
				"push:",
				"pull_request:",
				"branches: [main]",
			},
		},
		{
			path: ".github/workflows/lint.yml",
			contains: []string{
				"# STUB — not yet wired to real project",
				"runs-on: macos-latest",
				"make lint",
				"push:",
				"pull_request:",
				"branches: [main]",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			content := readFile(t, filepath.Join(root, tc.path))
			for _, fragment := range tc.contains {
				if !strings.Contains(content, fragment) {
					t.Fatalf("%s missing %q", tc.path, fragment)
				}
			}
		})
	}
}

func TestReadmeDocumentsCiStubs(t *testing.T) {
	content := readFile(t, filepath.Join(repoRoot(t), "README.md"))

	required := []string{
		"## CI/CD (STUB)",
		"CI entry points",
		"make setup",
		"make build",
		"make test",
		"make lint",
		"Adding new CI steps",
		"TEAM_ID",
		"PROVISIONING_PROFILE_SPECIFIER",
		"PROVISIONING_PROFILE_BASE64",
	}

	for _, fragment := range required {
		if !strings.Contains(content, fragment) {
			t.Fatalf("README.md missing %q", fragment)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	// file is tuist-starter/internal/config/cicd_stubs_test.go
	// repo root is three levels up
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

// goModuleRoot returns the Go module root (tuist-starter/), two levels up from internal/config/.
func goModuleRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return string(b)
}
