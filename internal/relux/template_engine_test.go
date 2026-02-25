package relux

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func sampleTemplateVariables() TemplateVariables {
	return TemplateVariables{
		ModuleName:           "Notes",
		ModuleNameLower:      "notes",
		InterfacePackageName: "Notes",
		ImplPackageName:      "NotesImpl",
	}
}

func TestTemplateNames(t *testing.T) {
	engine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("NewTemplateEngine() error = %v", err)
	}

	want := []string{
		"namespace.swift.tmpl",
		"module.swift.tmpl",
		"interface.swift.tmpl",
		"impl.swift.tmpl",
	}

	got := engine.TemplateNames()
	if len(got) != len(want) {
		t.Fatalf("TemplateNames() length = %d, want %d", len(got), len(want))
	}

	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("TemplateNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRenderAllMatchesGolden(t *testing.T) {
	engine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("NewTemplateEngine() error = %v", err)
	}

	rendered, err := engine.RenderAll(sampleTemplateVariables())
	if err != nil {
		t.Fatalf("RenderAll() error = %v", err)
	}

	outputNames := make([]string, 0, len(rendered))
	for outputName := range rendered {
		outputNames = append(outputNames, outputName)
	}
	sort.Strings(outputNames)

	for _, outputName := range outputNames {
		t.Run(outputName, func(t *testing.T) {
			goldenPath := filepath.Join("testdata", "golden", outputName)
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", goldenPath, err)
			}

			got := rendered[outputName]
			if string(got) != string(want) {
				t.Fatalf("rendered output mismatch for %s\nwant:\n%s\n\ngot:\n%s", outputName, string(want), string(got))
			}

			assertNoTemplateArtifacts(t, outputName, got)
		})
	}
}

func TestWriteAll(t *testing.T) {
	engine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("NewTemplateEngine() error = %v", err)
	}

	targetDir := t.TempDir()
	writtenPaths, err := engine.WriteAll(targetDir, sampleTemplateVariables())
	if err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}

	if len(writtenPaths) != len(engine.TemplateNames()) {
		t.Fatalf("WriteAll() wrote %d files, want %d", len(writtenPaths), len(engine.TemplateNames()))
	}

	for _, outputPath := range writtenPaths {
		contents, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", outputPath, err)
		}

		assertNoTemplateArtifacts(t, outputPath, contents)
	}
}

func TestRenderRequiresVariables(t *testing.T) {
	engine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("NewTemplateEngine() error = %v", err)
	}

	_, err = engine.Render("namespace.swift.tmpl", TemplateVariables{})
	if err == nil {
		t.Fatal("Render() expected error for empty variables, got nil")
	}
}

func assertNoTemplateArtifacts(t *testing.T, name string, content []byte) {
	t.Helper()

	asString := string(content)
	if strings.Contains(asString, "{{") || strings.Contains(asString, "}}") {
		t.Fatalf("%s still contains template markers: %s", name, preview(asString))
	}
}

func preview(content string) string {
	const max = 80
	if len(content) <= max {
		return content
	}
	return fmt.Sprintf("%s...", content[:max])
}
