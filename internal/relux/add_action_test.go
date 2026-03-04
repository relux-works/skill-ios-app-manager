package relux

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddActionCommandRunUpdatesActionsAndReducer(t *testing.T) {
	modulePath := scaffoldLegacyReluxModuleForTest(t, "Notes")

	command := NewAddActionCommand()
	updatedPaths, err := command.Run(context.Background(), AddActionInput{
		ModuleName: "Notes",
		ModulePath: modulePath,
		ActionName: "didSelect",
		ActionParams: []ActionParameter{
			{Name: "id", Type: "UUID"},
			{Name: "source", Type: "String"},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(updatedPaths) != 2 {
		t.Fatalf("Run() updated %d files, want 2", len(updatedPaths))
	}

	actionsPath := filepath.Join(modulePath+"Impl", "Sources", "actions.swift")
	reducerPath := filepath.Join(modulePath+"Impl", "Sources", "reducer.swift")

	actionsContent := readFileStringForTest(t, actionsPath)
	if !strings.Contains(actionsContent, "case didSelect(id: UUID, source: String)") {
		t.Fatalf("actions.swift missing inserted case:\n%s", actionsContent)
	}
	if !containsInOrder(actionsContent, "case didSelect(id: UUID, source: String)", "case forward(NotesPublicAction)") {
		t.Fatalf("new action case should be inserted before forward case:\n%s", actionsContent)
	}

	reducerContent := readFileStringForTest(t, reducerPath)
	if !strings.Contains(reducerContent, "case .didSelect(let id, let source):") {
		t.Fatalf("reducer.swift missing inserted case pattern:\n%s", reducerContent)
	}
	if !strings.Contains(reducerContent, "// TODO: handle didSelect") {
		t.Fatalf("reducer.swift missing TODO stub:\n%s", reducerContent)
	}
	if !containsInOrder(reducerContent, "case .didSelect(let id, let source):", "case .forward:") {
		t.Fatalf("new reducer case should be inserted before .forward:\n%s", reducerContent)
	}
}

func TestAddActionCommandRunRejectsDuplicateCase(t *testing.T) {
	modulePath := scaffoldLegacyReluxModuleForTest(t, "Notes")

	command := NewAddActionCommand()
	input := AddActionInput{
		ModuleName: "Notes",
		ModulePath: modulePath,
		ActionName: "didRetry",
	}

	if _, err := command.Run(context.Background(), input); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}

	if _, err := command.Run(context.Background(), input); err == nil {
		t.Fatal("second Run() expected duplicate error, got nil")
	}
}

// scaffoldLegacyReluxModuleForTest creates a module with old-style Relux files
// (actions.swift, reducer.swift) that AddAction and AddMiddleware commands expect.
func scaffoldLegacyReluxModuleForTest(t *testing.T, moduleName string) string {
	t.Helper()

	root := t.TempDir()
	modulePath := filepath.Join(root, moduleName)
	implPath := filepath.Join(root, moduleName+"Impl")
	implSourcesDir := filepath.Join(implPath, "Sources")

	if err := os.MkdirAll(implSourcesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", implSourcesDir, err)
	}

	actionsContent := `import Foundation
import ` + moduleName + `

public enum ` + moduleName + `Action: Sendable, Equatable {
    case appeared
    case setLoading(Bool)
    case forward(` + moduleName + `PublicAction)
}
`
	reducerContent := `import Foundation

public struct ` + moduleName + `Reducer: Sendable {
    public init() {}

    public nonisolated func reduce(state: ` + moduleName + `State, action: ` + moduleName + `Action) -> ` + moduleName + `State {
        var nextState = state

        switch action {
        case .appeared:
            nextState.isLoading = true
        case .setLoading(let isLoading):
            nextState.isLoading = isLoading
        case .forward:
            break
        }

        return nextState
    }
}
`
	middlewareContent := `import Foundation

public protocol ` + moduleName + `MiddlewareProtocol: Sendable {}

public actor ` + moduleName + `Middleware: ` + moduleName + `MiddlewareProtocol {
    public init() {}
}
`

	writeTestFile(t, filepath.Join(implSourcesDir, "actions.swift"), actionsContent)
	writeTestFile(t, filepath.Join(implSourcesDir, "reducer.swift"), reducerContent)
	writeTestFile(t, filepath.Join(implSourcesDir, "middleware.swift"), middlewareContent)

	return modulePath
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readFileStringForTest(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	return string(content)
}

func containsInOrder(content string, first string, second string) bool {
	firstIndex := strings.Index(content, first)
	secondIndex := strings.Index(content, second)
	if firstIndex < 0 || secondIndex < 0 {
		return false
	}
	return firstIndex < secondIndex
}
