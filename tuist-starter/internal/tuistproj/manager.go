package tuistproj

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/components"
)

const (
	defaultManagerRootDir         = "."
	defaultManagerModulesDir      = "Packages"
	defaultManagerProjectPath     = "Project.swift"
	defaultManagerWorkspace       = "Workspace.swift"
	defaultManagerPlatform        = "iOS(.v17)"
	swiftManifestFileName         = "Package.swift"
	swiftSourceDirectoryName      = "Sources"
	swiftTestsDirectoryName       = "Tests"
	moduleImplSuffix              = "Impl"
	productModuleTypeFeature      = "feature"
	productModuleTypeKit          = "kit"
	productModuleTypeUI           = "ui"
	productModuleTypeShared       = "shared"
	productModuleTypeProduct      = "product"
	productModuleTypeReluxFeature = "relux-feature"
	manifestOperationSeparator    = "_"
)

var (
	swiftIdentifierRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	errNilRunner         = errors.New("tuist runner is not configured")
)

type modulePackageSpec struct {
	ModuleName   string
	PackageName  string
	TargetName   string
	PackageType  PackageType
	ExternalDeps []ExternalProductDep
}

// ManagerOption configures TuistProjectManager.
type ManagerOption func(*TuistProjectManager)

// TuistProjectManager manages Tuist project scaffolding and command delegation.
type TuistProjectManager struct {
	runner Runner

	rootDir     string
	modulesDir  string
	projectPath string
	workspace   string
	platform    string
	mkdirAll    func(path string, perm os.FileMode) error
	writeFile   func(name string, data []byte, perm os.FileMode) error
	removeAll   func(path string) error
	stat        func(name string) (os.FileInfo, error)
}

var _ components.TuistProjectManager = (*TuistProjectManager)(nil)

// NewTuistProjectManager creates a TuistProjectManager with sensible defaults.
func NewTuistProjectManager(options ...ManagerOption) *TuistProjectManager {
	manager := &TuistProjectManager{
		runner:      NewTuistRunner(),
		rootDir:     defaultManagerRootDir,
		modulesDir:  defaultManagerModulesDir,
		projectPath: defaultManagerProjectPath,
		workspace:   defaultManagerWorkspace,
		platform:    defaultManagerPlatform,
		mkdirAll:    os.MkdirAll,
		writeFile:   os.WriteFile,
		removeAll:   os.RemoveAll,
		stat:        os.Stat,
	}

	for _, option := range options {
		if option != nil {
			option(manager)
		}
	}

	return manager
}

// NewManager is an alias constructor for TuistProjectManager.
func NewManager(options ...ManagerOption) *TuistProjectManager {
	return NewTuistProjectManager(options...)
}

// WithRunner overrides the command runner used for tuist invocations.
func WithRunner(runner Runner) ManagerOption {
	return func(m *TuistProjectManager) {
		if runner != nil {
			m.runner = runner
		}
	}
}

// WithRootDir sets the project root directory used for relative paths.
func WithRootDir(rootDir string) ManagerOption {
	return func(m *TuistProjectManager) {
		trimmed := strings.TrimSpace(rootDir)
		if trimmed != "" {
			m.rootDir = trimmed
		}
	}
}

// WithModulesDir sets the root directory where modules are scaffolded.
func WithModulesDir(modulesDir string) ManagerOption {
	return func(m *TuistProjectManager) {
		trimmed := strings.TrimSpace(modulesDir)
		if trimmed != "" {
			m.modulesDir = trimmed
		}
	}
}

// WithProjectManifestPath overrides the project manifest path used for ref updates.
func WithProjectManifestPath(path string) ManagerOption {
	return func(m *TuistProjectManager) {
		trimmed := strings.TrimSpace(path)
		if trimmed != "" {
			m.projectPath = trimmed
		}
	}
}

