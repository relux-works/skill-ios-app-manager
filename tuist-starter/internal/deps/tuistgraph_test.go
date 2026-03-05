package deps

import (
	"reflect"
	"testing"
)

func TestParseTuistGraphJSON(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"name": "MyApp",
		"path": "/project",
		"projects": {
			"/project": {
				"name": "MyApp",
				"path": "/project",
				"isExternal": false,
				"targets": [{
					"name": "MyApp",
					"product": "app",
					"dependencies": [
						{"project": {"path": "/project/Packages/Auth", "target": "Auth", "status": "required"}}
					]
				}]
			},
			"/project/Packages/Auth": {
				"name": "Auth",
				"path": "/project/Packages/Auth",
				"isExternal": true,
				"targets": [{
					"name": "Auth",
					"product": "framework",
					"dependencies": []
				}]
			}
		}
	}`)

	output, err := parseTuistGraphJSON(data)
	if err != nil {
		t.Fatalf("parseTuistGraphJSON() error = %v", err)
	}

	if output.Name != "MyApp" {
		t.Fatalf("name = %q, want %q", output.Name, "MyApp")
	}
	if len(output.Projects) != 2 {
		t.Fatalf("projects count = %d, want 2", len(output.Projects))
	}
}

func TestExtractGraphFromTuistOutput(t *testing.T) {
	t.Parallel()

	output := &tuistGraphOutput{
		Name: "XFlow",
		Path: "/project",
		Projects: map[string]tuistGraphProject{
			"/project": {
				Name:       "XFlow",
				Path:       "/project",
				IsExternal: false,
				Targets: []tuistGraphTarget{{
					Name:    "XFlow",
					Product: "app",
					Dependencies: []tuistGraphDependency{
						{Project: &tuistGraphProjectRef{Path: "/project/Packages/Auth", Target: "Auth"}},
						{Project: &tuistGraphProjectRef{Path: "/project/Packages/AuthImpl", Target: "AuthImpl"}},
						{Project: &tuistGraphProjectRef{Path: "/project/.build/checkouts/swift-ioc", Target: "SwiftIoC"}},
					},
				}},
			},
			"/project/Packages/Auth": {
				Name:       "Auth",
				Path:       "/project/Packages/Auth",
				IsExternal: true,
				Targets: []tuistGraphTarget{{
					Name:    "Auth",
					Product: "framework",
					Dependencies: []tuistGraphDependency{
						{Project: &tuistGraphProjectRef{Path: "/project/Packages/CoreKit", Target: "CoreKit"}},
						{Project: &tuistGraphProjectRef{Path: "/project/.build/checkouts/swift-relux", Target: "Relux"}},
					},
				}},
			},
			"/project/Packages/AuthImpl": {
				Name:       "AuthImpl",
				Path:       "/project/Packages/AuthImpl",
				IsExternal: true,
				Targets: []tuistGraphTarget{{
					Name:    "AuthImpl",
					Product: "framework",
					Dependencies: []tuistGraphDependency{
						{Project: &tuistGraphProjectRef{Path: "/project/Packages/Auth", Target: "Auth"}},
					},
				}},
			},
			"/project/Packages/CoreKit": {
				Name:       "CoreKit",
				Path:       "/project/Packages/CoreKit",
				IsExternal: true,
				Targets: []tuistGraphTarget{{
					Name:         "CoreKit",
					Product:      "framework",
					Dependencies: []tuistGraphDependency{},
				}},
			},
			"/project/.build/checkouts/swift-relux": {
				Name:       "swift-relux",
				Path:       "/project/.build/checkouts/swift-relux",
				IsExternal: true,
				Targets: []tuistGraphTarget{{
					Name:         "Relux",
					Product:      "framework",
					Dependencies: []tuistGraphDependency{},
				}},
			},
		},
	}

	graph := extractGraphFromTuistOutput(output, "/project/Packages")

	// Should only contain interface modules (no Impl, no external)
	if _, exists := graph["AuthImpl"]; exists {
		t.Fatal("graph should not contain AuthImpl")
	}
	if _, exists := graph["Relux"]; exists {
		t.Fatal("graph should not contain external dep Relux")
	}
	if _, exists := graph["XFlow"]; exists {
		t.Fatal("graph should not contain XFlow (app target, not in Packages)")
	}

	// Auth should depend on CoreKit (local) but not Relux (external)
	if got := graph["Auth"]; !reflect.DeepEqual(got, []string{"CoreKit"}) {
		t.Fatalf(`graph["Auth"] = %#v, want %#v`, got, []string{"CoreKit"})
	}

	// CoreKit should have no deps
	if got := graph["CoreKit"]; !reflect.DeepEqual(got, []string{}) {
		t.Fatalf(`graph["CoreKit"] = %#v, want empty slice`, got)
	}
}

