package blueprint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateMinimal(t *testing.T) {
	tmpDir := t.TempDir()
	modulesRoot := filepath.Join(tmpDir, "Packages")
	if err := os.MkdirAll(modulesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	bp := &Blueprint{
		Name: "Settings",
		Type: "relux-feature",
	}

	gen := NewGenerator(modulesRoot)
	written, err := gen.Generate(bp)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Should have exactly the always-generated files (13 total)
	expectedCount := 13
	if len(written) != expectedCount {
		t.Errorf("written files count = %d, want %d", len(written), expectedCount)
		for _, f := range written {
			t.Logf("  %s", f)
		}
	}

	// Verify key files exist and contain module name
	assertFileContains(t, filepath.Join(modulesRoot, "Settings", "Sources", "Settings.swift"), "public enum Settings")
	assertFileContains(t, filepath.Join(modulesRoot, "Settings", "Sources", "Business", "Settings.Business+Action.swift"), "scaffoldedSuccess")
	assertFileContains(t, filepath.Join(modulesRoot, "SettingsImpl", "Sources", "Business", "Settings.Business+State.swift"), "MaybeData")
	assertFileContains(t, filepath.Join(modulesRoot, "Settings", "Sources", "Business", "Middleware", "Settings.Business+IService.swift"), "IService")
	assertFileContains(t, filepath.Join(modulesRoot, "SettingsImpl", "Sources", "Business", "Middleware", "Settings.Business+Flow.swift"), "IService")

	// Should NOT have HTTP or UI files
	assertNoFile(t, filepath.Join(modulesRoot, "Settings", "Sources", "Data", "Api", "Http"))
	assertNoFile(t, filepath.Join(modulesRoot, "Settings", "Sources", "UI"))
}

func TestGenerateWithHTTP(t *testing.T) {
	tmpDir := t.TempDir()
	modulesRoot := filepath.Join(tmpDir, "Packages")
	if err := os.MkdirAll(modulesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	bp := &Blueprint{
		Name: "Auth",
		Type: "relux-feature",
		Data: &DataConfig{HTTP: true},
	}

	gen := NewGenerator(modulesRoot)
	written, err := gen.Generate(bp)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// 13 base + 4 HTTP = 17
	expectedCount := 17
	if len(written) != expectedCount {
		t.Errorf("written files count = %d, want %d", len(written), expectedCount)
		for _, f := range written {
			t.Logf("  %s", f)
		}
	}

	// HTTP-specific files (interface)
	assertFileContains(t, filepath.Join(modulesRoot, "Auth", "Sources", "Data", "Api", "Http", "Auth.Data+Api+IFetcher.swift"), "IFetcher")
	// HTTP-specific files (impl)
	assertFileContains(t, filepath.Join(modulesRoot, "AuthImpl", "Sources", "Data", "Api", "Http", "Auth.Data+Api+Fetcher.swift"), "actor Fetcher")
	assertFileContains(t, filepath.Join(modulesRoot, "Auth", "Sources", "Data", "Api", "Http", "Auth.Data+Api+Fetcher+Config.swift"), "scaffoldedEndpoint")
	assertFileContains(t, filepath.Join(modulesRoot, "Auth", "Sources", "Data", "Api", "DTO", "Auth.Data+Api+DTO+ScaffoldedResponse.swift"), "ScaffoldedResponse")

	// Namespace should include Http sub-namespace
	assertFileContains(t, filepath.Join(modulesRoot, "Auth", "Sources", "Auth.swift"), "public enum Http {}")

	// Impl should wire up fetcher via IRpcAsyncClient
	assertFileContains(t, filepath.Join(modulesRoot, "AuthImpl", "Sources", "Module", "Auth.Module+Impl.swift"), "import HttpClient")
	assertFileContains(t, filepath.Join(modulesRoot, "AuthImpl", "Sources", "Module", "Auth.Module+Impl.swift"), "client: any IRpcAsyncClient")

	// Service interface in iface, impl in impl
	assertFileContains(t, filepath.Join(modulesRoot, "Auth", "Sources", "Business", "Middleware", "Auth.Business+IService.swift"), "IService")
	assertFileContains(t, filepath.Join(modulesRoot, "AuthImpl", "Sources", "Business", "Middleware", "Auth.Business+Service.swift"), "fetcher.fetchScaffolded")
}

func TestGenerateWithUI(t *testing.T) {
	tmpDir := t.TempDir()
	modulesRoot := filepath.Join(tmpDir, "Packages")
	if err := os.MkdirAll(modulesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	bp := &Blueprint{
		Name: "Auth",
		Type: "relux-feature",
		UI: &UIConfig{
			Features:   []string{"Login", "Register"},
			Components: []string{"PasswordField"},
		},
	}

	gen := NewGenerator(modulesRoot)
	written, err := gen.Generate(bp)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// 13 base + 3 once-per-module UI + 2*4 per-feature + 1 component = 25
	expectedCount := 25
	if len(written) != expectedCount {
		t.Errorf("written files count = %d, want %d", len(written), expectedCount)
		for _, f := range written {
			t.Logf("  %s", f)
		}
	}

	implSrc := filepath.Join(modulesRoot, "AuthImpl", "Sources")
	ifaceSrc := filepath.Join(modulesRoot, "Auth", "Sources")

	// Once-per-module UI files (ViewState and Router are in impl because they access State)
	assertFileContains(t, filepath.Join(implSrc, "UI", "Auth.UI+ViewState.swift"), "ViewState")
	assertFileContains(t, filepath.Join(implSrc, "UI", "Auth.UI+Router.swift"), "Router")
	assertFileContains(t, filepath.Join(ifaceSrc, "UI", "Model", "Auth.UI+Model+Page.swift"), "case login")

	// Per-feature: Login
	assertFileContains(t, filepath.Join(implSrc, "UI", "Login", "Auth.UI+Login+Container.swift"), "Container")
	assertFileContains(t, filepath.Join(implSrc, "UI", "Login", "Auth.UI+Login+Container+LocalState.swift"), "LocalState")
	assertFileContains(t, filepath.Join(implSrc, "UI", "Login", "Page", "Auth.UI+Login+Page.swift"), "Page")
	assertFileContains(t, filepath.Join(implSrc, "UI", "Login", "Page", "Auth.UI+Login+Page+Props.swift"), "Props")

	// Per-feature: Register
	assertFileContains(t, filepath.Join(implSrc, "UI", "Register", "Auth.UI+Register+Container.swift"), "Container")

	// Component
	assertFileContains(t, filepath.Join(ifaceSrc, "UI", "Components", "Auth.UI+PasswordField.swift"), "PasswordField")
}

func TestGenerateEntryPointContainer(t *testing.T) {
	tmpDir := t.TempDir()
	modulesRoot := filepath.Join(tmpDir, "Packages")
	if err := os.MkdirAll(modulesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	bp := &Blueprint{
		Name: "Auth",
		Type: "relux-feature",
		UI: &UIConfig{
			EntryPoint: "Login",
			Features:   []string{"Login", "Register"},
		},
	}

	gen := NewGenerator(modulesRoot)
	if _, err := gen.Generate(bp); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	implSrc := filepath.Join(modulesRoot, "AuthImpl", "Sources")

	// Entry point container (Login) should have NavigationStack + navigationDestination
	loginContainer := readTestFile(t, filepath.Join(implSrc, "UI", "Login", "Auth.UI+Login+Container.swift"))
	assertContains(t, loginContainer, "NavigationStack", "entry container missing NavigationStack")
	assertContains(t, loginContainer, ".navigationDestination", "entry container missing .navigationDestination")
	assertContains(t, loginContainer, "Router.destination", "entry container missing Router.destination")

	// Non-entry container (Register) should NOT have NavigationStack
	registerContainer := readTestFile(t, filepath.Join(implSrc, "UI", "Register", "Auth.UI+Register+Container.swift"))
	assertNotContains(t, registerContainer, "NavigationStack", "non-entry container should not have NavigationStack")
	assertNotContains(t, registerContainer, ".navigationDestination", "non-entry container should not have .navigationDestination")

	// Router should be a static struct, not a View
	router := readTestFile(t, filepath.Join(implSrc, "UI", "Auth.UI+Router.swift"))
	assertContains(t, router, "public struct Router", "Router missing struct")
	assertContains(t, router, "public static func destination", "Router missing static destination")
	assertNotContains(t, router, ": View", "Router should not conform to View")
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, content, substr, msg string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("%s: content does not contain %q", msg, substr)
	}
}

func assertNotContains(t *testing.T, content, substr, msg string) {
	t.Helper()
	if strings.Contains(content, substr) {
		t.Errorf("%s: content should not contain %q", msg, substr)
	}
}

func TestGenerateFullBlueprint(t *testing.T) {
	tmpDir := t.TempDir()
	modulesRoot := filepath.Join(tmpDir, "Packages")
	if err := os.MkdirAll(modulesRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	bp := &Blueprint{
		Name: "Auth",
		Type: "relux-feature",
		Data: &DataConfig{HTTP: true},
		UI: &UIConfig{
			Features:   []string{"Login", "Register"},
			Components: []string{"PasswordField"},
		},
	}

	gen := NewGenerator(modulesRoot)
	written, err := gen.Generate(bp)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// 13 base + 4 HTTP + 3 once-per-module UI + 2*4 per-feature + 1 component = 29
	expectedCount := 29
	if len(written) != expectedCount {
		t.Errorf("written files count = %d, want %d", len(written), expectedCount)
		for _, f := range written {
			t.Logf("  %s", f)
		}
	}

	// Verify no template artifacts in generated Swift files
	verifyNoTemplateArtifacts(t, written)
}

func TestGenerateValidationError(t *testing.T) {
	gen := NewGenerator("/tmp/test")

	bp := &Blueprint{Name: "auth", Type: "relux-feature"}
	_, err := gen.Generate(bp)
	if err == nil {
		t.Fatal("expected validation error for non-PascalCase name")
	}
}

func assertFileContains(t *testing.T, path, substr string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("read %q: %v", path, err)
		return
	}
	if !strings.Contains(string(data), substr) {
		t.Errorf("file %q does not contain %q", filepath.Base(path), substr)
	}
}

func assertNoFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("path %q exists but should not", path)
	}
}

func verifyNoTemplateArtifacts(t *testing.T, paths []string) {
	t.Helper()
	artifacts := []string{"{{", "}}", "{%", "%}", "<#"}
	for _, path := range paths {
		if !strings.HasSuffix(path, ".swift") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %q: %v", path, err)
			continue
		}
		content := string(data)
		for _, artifact := range artifacts {
			if strings.Contains(content, artifact) {
				t.Errorf("file %q contains template artifact %q", filepath.Base(path), artifact)
			}
		}
	}
}
