package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureOutput(t *testing.T) {
	got := CaptureOutput(func() {
		fmt.Print("captured output")
	})

	if got != "captured output" {
		t.Fatalf("CaptureOutput() = %q, want %q", got, "captured output")
	}
}

func TestTempDir(t *testing.T) {
	dir := TempDir(t)

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("TempDir() path does not exist: %v", err)
	}

	if !info.IsDir() {
		t.Fatalf("TempDir() returned non-directory path: %s", dir)
	}

	file := filepath.Join(dir, "probe.txt")
	if err := os.WriteFile(file, []byte("ok"), 0o644); err != nil {
		t.Fatalf("failed to write file in TempDir(): %v", err)
	}
}
