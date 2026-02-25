package tuistproj

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ProjectGraph contains extracted graph information from tuist JSON output.
type ProjectGraph struct {
	Targets      []GraphTarget     `json:"targets"`
	Dependencies []GraphDependency `json:"dependencies"`
	ModuleTypes  map[string]string `json:"moduleTypes"`
}

// GraphTarget represents one target/module in the project graph.
type GraphTarget struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name"`
	ModuleType string `json:"moduleType,omitempty"`
	Product    string `json:"product,omitempty"`
}

// GraphDependency represents a directed dependency edge.
type GraphDependency struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// GraphParseError is returned when graph JSON cannot be parsed.
type GraphParseError struct {
	Output string
	Err    error
}

func (e *GraphParseError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("unable to parse tuist graph JSON: %v", e.Err)
}

// Unwrap returns the underlying parse error.
func (e *GraphParseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// LoadGraph runs `tuist graph --format json` and parses graph output.
func LoadGraph(ctx context.Context, runner Runner, extraArgs ...string) (ProjectGraph, error) {
	if runner == nil {
		return ProjectGraph{}, fmt.Errorf("runner is nil")
	}

	graphArgs := ensureGraphJSONArgs(extraArgs)
	result, err := runner.Run(ctx, CommandGraph, graphArgs...)
	if err != nil {
		return ProjectGraph{}, fmt.Errorf("run tuist graph --format json: %w", err)
	}

	output := strings.TrimSpace(result.Stdout)
	if output == "" {
		output = strings.TrimSpace(result.Stderr)
	}

	return ParseGraphJSON([]byte(output))
}

// GraphJSON runs and parses a JSON dependency graph for this runner.
func (r *TuistRunner) GraphJSON(ctx context.Context, extraArgs ...string) (ProjectGraph, error) {
	return LoadGraph(ctx, r, extraArgs...)
}

// ParseGraphJSON parses tuist graph JSON into extracted graph data.
func ParseGraphJSON(payload []byte) (ProjectGraph, error) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return ProjectGraph{}, &GraphParseError{
			Output: string(payload),
			Err:    fmt.Errorf("empty graph payload"),
		}
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &root); err != nil {
		return ProjectGraph{}, &GraphParseError{
			Output: string(trimmed),
			Err:    err,
		}
	}

	graph := ProjectGraph{
		Targets:      make([]GraphTarget, 0),
		Dependencies: make([]GraphDependency, 0),
		ModuleTypes:  make(map[string]string),
	}

	targetSeen := make(map[string]struct{})
	nodeIDToName := make(map[string]string)

	if rawTargets, ok := root["targets"]; ok {
		if err := appendTargetsFromRaw(rawTargets, &graph, targetSeen, nodeIDToName); err != nil {
			return ProjectGraph{}, &GraphParseError{
				Output: string(trimmed),
				Err:    err,
			}
		}
	}

	if len(graph.Targets) == 0 {
		if rawNodes, ok := root["nodes"]; ok {
			if err := appendTargetsFromRaw(rawNodes, &graph, targetSeen, nodeIDToName); err != nil {
				return ProjectGraph{}, &GraphParseError{
					Output: string(trimmed),
					Err:    err,
				}
			}
		}
	}

	dependencySeen := make(map[string]struct{})
	if rawDependencies, ok := root["dependencies"]; ok {
		if err := appendDependenciesFromRaw(rawDependencies, nodeIDToName, &graph, dependencySeen); err != nil {
			return ProjectGraph{}, &GraphParseError{
				Output: string(trimmed),
				Err:    err,
			}
		}
	}

	if rawEdges, ok := root["edges"]; ok {
		if err := appendDependenciesFromRaw(rawEdges, nodeIDToName, &graph, dependencySeen); err != nil {
			return ProjectGraph{}, &GraphParseError{
				Output: string(trimmed),
				Err:    err,
			}
		}
	}

	return graph, nil
}

func ensureGraphJSONArgs(extraArgs []string) []string {
	args := append([]string(nil), extraArgs...)

	for i := 0; i < len(args); i++ {
		if args[i] == "--format" {
			if i+1 < len(args) {
				args[i+1] = "json"
			} else {
				args = append(args, "json")
			}
			return args
		}

		if strings.HasPrefix(args[i], "--format=") {
			args[i] = "--format=json"
			return args
		}
	}

	return append([]string{"--format", "json"}, args...)
}

