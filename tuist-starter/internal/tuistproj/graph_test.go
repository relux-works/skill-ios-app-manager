package tuistproj

import (
	"context"
	"os"
	"reflect"
	"testing"
)

func TestParseGraphJSONFromGolden(t *testing.T) {
	t.Parallel()

	payload, err := os.ReadFile("testdata/graph.json")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	graph, err := ParseGraphJSON(payload)
	if err != nil {
		t.Fatalf("ParseGraphJSON() error = %v", err)
	}

	wantTargets := []GraphTarget{
		{ID: "App", Name: "App", ModuleType: "app", Product: "app"},
		{ID: "FeatureOne", Name: "FeatureOne", ModuleType: "feature", Product: "framework"},
		{ID: "CoreKit", Name: "CoreKit", ModuleType: "core", Product: "framework"},
	}
	if !reflect.DeepEqual(graph.Targets, wantTargets) {
		t.Fatalf("targets = %#v, want %#v", graph.Targets, wantTargets)
	}

	wantDependencies := []GraphDependency{
		{From: "App", To: "FeatureOne"},
		{From: "FeatureOne", To: "CoreKit"},
		{From: "App", To: "CoreKit"},
	}
	if !reflect.DeepEqual(graph.Dependencies, wantDependencies) {
		t.Fatalf("dependencies = %#v, want %#v", graph.Dependencies, wantDependencies)
	}

	wantModuleTypes := map[string]string{
		"App":        "app",
		"FeatureOne": "feature",
		"CoreKit":    "core",
	}
	if !reflect.DeepEqual(graph.ModuleTypes, wantModuleTypes) {
		t.Fatalf("module types = %#v, want %#v", graph.ModuleTypes, wantModuleTypes)
	}
}

func TestLoadGraphForcesJSONFormat(t *testing.T) {
	t.Parallel()

	var gotCommand string
	var gotArgs []string

	runner := mockRunner{
		runFn: func(_ context.Context, command string, extraArgs ...string) (RunResult, error) {
			gotCommand = command
			gotArgs = append([]string(nil), extraArgs...)
			return RunResult{
				Stdout: `{
					"nodes": [
						{"id": "0", "name": "App", "type": "app"},
						{"id": "1", "name": "CoreKit", "type": "framework"}
					],
					"edges": [{"from": "0", "to": "1"}]
				}`,
			}, nil
		},
	}

	graph, err := LoadGraph(context.Background(), runner, "--skip-test-targets")
	if err != nil {
		t.Fatalf("LoadGraph() error = %v", err)
	}

	if gotCommand != CommandGraph {
		t.Fatalf("command = %q, want %q", gotCommand, CommandGraph)
	}

	wantArgs := []string{"--format", "json", "--skip-test-targets"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}

	wantDependencies := []GraphDependency{
		{From: "App", To: "CoreKit"},
	}
	if !reflect.DeepEqual(graph.Dependencies, wantDependencies) {
		t.Fatalf("dependencies = %#v, want %#v", graph.Dependencies, wantDependencies)
	}
}

func TestParseGraphJSONDependencyMap(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"targets": [
			{"id":"0","name":"App","moduleType":"app"},
			{"id":"1","name":"CoreKit","moduleType":"core"},
			{"id":"2","name":"FeatureOne","moduleType":"feature"}
		],
		"dependencies": {
			"0": ["1","2"],
			"2": ["1"]
		}
	}`)

	graph, err := ParseGraphJSON(payload)
	if err != nil {
		t.Fatalf("ParseGraphJSON() error = %v", err)
	}

	wantDependencies := []GraphDependency{
		{From: "App", To: "CoreKit"},
		{From: "App", To: "FeatureOne"},
		{From: "FeatureOne", To: "CoreKit"},
	}
	if !reflect.DeepEqual(graph.Dependencies, wantDependencies) {
		t.Fatalf("dependencies = %#v, want %#v", graph.Dependencies, wantDependencies)
	}
}
