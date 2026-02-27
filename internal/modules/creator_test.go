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

func TestCreatorCreateReluxFeatureModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	creator := newCreatorForTest(t, root, "Packages")

	err := creator.Create(context.Background(), "Auth", "relux-feature", config.ProjectConfig{
		ModulesPath: filepath.Join(root, "Packages"),
	})
	if err != nil {
		t.Fatalf("Create(relux-feature) error = %v", err)
	}

	requireDir(t, filepath.Join(root, "Packages", "Auth"))
	requireDir(t, filepath.Join(root, "Packages", "AuthImpl"))

	// Interface package files
	requireFile(t, filepath.Join(root, "Packages", "Auth", "Sources", "Auth", "Auth.swift"))
	requireFile(t, filepath.Join(root, "Packages", "Auth", "Sources", "Auth", "Module", "Auth.Module.swift"))
	requireFile(t, filepath.Join(root, "Packages", "Auth", "Sources", "Auth", "Module", "Auth.Module+Interface.swift"))
	requireFile(t, filepath.Join(root, "Packages", "Auth", "Sources", "Auth", "Business", "Auth.Business+Action.swift"))
	requireFile(t, filepath.Join(root, "Packages", "Auth", "Sources", "Auth", "Business", "Auth.Business+Effect.swift"))

	// Impl package files
	requireFile(t, filepath.Join(root, "Packages", "AuthImpl", "Sources", "AuthImpl", "Module", "Auth.Module+Impl.swift"))
	requireFile(t, filepath.Join(root, "Packages", "AuthImpl", "Sources", "AuthImpl", "Business", "Auth.Business+State.swift"))
	requireFile(t, filepath.Join(root, "Packages", "AuthImpl", "Sources", "AuthImpl", "Business", "Auth.Business+Flow.swift"))

	// Verify Package.swift files contain swift-relux dependency
	interfaceManifest := readFileString(t, filepath.Join(root, "Packages", "Auth", "Package.swift"))
	if !strings.Contains(interfaceManifest, "swift-relux") {
		t.Fatalf("interface Package.swift missing swift-relux dependency:\n%s", interfaceManifest)
	}
	if !strings.Contains(interfaceManifest, `"Relux"`) {
		t.Fatalf("interface Package.swift missing Relux product:\n%s", interfaceManifest)
	}

	implManifest := readFileString(t, filepath.Join(root, "Packages", "AuthImpl", "Package.swift"))
	if !strings.Contains(implManifest, "swift-relux") {
		t.Fatalf("impl Package.swift missing swift-relux dependency:\n%s", implManifest)
	}
	if !strings.Contains(implManifest, `"Relux"`) {
		t.Fatalf("impl Package.swift missing Relux product:\n%s", implManifest)
	}
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

func TestCreatorCreateReluxFeatureModuleContent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	creator := newCreatorForTest(t, root, "Packages")

	err := creator.Create(context.Background(), "Auth", "relux-feature", config.ProjectConfig{
		ModulesPath: filepath.Join(root, "Packages"),
	})
	if err != nil {
		t.Fatalf("Create(relux-feature) error = %v", err)
	}

	interfaceSrc := filepath.Join(root, "Packages", "Auth", "Sources", "Auth")
	implSrc := filepath.Join(root, "Packages", "AuthImpl", "Sources", "AuthImpl")

	// Verify namespace has Business enum
	namespace := readFileString(t, filepath.Join(interfaceSrc, "Auth.swift"))
	for _, want := range []string{"public enum Auth", "public enum Business"} {
		if !strings.Contains(namespace, want) {
			t.Fatalf("namespace file missing %q:\n%s", want, namespace)
		}
	}

	// Verify interface has Relux.Module
	iface := readFileString(t, filepath.Join(interfaceSrc, "Module", "Auth.Module+Interface.swift"))
	for _, want := range []string{"import Relux", "Relux.Module"} {
		if !strings.Contains(iface, want) {
			t.Fatalf("interface file missing %q:\n%s", want, iface)
		}
	}

	// Verify action has Relux.Action
	action := readFileString(t, filepath.Join(interfaceSrc, "Business", "Auth.Business+Action.swift"))
	for _, want := range []string{"import Relux", "Relux.Action"} {
		if !strings.Contains(action, want) {
			t.Fatalf("action file missing %q:\n%s", want, action)
		}
	}

	// Verify effect has Relux.Effect
	effect := readFileString(t, filepath.Join(interfaceSrc, "Business", "Auth.Business+Effect.swift"))
	for _, want := range []string{"import Relux", "Relux.Effect"} {
		if !strings.Contains(effect, want) {
			t.Fatalf("effect file missing %q:\n%s", want, effect)
		}
	}

	// Verify impl has Relux.AnyState and Relux.Saga
	impl := readFileString(t, filepath.Join(implSrc, "Module", "Auth.Module+Impl.swift"))
	for _, want := range []string{"import Auth", "import Relux", "Relux.AnyState", "Relux.Saga"} {
		if !strings.Contains(impl, want) {
			t.Fatalf("impl file missing %q:\n%s", want, impl)
		}
	}

	// Verify state has Relux.HybridState
	state := readFileString(t, filepath.Join(implSrc, "Business", "Auth.Business+State.swift"))
	for _, want := range []string{"import Auth", "import Relux", "Relux.HybridState", "@Observable"} {
		if !strings.Contains(state, want) {
			t.Fatalf("state file missing %q:\n%s", want, state)
		}
	}

	// Verify flow has Relux.Flow
	flow := readFileString(t, filepath.Join(implSrc, "Business", "Auth.Business+Flow.swift"))
	for _, want := range []string{"import Auth", "import Relux", "Relux.Flow", "public actor Flow"} {
		if !strings.Contains(flow, want) {
			t.Fatalf("flow file missing %q:\n%s", want, flow)
		}
	}

	// Verify no template artifacts in any generated file
	allFiles := []string{
		filepath.Join(interfaceSrc, "Auth.swift"),
		filepath.Join(interfaceSrc, "Module", "Auth.Module.swift"),
		filepath.Join(interfaceSrc, "Module", "Auth.Module+Interface.swift"),
		filepath.Join(interfaceSrc, "Business", "Auth.Business+Action.swift"),
		filepath.Join(interfaceSrc, "Business", "Auth.Business+Effect.swift"),
		filepath.Join(implSrc, "Module", "Auth.Module+Impl.swift"),
		filepath.Join(implSrc, "Business", "Auth.Business+State.swift"),
		filepath.Join(implSrc, "Business", "Auth.Business+Flow.swift"),
	}
	for _, path := range allFiles {
		content := readFileString(t, path)
		for _, token := range []string{"{{", "}}", "{%", "%}", "<#"} {
			if strings.Contains(content, token) {
				t.Fatalf("file %q contains template artifact %q", path, token)
			}
		}
	}
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