// WithWorkspaceManifestPath overrides the workspace manifest path used for ref updates.
func WithWorkspaceManifestPath(path string) ManagerOption {
	return func(m *TuistProjectManager) {
		trimmed := strings.TrimSpace(path)
		if trimmed != "" {
			m.workspace = trimmed
		}
	}
}

// WithPackagePlatform sets the SwiftPM platform used when generating Package.swift.
func WithPackagePlatform(platform string) ManagerOption {
	return func(m *TuistProjectManager) {
		trimmed := strings.TrimSpace(platform)
		if trimmed != "" {
			m.platform = trimmed
		}
	}
}

// Generate delegates to `tuist generate`.
func (m *TuistProjectManager) Generate(ctx context.Context, opts components.GenerateOpts) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	runner, err := m.requireRunner()
	if err != nil {
		return err
	}

	args := make([]string, 0, 3)
	if opts.Open {
		args = append(args, "--open")
	} else {
		args = append(args, "--no-open")
	}
	if configPath := strings.TrimSpace(opts.ConfigPath); configPath != "" {
		args = append(args, "--path", configPath)
	}

	if _, err := runner.Run(ctx, CommandGenerate, args...); err != nil {
		return fmt.Errorf("run tuist generate: %w", err)
	}

	return nil
}

// Install delegates to `tuist install`.
func (m *TuistProjectManager) Install(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	runner, err := m.requireRunner()
	if err != nil {
		return err
	}

	if _, err := runner.Run(ctx, CommandInstall); err != nil {
		return fmt.Errorf("run tuist install: %w", err)
	}

	return nil
}

// Graph delegates to `tuist graph` and validates JSON payload when requested.
func (m *TuistProjectManager) Graph(ctx context.Context, format string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	runner, err := m.requireRunner()
	if err != nil {
		return nil, err
	}

	graphFormat := normalizeGraphFormat(format)
	result, err := runner.Run(ctx, CommandGraph, "--format", graphFormat)
	if err != nil {
		return nil, fmt.Errorf("run tuist graph --format %s: %w", graphFormat, err)
	}

	output := strings.TrimSpace(result.Stdout)
	if output == "" {
		output = strings.TrimSpace(result.Stderr)
	}

	if graphFormat == "json" {
		if _, parseErr := ParseGraphJSON([]byte(output)); parseErr != nil {
			return nil, parseErr
		}
	}

	return []byte(output), nil
}

// Clean delegates to `tuist clean`.
func (m *TuistProjectManager) Clean(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	runner, err := m.requireRunner()
	if err != nil {
		return err
	}

	if _, err := runner.Run(ctx, CommandClean); err != nil {
		return fmt.Errorf("run tuist clean: %w", err)
	}

	return nil
}

// CreateModule scaffolds module packages under Packages/ (or configured path).
func (m *TuistProjectManager) CreateModule(ctx context.Context, opts components.ModuleOpts) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	moduleName, err := normalizeModuleName(opts.Name)
	if err != nil {
		return err
	}

	moduleType := normalizeModuleType(opts.Type)
	if moduleType == "" {
		return errors.New("module type is required")
	}

	externalDeps := convertComponentExternalDeps(opts.ExternalDeps)

	specs := buildModulePackageSpecs(moduleName, moduleType)
	for i := range specs {
		specs[i].ExternalDeps = externalDeps
	}
	createdPaths := make([]string, 0, len(specs))
	for _, spec := range specs {
		packagePath, createErr := m.createModulePackage(spec)
		if createErr != nil {
			m.rollbackCreatedPackages(createdPaths)
			return createErr
		}
		createdPaths = append(createdPaths, packagePath)
	}

	// Write .module-type marker in the interface package root for IoC registry grouping.
	moduleTypeFilePath := filepath.Join(m.modulesRootPath(), moduleName, ".module-type")
	if writeErr := m.writeFile(moduleTypeFilePath, []byte(moduleType+"\n"), 0o644); writeErr != nil {
		m.rollbackCreatedPackages(createdPaths)
		return fmt.Errorf("write .module-type: %w", writeErr)
	}

	packageNames := make([]string, 0, len(specs))
	for _, spec := range specs {
		packageNames = append(packageNames, spec.PackageName)
	}
	if err := m.addModuleReferences(packageNames); err != nil {
		m.rollbackCreatedPackages(createdPaths)
		return err
	}

	if len(externalDeps) > 0 {
		if err := m.addExternalDepsToRootManifest(externalDeps); err != nil {
			m.rollbackCreatedPackages(createdPaths)
			return err
		}
	}

	return nil
}

