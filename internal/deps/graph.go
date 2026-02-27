package deps

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/tuistproj"
)

const (
	defaultModulesPath = "Packages"
	moduleImplSuffix   = "Impl"
	moduleManifestName = "Package.swift"
	cyclePathSeparator = " → "
)

// Graph is a directed module dependency graph: module -> dependencies.
type Graph map[string][]string

// BuildDependencyGraph builds the internal module dependency graph.
// It tries tuist graph first, falling back to manifest parsing if tuist is unavailable.
func BuildDependencyGraph(modulesPath string) (Graph, error) {
	return buildDependencyGraph(modulesPath, defaultGraphSource)
}

// buildDependencyGraph builds the graph using the provided source function.
func buildDependencyGraph(modulesPath string, source GraphSourceFunc) (Graph, error) {
	return source(modulesPath)
}

// defaultGraphSource tries tuist graph first, falls back to manifest parsing.
func defaultGraphSource(modulesPath string) (Graph, error) {
	graph, err := tuistGraphSource(modulesPath)
	if err == nil {
		return graph, nil
	}
	return manifestGraphSource(modulesPath)
}

// DetectCircularDependencies detects cycles in the module dependency graph.
func DetectCircularDependencies(modulesPath string) error {
	return detectCircularDependencies(modulesPath, defaultGraphSource)
}

func detectCircularDependencies(modulesPath string, source GraphSourceFunc) error {
	graph, err := buildDependencyGraph(modulesPath, source)
	if err != nil {
		return err
	}
	return detectCircularDependencyInGraph(graph)
}

func detectCircularDependencyInGraph(graph Graph) error {
	cycle, ok := detectCyclePath(graph)
	if !ok {
		return nil
	}
	return fmt.Errorf("circular dependency: %s", strings.Join(cycle, cyclePathSeparator))
}

func detectCyclePath(graph Graph) ([]string, bool) {
	type visitState uint8
	const (
		visitStateUnvisited visitState = iota
		visitStateVisiting
		visitStateVisited
	)

	states := make(map[string]visitState, len(graph))
	position := make(map[string]int, len(graph))
	stack := make([]string, 0, len(graph))

	nodes := make([]string, 0, len(graph))
	for node := range graph {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)

	var dfs func(node string) ([]string, bool)
	dfs = func(node string) ([]string, bool) {
		states[node] = visitStateVisiting
		position[node] = len(stack)
		stack = append(stack, node)

		neighbors := append([]string(nil), graph[node]...)
		sort.Strings(neighbors)
		for _, neighbor := range neighbors {
			if _, exists := graph[neighbor]; !exists {
				continue
			}

			switch states[neighbor] {
			case visitStateUnvisited:
				if cycle, ok := dfs(neighbor); ok {
					return cycle, true
				}
			case visitStateVisiting:
				start := position[neighbor]
				cycle := append([]string{}, stack[start:]...)
				cycle = append(cycle, neighbor)
				return cycle, true
			}
		}

		stack = stack[:len(stack)-1]
		delete(position, node)
		states[node] = visitStateVisited
		return nil, false
	}

	for _, node := range nodes {
		if states[node] != visitStateUnvisited {
			continue
		}
		if cycle, ok := dfs(node); ok {
			return cycle, true
		}
	}

	return nil, false
}

// manifestGraphSource builds the graph by scanning Package.swift files with regex parsing.
// Used in unit tests where tuist CLI is not available.
func manifestGraphSource(modulesPath string) (Graph, error) {
	modulesRoot := normalizeModulesPath(modulesPath)
	entries, err := os.ReadDir(modulesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return Graph{}, nil
		}
		return nil, fmt.Errorf("scan modules directory %q: %w", modulesRoot, err)
	}

	moduleNames := make([]string, 0, len(entries))
	moduleSet := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") || strings.HasSuffix(name, moduleImplSuffix) {
			continue
		}

		manifestPath := filepath.Join(modulesRoot, name, moduleManifestName)
		exists, statErr := pathExists(manifestPath)
		if statErr != nil {
			return nil, fmt.Errorf("stat module manifest %q: %w", manifestPath, statErr)
		}
		if !exists {
			continue
		}

		moduleNames = append(moduleNames, name)
		moduleSet[name] = struct{}{}
	}
	sort.Strings(moduleNames)

	graph := make(Graph, len(moduleNames))
	for _, moduleName := range moduleNames {
		graph[moduleName] = []string{}
	}

	for _, moduleName := range moduleNames {
		manifestPath := filepath.Join(modulesRoot, moduleName, moduleManifestName)
		manifest, manifestErr := tuistproj.ReadManifestFile(manifestPath)
		if manifestErr != nil {
			return nil, fmt.Errorf("read module manifest %q: %w", manifestPath, manifestErr)
		}

		dependencySet := make(map[string]struct{}, len(manifest.Dependencies))
		for _, item := range manifest.Dependencies {
			dependencyName := strings.TrimSpace(item.Name)
			if dependencyName == "" || strings.HasSuffix(dependencyName, moduleImplSuffix) {
				continue
			}
			if _, ok := moduleSet[dependencyName]; !ok {
				continue
			}
			dependencySet[dependencyName] = struct{}{}
		}

		dependencies := make([]string, 0, len(dependencySet))
		for dependencyName := range dependencySet {
			dependencies = append(dependencies, dependencyName)
		}
		sort.Strings(dependencies)
		graph[moduleName] = dependencies
	}

	return graph, nil
}

func normalizeModulesPath(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = defaultModulesPath
	}
	return filepath.Clean(value)
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
