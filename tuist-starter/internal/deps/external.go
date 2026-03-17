package deps

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const externalDependencyScopeRoot = "root"

var (
	externalVersionInputPattern = regexp.MustCompile(`(?i)^(from|exact|branch|revision)\s*:\s*(.+)$`)
	externalURLArgumentPattern  = regexp.MustCompile(`\burl\s*:\s*"([^"]+)"`)

	externalRequirementExtractors = []struct {
		kind    string
		pattern *regexp.Regexp
	}{
		{kind: "from", pattern: regexp.MustCompile(`\bfrom\s*:\s*"([^"]+)"`)},
		{kind: "exact", pattern: regexp.MustCompile(`\bexact\s*:\s*"([^"]+)"`)},
		{kind: "branch", pattern: regexp.MustCompile(`\bbranch\s*:\s*"([^"]+)"`)},
		{kind: "revision", pattern: regexp.MustCompile(`\brevision\s*:\s*"([^"]+)"`)},
	}
)

// ExternalDependency is one remote package dependency discovered in manifests.
type ExternalDependency struct {
	Scope        string
	PackageName  string
	URL          string
	Requirement  string
	ManifestPath string
}

type externalDependencySpec struct {
	packageName string
	url         string
	version     externalVersionRequirement
}

type externalVersionRequirement struct {
	kind  string
	value string
}

type externalDependencyManifest struct {
	scope string
	path  string
}

// AddExternalDep adds remote SPM dependency to project Package.swift and optional module Package.swift.
func AddExternalDep(url string, version string, packageName string, targetModule string, modulesPath string) error {
	trimmedURL := strings.TrimSpace(url)
	if trimmedURL == "" {
		return fmt.Errorf("external dependency URL is required")
	}

	resolvedPackageName, err := resolveExternalPackageName(packageName, trimmedURL)
	if err != nil {
		return err
	}

	versionRequirement, err := parseExternalVersionRequirement(version)
	if err != nil {
		return err
	}

	spec := externalDependencySpec{
		packageName: resolvedPackageName,
		url:         trimmedURL,
		version:     versionRequirement,
	}

	modulesRoot := normalizeModulesPath(modulesPath)
	projectManifestPath := rootManifestPathForModules(modulesRoot)
	if exists, err := pathExists(projectManifestPath); err != nil {
		return fmt.Errorf("stat project manifest %q: %w", projectManifestPath, err)
	} else if !exists {
		return fmt.Errorf("project manifest %q was not found", projectManifestPath)
	}

	if err := addExternalDependencyToManifest(projectManifestPath, spec); err != nil {
		return fmt.Errorf(
			"add external dependency %q to project manifest: %w",
			spec.packageName,
			err,
		)
	}
	if err := tuistproj.EnsureFrameworkProductTypes(projectManifestPath, spec.packageName); err != nil {
		return fmt.Errorf(
			"ensure framework product type for %q in project manifest: %w",
			spec.packageName,
			err,
		)
	}

	targetModuleName := strings.TrimSpace(targetModule)
	if targetModuleName == "" {
		return nil
	}

	moduleName, err := normalizeInterfaceModuleName(targetModuleName)
	if err != nil {
		return rollbackExternalDependencyAdd(projectManifestPath, spec.packageName, err)
	}

	moduleManifestPath, err := requireModuleManifest(moduleName, modulesRoot)
	if err != nil {
		return rollbackExternalDependencyAdd(projectManifestPath, spec.packageName, err)
	}

	if err := addExternalDependencyToModuleManifest(moduleManifestPath, spec); err != nil {
		rollbackErr := removeExternalDependencyFromManifest(projectManifestPath, spec.packageName, true)
		if rollbackErr != nil {
			return fmt.Errorf(
				"add external dependency %q to module %q: %w (rollback failed: %v)",
				spec.packageName,
				moduleName,
				err,
				rollbackErr,
			)
		}
		return fmt.Errorf("add external dependency %q to module %q: %w", spec.packageName, moduleName, err)
	}

	return nil
}

