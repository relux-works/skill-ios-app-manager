package modules

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/components"
	"github.com/relux-works/ios-app-manager/internal/config"
)

const (
	defaultCreatorModulesPath = "Packages"
	moduleImplSuffix          = "Impl"
)

var moduleNamePascalCasePattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

// Creator orchestrates module scaffolding across Tuist and Relux managers.
type Creator struct {
	tuist components.TuistProjectManager
	relux components.ReluxManager
	stat  func(name string) (os.FileInfo, error)
}

// NewCreator constructs a module creator facade.
func NewCreator(tuist components.TuistProjectManager, relux components.ReluxManager) *Creator {
	return &Creator{
		tuist: tuist,
		relux: relux,
		stat:  os.Stat,
	}
}

// Create scaffolds a module using module type descriptor rules.
func (c *Creator) Create(
	ctx context.Context,
	moduleName string,
	moduleType string,
	cfg config.ProjectConfig,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	name, err := ValidateModuleName(moduleName)
	if err != nil {
		return err
	}

	descriptor, err := GetModuleType(moduleType)
	if err != nil {
		return err
	}

	if c.tuist == nil {
		return errors.New("tuist project manager is not configured")
	}

	if err := c.ensureNoConflicts(name, descriptor, cfg.ModulesPath); err != nil {
		return err
	}

	if err := c.tuist.CreateModule(ctx, components.ModuleOpts{
		Name:         name,
		Type:         string(descriptor.Type),
		ExternalDeps: convertExternalDeps(descriptor.ExternalDeps),
	}); err != nil {
		return fmt.Errorf("create module in tuist project: %w", err)
	}

	if !descriptor.HasRelux {
		return nil
	}

	if c.relux == nil {
		return errors.New("relux manager is not configured")
	}

	if err := c.relux.InitModule(ctx, name, string(descriptor.Type)); err != nil {
		return fmt.Errorf("initialize relux module: %w", err)
	}

	return nil
}

// ValidateModuleName validates a module name as PascalCase.
func ValidateModuleName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", errors.New("module name is required")
	}

	if !moduleNamePascalCasePattern.MatchString(name) {
		return "", fmt.Errorf("module name %q must be PascalCase", name)
	}

	return name, nil
}

func (c *Creator) ensureNoConflicts(moduleName string, descriptor ModuleTypeDescriptor, modulesPath string) error {
	basePath := normalizeModulesPath(modulesPath)
	for _, packageName := range modulePackageNames(moduleName, descriptor) {
		packagePath := filepath.Join(basePath, packageName)
		exists, err := c.pathExists(packagePath)
		if err != nil {
			return fmt.Errorf("check module package path %q: %w", packagePath, err)
		}
		if exists {
			return fmt.Errorf("module package already exists: %q", packagePath)
		}
	}

	return nil
}

func (c *Creator) pathExists(path string) (bool, error) {
	_, err := c.stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func normalizeModulesPath(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = defaultCreatorModulesPath
	}
	return filepath.Clean(value)
}

func modulePackageNames(moduleName string, descriptor ModuleTypeDescriptor) []string {
	if descriptor.HasInterfaceImplSplit {
		return []string{
			moduleName,
			moduleName + moduleImplSuffix,
		}
	}

	return []string{moduleName}
}

func convertExternalDeps(deps []ExternalDep) []components.ExternalDep {
	if len(deps) == 0 {
		return nil
	}
	out := make([]components.ExternalDep, len(deps))
	for i, d := range deps {
		out[i] = components.ExternalDep{
			PackageName: d.PackageName,
			ProductName: d.ProductName,
			URL:         d.URL,
			Version:     d.Version,
		}
	}
	return out
}
