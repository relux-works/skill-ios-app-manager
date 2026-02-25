package components

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/config"
)

type tuistManagerMock struct {
	generateFn     func(ctx context.Context, opts GenerateOpts) error
	installFn      func(ctx context.Context) error
	graphFn        func(ctx context.Context, format string) ([]byte, error)
	cleanFn        func(ctx context.Context) error
	createModuleFn func(ctx context.Context, opts ModuleOpts) error
	editManifestFn func(ctx context.Context, path string, edits []ManifestEdit) error
}

func (m *tuistManagerMock) Generate(ctx context.Context, opts GenerateOpts) error {
	if m.generateFn == nil {
		return nil
	}
	return m.generateFn(ctx, opts)
}

func (m *tuistManagerMock) Install(ctx context.Context) error {
	if m.installFn == nil {
		return nil
	}
	return m.installFn(ctx)
}

func (m *tuistManagerMock) Graph(ctx context.Context, format string) ([]byte, error) {
	if m.graphFn == nil {
		return nil, nil
	}
	return m.graphFn(ctx, format)
}

func (m *tuistManagerMock) Clean(ctx context.Context) error {
	if m.cleanFn == nil {
		return nil
	}
	return m.cleanFn(ctx)
}

func (m *tuistManagerMock) CreateModule(ctx context.Context, opts ModuleOpts) error {
	if m.createModuleFn == nil {
		return nil
	}
	return m.createModuleFn(ctx, opts)
}

func (m *tuistManagerMock) EditManifest(ctx context.Context, path string, edits []ManifestEdit) error {
	if m.editManifestFn == nil {
		return nil
	}
	return m.editManifestFn(ctx, path, edits)
}

type reluxManagerMock struct {
	initModuleFn    func(ctx context.Context, moduleName string, moduleType string) error
	addActionFn     func(ctx context.Context, moduleName string, actionName string) error
	addMiddlewareFn func(ctx context.Context, moduleName string, mwName string) error
}

func (m *reluxManagerMock) InitModule(ctx context.Context, moduleName string, moduleType string) error {
	if m.initModuleFn == nil {
		return nil
	}
	return m.initModuleFn(ctx, moduleName, moduleType)
}

func (m *reluxManagerMock) AddAction(ctx context.Context, moduleName string, actionName string) error {
	if m.addActionFn == nil {
		return nil
	}
	return m.addActionFn(ctx, moduleName, actionName)
}

func (m *reluxManagerMock) AddMiddleware(ctx context.Context, moduleName string, mwName string) error {
	if m.addMiddlewareFn == nil {
		return nil
	}
	return m.addMiddlewareFn(ctx, moduleName, mwName)
}

