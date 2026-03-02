package modules_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/modules"
	"github.com/relux-works/ios-app-manager/internal/relux"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

func TestCreatorCreateFeatureModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	creator := newCreatorForTest(t, root, "Packages")

	err := creator.Create(context.Background(), "Auth", "feature", config.ProjectConfig{
		ModulesPath: filepath.Join(root, "Packages"),
	})
	if err != nil {
		t.Fatalf("Create(feature) error = %v", err)
	}

	requireDir(t, filepath.Join(root, "Packages", "Auth"))
	requireDir(t, filepath.Join(root, "Packages", "AuthImpl"))

	requireFile(t, filepath.Join(root, "Packages", "Auth", "Sources", "Auth", "Auth.swift"))
	requireFile(t, filepath.Join(root, "Packages", "Auth", "Sources", "Auth", "Module", "Auth.Module.swift"))
	requireFile(t, filepath.Join(root, "Packages", "Auth", "Sources", "Auth", "Module", "Auth.Module+Interface.swift"))
	requireFile(t, filepath.Join(root, "Packages", "AuthImpl", "Sources", "AuthImpl", "Module", "Auth.Module+Impl.swift"))
}

func TestCreatorCreateUtilityModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	creator := newCreatorForTest(t, root, "Packages")

	err := creator.Create(context.Background(), "Logger", "utility", config.ProjectConfig{
		ModulesPath: filepath.Join(root, "Packages"),
	})
	if err != nil {
		t.Fatalf("Create(utility) error = %v", err)
	}

	requireDir(t, filepath.Join(root, "Packages", "Logger"))
	requireNotExists(t, filepath.Join(root, "Packages", "LoggerImpl"))

	requireNotExists(t, filepath.Join(root, "Packages", "Logger", "Sources", "Logger", "Logger.swift"))
	requireNotExists(t, filepath.Join(root, "Packages", "Logger", "Sources", "Logger", "Logger.Module+Impl.swift"))
}

func TestCreatorCreateKitModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	creator := newCreatorForTest(t, root, "Packages")

	err := creator.Create(context.Background(), "Networking", "kit", config.ProjectConfig{
		ModulesPath: filepath.Join(root, "Packages"),
	})
	if err != nil {
		t.Fatalf("Create(kit) error = %v", err)
	}

	requireDir(t, filepath.Join(root, "Packages", "Networking"))
	requireDir(t, filepath.Join(root, "Packages", "NetworkingImpl"))

	requireFile(t, filepath.Join(root, "Packages", "Networking", "Sources", "Networking", "Networking.swift"))
	requireFile(t, filepath.Join(root, "Packages", "Networking", "Sources", "Networking", "Module", "Networking.Module.swift"))
	requireFile(t, filepath.Join(root, "Packages", "Networking", "Sources", "Networking", "Module", "Networking.Module+Interface.swift"))
	requireFile(t, filepath.Join(root, "Packages", "NetworkingImpl", "Sources", "NetworkingImpl", "Module", "Networking.Module+Impl.swift"))

	// Kit modules should not have swift-relux dependency
	interfaceManifest := readFileString(t, filepath.Join(root, "Packages", "Networking", "Package.swift"))
	if strings.Contains(interfaceManifest, "swift-relux") {
		t.Fatalf("kit interface Package.swift should not have swift-relux dependency:\n%s", interfaceManifest)
	}
}

func TestCreatorCreateSharedModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	creator := newCreatorForTest(t, root, "Packages")

	err := creator.Create(context.Background(), "Storage", "shared", config.ProjectConfig{
		ModulesPath: filepath.Join(root, "Packages"),
	})
	if err != nil {
		t.Fatalf("Create(shared) error = %v", err)
	}

	requireDir(t, filepath.Join(root, "Packages", "Storage"))
	requireDir(t, filepath.Join(root, "Packages", "StorageImpl"))

	requireFile(t, filepath.Join(root, "Packages", "Storage", "Sources", "Storage", "Storage.swift"))
	requireFile(t, filepath.Join(root, "Packages", "Storage", "Sources", "Storage", "Module", "Storage.Module.swift"))
	requireFile(t, filepath.Join(root, "Packages", "Storage", "Sources", "Storage", "Module", "Storage.Module+Interface.swift"))
	requireFile(t, filepath.Join(root, "Packages", "StorageImpl", "Sources", "StorageImpl", "Module", "Storage.Module+Impl.swift"))
}

func TestCreatorCreateUIModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	creator := newCreatorForTest(t, root, "Packages")

	err := creator.Create(context.Background(), "DesignSystem", "ui", config.ProjectConfig{
		ModulesPath: filepath.Join(root, "Packages"),
	})
	if err != nil {
		t.Fatalf("Create(ui) error = %v", err)
	}

	requireDir(t, filepath.Join(root, "Packages", "DesignSystem"))
	requireDir(t, filepath.Join(root, "Packages", "DesignSystemImpl"))

	requireFile(t, filepath.Join(root, "Packages", "DesignSystem", "Sources", "DesignSystem", "DesignSystem.swift"))
	requireFile(t, filepath.Join(root, "Packages", "DesignSystem", "Sources", "DesignSystem", "Module", "DesignSystem.Module.swift"))
	requireFile(t, filepath.Join(root, "Packages", "DesignSystem", "Sources", "DesignSystem", "Module", "DesignSystem.Module+Interface.swift"))
	requireFile(t, filepath.Join(root, "Packages", "DesignSystemImpl", "Sources", "DesignSystemImpl", "Module", "DesignSystem.Module+Impl.swift"))
}

func TestCreatorCreateDetectsConflicts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	creator := newCreatorForTest(t, root, "Packages")

	if err := os.MkdirAll(filepath.Join(root, "Packages", "Auth"), 0o755); err != nil {
		t.Fatalf("MkdirAll(Auth) error = %v", err)
	}

	err := creator.Create(context.Background(), "Auth", "feature", config.ProjectConfig{
		ModulesPath: filepath.Join(root, "Packages"),
	})
	if err == nil {
		t.Fatal("Create(feature) error = nil, want conflict error")
	}

	if !strings.Contains(err.Error(), "module package already exists") {
		t.Fatalf("Create(feature) error = %q, want conflict message", err.Error())
	}
}

func TestValidateModuleName(t *testing.T) {
	t.Parallel()

	if _, err := modules.ValidateModuleName("AuthKit"); err != nil {
		t.Fatalf("ValidateModuleName(AuthKit) error = %v", err)
	}

	_, err := modules.ValidateModuleName("authKit")
	if err == nil {
		t.Fatal("ValidateModuleName(authKit) error = nil, want error")
	}
	if !strings.Contains(err.Error(), "PascalCase") {
		t.Fatalf("ValidateModuleName(authKit) error = %q, want PascalCase message", err.Error())
	}
}

func newCreatorForTest(t *testing.T, rootDir string, modulesDir string) *modules.Creator {
	t.Helper()

	tuistManager := tuistproj.NewTuistProjectManager(
		tuistproj.WithRootDir(rootDir),
		tuistproj.WithModulesDir(modulesDir),
		tuistproj.WithPackagePlatform("iOS(.v16)"),
	)

	reluxManager, err := relux.NewReluxManager(filepath.Join(rootDir, modulesDir))
	if err != nil {
		t.Fatalf("NewReluxManager() error = %v", err)
	}

	return modules.NewCreator(tuistManager, reluxManager)
}

func requireDir(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", path)
	}
}

func requireFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("%q is a directory, want file", path)
	}
}

func requireNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("path %q exists, want missing", path)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(%q) error = %v, want os.ErrNotExist", path, err)
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}
