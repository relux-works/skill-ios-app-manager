package ioc

import (
	"strings"
	"testing"
)

const matureAppBootstrapFixture = `import CustomRuntime
import SwiftUI

@main
struct DemoApp: App {
    init() {
        CustomRuntime.prepare()
        Registry.configure(runtimeMode: CustomRuntimeMode.current())
    }

    var body: some Scene {
        WindowGroup {
            Text("Demo")
        }
    }
}
`

func TestConvergeManagedAppBootstrapContentInsertsBeforeRegistryConfiguration(t *testing.T) {
	t.Parallel()

	patch := AppManagedBootstrapPatch{
		ID:   "fireauth-relux",
		Call: "Registry.configureFireAuthReluxFromProcess()",
	}
	first, err := ConvergeManagedAppBootstrapContent(matureAppBootstrapFixture, patch)
	if err != nil {
		t.Fatalf("ConvergeManagedAppBootstrapContent() error = %v", err)
	}
	second, err := ConvergeManagedAppBootstrapContent(first, patch)
	if err != nil {
		t.Fatalf("second ConvergeManagedAppBootstrapContent() error = %v", err)
	}
	if second != first {
		t.Fatalf("managed app bootstrap patch is not byte-idempotent:\n%s", second)
	}

	for _, preserved := range []string{
		"import CustomRuntime",
		"CustomRuntime.prepare()",
		"Registry.configure(runtimeMode: CustomRuntimeMode.current())",
		`Text("Demo")`,
	} {
		if !strings.Contains(first, preserved) {
			t.Fatalf("managed app bootstrap patch lost %q:\n%s", preserved, first)
		}
	}

	managedCall := strings.Index(first, "Registry.configureFireAuthReluxFromProcess()")
	registryConfigure := strings.Index(first, "Registry.configure(runtimeMode:")
	if managedCall < 0 || registryConfigure < 0 || managedCall >= registryConfigure {
		t.Fatalf("managed FireAuth call must precede Registry.configure(...):\n%s", first)
	}
	if strings.Count(first, "// ios-app-manager:fireauth-relux-bootstrap:begin") != 1 ||
		strings.Count(first, "Registry.configureFireAuthReluxFromProcess()") != 1 {
		t.Fatalf("managed app bootstrap block is duplicated:\n%s", first)
	}
}

func TestConvergeManagedAppBootstrapContentUpdatesOwnedCall(t *testing.T) {
	t.Parallel()

	first, err := ConvergeManagedAppBootstrapContent(matureAppBootstrapFixture, AppManagedBootstrapPatch{
		ID:   "fireauth-relux",
		Call: "Registry.installLegacyFireAuthSelection()",
	})
	if err != nil {
		t.Fatalf("initial ConvergeManagedAppBootstrapContent() error = %v", err)
	}
	updated, err := ConvergeManagedAppBootstrapContent(first, AppManagedBootstrapPatch{
		ID:   "fireauth-relux",
		Call: "Registry.configureFireAuthReluxFromProcess()",
	})
	if err != nil {
		t.Fatalf("updated ConvergeManagedAppBootstrapContent() error = %v", err)
	}
	if strings.Contains(updated, "installLegacyFireAuthSelection") {
		t.Fatalf("managed app bootstrap retained stale call:\n%s", updated)
	}
	if strings.Count(updated, "Registry.configureFireAuthReluxFromProcess()") != 1 {
		t.Fatalf("managed app bootstrap did not converge current call:\n%s", updated)
	}
}

func TestConvergeManagedAppBootstrapContentRejectsUnsupportedConfigureShape(t *testing.T) {
	t.Parallel()

	for name, content := range map[string]string{
		"missing": strings.Replace(
			matureAppBootstrapFixture,
			"        Registry.configure(runtimeMode: CustomRuntimeMode.current())\n",
			"",
			1,
		),
		"multiple": strings.Replace(
			matureAppBootstrapFixture,
			"        Registry.configure(runtimeMode: CustomRuntimeMode.current())",
			"        Registry.configure()\n        Registry.configure(runtimeMode: CustomRuntimeMode.current())",
			1,
		),
		"multiline": strings.Replace(
			matureAppBootstrapFixture,
			"        Registry.configure(runtimeMode: CustomRuntimeMode.current())",
			"        Registry.configure(\n            runtimeMode: CustomRuntimeMode.current()\n        )",
			1,
		),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := ConvergeManagedAppBootstrapContent(content, AppManagedBootstrapPatch{
				ID:   "fireauth-relux",
				Call: "Registry.configureFireAuthReluxFromProcess()",
			})
			if err == nil || !strings.Contains(err.Error(), "supported Registry.configure") {
				t.Fatalf("error = %v, want explicit unsupported Registry.configure error", err)
			}
		})
	}
}
