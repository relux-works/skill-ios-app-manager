package deps

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// GraphSourceFunc builds a dependency graph from a modules path.
type GraphSourceFunc func(modulesPath string) (Graph, error)

// tuistGraphOutput matches the top-level schema of `tuist graph --format legacyJSON`.
type tuistGraphOutput struct {
	Name     string                       `json:"name"`
	Path     string                       `json:"path"`
	Projects map[string]tuistGraphProject `json:"projects"`
}

type tuistGraphProject struct {
	Name       string             `json:"name"`
	Path       string             `json:"path"`
	IsExternal bool               `json:"isExternal"`
	Targets    []tuistGraphTarget `json:"targets"`
}

type tuistGraphTarget struct {
	Name         string                 `json:"name"`
	Product      string                 `json:"product"`
	Dependencies []tuistGraphDependency `json:"dependencies"`
}

type tuistGraphDependency struct {
	Project *tuistGraphProjectRef `json:"project,omitempty"`
}

type tuistGraphProjectRef struct {
	Path   string `json:"path"`
	Target string `json:"target"`
	Status string `json:"status"`
}

// runTuistGraph is the default function for executing the tuist graph CLI.
// It can be replaced in tests.
var runTuistGraph = defaultRunTuistGraph

func defaultRunTuistGraph(projectRoot string) ([]byte, error) {
	tempDir, err := os.MkdirTemp("", "tuist-graph-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir for tuist graph: %w", err)
	}
	defer os.RemoveAll(tempDir)

	cmd := exec.Command(
		"tuist", "graph",
		"--format", "legacyJSON",
		"--no-open",
		"--path", projectRoot,
		"--output-path", tempDir,
	)
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tuist graph failed: %w\noutput: %s", err, string(output))
	}

	graphPath := filepath.Join(tempDir, "graph.json")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		return nil, fmt.Errorf("read tuist graph output %q: %w", graphPath, err)
	}

	return data, nil
}

// parseTuistGraphJSON parses the legacyJSON output from tuist graph.
func parseTuistGraphJSON(data []byte) (*tuistGraphOutput, error) {
	var output tuistGraphOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parse tuist graph JSON: %w", err)
	}
	return &output, nil
}

// extractGraphFromTuistOutput builds an internal module dependency graph from tuist graph output.
// It filters to only interface modules (no Impl suffix) within the given modulesRoot path.
func extractGraphFromTuistOutput(output *tuistGraphOutput, modulesRoot string) Graph {
	absModulesRoot := filepath.Clean(modulesRoot)

	// Collect all local interface module names by checking project paths.
	moduleSet := make(map[string]struct{})
	for _, project := range output.Projects {
		projectPath := filepath.Clean(project.Path)
		if !isSubpath(projectPath, absModulesRoot) {
			continue
		}
		if strings.HasSuffix(project.Name, moduleImplSuffix) {
			continue
		}
		moduleSet[project.Name] = struct{}{}
	}

	graph := make(Graph, len(moduleSet))
	for name := range moduleSet {
		graph[name] = []string{}
	}

	// For each interface module, find its target and extract deps that are also local interface modules.
	for _, project := range output.Projects {
		if _, ok := moduleSet[project.Name]; !ok {
			continue
		}

		for _, target := range project.Targets {
			if target.Name != project.Name {
				continue
			}

			depSet := make(map[string]struct{})
			for _, dep := range target.Dependencies {
				if dep.Project == nil {
					continue
				}
				depPath := filepath.Clean(dep.Project.Path)
				if !isSubpath(depPath, absModulesRoot) {
					continue
				}
				depName := dep.Project.Target
				if strings.HasSuffix(depName, moduleImplSuffix) {
					continue
				}
				if _, ok := moduleSet[depName]; !ok {
					continue
				}
				depSet[depName] = struct{}{}
			}

			deps := make([]string, 0, len(depSet))
			for d := range depSet {
				deps = append(deps, d)
			}
			sort.Strings(deps)
			graph[project.Name] = deps
		}
	}

	return graph
}

// tuistGraphSource builds the dependency graph by running tuist graph CLI.
func tuistGraphSource(modulesPath string) (Graph, error) {
	modulesRoot := normalizeModulesPath(modulesPath)
	projectRoot := filepath.Dir(modulesRoot)

	data, err := runTuistGraph(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("build dependency graph via tuist: %w", err)
	}

	output, err := parseTuistGraphJSON(data)
	if err != nil {
		return nil, err
	}

	return extractGraphFromTuistOutput(output, modulesRoot), nil
}

// isSubpath reports whether child is a direct child directory of parent.
func isSubpath(child string, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	// Must be a single directory name (no path separators, no "..")
	return rel != "." && !strings.Contains(rel, string(filepath.Separator)) && !strings.HasPrefix(rel, "..")
}
