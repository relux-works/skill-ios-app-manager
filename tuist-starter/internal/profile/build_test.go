package profile

import (
	"reflect"
	"testing"
)

func TestParseBuildTimingSummary(t *testing.T) {
	t.Parallel()

	raw := `Build Timing Summary
CompileSwiftSources normal arm64 (in target 'Auth' from project 'Auth')
    12.500 seconds
Ld /tmp/DemoApp (in target 'DemoApp' from project 'DemoApp') 3.25 seconds
PhaseScriptExecution RunScript 0.75 seconds
`

	entries := ParseBuildTimingSummary(raw)
	if len(entries) != 3 {
		t.Fatalf("entry count = %d, want 3: %#v", len(entries), entries)
	}

	if entries[0].Command != "CompileSwiftSources" || entries[0].Target != "Auth" || entries[0].Project != "Auth" || entries[0].Duration != 12.5 {
		t.Fatalf("first entry = %#v", entries[0])
	}
	if entries[1].Command != "Ld" || entries[1].Target != "DemoApp" || entries[1].Duration != 3.25 {
		t.Fatalf("second entry = %#v", entries[1])
	}
	if entries[2].Command != "PhaseScriptExecution" || entries[2].Target != "" || entries[2].Duration != 0.75 {
		t.Fatalf("third entry = %#v", entries[2])
	}
}

func TestAnalyzeBuildLogComputesTargetWorkAndCriticalPath(t *testing.T) {
	t.Parallel()

	raw := `Build Timing Summary
CompileSwiftSources (in target 'Core' from project 'Core') 4.0 seconds
CompileSwiftSources (in target 'Auth' from project 'Auth') 8.0 seconds
CompileSwiftSources (in target 'Feed' from project 'Feed') 5.0 seconds
Ld (in target 'DemoApp' from project 'DemoApp') 2.0 seconds
`

	graph := &TargetGraph{Nodes: map[string][]string{
		"Core":    {},
		"Auth":    {"Core"},
		"Feed":    {"Core"},
		"DemoApp": {"Auth", "Feed"},
	}}

	report := AnalyzeBuildLog(raw, graph)
	if report.TotalWorkSeconds != 19 {
		t.Fatalf("total work = %v, want 19", report.TotalWorkSeconds)
	}

	wantPath := []CriticalPathNode{
		{Target: "Core", Duration: 4},
		{Target: "Auth", Duration: 8},
		{Target: "DemoApp", Duration: 2},
	}
	if !reflect.DeepEqual(report.CriticalPath, wantPath) {
		t.Fatalf("critical path = %#v, want %#v", report.CriticalPath, wantPath)
	}
	if report.IdealParallelism <= 1.3 || report.IdealParallelism >= 1.4 {
		t.Fatalf("ideal parallelism = %v, want about 1.36", report.IdealParallelism)
	}
}

func TestParseTuistTargetGraph(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"name": "DemoApp",
		"path": "/project",
		"projects": {
			"/project": {
				"name": "DemoApp",
				"path": "/project",
				"targets": [{
					"name": "DemoApp",
					"product": "app",
					"dependencies": [
						{"project": {"path": "/project/Packages/Auth", "target": "Auth"}},
						{"project": {"path": "/project/.build/checkouts/Remote", "target": "Remote"}}
					]
				}]
			},
			"/project/Packages/Auth": {
				"name": "Auth",
				"path": "/project/Packages/Auth",
				"targets": [{
					"name": "Auth",
					"product": "framework",
					"dependencies": [
						{"project": {"path": "/project/Packages/Core", "target": "Core"}}
					]
				}]
			},
			"/project/Packages/Core": {
				"name": "Core",
				"path": "/project/Packages/Core",
				"targets": [{"name": "Core", "product": "framework", "dependencies": []}]
			},
			"/project/.build/checkouts/Remote": {
				"name": "Remote",
				"path": "/project/.build/checkouts/Remote",
				"targets": [{"name": "Remote", "product": "framework", "dependencies": []}]
			}
		}
	}`)

	graph, err := ParseTuistTargetGraph(data, "/project")
	if err != nil {
		t.Fatalf("ParseTuistTargetGraph() error = %v", err)
	}

	if _, exists := graph.Nodes["Remote"]; exists {
		t.Fatalf("graph should not include external checkout target: %#v", graph.Nodes)
	}
	if !reflect.DeepEqual(graph.Nodes["DemoApp"], []string{"Auth"}) {
		t.Fatalf("DemoApp deps = %#v, want Auth", graph.Nodes["DemoApp"])
	}
	if !reflect.DeepEqual(graph.Nodes["Auth"], []string{"Core"}) {
		t.Fatalf("Auth deps = %#v, want Core", graph.Nodes["Auth"])
	}
}
