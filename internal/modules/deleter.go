package modules

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const (
	defaultProjectManifestPath = "Project.swift"
	defaultRootPackagePath     = "Package.swift"
)

// ErrDeleteModuleCanceled is returned when a module deletion is aborted by confirmation.
var ErrDeleteModuleCanceled = errors.New("module deletion canceled")

// DeleteOptions configures module deletion behavior.
type DeleteOptions struct {
	ModulesPath             string
	ProjectRoot             string
	ProjectManifestPath     string
	RootPackageManifestPath string
	Force                   bool
	Confirm                 func(module ModuleInfo) (bool, error)
}

// DeleteResult contains artifacts of one module deletion run.
type DeleteResult struct {
	Module           ModuleInfo
	DeletedPaths     []string
	UpdatedManifests []string
}

// Deleter deletes module packages and removes dependency references.
type Deleter struct {
	lister             *Lister
	readDir            func(name string) ([]os.DirEntry, error)
	removeAll          func(path string) error
	stat               func(name string) (os.FileInfo, error)
	readManifest       func(path string) (tuistproj.Manifest, error)
	applyManifestEdits func(path string, edits ...tuistproj.ManifestEdit) error
}

// NewDeleter constructs a module deleter with default filesystem behavior.
func NewDeleter() *Deleter {
	return &Deleter{
		lister:             NewLister(),
		readDir:            os.ReadDir,
		removeAll:          os.RemoveAll,
		stat:               os.Stat,
		readManifest:       tuistproj.ReadManifestFile,
		applyManifestEdits: tuistproj.ApplyManifestEditsToFile,
	}
}

// Delete removes the requested module and all references to it.
func (d *Deleter) Delete(ctx context.Context, moduleName string, opts DeleteOptions) (DeleteResult, error) {
	if err := ctx.Err(); err != nil {
		return DeleteResult{}, err
	}

	name := strings.TrimSpace(moduleName)
	if name == "" {
		return DeleteResult{}, errors.New("module name is required")
	}

	modulesRoot := resolveDeleteModulesRoot(opts.ProjectRoot, opts.ModulesPath)
	moduleInfo, err := d.lookupModule(ctx, modulesRoot, name)
	if err != nil {
		return DeleteResult{}, err
	}

	if !opts.Force {
		if opts.Confirm == nil {
			return DeleteResult{}, errors.New("deletion confirmation is required when --force is not set")
		}

		confirmed, err := opts.Confirm(moduleInfo)
		if err != nil {
			return DeleteResult{}, err
		}
		if !confirmed {
			return DeleteResult{}, ErrDeleteModuleCanceled
		}
	}

	packageNames := moduleInfo.PackageNames()
	manifestPaths, err := d.collectManifestPaths(modulesRoot, packageNames, opts)
	if err != nil {
		return DeleteResult{}, err
	}

	updated := make(map[string]struct{}, len(manifestPaths))
	for _, manifestPath := range manifestPaths {
		if err := ctx.Err(); err != nil {
			return DeleteResult{}, err
		}

		changed, err := d.cleanupManifest(manifestPath, packageNames)
		if err != nil {
			return DeleteResult{}, err
		}
		if changed {
			updated[manifestPath] = struct{}{}
		}
	}

	deletedPaths := make([]string, 0, len(packageNames))
	for _, packageName := range packageNames {
		if err := ctx.Err(); err != nil {
			return DeleteResult{}, err
		}

		packagePath := filepath.Join(modulesRoot, packageName)
		if err := d.removeAll(packagePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return DeleteResult{}, fmt.Errorf("remove module package %q: %w", packagePath, err)
		}
		deletedPaths = append(deletedPaths, packagePath)
	}

	updatedManifests := make([]string, 0, len(updated))
	for path := range updated {
		updatedManifests = append(updatedManifests, path)
	}
	sort.Strings(updatedManifests)

	return DeleteResult{
		Module:           moduleInfo,
		DeletedPaths:     deletedPaths,
		UpdatedManifests: updatedManifests,
	}, nil
}

func (d *Deleter) lookupModule(ctx context.Context, modulesRoot string, moduleName string) (ModuleInfo, error) {
	modulesList, err := d.lister.List(ctx, modulesRoot)
	if err != nil {
		return ModuleInfo{}, err
	}

	for _, module := range modulesList {
		if module.Name == moduleName {
			return module, nil
		}
	}

	return ModuleInfo{}, fmt.Errorf("module %q was not found in %q", moduleName, modulesRoot)
}