func appendTargetsFromRaw(
	raw json.RawMessage,
	graph *ProjectGraph,
	targetSeen map[string]struct{},
	nodeIDToName map[string]string,
) error {
	objects, err := parseObjectArray(raw)
	if err != nil {
		return err
	}

	for index, obj := range objects {
		id := firstString(obj, "id")
		if id == "" {
			id = strconv.Itoa(index)
		}

		name := firstString(obj, "name", "target", "label", "id")
		if name == "" {
			name = id
		}

		moduleType := firstString(obj, "moduleType", "module_type", "type", "kind", "productType", "product")
		product := firstString(obj, "product", "productType")

		if _, exists := targetSeen[name]; !exists {
			graph.Targets = append(graph.Targets, GraphTarget{
				ID:         id,
				Name:       name,
				ModuleType: moduleType,
				Product:    product,
			})
			targetSeen[name] = struct{}{}
		}

		if moduleType != "" {
			graph.ModuleTypes[name] = moduleType
		}
		nodeIDToName[id] = name
	}

	return nil
}

func appendDependenciesFromRaw(
	raw json.RawMessage,
	nodeIDToName map[string]string,
	graph *ProjectGraph,
	dependencySeen map[string]struct{},
) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}

	switch trimmed[0] {
	case '[':
		objects, err := parseObjectArray(trimmed)
		if err != nil {
			return err
		}

		for _, obj := range objects {
			from := resolveNodeName(firstValue(obj, "from", "source", "origin", "u"), nodeIDToName)
			to := resolveNodeName(firstValue(obj, "to", "target", "destination", "v"), nodeIDToName)
			addDependency(graph, dependencySeen, from, to)
		}

	case '{':
		var adjacency map[string]any
		if err := json.Unmarshal(trimmed, &adjacency); err != nil {
			return err
		}

		keys := make([]string, 0, len(adjacency))
		for key := range adjacency {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			from := resolveNodeName(key, nodeIDToName)
			targets := flattenStrings(adjacency[key])
			for _, toTarget := range targets {
				to := resolveNodeName(toTarget, nodeIDToName)
				addDependency(graph, dependencySeen, from, to)
			}
		}

	default:
		return fmt.Errorf("unexpected graph dependency payload")
	}

	return nil
}

func parseObjectArray(raw json.RawMessage) ([]map[string]any, error) {
	var objects []map[string]any
	if err := json.Unmarshal(raw, &objects); err == nil {
		return objects, nil
	}

	var values []any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}

	objects = make([]map[string]any, 0, len(values))
	for _, value := range values {
		object, ok := value.(map[string]any)
		if !ok {
			continue
		}
		objects = append(objects, object)
	}

	if len(values) > 0 && len(objects) == 0 {
		return nil, fmt.Errorf("array does not contain JSON objects")
	}

	return objects, nil
}

func addDependency(
	graph *ProjectGraph,
	dependencySeen map[string]struct{},
	from string,
	to string,
) {
	if from == "" || to == "" {
		return
	}

	key := from + "->" + to
	if _, exists := dependencySeen[key]; exists {
		return
	}

	dependencySeen[key] = struct{}{}
	graph.Dependencies = append(graph.Dependencies, GraphDependency{
		From: from,
		To:   to,
	})
}

func firstValue(values map[string]any, keys ...string) any {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}

		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return typed
			}
		default:
			if value != nil {
				return value
			}
		}
	}
	return nil
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		if stringValue := asString(value); stringValue != "" {
			return stringValue
		}
	}
	return ""
}

func resolveNodeName(value any, nodeIDToName map[string]string) string {
	key := asString(value)
	if key == "" {
		return ""
	}
	if name, ok := nodeIDToName[key]; ok {
		return name
	}
	return key
}

func flattenStrings(value any) []string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}

	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			stringValue := asString(item)
			if stringValue != "" {
				values = append(values, stringValue)
			}
		}
		return values

	default:
		stringValue := asString(value)
		if stringValue == "" {
			return nil
		}
		return []string{stringValue}
	}
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		if typed == float32(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int8:
		return strconv.FormatInt(int64(typed), 10)
	case int16:
		return strconv.FormatInt(int64(typed), 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint8:
		return strconv.FormatUint(uint64(typed), 10)
	case uint16:
		return strconv.FormatUint(uint64(typed), 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	default:
		return ""
	}
}