func TestExtractGraphFiltersImplModules(t *testing.T) {
	t.Parallel()

	output := &tuistGraphOutput{
		Name: "App",
		Path: "/project",
		Projects: map[string]tuistGraphProject{
			"/project/Packages/Feed": {
				Name:       "Feed",
				Path:       "/project/Packages/Feed",
				IsExternal: true,
				Targets: []tuistGraphTarget{{
					Name:    "Feed",
					Product: "framework",
					Dependencies: []tuistGraphDependency{
						{Project: &tuistGraphProjectRef{Path: "/project/Packages/Auth", Target: "Auth"}},
						{Project: &tuistGraphProjectRef{Path: "/project/Packages/AuthImpl", Target: "AuthImpl"}},
					},
				}},
			},
			"/project/Packages/Auth": {
				Name:       "Auth",
				Path:       "/project/Packages/Auth",
				IsExternal: true,
				Targets: []tuistGraphTarget{{
					Name:         "Auth",
					Product:      "framework",
					Dependencies: []tuistGraphDependency{},
				}},
			},
			"/project/Packages/AuthImpl": {
				Name:       "AuthImpl",
				Path:       "/project/Packages/AuthImpl",
				IsExternal: true,
				Targets: []tuistGraphTarget{{
					Name:    "AuthImpl",
					Product: "framework",
					Dependencies: []tuistGraphDependency{
						{Project: &tuistGraphProjectRef{Path: "/project/Packages/Auth", Target: "Auth"}},
					},
				}},
			},
		},
	}

	graph := extractGraphFromTuistOutput(output, "/project/Packages")

	if len(graph) != 2 {
		t.Fatalf("graph has %d modules, want 2 (Feed, Auth)", len(graph))
	}

	if got := graph["Feed"]; !reflect.DeepEqual(got, []string{"Auth"}) {
		t.Fatalf(`graph["Feed"] = %#v, want ["Auth"]`, got)
	}
	if got := graph["Auth"]; !reflect.DeepEqual(got, []string{}) {
		t.Fatalf(`graph["Auth"] = %#v, want empty`, got)
	}
}

func TestIsSubpath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		child  string
		parent string
		want   bool
	}{
		{"/project/Packages/Auth", "/project/Packages", true},
		{"/project/Packages/AuthImpl", "/project/Packages", true},
		{"/project/.build/checkouts/pkg", "/project/Packages", false},
		{"/project/Packages", "/project/Packages", false},                       // same dir
		{"/project/Packages/Auth/Sources", "/project/Packages", false},          // nested too deep
		{"/other/Packages/Auth", "/project/Packages", false},                    // different root
		{"/project/Packages/../.build/x", "/project/Packages", false},           // path traversal
	}

	for _, tt := range tests {
		if got := isSubpath(tt.child, tt.parent); got != tt.want {
			t.Errorf("isSubpath(%q, %q) = %v, want %v", tt.child, tt.parent, got, tt.want)
		}
	}
}
