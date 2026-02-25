package modules

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const modulePackageManifestName = "Package.swift"

// ModuleInfo represents one discovered module package (single or split).
type ModuleInfo struct {
	Name               string
	Type               ModuleType
	InterfacePath      string
	ImplementationPath string
	DependencyCount    int
}

// PackageNames returns package names used by this module in the modules directory.
func (m ModuleInfo) PackageNames() []string {
	names := []string{m.Name}
	if strings.TrimSpace(m.ImplementationPath) != "" {
		names = append(names, m.Name+moduleImplSuffix)
	}
	return names
}

// Lister scans module packages and derives module metadata.
type Lister struct {
	readDir      func(name string) ([]os.DirEntry, error)
	stat         func(name string) (os.FileInfo, error)
	walkDir      func(root string, fn fs.WalkDirFunc) error
	readManifest func(path string) (tuistproj.Manifest, error)
}

// NewLister constructs a module lister with OS-backed dependencies.
func NewLister() *Lister {
	return &Lister{
		readDir:      os.ReadDir,
		stat:         os.Stat,
		walkDir:      filepath.WalkDir,
		readManifest: tuistproj.ReadManifestFile,
	}
}

// List discovers modules from the configured modules path.
func (l *Lister) List(ctx context.Context, modulesPath string) ([]ModuleInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	modulesRoot := normalizeModulesPath(modulesPath)
	entries, err := l.readDir(modulesRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []ModuleInfo{}, nil
		}
		return nil, fmt.Errorf("scan modules directory %q: %w", modulesRoot, err)
	}

	directories := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}

		directories[name] = struct{}{}
	}

	candidates := make([]string, 0, len(directories))
	for name := range directories {
		if strings.HasSuffix(name, moduleImplSuffix) {
			baseName := strings.TrimSuffix(name, moduleImplSuffix)
			if baseName != "" {
				if _, ok := directories[baseName]; ok {
					continue
				}
			}
		}
		candidates = append(candidates, name)
	}
	sort.Strings(candidates)

	modulesList := make([]ModuleInfo, 0, len(candidates))
	for _, name := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		interfacePath := filepath.Join(modulesRoot, name)
		info := ModuleInfo{
			Name:          name,
			Type:          ModuleTypeUtility,
			InterfacePath: interfacePath,
		}

		implName := name + moduleImplSuffix
		if _, ok := directories[implName]; ok {
			info.ImplementationPath = filepath.Join(modulesRoot, implName)
			moduleType, err := l.detectSplitModuleType(interfacePath)
			if err != nil {
				return nil, err
			}
			info.Type = moduleType
		}

		dependencyCount, err := l.countDependencies(info)
		if err != nil {
			return nil, err
		}
		info.DependencyCount = dependencyCount

		modulesList = append(modulesList, info)
	}

	return modulesList, nil
}

func (l *Lister) detectSplitModuleType(interfacePath string) (ModuleType, error) {
	swiftFiles := make(map[string]struct{})
	err := l.walkDir(interfacePath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".swift") {
			return nil
		}

		swiftFiles[strings.ToLower(entry.Name())] = struct{}{}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("scan module files in %q: %w", interfacePath, err)
	}

	has := func(name string) bool {
		_, ok := swiftFiles[strings.ToLower(name)]
		return ok
	}

	// New-style module detection: namespace + module + interface templates.
	hasNewModuleFiles := has("namespace.swift") ||
		has("module.swift") ||
		has("module+interface.swift")

	if hasNewModuleFiles {
		// All new-style split modules with Relux produce the same 4 files.
		// Default to feature as the most common split module type.
		return ModuleTypeFeature, nil
	}

	// Legacy detection: old-style Relux file names.
	hasCoreReluxFiles := has("store.swift") ||
		has("reducer.swift") ||
		has("actions.swift") ||
		has("state.swift") ||
		has("middleware.swift") ||
		has("actions_public.swift") ||
		has("module_registration.swift")

	if hasCoreReluxFiles {
		if has("view.swift") {
			return ModuleTypeFeature, nil
		}
		return ModuleTypeKit, nil
	}

	if has("dto.swift") || has("ioc_registration.swift") || has("ioc_resolver.swift") {
		return ModuleTypeShared, nil
	}

	return ModuleTypeUI, nil
}

func (l *Lister) countDependencies(moduleInfo ModuleInfo) (int, error) {
	excluded := make(map[string]struct{}, 2)
	for _, packageName := range moduleInfo.PackageNames() {
		excluded[packageName] = struct{}{}
	}

	dependencies := make(map[string]struct{})
	for _, packageName := range moduleInfo.PackageNames() {
		manifestPath := filepath.Join(filepath.Dir(moduleInfo.InterfacePath), packageName, modulePackageManifestName)
		exists, err := l.pathExists(manifestPath)
		if err != nil {
			return 0, fmt.Errorf("stat module manifest %q: %w", manifestPath, err)
		}
		if !exists {
			continue
		}

		manifest, err := l.readManifest(manifestPath)
		if err != nil {
			return 0, fmt.Errorf("read module manifest %q: %w", manifestPath, err)
		}
		for _, dependency := range manifest.Dependencies {
			name := strings.TrimSpace(dependency.Name)
			if name == "" {
				continue
			}
			if _, skip := excluded[name]; skip {
				continue
			}
			dependencies[name] = struct{}{}
		}
	}

	return len(dependencies), nil
}

func (l *Lister) pathExists(path string) (bool, error) {
	_, err := l.stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
