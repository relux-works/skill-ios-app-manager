package clean

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestQuickCleanRemovesLocalTargetsAndRunsTuist(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(t.TempDir(), "DemoApp")
	expectedPaths := quickCleanPaths(projectRoot)

	var removed []string
	var tuistRoot string
	sizeByPath := map[string]int64{
		filepath.Clean(expectedPaths[0]): 4096,
		filepath.Clean(expectedPaths[1]): 1024,
	}

	manager := NewManager(
		WithRemoveAll(func(path string) error {
			removed = append(removed, filepath.Clean(path))
			return nil
		}),
		WithEstimatePathSize(func(path string) (int64, error) {
			return sizeByPath[filepath.Clean(path)], nil
		}),
		WithTuistClean(func(_ context.Context, root string) error {
			tuistRoot = filepath.Clean(root)
			return nil
		}),
	)

	result, err := manager.QuickClean(projectRoot)
	if err != nil {
		t.Fatalf("QuickClean() error = %v", err)
	}

	if !reflect.DeepEqual(result.CleanedPaths, cleanPaths(expectedPaths)) {
		t.Fatalf("CleanedPaths = %#v, want %#v", result.CleanedPaths, cleanPaths(expectedPaths))
	}
	if !reflect.DeepEqual(removed, cleanPaths(expectedPaths)) {
		t.Fatalf("removed = %#v, want %#v", removed, cleanPaths(expectedPaths))
	}
	if tuistRoot != filepath.Clean(projectRoot) {
		t.Fatalf("tuist clean root = %q, want %q", tuistRoot, filepath.Clean(projectRoot))
	}

	const wantFreed = int64(5120)
	if result.FreedBytes != wantFreed {
		t.Fatalf("FreedBytes = %d, want %d", result.FreedBytes, wantFreed)
	}
}

func TestDeepCleanIncludesGlobalTargets(t *testing.T) {
	t.Parallel()

	projectRoot := filepath.Join(t.TempDir(), "DemoApp")
	home := filepath.Join(t.TempDir(), "home")

	expected := append(cleanPaths(quickCleanPaths(projectRoot)), cleanPaths(deepCleanPaths(home))...)

	var removed []string
	tuistCalls := 0

	manager := NewManager(
		WithRemoveAll(func(path string) error {
			removed = append(removed, filepath.Clean(path))
			return nil
		}),
		WithEstimatePathSize(func(_ string) (int64, error) {
			return 1, nil
		}),
		WithTuistClean(func(_ context.Context, _ string) error {
			tuistCalls++
			return nil
		}),
		WithUserHomeDir(func() (string, error) {
			return home, nil
		}),
	)

	result, err := manager.DeepClean(projectRoot)
	if err != nil {
		t.Fatalf("DeepClean() error = %v", err)
	}

	if !reflect.DeepEqual(result.CleanedPaths, expected) {
		t.Fatalf("CleanedPaths = %#v, want %#v", result.CleanedPaths, expected)
	}
	if !reflect.DeepEqual(removed, expected) {
		t.Fatalf("removed = %#v, want %#v", removed, expected)
	}
	if tuistCalls != 1 {
		t.Fatalf("tuistCalls = %d, want 1", tuistCalls)
	}
	if result.FreedBytes != int64(len(expected)) {
		t.Fatalf("FreedBytes = %d, want %d", result.FreedBytes, len(expected))
	}
}

func TestKillXcodeDelegatesToProcessKiller(t *testing.T) {
	t.Parallel()

	called := false
	manager := NewManager(
		WithKillXcode(func(_ context.Context) error {
			called = true
			return nil
		}),
	)

	if err := manager.KillXcode(); err != nil {
		t.Fatalf("KillXcode() error = %v", err)
	}
	if !called {
		t.Fatal("KillXcode() did not call configured kill function")
	}
}

func TestKillXcodeReturnsWrappedError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	manager := NewManager(
		WithKillXcode(func(_ context.Context) error {
			return wantErr
		}),
	)

	err := manager.KillXcode()
	if err == nil {
		t.Fatal("KillXcode() error = nil, want non-nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("KillXcode() error = %v, want wrapped %v", err, wantErr)
	}
}

func cleanPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, filepath.Clean(path))
	}
	return out
}
