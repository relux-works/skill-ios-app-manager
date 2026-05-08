package components

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
)

const (
	defaultPackagesDir = "Packages"
	defaultManifest    = "Project.swift"
)

var errManagerNotInitialized = errors.New("app manager is not initialized")

type appManager struct {
	tuist TuistProjectManager
	relux ReluxManager

	configPath   string
	packagesDir  string
	manifestPath string

	loadConfig func(path string) (config.ProjectConfig, error)
	readDir    func(name string) ([]os.DirEntry, error)
	removeAll  func(path string) error
}

var _ AppManager = (*appManager)(nil)

// NewAppManager creates the integration manager used by CLI root commands.
func NewAppManager(tuist TuistProjectManager, relux ReluxManager) AppManager {
	return &appManager{
		tuist:        tuist,
		relux:        relux,
		configPath:   config.DefaultConfigPath,
		packagesDir:  defaultPackagesDir,
		manifestPath: defaultManifest,
		loadConfig:   config.LoadProjectConfig,
		readDir:      os.ReadDir,
		removeAll:    os.RemoveAll,
	}
}

func (m *appManager) Init(_ context.Context, configPath string) error {
	path := strings.TrimSpace(configPath)
	if path == "" {
		path = config.DefaultConfigPath
	}

	if _, err := m.loadConfig(path); err != nil {
		return err
	}

	m.configPath = path
	return nil
}

func (m *appManager) Status(_ context.Context) (*ProjectStatus, error) {
	path := strings.TrimSpace(m.configPath)
	if path == "" {
		return nil, errManagerNotInitialized
	}

	cfg, err := m.loadConfig(path)
	if err != nil {
		return nil, err
	}

	modulesPath := strings.TrimSpace(cfg.ModulesPath)
	if modulesPath == "" {
		modulesPath = m.packagesDir
	}

	modules, err := m.scanModules(modulesPath)
	if err != nil {
		return nil, err
	}

	return &ProjectStatus{
		ConfigPath:            path,
		Config:                cfg,
		ModulesPath:           modulesPath,
		Modules:               modules,
		DependencyGraphHealth: "unknown",
	}, nil
}

func (m *appManager) CreateModule(ctx context.Context, name string, moduleType string) error {
	moduleName := strings.TrimSpace(name)
	if moduleName == "" {
		return errors.New("module name is required")
	}

	moduleKind := strings.TrimSpace(moduleType)
	if moduleKind == "" {
		return errors.New("module type is required")
	}

	if m.tuist == nil {
		return errors.New("tuist project manager is not configured")
	}
	if m.relux == nil {
		return errors.New("relux manager is not configured")
	}

	moduleConfig := config.ProjectConfig{}
	if m.loadConfig != nil {
		path := strings.TrimSpace(m.configPath)
		if path == "" {
			path = config.DefaultConfigPath
		}

		loadedConfig, err := m.loadConfig(path)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		moduleConfig = loadedConfig
	}

	if err := m.tuist.CreateModule(ctx, ModuleOpts{
		Name:   moduleName,
		Type:   moduleKind,
		Config: moduleConfig,
	}); err != nil {
		return fmt.Errorf("create module in tuist project: %w", err)
	}

	if err := m.relux.InitModule(ctx, moduleName, moduleKind); err != nil {
		return fmt.Errorf("initialize relux module: %w", err)
	}

	return nil
}

func (m *appManager) DeleteModule(ctx context.Context, name string) error {
	moduleName := strings.TrimSpace(name)
	if moduleName == "" {
		return errors.New("module name is required")
	}

	if m.tuist == nil {
		return errors.New("tuist project manager is not configured")
	}

	edits := []ManifestEdit{
		{
			Operation: "delete_module",
			Path:      moduleName,
		},
	}

	if err := m.tuist.EditManifest(ctx, m.manifestPath, edits); err != nil {
		return fmt.Errorf("edit tuist manifest: %w", err)
	}

	modulePath := filepath.Join(m.packagesDir, moduleName)
	if err := m.removeAll(modulePath); err != nil {
		return fmt.Errorf("remove module directory %q: %w", modulePath, err)
	}

	return nil
}

func (m *appManager) scanModules(modulesPath string) ([]string, error) {
	entries, err := m.readDir(modulesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}

		return nil, fmt.Errorf("count modules in %q: %w", modulesPath, err)
	}

	modules := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		modules = append(modules, entry.Name())
	}
	sort.Strings(modules)

	return modules, nil
}
