package components

import (
	"context"

	"github.com/relux-works/ios-app-manager/internal/config"
)

// GenerateOpts configures Tuist project generation.
type GenerateOpts struct {
	ConfigPath string
	Open       bool
}

// ExternalDep describes an external package dependency with distinct package/product names.
type ExternalDep struct {
	PackageName string // e.g., "swift-relux"
	ProductName string // e.g., "Relux"
	URL         string // e.g., "https://github.com/relux-works/swift-relux.git"
	Version     string // e.g., `from: "9.0.0"`
}

// PlatformTarget describes a generated SwiftPM platform and its minimum version.
type PlatformTarget struct {
	Platform  Platform // e.g., PlatformIOS
	MinTarget string   // e.g., "16.0"
}

// ModuleOpts identifies a module operation target.
type ModuleOpts struct {
	Name         string
	Type         string
	ExternalDeps []ExternalDep
	Platforms    []PlatformTarget
	Config       config.ProjectConfig
}

// ManifestEdit describes one manifest mutation.
type ManifestEdit struct {
	Operation string
	Path      string
	Value     string
}

// ProjectStatus is the consolidated status payload exposed by AppManager.
type ProjectStatus struct {
	ConfigPath            string
	Config                config.ProjectConfig
	ModulesPath           string
	Modules               []string
	DependencyGraphHealth string
}

// TuistProjectManager manages Tuist project-level operations.
type TuistProjectManager interface {
	Generate(ctx context.Context, opts GenerateOpts) error
	Install(ctx context.Context) error
	Graph(ctx context.Context, format string) ([]byte, error)
	Clean(ctx context.Context) error
	CreateModule(ctx context.Context, opts ModuleOpts) error
	EditManifest(ctx context.Context, path string, edits []ManifestEdit) error
}

// ReluxManager manages Relux-specific module workflows.
type ReluxManager interface {
	InitModule(ctx context.Context, moduleName string, moduleType string) error
	AddAction(ctx context.Context, moduleName string, actionName string) error
	AddMiddleware(ctx context.Context, moduleName string, mwName string) error
}

// AppManager orchestrates root project workflows and delegates to components.
type AppManager interface {
	Init(ctx context.Context, configPath string) error
	Status(ctx context.Context) (*ProjectStatus, error)
	CreateModule(ctx context.Context, name string, moduleType string) error
	DeleteModule(ctx context.Context, name string) error
}
