package clean

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	localDerivedDataDir = "DerivedData"
	localBuildDir       = ".build"
)

// Result captures cleaned targets and the estimated reclaimed disk usage.
type Result struct {
	CleanedPaths []string
	FreedBytes   int64
}

// Manager performs local/global cleanup operations for ios-app-manager.
type Manager struct {
	removeAll   func(path string) error
	estimateDir func(path string) (int64, error)
	tuistClean  func(ctx context.Context, projectRoot string) error
	killXcode   func(ctx context.Context) error
	userHomeDir func() (string, error)
}

// Option configures manager internals for tests and custom runtime behavior.
type Option func(*Manager)

// NewManager creates a clean manager with production defaults.
func NewManager(options ...Option) *Manager {
	manager := &Manager{
		removeAll:   os.RemoveAll,
		estimateDir: estimatePathSize,
		tuistClean:  runTuistClean,
		killXcode:   killXcodeProcess,
		userHomeDir: os.UserHomeDir,
	}

	for _, option := range options {
		if option != nil {
			option(manager)
		}
	}

	return manager
}

// WithRemoveAll overrides directory deletion behavior.
func WithRemoveAll(removeAll func(path string) error) Option {
	return func(m *Manager) {
		if removeAll != nil {
			m.removeAll = removeAll
		}
	}
}

// WithEstimatePathSize overrides disk-usage estimation behavior.
func WithEstimatePathSize(estimate func(path string) (int64, error)) Option {
	return func(m *Manager) {
		if estimate != nil {
			m.estimateDir = estimate
		}
	}
}

// WithTuistClean overrides tuist clean execution.
func WithTuistClean(cleanFn func(ctx context.Context, projectRoot string) error) Option {
	return func(m *Manager) {
		if cleanFn != nil {
			m.tuistClean = cleanFn
		}
	}
}

// WithKillXcode overrides Xcode process termination behavior.
func WithKillXcode(killFn func(ctx context.Context) error) Option {
	return func(m *Manager) {
		if killFn != nil {
			m.killXcode = killFn
		}
	}
}

// WithUserHomeDir overrides home directory resolution.
func WithUserHomeDir(homeFn func() (string, error)) Option {
	return func(m *Manager) {
		if homeFn != nil {
			m.userHomeDir = homeFn
		}
	}
}

// QuickClean removes local build artifacts and executes tuist clean.
func (m *Manager) QuickClean(projectRoot string) (Result, error) {
	root, err := normalizeProjectRoot(projectRoot)
	if err != nil {
		return Result{}, err
	}

	localPaths := quickCleanPaths(root)
	result, err := m.cleanPaths(localPaths)
	if err != nil {
		return Result{}, err
	}

	if err := m.tuistClean(context.Background(), root); err != nil {
		return Result{}, fmt.Errorf("run tuist clean: %w", err)
	}

	return result, nil
}

// DeepClean performs QuickClean and then removes global caches.
func (m *Manager) DeepClean(projectRoot string) (Result, error) {
	quickResult, err := m.QuickClean(projectRoot)
	if err != nil {
		return Result{}, err
	}

	homeDir, err := m.userHomeDir()
	if err != nil {
		return Result{}, fmt.Errorf("resolve user home directory: %w", err)
	}

	deepResult, err := m.cleanPaths(deepCleanPaths(homeDir))
	if err != nil {
		return Result{}, err
	}

	return Result{
		CleanedPaths: append(quickResult.CleanedPaths, deepResult.CleanedPaths...),
		FreedBytes:   quickResult.FreedBytes + deepResult.FreedBytes,
	}, nil
}

// KillXcode terminates the Xcode app process when requested by the user.
func (m *Manager) KillXcode() error {
	if err := m.killXcode(context.Background()); err != nil {
		return fmt.Errorf("kill Xcode process: %w", err)
	}

	return nil
}

func (m *Manager) cleanPaths(paths []string) (Result, error) {
	result := Result{
		CleanedPaths: make([]string, 0, len(paths)),
	}

	for _, rawPath := range paths {
		path := filepath.Clean(rawPath)
		size, err := m.estimateDir(path)
		if err != nil {
			return Result{}, fmt.Errorf("estimate %q size: %w", path, err)
		}

		if err := m.removeAll(path); err != nil {
			return Result{}, fmt.Errorf("remove %q: %w", path, err)
		}

		result.CleanedPaths = append(result.CleanedPaths, path)
		result.FreedBytes += size
	}

	return result, nil
}

func quickCleanPaths(projectRoot string) []string {
	return []string{
		filepath.Join(projectRoot, localDerivedDataDir),
		filepath.Join(projectRoot, localBuildDir),
	}
}

func deepCleanPaths(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, "Library", "Developer", "Xcode", "DerivedData"),
		filepath.Join(homeDir, "Library", "Caches", "org.swift.swiftpm"),
		filepath.Join(homeDir, "Library", "org.swift.swiftpm"),
		filepath.Join(homeDir, ".swiftpm"),
		filepath.Join(homeDir, "Library", "Caches", "com.apple.dt.Xcode"),
	}
}

func normalizeProjectRoot(projectRoot string) (string, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		root = "."
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root %q: %w", root, err)
	}

	return filepath.Clean(absoluteRoot), nil
}

func estimatePathSize(path string) (int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}

	if !info.IsDir() {
		return info.Size(), nil
	}

	var total int64
	walkErr := filepath.WalkDir(path, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, os.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		entryInfo, err := entry.Info()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		total += entryInfo.Size()

		return nil
	})
	if walkErr != nil {
		return 0, walkErr
	}

	return total, nil
}

func runTuistClean(ctx context.Context, projectRoot string) error {
	cmd := exec.CommandContext(ctx, "tuist", "clean")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			return fmt.Errorf("%w: %s", err, trimmed)
		}
		return err
	}

	return nil
}

func killXcodeProcess(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "pkill", "-x", "Xcode")
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		// Exit code 1 means no matching process was found.
		return nil
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed != "" {
		return fmt.Errorf("%w: %s", err, trimmed)
	}
	return err
}

// FormatBytes renders a human-readable byte size for CLI output.
func FormatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}

	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(size)
	for i, unit := range units {
		value /= 1024
		if value < 1024 || i == len(units)-1 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}

	return fmt.Sprintf("%d B", size)
}
