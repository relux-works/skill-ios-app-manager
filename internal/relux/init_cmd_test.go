package relux

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestInitCommandRunCreatesAllFiles(t *testing.T) {
	engine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("NewTemplateEngine() error = %v", err)
	}

	command, err := NewInitCommand(engine)
	if err != nil {
		t.Fatalf("NewInitCommand() error = %v", err)
	}

	modulePath := filepath.Join(t.TempDir(), "Notes")
	written, err := command.Run(context.Background(), InitModuleInput{
		ModuleName: "Notes",
		ModulePath: modulePath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	implPath := modulePath + "Impl"
	want := []string{
		filepath.Join(modulePath, "Sources", "Notes.swift"),
		filepath.Join(modulePath, "Sources", "Module", "Notes.Module.swift"),
		filepath.Join(modulePath, "Sources", "Module", "Notes.Module+Interface.swift"),
		filepath.Join(implPath, "Sources", "Module", "Notes.Module+Impl.swift"),
	}
	sort.Strings(want)

	if len(written) != len(want) {
		t.Fatalf("Run() wrote %d files, want %d\nwrote: %v", len(written), len(want), written)
	}

	for i := range want {
		if written[i] != want[i] {
			t.Fatalf("Run() wrote path[%d] = %q, want %q", i, written[i], want[i])
		}
	}

	for _, path := range want {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, readErr)
		}

		if strings.Contains(string(content), "{{") || strings.Contains(string(content), "}}") {
			t.Fatalf("%q still contains template tokens", path)
		}
	}
}

func TestInitCommandRunUsesExistingInterfaceAndImplDirs(t *testing.T) {
	engine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("NewTemplateEngine() error = %v", err)
	}

	command, err := NewInitCommand(engine)
	if err != nil {
		t.Fatalf("NewInitCommand() error = %v", err)
	}

	modulePath := filepath.Join(t.TempDir(), "Notes")
	interfaceSources := filepath.Join(modulePath, "Interface", "Sources")
	implSources := filepath.Join(modulePath, "Impl", "Sources")

	if err := os.MkdirAll(interfaceSources, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", interfaceSources, err)
	}
	if err := os.MkdirAll(implSources, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", implSources, err)
	}

	_, err = command.Run(context.Background(), InitModuleInput{
		ModuleName: "Notes",
		ModulePath: modulePath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// When Interface/Sources/ and Impl/Sources/ exist, they are used as the layout dirs.
	checks := []string{
		filepath.Join(interfaceSources, "Notes.swift"),
		filepath.Join(interfaceSources, "Module", "Notes.Module.swift"),
		filepath.Join(interfaceSources, "Module", "Notes.Module+Interface.swift"),
		filepath.Join(implSources, "Module", "Notes.Module+Impl.swift"),
	}

	for _, path := range checks {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected generated file %q: %v", path, err)
		}
	}
}

func TestInitCommandRunReluxFeatureTemplateSet(t *testing.T) {
	engine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("NewTemplateEngine() error = %v", err)
	}

	command, err := NewInitCommand(engine)
	if err != nil {
		t.Fatalf("NewInitCommand() error = %v", err)
	}

	modulePath := filepath.Join(t.TempDir(), "Auth")
	reluxFeatureSet := []string{
		"relux_namespace", "module", "relux_interface",
		"relux_action", "relux_effect",
		"relux_impl", "relux_state", "relux_flow",
	}
	written, err := command.Run(context.Background(), InitModuleInput{
		ModuleName:  "Auth",
		ModulePath:  modulePath,
		TemplateSet: reluxFeatureSet,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	implPath := modulePath + "Impl"
	want := []string{
		filepath.Join(modulePath, "Sources", "Auth.swift"),
		filepath.Join(modulePath, "Sources", "Module", "Auth.Module.swift"),
		filepath.Join(modulePath, "Sources", "Module", "Auth.Module+Interface.swift"),
		filepath.Join(modulePath, "Sources", "Business", "Auth.Business+Action.swift"),
		filepath.Join(modulePath, "Sources", "Business", "Auth.Business+Effect.swift"),
		filepath.Join(implPath, "Sources", "Module", "Auth.Module+Impl.swift"),
		filepath.Join(implPath, "Sources", "Business", "Auth.Business+State.swift"),
		filepath.Join(implPath, "Sources", "Business", "Auth.Business+Flow.swift"),
	}
	sort.Strings(want)

	if len(written) != len(want) {
		t.Fatalf("Run() wrote %d files, want %d\nwrote: %v\nwant:  %v", len(written), len(want), written, want)
	}
	for i := range want {
		if written[i] != want[i] {
			t.Fatalf("Run() wrote path[%d] = %q, want %q", i, written[i], want[i])
		}
	}

	for _, path := range want {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, readErr)
		}
		if strings.Contains(string(content), "{{") || strings.Contains(string(content), "}}") {
			t.Fatalf("%q still contains template tokens", path)
		}
	}
}

func TestInitCommandRunTemplateSet(t *testing.T) {
	engine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("NewTemplateEngine() error = %v", err)
	}

	command, err := NewInitCommand(engine)
	if err != nil {
		t.Fatalf("NewInitCommand() error = %v", err)
	}

	modulePath := filepath.Join(t.TempDir(), "Notes")
	written, err := command.Run(context.Background(), InitModuleInput{
		ModuleName:  "Notes",
		ModulePath:  modulePath,
		TemplateSet: []string{"namespace", "impl"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	implPath := modulePath + "Impl"
	want := []string{
		filepath.Join(modulePath, "Sources", "Notes.swift"),
		filepath.Join(implPath, "Sources", "Module", "Notes.Module+Impl.swift"),
	}
	sort.Strings(want)

	if len(written) != len(want) {
		t.Fatalf("Run() wrote %d files, want %d\nwrote: %v", len(written), len(want), written)
	}
	for i := range want {
		if written[i] != want[i] {
			t.Fatalf("Run() wrote path[%d] = %q, want %q", i, written[i], want[i])
		}
	}

	for _, path := range want {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected generated file %q: %v", path, err)
		}
	}
}