// RemoveExternalDep removes external dependency from project Package.swift and module Package.swift files.
func RemoveExternalDep(packageName string, modulesPath string) error {
	resolvedPackageName := strings.TrimSpace(packageName)
	if resolvedPackageName == "" {
		return fmt.Errorf("external package name is required")
	}

	modulesRoot := normalizeModulesPath(modulesPath)
	projectManifestPath := rootManifestPathForModules(modulesRoot)
	if exists, err := pathExists(projectManifestPath); err != nil {
		return fmt.Errorf("stat project manifest %q: %w", projectManifestPath, err)
	} else if !exists {
		return fmt.Errorf("project manifest %q was not found", projectManifestPath)
	}

	if err := removeExternalDependencyFromManifest(projectManifestPath, resolvedPackageName, true); err != nil {
		return fmt.Errorf(
			"remove external dependency %q from project manifest: %w",
			resolvedPackageName,
			err,
		)
	}
	if err := tuistproj.RemoveFrameworkProductTypes(projectManifestPath, resolvedPackageName); err != nil {
		return fmt.Errorf(
			"remove framework product type for %q from project manifest: %w",
			resolvedPackageName,
			err,
		)
	}

	moduleManifests, err := listModuleManifestPaths(modulesRoot)
	if err != nil {
		return err
	}

	for _, moduleManifestPath := range moduleManifests {
		if err := removeExternalDependencyFromManifest(moduleManifestPath, resolvedPackageName, false); err != nil {
			return fmt.Errorf(
				"remove external dependency %q from module manifest %q: %w",
				resolvedPackageName,
				moduleManifestPath,
				err,
			)
		}
	}

	return nil
}

// ListExternalDeps scans root + module manifests for .package(url: "...") dependencies.
func ListExternalDeps(modulesPath string) ([]ExternalDependency, error) {
	modulesRoot := normalizeModulesPath(modulesPath)
	manifests, err := listExternalDependencyManifests(modulesRoot)
	if err != nil {
		return nil, err
	}

	dependencies := make([]ExternalDependency, 0)
	for _, manifestRef := range manifests {
		manifest, err := tuistproj.ReadManifestFile(manifestRef.path)
		if err != nil {
			return nil, fmt.Errorf("read manifest %q: %w", manifestRef.path, err)
		}

		for _, item := range manifest.Dependencies {
			if !isExternalDependencyContent(item.Content) {
				continue
			}

			url := extractExternalDependencyURL(item.Content)
			if url == "" {
				continue
			}

			resolvedPackageName := strings.TrimSpace(item.Name)
			if resolvedPackageName == "" {
				inferredPackageName, inferErr := resolveExternalPackageName("", url)
				if inferErr == nil {
					resolvedPackageName = inferredPackageName
				}
			}

			requirement := extractExternalDependencyRequirement(item.Content)
			if requirement == "" {
				requirement = "-"
			}

			dependencies = append(dependencies, ExternalDependency{
				Scope:        manifestRef.scope,
				PackageName:  resolvedPackageName,
				URL:          url,
				Requirement:  requirement,
				ManifestPath: manifestRef.path,
			})
		}
	}

	sort.Slice(dependencies, func(i, j int) bool {
		left := dependencies[i]
		right := dependencies[j]

		if externalDependencyScopeRank(left.Scope) != externalDependencyScopeRank(right.Scope) {
			return externalDependencyScopeRank(left.Scope) < externalDependencyScopeRank(right.Scope)
		}
		if left.Scope != right.Scope {
			return left.Scope < right.Scope
		}
		if left.PackageName != right.PackageName {
			return left.PackageName < right.PackageName
		}
		if left.URL != right.URL {
			return left.URL < right.URL
		}
		return left.Requirement < right.Requirement
	})

	return dependencies, nil
}

func addExternalDependencyToManifest(manifestPath string, spec externalDependencySpec) error {
	edit := tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    spec.packageName,
		Content: spec.manifestEntry(),
	}
	return tuistproj.ApplyManifestEditsToFile(manifestPath, edit)
}

func addExternalDependencyToModuleManifest(manifestPath string, spec externalDependencySpec) error {
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read module manifest %q: %w", manifestPath, err)
	}

	updated, err := tuistproj.ApplyManifestEdits(string(payload), tuistproj.ManifestEdit{
		Type:    tuistproj.AddDependency,
		Name:    spec.packageName,
		Content: spec.manifestEntry(),
	})
	if err != nil {
		return err
	}

	updated, err = addTargetProductDependency(updated, spec.packageName)
	if err != nil {
		return err
	}

	if err := os.WriteFile(manifestPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write module manifest %q: %w", manifestPath, err)
	}
	return nil
}