// DeleteModule removes module package directories and best-effort manifest references.
func (m *TuistProjectManager) DeleteModule(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	moduleName, err := normalizeModuleName(name)
	if err != nil {
		return err
	}

	packageNames := []string{moduleName, moduleName + moduleImplSuffix}
	for _, packageName := range packageNames {
		path := filepath.Join(m.modulesRootPath(), packageName)
		if err := m.removeAll(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove module package %q: %w", path, err)
		}
	}

	if err := m.removeModuleReferences(packageNames); err != nil {
		return err
	}

	return nil
}

// EditManifest applies supported manifest operations.
func (m *TuistProjectManager) EditManifest(ctx context.Context, path string, edits []components.ManifestEdit) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if len(edits) == 0 {
		return nil
	}

	manifestPath := m.resolveManifestPath(path)
	converted := make([]ManifestEdit, 0, len(edits))
	for _, edit := range edits {
		op := normalizeManifestOperation(edit.Operation)
		if op == "deletemodule" {
			moduleName, err := normalizeModuleName(edit.Path)
			if err != nil {
				return err
			}
			if err := m.removeManifestReferences(manifestPath, []string{moduleName, moduleName + moduleImplSuffix}, true); err != nil {
				return err
			}
			continue
		}

		mapped, err := convertComponentManifestEdit(op, edit)
		if err != nil {
			return err
		}
		converted = append(converted, mapped)
	}

	if len(converted) == 0 {
		return nil
	}

	if err := ApplyManifestEditsToFile(manifestPath, converted...); err != nil {
		return fmt.Errorf("apply manifest edits to %q: %w", manifestPath, err)
	}

	return nil
}

func (m *TuistProjectManager) requireRunner() (Runner, error) {
	if m.runner == nil {
		return nil, errNilRunner
	}
	return m.runner, nil
}

func (m *TuistProjectManager) createModulePackage(spec modulePackageSpec) (string, error) {
	packagePath := filepath.Join(m.modulesRootPath(), spec.PackageName)
	exists, err := m.pathExists(packagePath)
	if err != nil {
		return "", fmt.Errorf("stat module package %q: %w", packagePath, err)
	}
	if exists {
		return "", fmt.Errorf("module package already exists: %q", packagePath)
	}

	if err := m.mkdirAll(packagePath, 0o755); err != nil {
		return "", fmt.Errorf("create module package directory %q: %w", packagePath, err)
	}

	manifest, err := GeneratePackageSwift(PackageGenerationInput{
		ModuleName:   spec.ModuleName,
		Type:         spec.PackageType,
		ExternalDeps: spec.ExternalDeps,
		Platform:     m.platform,
	})
	if err != nil {
		return "", fmt.Errorf("generate Package.swift for %q: %w", spec.PackageName, err)
	}

	manifestPath := filepath.Join(packagePath, swiftManifestFileName)
	if err := m.writeFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return "", fmt.Errorf("write package manifest %q: %w", manifestPath, err)
	}

	sourceDir := filepath.Join(packagePath, swiftSourceDirectoryName)
	if err := m.mkdirAll(sourceDir, 0o755); err != nil {
		return "", fmt.Errorf("create sources directory %q: %w", sourceDir, err)
	}

	testsDir := filepath.Join(packagePath, swiftTestsDirectoryName, spec.TargetName+"Tests")
	if err := m.mkdirAll(testsDir, 0o755); err != nil {
		return "", fmt.Errorf("create tests directory %q: %w", testsDir, err)
	}

	return packagePath, nil
}