func TestAppManagerInit(t *testing.T) {
	t.Parallel()

	var loadPath string
	manager := &appManager{
		configPath: config.DefaultConfigPath,
		loadConfig: func(path string) (config.ProjectConfig, error) {
			loadPath = path
			return validProjectConfig(t), nil
		},
	}

	if err := manager.Init(context.Background(), "custom.json"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if loadPath != "custom.json" {
		t.Fatalf("load path = %q, want %q", loadPath, "custom.json")
	}
	if manager.configPath != "custom.json" {
		t.Fatalf("configPath = %q, want %q", manager.configPath, "custom.json")
	}
}

func TestAppManagerCreateModuleOrchestrates(t *testing.T) {
	t.Parallel()

	var order []string
	var gotOpts ModuleOpts
	var gotInitName string
	var gotInitType string

	manager := &appManager{
		tuist: &tuistManagerMock{
			createModuleFn: func(_ context.Context, opts ModuleOpts) error {
				order = append(order, "tuist")
				gotOpts = opts
				return nil
			},
		},
		relux: &reluxManagerMock{
			initModuleFn: func(_ context.Context, moduleName string, moduleType string) error {
				order = append(order, "relux")
				gotInitName = moduleName
				gotInitType = moduleType
				return nil
			},
		},
	}

	err := manager.CreateModule(context.Background(), "FeatureA", "feature")
	if err != nil {
		t.Fatalf("CreateModule() error = %v", err)
	}

	if !reflect.DeepEqual(order, []string{"tuist", "relux"}) {
		t.Fatalf("order = %#v, want %#v", order, []string{"tuist", "relux"})
	}
	if gotOpts != (ModuleOpts{Name: "FeatureA", Type: "feature"}) {
		t.Fatalf("create opts = %#v, want %#v", gotOpts, ModuleOpts{Name: "FeatureA", Type: "feature"})
	}
	if gotInitName != "FeatureA" || gotInitType != "feature" {
		t.Fatalf(
			"init args = (%q, %q), want (%q, %q)",
			gotInitName,
			gotInitType,
			"FeatureA",
			"feature",
		)
	}
}

func TestAppManagerCreateModuleTuistError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("tuist failed")
	reluxCalled := false

	manager := &appManager{
		tuist: &tuistManagerMock{
			createModuleFn: func(_ context.Context, _ ModuleOpts) error {
				return wantErr
			},
		},
		relux: &reluxManagerMock{
			initModuleFn: func(_ context.Context, _ string, _ string) error {
				reluxCalled = true
				return nil
			},
		},
	}

	err := manager.CreateModule(context.Background(), "FeatureA", "feature")
	if err == nil {
		t.Fatal("CreateModule() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "create module in tuist project") {
		t.Fatalf("CreateModule() error = %q, want wrapped tuist message", err.Error())
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("CreateModule() error should wrap %q", wantErr.Error())
	}
	if reluxCalled {
		t.Fatal("relux InitModule should not be called when tuist CreateModule fails")
	}
}

func TestAppManagerCreateModuleReluxError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("relux failed")

	manager := &appManager{
		tuist: &tuistManagerMock{
			createModuleFn: func(_ context.Context, _ ModuleOpts) error {
				return nil
			},
		},
		relux: &reluxManagerMock{
			initModuleFn: func(_ context.Context, _ string, _ string) error {
				return wantErr
			},
		},
	}

	err := manager.CreateModule(context.Background(), "FeatureA", "feature")
	if err == nil {
		t.Fatal("CreateModule() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "initialize relux module") {
		t.Fatalf("CreateModule() error = %q, want wrapped relux message", err.Error())
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("CreateModule() error should wrap %q", wantErr.Error())
	}
}

func TestAppManagerStatusLoadsConfigAndScansModules(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	packagesDir := filepath.Join(dir, "Packages")
	for _, moduleName := range []string{"FeatureA", "CoreKit"} {
		if err := os.MkdirAll(filepath.Join(packagesDir, moduleName), 0o755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
	}
	if err := os.MkdirAll(filepath.Join(packagesDir, ".hidden"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(.hidden) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(packagesDir, "README.md"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg := validProjectConfig(t)
	manager := &appManager{
		configPath:  "config.json",
		packagesDir: packagesDir,
		loadConfig: func(path string) (config.ProjectConfig, error) {
			if path != "config.json" {
				t.Fatalf("load config path = %q, want %q", path, "config.json")
			}
			return cfg, nil
		},
		readDir: os.ReadDir,
	}

	status, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if status.ConfigPath != "config.json" {
		t.Fatalf("status.ConfigPath = %q, want %q", status.ConfigPath, "config.json")
	}
	if !reflect.DeepEqual(status.Config, cfg) {
		t.Fatalf("status.Config = %#v, want %#v", status.Config, cfg)
	}
	if status.ModulesPath != packagesDir {
		t.Fatalf("status.ModulesPath = %q, want %q", status.ModulesPath, packagesDir)
	}
	if !reflect.DeepEqual(status.Modules, []string{"CoreKit", "FeatureA"}) {
		t.Fatalf("status.Modules = %#v, want %#v", status.Modules, []string{"CoreKit", "FeatureA"})
	}
	if status.DependencyGraphHealth != "unknown" {
		t.Fatalf("status.DependencyGraphHealth = %q, want %q", status.DependencyGraphHealth, "unknown")
	}
}

func TestAppManagerStatusMissingPackagesDir(t *testing.T) {
	t.Parallel()

	cfg := validProjectConfig(t)
	manager := &appManager{
		configPath:  "config.json",
		packagesDir: filepath.Join(t.TempDir(), "Packages"),
		loadConfig: func(_ string) (config.ProjectConfig, error) {
			return cfg, nil
		},
		readDir: os.ReadDir,
	}

	status, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if len(status.Modules) != 0 {
		t.Fatalf("status.Modules len = %d, want 0", len(status.Modules))
	}
}

func TestAppManagerDeleteModule(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotEdits []ManifestEdit
	var removedPath string

	manager := &appManager{
		tuist: &tuistManagerMock{
			editManifestFn: func(_ context.Context, path string, edits []ManifestEdit) error {
				gotPath = path
				gotEdits = append([]ManifestEdit(nil), edits...)
				return nil
			},
		},
		packagesDir:  "Packages",
		manifestPath: "Project.swift",
		removeAll: func(path string) error {
			removedPath = path
			return nil
		},
	}

	if err := manager.DeleteModule(context.Background(), "FeatureA"); err != nil {
		t.Fatalf("DeleteModule() error = %v", err)
	}

	if gotPath != "Project.swift" {
		t.Fatalf("manifest path = %q, want %q", gotPath, "Project.swift")
	}
	wantEdits := []ManifestEdit{
		{
			Operation: "delete_module",
			Path:      "FeatureA",
		},
	}
	if !reflect.DeepEqual(gotEdits, wantEdits) {
		t.Fatalf("edits = %#v, want %#v", gotEdits, wantEdits)
	}

	if removedPath != filepath.Join("Packages", "FeatureA") {
		t.Fatalf("removed path = %q, want %q", removedPath, filepath.Join("Packages", "FeatureA"))
	}
}

func TestAppManagerDeleteModuleManifestError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("manifest edit failed")

	manager := &appManager{
		tuist: &tuistManagerMock{
			editManifestFn: func(_ context.Context, _ string, _ []ManifestEdit) error {
				return wantErr
			},
		},
		packagesDir:  "Packages",
		manifestPath: "Project.swift",
		removeAll: func(_ string) error {
			t.Fatal("removeAll should not be called when EditManifest fails")
			return nil
		},
	}

	err := manager.DeleteModule(context.Background(), "FeatureA")
	if err == nil {
		t.Fatal("DeleteModule() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "edit tuist manifest") {
		t.Fatalf("DeleteModule() error = %q, want wrapped manifest message", err.Error())
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("DeleteModule() error should wrap %q", wantErr.Error())
	}
}

func validProjectConfig(t *testing.T) config.ProjectConfig {
	t.Helper()

	cfg := config.ProjectConfig{
		AppName:          "DemoApp",
		BundleID:         "com.example.demo",
		TeamID:           "ABCDE12345",
		URLScheme:        "demoapp",
		AppGroups:        []string{"group.com.example.demo"},
		MinTarget:        "17.0",
		SwiftVersion:     "6.0",
		MarketingVersion: "1.2.3",
		ProjectVersion:   "123",
		ProductName:      "Demo Product",
		OrgName:          "Example Org",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validProjectConfig().Validate() error = %v", err)
	}

	return cfg
}