func removeExternalDependencyFromManifest(manifestPath string, packageName string, strict bool) error {
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest %q: %w", manifestPath, err)
	}

	updated := string(payload)
	modified := false

	updated, removedDependency, err := removeDependencyItemIf(updated, func(item tuistproj.ManifestItem) bool {
		if strings.TrimSpace(item.Name) != packageName {
			return false
		}
		return isExternalDependencyContent(item.Content)
	})
	if err != nil {
		return err
	}
	if strict && !removedDependency {
		return fmt.Errorf("external dependency %q was not found", packageName)
	}
	if removedDependency {
		modified = true
	}

	updated, removedTargetDependency, err := removeTargetProductDependencyIfPresent(updated, packageName)
	if err != nil {
		return err
	}
	if removedTargetDependency {
		modified = true
	}

	if !modified {
		return nil
	}

	if err := os.WriteFile(manifestPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write manifest %q: %w", manifestPath, err)
	}
	return nil
}

func removeDependencyItemIf(
	content string,
	match func(item tuistproj.ManifestItem) bool,
) (string, bool, error) {
	manifest, err := tuistproj.ParseManifest(content)
	if err != nil {
		return "", false, err
	}

	removeItemIndex := -1
	for index, dependency := range manifest.Dependencies {
		if !match(dependency) {
			continue
		}
		removeItemIndex = index
		break
	}
	if removeItemIndex == -1 {
		return content, false, nil
	}

	removeItem := manifest.Dependencies[removeItemIndex]
	lines, hasTrailingNewline := splitEditableLines(content)

	start := removeItem.StartLine - 1
	end := removeItem.EndLine - 1
	if start < 0 || end < start || end >= len(lines) {
		return "", false, fmt.Errorf(
			"invalid removal range %d:%d for dependency %q",
			removeItem.StartLine,
			removeItem.EndLine,
			removeItem.Name,
		)
	}

	updatedLines := make([]string, 0, len(lines)-(end-start+1))
	updatedLines = append(updatedLines, lines[:start]...)
	updatedLines = append(updatedLines, lines[end+1:]...)

	return joinEditableLines(updatedLines, hasTrailingNewline), true, nil
}

func removeTargetProductDependencyIfPresent(content string, packageName string) (string, bool, error) {
	lines, hasTrailingNewline := splitEditableLines(content)
	arraySection, err := findTargetDependencyArray(lines)
	if err != nil {
		if strings.Contains(err.Error(), "target dependencies section not found") {
			return content, false, nil
		}
		return "", false, err
	}

	pattern := productDependencyPattern(packageName)
	removeLine := -1
	for lineIndex := arraySection.openLine + 1; lineIndex < arraySection.closeLine; lineIndex++ {
		if !pattern.MatchString(lines[lineIndex]) {
			continue
		}
		removeLine = lineIndex
		break
	}
	if removeLine == -1 {
		return content, false, nil
	}

	updatedLines := make([]string, 0, len(lines)-1)
	updatedLines = append(updatedLines, lines[:removeLine]...)
	updatedLines = append(updatedLines, lines[removeLine+1:]...)

	return joinEditableLines(updatedLines, hasTrailingNewline), true, nil
}

func listExternalDependencyManifests(modulesRoot string) ([]externalDependencyManifest, error) {
	manifests := make([]externalDependencyManifest, 0)

	rootManifestPath := rootManifestPathForModules(modulesRoot)
	if exists, err := pathExists(rootManifestPath); err != nil {
		return nil, fmt.Errorf("stat project manifest %q: %w", rootManifestPath, err)
	} else if exists {
		manifests = append(manifests, externalDependencyManifest{
			scope: externalDependencyScopeRoot,
			path:  rootManifestPath,
		})
	}

	moduleManifestPaths, err := listModuleManifestPaths(modulesRoot)
	if err != nil {
		return nil, err
	}

	for _, moduleManifestPath := range moduleManifestPaths {
		moduleName := filepath.Base(filepath.Dir(moduleManifestPath))
		manifests = append(manifests, externalDependencyManifest{
			scope: moduleName,
			path:  moduleManifestPath,
		})
	}

	return manifests, nil
}

func listModuleManifestPaths(modulesRoot string) ([]string, error) {
	entries, err := os.ReadDir(modulesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan modules directory %q: %w", modulesRoot, err)
	}

	moduleManifestPaths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		moduleName := strings.TrimSpace(entry.Name())
		if moduleName == "" || strings.HasPrefix(moduleName, ".") {
			continue
		}

		moduleManifestPath := filepath.Join(modulesRoot, moduleName, moduleManifestName)
		exists, err := pathExists(moduleManifestPath)
		if err != nil {
			return nil, fmt.Errorf("stat module manifest %q: %w", moduleManifestPath, err)
		}
		if !exists {
			continue
		}

		moduleManifestPaths = append(moduleManifestPaths, moduleManifestPath)
	}

	sort.Strings(moduleManifestPaths)
	return moduleManifestPaths, nil
}

