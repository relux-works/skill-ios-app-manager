package profile

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

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

// TargetGraph is a local Xcode/Tuist target dependency graph.
type TargetGraph struct {
	Nodes map[string][]string `json:"nodes"`
}

// ParseTuistTargetGraph parses Tuist legacyJSON graph output.
func ParseTuistTargetGraph(data []byte, projectRoot string) (*TargetGraph, error) {
	var output tuistGraphOutput
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parse Tuist graph JSON: %w", err)
	}

	root := filepath.Clean(projectRoot)
	nodes := make(map[string][]string)
	localTargets := make(map[string]struct{})

	for _, project := range output.Projects {
		if !isLocalProjectPath(project.Path, root) {
			continue
		}
		for _, target := range project.Targets {
			if strings.TrimSpace(target.Name) == "" {
				continue
			}
			localTargets[target.Name] = struct{}{}
			if _, exists := nodes[target.Name]; !exists {
				nodes[target.Name] = []string{}
			}
		}
	}

	for _, project := range output.Projects {
		if !isLocalProjectPath(project.Path, root) {
			continue
		}
		for _, target := range project.Targets {
			if _, exists := localTargets[target.Name]; !exists {
				continue
			}
			depSet := make(map[string]struct{})
			for _, dep := range target.Dependencies {
				if dep.Project == nil {
					continue
				}
				if _, exists := localTargets[dep.Project.Target]; !exists {
					continue
				}
				if dep.Project.Target == target.Name {
					continue
				}
				depSet[dep.Project.Target] = struct{}{}
			}
			deps := make([]string, 0, len(depSet))
			for dep := range depSet {
				deps = append(deps, dep)
			}
			sort.Strings(deps)
			nodes[target.Name] = deps
		}
	}

	return &TargetGraph{Nodes: nodes}, nil
}

func isLocalProjectPath(path string, root string) bool {
	cleaned := filepath.Clean(path)
	rel, err := filepath.Rel(root, cleaned)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, "..") {
		return false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) > 0 && parts[0] == ".build" {
		return false
	}
	return true
}

// EdgeCount returns the number of dependency edges.
func (g *TargetGraph) EdgeCount() int {
	if g == nil {
		return 0
	}
	count := 0
	for _, deps := range g.Nodes {
		count += len(deps)
	}
	return count
}

// CriticalPath returns the longest weighted dependency chain.
func (g *TargetGraph) CriticalPath(weights map[string]float64) []CriticalPathNode {
	if g == nil || len(g.Nodes) == 0 {
		return nil
	}

	memo := make(map[string][]CriticalPathNode)
	visiting := make(map[string]bool)
	best := []CriticalPathNode{}

	names := make([]string, 0, len(g.Nodes))
	for name := range g.Nodes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		path := g.criticalPathEndingAt(name, weights, memo, visiting)
		if pathDuration(path) > pathDuration(best) {
			best = path
		}
	}

	return best
}

func (g *TargetGraph) criticalPathEndingAt(name string, weights map[string]float64, memo map[string][]CriticalPathNode, visiting map[string]bool) []CriticalPathNode {
	if path, ok := memo[name]; ok {
		return path
	}
	if visiting[name] {
		return []CriticalPathNode{{Target: name, Duration: weights[name]}}
	}
	visiting[name] = true
	defer delete(visiting, name)

	bestDeps := []CriticalPathNode{}
	for _, dep := range g.Nodes[name] {
		path := g.criticalPathEndingAt(dep, weights, memo, visiting)
		if pathDuration(path) > pathDuration(bestDeps) {
			bestDeps = path
		}
	}

	result := append([]CriticalPathNode(nil), bestDeps...)
	result = append(result, CriticalPathNode{Target: name, Duration: weights[name]})
	memo[name] = result
	return result
}

func pathDuration(path []CriticalPathNode) float64 {
	total := 0.0
	for _, node := range path {
		total += node.Duration
	}
	return total
}
