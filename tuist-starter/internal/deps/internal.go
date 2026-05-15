package deps

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/modules"
	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

var dependenciesLabelPattern = regexp.MustCompile(`\bdependencies\s*:`)

// AddInternalDep wires moduleName -> dependsOn via interface package dependency.
func AddInternalDep(moduleName string, dependsOn string, modulesPath string) error {
	return addInternalDep(moduleName, dependsOn, modulesPath, defaultGraphSource)
}

func addInternalDep(moduleName string, dependsOn string, modulesPath string, source GraphSourceFunc) error {
	sourceModule, err := normalizeInterfaceModuleName(moduleName)
	if err != nil {
		return err
	}
	dependencyModule, err := normalizeInterfaceModuleName(dependsOn)
	if err != nil {
		return err
	}
	if sourceModule == dependencyModule {
		return fmt.Errorf("module %q cannot depend on itself", sourceModule)
	}

	modulesRoot := normalizeModulesPath(modulesPath)
	sourceManifestPath, err := requireModuleManifest(sourceModule, modulesRoot)
	if err != nil {
		return err
	}
	if _, err := requireModuleManifest(dependencyModule, modulesRoot); err != nil {
		return err
	}

	graph, err := buildDependencyGraph(modulesRoot, source)
	if err != nil {
		return err
	}
	if _, exists := graph[sourceModule]; !exists {
		return fmt.Errorf("module %q was not found in %q", sourceModule, modulesRoot)
	}
	if _, exists := graph[dependencyModule]; !exists {
		return fmt.Errorf("module %q was not found in %q", dependencyModule, modulesRoot)
	}
	if stringSliceContains(graph[sourceModule], dependencyModule) {
		return fmt.Errorf("module %q already depends on %q", sourceModule, dependencyModule)
	}

	graphCandidate := cloneGraph(graph)
	graphCandidate[sourceModule] = append(graphCandidate[sourceModule], dependencyModule)
	sort.Strings(graphCandidate[sourceModule])
	if err := detectCircularDependencyInGraph(graphCandidate); err != nil {
		return err
	}

	if err := applyInternalDependencyEdit(sourceManifestPath, dependencyModule, true); err != nil {
		return fmt.Errorf("add internal dependency %q -> %q: %w", sourceModule, dependencyModule, err)
	}

	return nil
}

// RemoveInternalDep removes moduleName -> dependsOn from package + target dependencies.
func RemoveInternalDep(moduleName string, dependsOn string, modulesPath string) error {
	sourceModule, err := normalizeInterfaceModuleName(moduleName)
	if err != nil {
		return err
	}
	dependencyModule, err := normalizeInterfaceModuleName(dependsOn)
	if err != nil {
		return err
	}

	modulesRoot := normalizeModulesPath(modulesPath)
	sourceManifestPath, err := requireModuleManifest(sourceModule, modulesRoot)
	if err != nil {
		return err
	}

	if err := applyInternalDependencyEdit(sourceManifestPath, dependencyModule, false); err != nil {
		return fmt.Errorf("remove internal dependency %q -> %q: %w", sourceModule, dependencyModule, err)
	}

	return nil
}

// ListInternalDeps lists internal interface dependencies for one module or all modules.
func ListInternalDeps(moduleName string, modulesPath string) (map[string][]string, error) {
	return listInternalDeps(moduleName, modulesPath, defaultGraphSource)
}

func listInternalDeps(moduleName string, modulesPath string, source GraphSourceFunc) (map[string][]string, error) {
	modulesRoot := normalizeModulesPath(modulesPath)
	graph, err := buildDependencyGraph(modulesRoot, source)
	if err != nil {
		return nil, err
	}

	trimmedModuleName := strings.TrimSpace(moduleName)
	if trimmedModuleName == "" {
		return cloneGraph(graph), nil
	}

	selectedModule, err := normalizeInterfaceModuleName(trimmedModuleName)
	if err != nil {
		return nil, err
	}
	dependencies, exists := graph[selectedModule]
	if !exists {
		return nil, fmt.Errorf("module %q was not found in %q", selectedModule, modulesRoot)
	}

	return map[string][]string{
		selectedModule: append([]string(nil), dependencies...),
	}, nil
}