func (m *TuistProjectManager) rollbackCreatedPackages(paths []string) {
	for i := len(paths) - 1; i >= 0; i-- {
		_ = m.removeAll(paths[i])
	}
}

func (m *TuistProjectManager) addModuleReferences(packageNames []string) error {
	// Project.swift: .external(name: "PackageName")
	projectPath := m.resolvePath(m.projectPath)
	if err := m.addManifestDepsWithFormat(projectPath, packageNames, true, func(name string) string {
		return fmt.Sprintf(`.external(name: "%s")`, name)
	}); err != nil {
		return err
	}

	// Root Package.swift: .package(path: "Packages/PackageName")
	rootPkgPath := m.resolvePath("Package.swift")
	if err := m.addManifestDepsWithFormat(rootPkgPath, packageNames, false, func(name string) string {
		return fmt.Sprintf(`.package(path: "%s")`, m.packageReferencePath(name))
	}); err != nil {
		return err
	}

	// Workspace.swift: best-effort
	workspacePath := m.resolvePath(m.workspace)
	if err := m.addManifestDepsWithFormat(workspacePath, packageNames, false, func(name string) string {
		return fmt.Sprintf(`.package(path: "%s")`, m.packageReferencePath(name))
	}); err != nil {
		return err
	}

	return nil
}

func (m *TuistProjectManager) removeModuleReferences(packageNames []string) error {
	projectPath := m.resolvePath(m.projectPath)
	if err := m.removeManifestReferences(projectPath, packageNames, true); err != nil {
		return err
	}

	workspacePath := m.resolvePath(m.workspace)
	if err := m.removeManifestReferences(workspacePath, packageNames, false); err != nil {
		return err
	}

	return nil
}

func (m *TuistProjectManager) addManifestDepsWithFormat(path string, packageNames []string, strict bool, formatFn func(string) string) error {
	exists, err := m.pathExists(path)
	if err != nil {
		if strict {
			return fmt.Errorf("stat manifest %q: %w", path, err)
		}
		return nil
	}
	if !exists {
		return nil
	}

	manifest, err := ReadManifestFile(path)
	if err != nil {
		if strict {
			return fmt.Errorf("read manifest %q: %w", path, err)
		}
		return nil
	}

	existing := manifestItemNameSet(manifest.Dependencies)
	edits := make([]ManifestEdit, 0, len(packageNames))
	for _, packageName := range uniqueNonEmpty(packageNames) {
		if _, ok := existing[packageName]; ok {
			continue
		}
		edits = append(edits, ManifestEdit{
			Type:    AddDependency,
			Name:    packageName,
			Content: formatFn(packageName),
		})
	}

	if len(edits) == 0 {
		return nil
	}

	if err := ApplyManifestEditsToFile(path, edits...); err != nil {
		if !strict || isManifestSectionNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("apply manifest dependency edits to %q: %w", path, err)
	}

	return nil
}