func (d *Deleter) collectManifestPaths(
	modulesRoot string,
	deletedPackageNames []string,
	opts DeleteOptions,
) ([]string, error) {
	deleted := make(map[string]struct{}, len(deletedPackageNames))
	for _, name := range deletedPackageNames {
		deleted[name] = struct{}{}
	}

	paths := make(map[string]struct{})
	projectManifestPath := resolveDeletePath(opts.ProjectRoot, opts.ProjectManifestPath, defaultProjectManifestPath)
	if exists, err := d.pathExists(projectManifestPath); err != nil {
		return nil, fmt.Errorf("stat project manifest %q: %w", projectManifestPath, err)
	} else if exists {
		paths[projectManifestPath] = struct{}{}
	}

	rootPackagePath := resolveDeletePath(opts.ProjectRoot, opts.RootPackageManifestPath, defaultRootPackagePath)
	if exists, err := d.pathExists(rootPackagePath); err != nil {
		return nil, fmt.Errorf("stat root package manifest %q: %w", rootPackagePath, err)
	} else if exists {
		paths[rootPackagePath] = struct{}{}
	}

	entries, err := d.readDir(modulesRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("modules directory %q does not exist", modulesRoot)
		}
		return nil, fmt.Errorf("scan modules directory %q: %w", modulesRoot, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		if _, skip := deleted[name]; skip {
			continue
		}

		manifestPath := filepath.Join(modulesRoot, name, modulePackageManifestName)
		exists, err := d.pathExists(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("stat module manifest %q: %w", manifestPath, err)
		}
		if !exists {
			continue
		}

		paths[manifestPath] = struct{}{}
	}

	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	sort.Strings(result)

	return result, nil
}

func (d *Deleter) cleanupManifest(path string, packageNames []string) (bool, error) {
	modified := false
	manifest, manifestErr := d.readManifest(path)
	if manifestErr == nil {
		dependencies := manifestNameSet(manifest.Dependencies)
		edits := make([]tuistproj.ManifestEdit, 0, len(packageNames))
		for _, packageName := range packageNames {
			if _, ok := dependencies[packageName]; !ok {
				continue
			}
			edits = append(edits, tuistproj.ManifestEdit{
				Type: tuistproj.RemoveDependency,
				Name: packageName,
			})
		}

		if len(edits) > 0 {
			if err := d.applyManifestEdits(path, edits...); err != nil {
				return false, fmt.Errorf("remove dependency references from %q: %w", path, err)
			}
			modified = true
		}
	}

	changed, err := removeManifestLinesByName(path, packageNames)
	if err != nil {
		return false, err
	}
	if changed {
		modified = true
	}

	return modified, nil
}

func (d *Deleter) pathExists(path string) (bool, error) {
	_, err := d.stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func resolveDeleteModulesRoot(projectRoot string, modulesPath string) string {
	normalizedModulesPath := normalizeModulesPath(modulesPath)
	if filepath.IsAbs(normalizedModulesPath) {
		return filepath.Clean(normalizedModulesPath)
	}

	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return filepath.Clean(normalizedModulesPath)
	}
	return filepath.Clean(filepath.Join(root, normalizedModulesPath))
}

func resolveDeletePath(projectRoot string, path string, fallback string) string {
	resolved := strings.TrimSpace(path)
	if resolved == "" {
		resolved = fallback
	}

	if filepath.IsAbs(resolved) {
		return filepath.Clean(resolved)
	}

	root := strings.TrimSpace(projectRoot)
	if root == "" {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(filepath.Join(root, resolved))
}

func manifestNameSet(items []tuistproj.ManifestItem) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return set
}

func removeManifestLinesByName(path string, packageNames []string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read manifest file %q: %w", path, err)
	}

	lines := strings.Split(string(content), "\n")
	filtered := make([]string, 0, len(lines))
	changed := false
	for _, line := range lines {
		if shouldRemoveManifestLine(line, packageNames) {
			changed = true
			continue
		}
		filtered = append(filtered, line)
	}

	if !changed {
		return false, nil
	}

	updated := strings.Join(filtered, "\n")
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write manifest file %q: %w", path, err)
	}

	return true, nil
}

func shouldRemoveManifestLine(line string, packageNames []string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "//") {
		return false
	}

	if !strings.Contains(line, "\"") {
		return false
	}

	for _, packageName := range packageNames {
		quotedName := `"` + packageName + `"`
		if !strings.Contains(line, quotedName) {
			continue
		}

		if strings.Contains(line, "path:") ||
			strings.Contains(line, "product:") ||
			strings.Contains(line, "package:") ||
			strings.Contains(line, "name:") ||
			strings.Contains(line, "targets:") ||
			strings.Contains(line, ".package(") ||
			strings.Contains(line, ".product(") ||
			strings.Contains(line, ".target(") ||
			strings.Contains(line, ".library(") {
			return true
		}
	}

	return false
}