func normalizeInterfaceModuleName(raw string) (string, error) {
	moduleName, err := modules.ValidateModuleName(raw)
	if err != nil {
		return "", err
	}
	if strings.HasSuffix(moduleName, moduleImplSuffix) {
		return "", fmt.Errorf("internal dependency must reference interface package, got %q", moduleName)
	}
	return moduleName, nil
}

func requireModuleManifest(moduleName string, modulesRoot string) (string, error) {
	modulePath := filepath.Join(modulesRoot, moduleName)
	if exists, err := pathExists(modulePath); err != nil {
		return "", fmt.Errorf("stat module path %q: %w", modulePath, err)
	} else if !exists {
		return "", fmt.Errorf("module %q was not found in %q", moduleName, modulesRoot)
	}

	manifestPath := filepath.Join(modulePath, moduleManifestName)
	if exists, err := pathExists(manifestPath); err != nil {
		return "", fmt.Errorf("stat module manifest %q: %w", manifestPath, err)
	} else if !exists {
		return "", fmt.Errorf("module manifest %q was not found", manifestPath)
	}

	return manifestPath, nil
}

func applyInternalDependencyEdit(manifestPath string, dependsOn string, add bool) error {
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read module manifest %q: %w", manifestPath, err)
	}

	updated := string(payload)
	if add {
		updated, err = tuistproj.ApplyManifestEdits(updated, tuistproj.ManifestEdit{
			Type:    tuistproj.AddDependency,
			Name:    dependsOn,
			Content: fmt.Sprintf(`.package(path: "../%s")`, dependsOn),
		})
		if err != nil {
			return err
		}

		updated, err = addTargetProductDependency(updated, dependsOn)
		if err != nil {
			return err
		}
	} else {
		updated, err = tuistproj.ApplyManifestEdits(updated, tuistproj.ManifestEdit{
			Type: tuistproj.RemoveDependency,
			Name: dependsOn,
		})
		if err != nil {
			return err
		}

		updated, err = removeTargetProductDependency(updated, dependsOn)
		if err != nil {
			return err
		}
	}

	if err := os.WriteFile(manifestPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write module manifest %q: %w", manifestPath, err)
	}
	return nil
}

func addTargetProductDependency(content string, dependsOn string) (string, error) {
	return addTargetProductDependencyFromPackage(content, dependsOn, dependsOn)
}

func addTargetProductDependencyFromPackage(content string, productName string, packageName string) (string, error) {
	lines, hasTrailingNewline := splitEditableLines(content)
	arraySection, err := findTargetDependencyArray(lines)
	if err != nil {
		return "", err
	}

	pattern := productDependencyPattern(productName)
	for lineIndex := arraySection.openLine + 1; lineIndex < arraySection.closeLine; lineIndex++ {
		if pattern.MatchString(lines[lineIndex]) {
			return "", fmt.Errorf("target dependencies already contain %q", productName)
		}
	}

	indent := targetDependencyInsertionIndent(lines, arraySection)
	newLine := indent + fmt.Sprintf(`.product(name: "%s", package: "%s"),`, productName, packageName)

	updatedLines := make([]string, 0, len(lines)+1)
	updatedLines = append(updatedLines, lines[:arraySection.closeLine]...)
	updatedLines = append(updatedLines, newLine)
	updatedLines = append(updatedLines, lines[arraySection.closeLine:]...)

	return joinEditableLines(updatedLines, hasTrailingNewline), nil
}

func removeTargetProductDependency(content string, dependsOn string) (string, error) {
	lines, hasTrailingNewline := splitEditableLines(content)
	arraySection, err := findTargetDependencyArray(lines)
	if err != nil {
		return "", err
	}

	pattern := productDependencyPattern(dependsOn)
	removeLine := -1
	for lineIndex := arraySection.openLine + 1; lineIndex < arraySection.closeLine; lineIndex++ {
		if !pattern.MatchString(lines[lineIndex]) {
			continue
		}
		removeLine = lineIndex
		break
	}
	if removeLine == -1 {
		return "", fmt.Errorf("target dependencies do not contain %q", dependsOn)
	}

	updatedLines := make([]string, 0, len(lines)-1)
	updatedLines = append(updatedLines, lines[:removeLine]...)
	updatedLines = append(updatedLines, lines[removeLine+1:]...)

	return joinEditableLines(updatedLines, hasTrailingNewline), nil
}

