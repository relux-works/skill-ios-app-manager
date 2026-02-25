package relux

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReluxManagerInitAndAddAction(t *testing.T) {
	root := t.TempDir()

	manager, err := NewReluxManager(root)
	if err != nil {
		t.Fatalf("NewReluxManager() error = %v", err)
	}

	const moduleName = "Notes"
	if err := manager.InitModule(context.Background(), moduleName, "feature"); err != nil {
		t.Fatalf("InitModule() error = %v", err)
	}

	// Verify new-style module files were created.
	interfaceDir := filepath.Join(root, moduleName, "Sources", moduleName)
	implDir := filepath.Join(root, moduleName+"Impl", "Sources", moduleName+"Impl")

	for _, expected := range []string{
		filepath.Join(interfaceDir, moduleName+".swift"),
		filepath.Join(interfaceDir, "Module", moduleName+".Module.swift"),
		filepath.Join(interfaceDir, "Module", moduleName+".Module+Interface.swift"),
		filepath.Join(implDir, "Module", moduleName+".Module+Impl.swift"),
	} {
		if _, err := os.Stat(expected); err != nil {
			t.Fatalf("InitModule() expected file %q: %v", expected, err)
		}
	}

	// AddAction operates on legacy actions.swift / reducer.swift files.
	// Seed the legacy files for the AddAction test.
	actionsContent := `import Foundation
import Notes

public enum NotesAction: Sendable, Equatable {
    case appeared
    case forward(NotesPublicAction)
}
`
	reducerContent := `import Foundation

public struct NotesReducer: Sendable {
    public init() {}

    public nonisolated func reduce(state: NotesState, action: NotesAction) -> NotesState {
        var nextState = state

        switch action {
        case .appeared:
            nextState.isLoading = true
        case .forward:
            break
        }

        return nextState
    }
}
`
	writeTestFile(t, filepath.Join(implDir, "actions.swift"), actionsContent)
	writeTestFile(t, filepath.Join(implDir, "reducer.swift"), reducerContent)

	if err := manager.AddAction(context.Background(), moduleName, "didRetry"); err != nil {
		t.Fatalf("AddAction() error = %v", err)
	}

	actionsPath := filepath.Join(implDir, "actions.swift")
	result := readFileStringForTest(t, actionsPath)
	if !containsInOrder(result, "case didRetry", "case forward(NotesPublicAction)") {
		t.Fatalf("expected action case insertion in %q, got:\n%s", actionsPath, result)
	}
}