func (m *TuistProjectManager) removeManifestReferences(path string, packageNames []string, strict bool) error {
	exists, err := m.pathExists(path)
	if err != nil {
		if strict {
			return fmt.Errorf("stat manifest %q: %w", path, err)
		}
		return nil
	}
	if !exists {
		return nil
	}

	manifest, err := ReadManifestFile(path)
	if err != nil {
		if strict {
			return fmt.Errorf("read manifest %q: %w", path, err)
		}
		return nil
	}

	dependencies := manifestItemNameSet(manifest.Dependencies)
	targets := manifestItemNameSet(manifest.Targets)
	products := manifestItemNameSet(manifest.Products)

	edits := make([]ManifestEdit, 0, len(packageNames)*3)
	for _, name := range uniqueNonEmpty(packageNames) {
		if _, ok := dependencies[name]; ok {
			edits = append(edits, ManifestEdit{
				Type: RemoveDependency,
				Name: name,
			})
		}
		if _, ok := targets[name]; ok {
			edits = append(edits, ManifestEdit{
				Type: RemoveTarget,
				Name: name,
			})
		}
		if _, ok := products[name]; ok {
			edits = append(edits, ManifestEdit{
				Type: RemoveProduct,
				Name: name,
			})
		}
	}

	if len(edits) == 0 {
		return nil
	}

	if err := ApplyManifestEditsToFile(path, edits...); err != nil {
		if !strict || isManifestSectionNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("apply manifest cleanup edits to %q: %w", path, err)
	}

	return nil
}

func (m *TuistProjectManager) resolveManifestPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = m.projectPath
	}
	return m.resolvePath(trimmed)
}

func (m *TuistProjectManager) resolvePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return filepath.Clean(m.rootDir)
	}

	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed)
	}

	return filepath.Clean(filepath.Join(m.rootDir, trimmed))
}

func (m *TuistProjectManager) modulesRootPath() string {
	return m.resolvePath(m.modulesDir)
}

func (m *TuistProjectManager) packageReferencePath(packageName string) string {
	reference := filepath.Join(m.modulesDir, packageName)
	return filepath.ToSlash(reference)
}

func (m *TuistProjectManager) pathExists(path string) (bool, error) {
	_, err := m.stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func normalizeModuleName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", errors.New("module name is required")
	}
	if !swiftIdentifierRegex.MatchString(name) {
		return "", fmt.Errorf("module name %q is not a valid Swift identifier", name)
	}
	return name, nil
}

func normalizeModuleType(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func buildModulePackageSpecs(moduleName string, moduleType string) []modulePackageSpec {
	if isProductModuleType(moduleType) {
		return []modulePackageSpec{
			{
				ModuleName:  moduleName,
				PackageName: moduleName,
				TargetName:  moduleName,
				PackageType: PackageTypeInterface,
			},
			{
				ModuleName:  moduleName,
				PackageName: moduleName + moduleImplSuffix,
				TargetName:  moduleName + moduleImplSuffix,
				PackageType: PackageTypeImpl,
			},
		}
	}

	return []modulePackageSpec{
		{
			ModuleName:  moduleName,
			PackageName: moduleName,
			TargetName:  moduleName,
			PackageType: PackageTypeInterface,
		},
	}
}

func isProductModuleType(moduleType string) bool {
	switch normalizeModuleType(moduleType) {
	case productModuleTypeFeature:
		return true
	case productModuleTypeKit:
		return true
	case productModuleTypeUI:
		return true
	case productModuleTypeShared:
		return true
	case productModuleTypeProduct:
		return true
	case productModuleTypeReluxFeature:
		return true
	default:
		return false
	}
}

func manifestItemNameSet(items []ManifestItem) map[string]struct{} {
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

func uniqueNonEmpty(values []string) []string {
	unique := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		unique = append(unique, trimmed)
	}
	return unique
}

func normalizeGraphFormat(raw string) string {
	format := strings.ToLower(strings.TrimSpace(raw))
	if format == "" {
		return "json"
	}
	return format
}

func convertComponentManifestEdit(
	operation string,
	edit components.ManifestEdit,
) (ManifestEdit, error) {
	switch operation {
	case "addtarget":
		return buildManifestEdit(AddTarget, manifestSectionTargets, edit)
	case "removetarget":
		return buildManifestEdit(RemoveTarget, manifestSectionTargets, edit)
	case "adddependency":
		return buildManifestEdit(AddDependency, manifestSectionDependencies, edit)
	case "removedependency":
		return buildManifestEdit(RemoveDependency, manifestSectionDependencies, edit)
	case "addproduct":
		return buildManifestEdit(AddProduct, manifestSectionProducts, edit)
	case "removeproduct":
		return buildManifestEdit(RemoveProduct, manifestSectionProducts, edit)
	default:
		return ManifestEdit{}, fmt.Errorf("unsupported manifest operation %q", edit.Operation)
	}
}

func buildManifestEdit(
	editType EditType,
	section manifestSectionKind,
	componentEdit components.ManifestEdit,
) (ManifestEdit, error) {
	name := strings.TrimSpace(componentEdit.Path)
	content := strings.TrimSpace(componentEdit.Value)

	isAdd := false
	switch editType {
	case AddTarget, AddDependency, AddProduct:
		isAdd = true
	}

	if isAdd {
		if content == "" {
			return ManifestEdit{}, fmt.Errorf("operation %q requires Value", componentEdit.Operation)
		}
		if name == "" {
			name = extractManifestItemName(section, content)
		}
		if name == "" {
			return ManifestEdit{}, fmt.Errorf("operation %q requires Path or identifiable Value", componentEdit.Operation)
		}
		return ManifestEdit{
			Type:    editType,
			Name:    name,
			Content: content,
		}, nil
	}

	if name == "" {
		return ManifestEdit{}, fmt.Errorf("operation %q requires Path", componentEdit.Operation)
	}

	return ManifestEdit{
		Type: editType,
		Name: name,
	}, nil
}

func normalizeManifestOperation(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, "-", manifestOperationSeparator)
	value = strings.ReplaceAll(value, " ", manifestOperationSeparator)
	value = strings.ReplaceAll(value, manifestOperationSeparator, "")
	return value
}

func isManifestSectionNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	message := err.Error()
	return strings.Contains(message, "manifest section") && strings.Contains(message, "not found")
}

func convertComponentExternalDeps(deps []components.ExternalDep) []ExternalProductDep {
	if len(deps) == 0 {
		return nil
	}
	out := make([]ExternalProductDep, len(deps))
	for i, d := range deps {
		out[i] = ExternalProductDep{
			PackageName: d.PackageName,
			ProductName: d.ProductName,
			URL:         d.URL,
			Version:     d.Version,
		}
	}
	return out
}

func (m *TuistProjectManager) addExternalDepsToRootManifest(deps []ExternalProductDep) error {
	rootPkgPath := m.resolvePath("Package.swift")
	exists, err := m.pathExists(rootPkgPath)
	if err != nil || !exists {
		return nil
	}

	manifest, err := ReadManifestFile(rootPkgPath)
	if err != nil {
		return nil
	}

	existing := manifestItemNameSet(manifest.Dependencies)
	edits := make([]ManifestEdit, 0, len(deps))
	for _, dep := range deps {
		if _, ok := existing[dep.PackageName]; ok {
			continue
		}
		edits = append(edits, ManifestEdit{
			Type:    AddDependency,
			Name:    dep.PackageName,
			Content: fmt.Sprintf(`.package(url: "%s", %s)`, dep.URL, dep.Version),
		})
	}

	if len(edits) == 0 {
		return EnsureFrameworkProductTypes(rootPkgPath, collectExternalProductNames(deps)...)
	}

	if err := ApplyManifestEditsToFile(rootPkgPath, edits...); err != nil {
		if isManifestSectionNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("add external deps to root Package.swift: %w", err)
	}

	if err := EnsureFrameworkProductTypes(rootPkgPath, collectExternalProductNames(deps)...); err != nil {
		return fmt.Errorf("ensure framework product types in root Package.swift: %w", err)
	}

	return nil
}

func collectExternalProductNames(deps []ExternalProductDep) []string {
	out := make([]string, 0, len(deps))
	for _, dep := range deps {
		if strings.TrimSpace(dep.ProductName) == "" {
			continue
		}
		out = append(out, dep.ProductName)
	}
	return out
}