type dependencyArrayRange struct {
	openLine  int
	closeLine int
	indent    int
}

func findTargetDependencyArray(lines []string) (dependencyArrayRange, error) {
	candidates := make([]dependencyArrayRange, 0, 2)
	for lineIndex, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}

		match := dependenciesLabelPattern.FindStringIndex(line)
		if match == nil {
			continue
		}

		openColumn := indexOutsideStringAndComment(line, match[1], '[')
		if openColumn < 0 {
			continue
		}

		closeLine, ok := findMatchingBracket(lines, lineIndex, openColumn)
		if !ok {
			return dependencyArrayRange{}, fmt.Errorf("dependencies array has no matching closing bracket")
		}

		candidates = append(candidates, dependencyArrayRange{
			openLine:  lineIndex,
			closeLine: closeLine,
			indent:    leadingIndentWidth(line),
		})
	}

	if len(candidates) == 0 {
		return dependencyArrayRange{}, fmt.Errorf("target dependencies section not found")
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].indent == candidates[j].indent {
			return candidates[i].openLine < candidates[j].openLine
		}
		return candidates[i].indent > candidates[j].indent
	})

	return candidates[0], nil
}

func targetDependencyInsertionIndent(lines []string, arraySection dependencyArrayRange) string {
	for lineIndex := arraySection.openLine + 1; lineIndex < arraySection.closeLine; lineIndex++ {
		if strings.TrimSpace(lines[lineIndex]) == "" {
			continue
		}
		return leadingIndentPrefix(lines[lineIndex])
	}

	return leadingIndentPrefix(lines[arraySection.closeLine]) + "    "
}

func splitEditableLines(content string) ([]string, bool) {
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	lines := strings.Split(content, "\n")
	if hasTrailingNewline && len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, hasTrailingNewline
	}
	return lines, hasTrailingNewline
}

func joinEditableLines(lines []string, hasTrailingNewline bool) string {
	joined := strings.Join(lines, "\n")
	if hasTrailingNewline {
		return joined + "\n"
	}
	return joined
}

func productDependencyPattern(moduleName string) *regexp.Regexp {
	return regexp.MustCompile(`\.product\s*\(\s*name\s*:\s*"` + regexp.QuoteMeta(moduleName) + `"`)
}

func findMatchingBracket(lines []string, openLine int, openColumn int) (int, bool) {
	depth := 0
	inString := false
	escaped := false

	for lineIndex := openLine; lineIndex < len(lines); lineIndex++ {
		line := lines[lineIndex]
		columnStart := 0
		if lineIndex == openLine {
			columnStart = openColumn
		}

		for column := columnStart; column < len(line); column++ {
			ch := line[column]

			if inString {
				if escaped {
					escaped = false
					continue
				}
				if ch == '\\' {
					escaped = true
					continue
				}
				if ch == '"' {
					inString = false
				}
				continue
			}

			if ch == '/' && column+1 < len(line) && line[column+1] == '/' {
				break
			}
			if ch == '"' {
				inString = true
				continue
			}

			switch ch {
			case '[':
				depth++
			case ']':
				if depth > 0 {
					depth--
				}
				if depth == 0 {
					return lineIndex, true
				}
			}
		}
	}

	return 0, false
}

func indexOutsideStringAndComment(line string, start int, target byte) int {
	inString := false
	escaped := false

	for index := start; index < len(line); index++ {
		ch := line[index]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '/' && index+1 < len(line) && line[index+1] == '/' {
			return -1
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == target {
			return index
		}
	}

	return -1
}

func leadingIndentPrefix(line string) string {
	end := 0
	for end < len(line) {
		if line[end] != ' ' && line[end] != '\t' {
			break
		}
		end++
	}
	return line[:end]
}

func leadingIndentWidth(line string) int {
	width := 0
	for index := 0; index < len(line); index++ {
		switch line[index] {
		case ' ', '\t':
			width++
		default:
			return width
		}
	}
	return width
}

func stringSliceContains(values []string, needle string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}

func cloneGraph(graph Graph) Graph {
	cloned := make(Graph, len(graph))
	for node, dependencies := range graph {
		clonedDependencies := make([]string, len(dependencies))
		copy(clonedDependencies, dependencies)
		cloned[node] = clonedDependencies
	}
	return cloned
}