func resolveExternalPackageName(packageName string, url string) (string, error) {
	trimmedPackageName := strings.TrimSpace(packageName)
	if trimmedPackageName != "" {
		return trimmedPackageName, nil
	}

	trimmedURL := strings.TrimSpace(url)
	if trimmedURL == "" {
		return "", fmt.Errorf("package name or URL is required")
	}

	cleanedURL := strings.TrimSuffix(trimmedURL, "/")
	lastSlash := strings.LastIndex(cleanedURL, "/")
	segment := cleanedURL
	if lastSlash >= 0 && lastSlash+1 < len(cleanedURL) {
		segment = cleanedURL[lastSlash+1:]
	}

	if colon := strings.LastIndex(segment, ":"); colon >= 0 && colon+1 < len(segment) {
		segment = segment[colon+1:]
	}

	if queryIndex := strings.IndexAny(segment, "?#"); queryIndex >= 0 {
		segment = segment[:queryIndex]
	}

	segment = strings.TrimSuffix(segment, ".git")
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return "", fmt.Errorf("cannot derive package name from URL %q", trimmedURL)
	}

	return segment, nil
}

func parseExternalVersionRequirement(raw string) (externalVersionRequirement, error) {
	trimmedVersion := strings.TrimSpace(raw)
	if trimmedVersion == "" {
		return externalVersionRequirement{}, fmt.Errorf("external dependency version is required")
	}

	match := externalVersionInputPattern.FindStringSubmatch(trimmedVersion)
	if len(match) == 3 {
		return newExternalVersionRequirement(strings.ToLower(strings.TrimSpace(match[1])), match[2])
	}

	return newExternalVersionRequirement("from", trimmedVersion)
}

func newExternalVersionRequirement(kind string, rawValue string) (externalVersionRequirement, error) {
	switch kind {
	case "from", "exact", "branch", "revision":
	default:
		return externalVersionRequirement{}, fmt.Errorf("unsupported external dependency version kind %q", kind)
	}

	value := strings.TrimSpace(rawValue)
	value = strings.TrimSuffix(value, ",")
	value = strings.TrimSpace(value)
	value = trimOptionalQuotes(value)
	if value == "" {
		return externalVersionRequirement{}, fmt.Errorf("version value for %q is required", kind)
	}

	return externalVersionRequirement{
		kind:  kind,
		value: value,
	}, nil
}

func trimOptionalQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	first := value[0]
	last := value[len(value)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

func extractExternalDependencyURL(content string) string {
	match := externalURLArgumentPattern.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func extractExternalDependencyRequirement(content string) string {
	for _, extractor := range externalRequirementExtractors {
		match := extractor.pattern.FindStringSubmatch(content)
		if len(match) < 2 {
			continue
		}
		value := strings.TrimSpace(match[1])
		if value == "" {
			continue
		}
		return fmt.Sprintf(`%s: "%s"`, extractor.kind, escapeSwiftString(value))
	}
	return ""
}

func isExternalDependencyContent(content string) bool {
	return externalURLArgumentPattern.MatchString(content)
}

func rollbackExternalDependencyAdd(projectManifestPath string, packageName string, reason error) error {
	rollbackErr := removeExternalDependencyFromManifest(projectManifestPath, packageName, true)
	if rollbackErr != nil {
		return fmt.Errorf("%w (rollback failed: %v)", reason, rollbackErr)
	}
	if cleanupErr := tuistproj.RemoveFrameworkProductTypes(projectManifestPath, packageName); cleanupErr != nil {
		return fmt.Errorf("%w (framework cleanup failed: %v)", reason, cleanupErr)
	}
	return reason
}

func rootManifestPathForModules(modulesRoot string) string {
	return filepath.Join(filepath.Dir(modulesRoot), moduleManifestName)
}

func externalDependencyScopeRank(scope string) int {
	if scope == externalDependencyScopeRoot {
		return 0
	}
	return 1
}

func (s externalDependencySpec) manifestEntry() string {
	return fmt.Sprintf(
		`.package(name: "%s", url: "%s", %s)`,
		escapeSwiftString(s.packageName),
		escapeSwiftString(s.url),
		s.version.clause(),
	)
}

func (r externalVersionRequirement) clause() string {
	return fmt.Sprintf(`%s: "%s"`, r.kind, escapeSwiftString(r.value))
}

func escapeSwiftString(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return escaped
}
