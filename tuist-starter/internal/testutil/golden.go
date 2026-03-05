package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateGolden = flag.Bool("update", false, "update golden files")

// AssertGoldenFile compares got with the named golden file in testdata/.
func AssertGoldenFile(t *testing.T, name, got string) {
	t.Helper()

	root := moduleRoot(t)
	path := filepath.Join(root, "testdata", name+".golden")

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("failed to update golden file %q: %v", path, err)
		}
		return
	}

	wantBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file %q: %v", path, err)
	}

	want := string(wantBytes)
	if got != want {
		t.Fatalf("golden mismatch for %q\nwant:\n%q\ngot:\n%q", path, want, got)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to determine working directory: %v", err)
	}

	for {
		modFile := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modFile); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("failed to locate module root from %q", dir)
		}
		dir = parent
	}
}
