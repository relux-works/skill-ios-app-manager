package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/ios-app-manager/internal/registry"
)

func TestDiagramHelpShowsOutputFlag(t *testing.T) {
	t.Parallel()

	output, err := executeRootCommand("diagram", "--help")
	if err != nil {
		t.Fatalf("executeRootCommand(diagram --help) error = %v", err)
	}

	for _, expected := range []string{"--output", "--format", defaultDiagramOutputPath} {
		if !strings.Contains(output, expected) {
			t.Fatalf("diagram --help output missing %q:\n%s", expected, output)
		}
	}
}

func TestDiagramCommandWritesOutputFile(t *testing.T) {
	t.Parallel()

	outputPath := filepath.Join(t.TempDir(), "generated", "diagram.puml")
	output, err := executeRootCommand("diagram", "--output", outputPath)
	if err != nil {
		t.Fatalf("executeRootCommand(diagram --output) error = %v", err)
	}

	if !strings.Contains(output, "Diagram written to "+outputPath) {
		t.Fatalf("diagram output missing success message:\n%s", output)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", outputPath, err)
	}

	diagram := string(content)
	for _, marker := range []string{"@startuml", "@enduml"} {
		if !strings.Contains(diagram, marker) {
			t.Fatalf("generated diagram missing marker %q:\n%s", marker, diagram)
		}
	}

	for _, mod := range registry.AllSorted() {
		if !strings.Contains(diagram, mod.Name) {
			t.Fatalf("generated diagram missing module name %q:\n%s", mod.Name, diagram)
		}
	}
}

func TestDiagramCommandRejectsInvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := executeRootCommand("diagram", "--format", "xyz")
	if err == nil {
		t.Fatal("executeRootCommand(diagram --format xyz) error = nil, want validation error")
	}
	for _, expected := range []string{`unsupported format "xyz"`, "supported: puml, png, svg"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("error = %q, want %q", err.Error(), expected)
		}
	}
}

func TestDiagramCommandUsesFormatSpecificDefaultOutputPath(t *testing.T) {
	workdir := t.TempDir()
	setWorkingDirectory(t, workdir)

	binDir := filepath.Join(workdir, "bin")
	writeFakePlantUML(t, binDir)
	t.Setenv("PATH", binDir)

	testCases := []struct {
		name   string
		format string
	}{
		{name: "puml", format: "puml"},
		{name: "png", format: "png"},
		{name: "svg", format: "svg"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"diagram"}
			if tc.format != "" {
				args = append(args, "--format", tc.format)
			}

			output, err := executeRootCommand(args...)
			if err != nil {
				t.Fatalf("executeRootCommand(%v) error = %v", args, err)
			}

			expectedPath := defaultDiagramOutputPathForFormat(tc.format)
			if !strings.Contains(output, "Diagram written to "+expectedPath) {
				t.Fatalf("diagram output missing success message for %q:\n%s", expectedPath, output)
			}

			absoluteOutputPath := filepath.Join(workdir, expectedPath)
			if _, err := os.Stat(absoluteOutputPath); err != nil {
				t.Fatalf("Stat(%q) error = %v", absoluteOutputPath, err)
			}

			if tc.format == defaultDiagramFormat {
				return
			}

			sourcePath := filepath.Join(workdir, defaultDiagramOutputPathForFormat(defaultDiagramFormat))
			if _, err := os.Stat(sourcePath); err != nil {
				t.Fatalf("Stat(%q) error = %v", sourcePath, err)
			}
		})
	}
}

func TestDiagramCommandRendersPNGAndSVG(t *testing.T) {
	binDir := t.TempDir()
	writeFakePlantUML(t, binDir)
	t.Setenv("PATH", binDir)

	testCases := []struct {
		name   string
		format string
	}{
		{name: "png", format: "png"},
		{name: "svg", format: "svg"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			outputPath := filepath.Join(t.TempDir(), "generated", "diagram."+tc.format)
			output, err := executeRootCommand("diagram", "--format", tc.format, "--output", outputPath)
			if err != nil {
				t.Fatalf("executeRootCommand(diagram --format %s --output) error = %v", tc.format, err)
			}

			if !strings.Contains(output, "Diagram written to "+outputPath) {
				t.Fatalf("diagram output missing success message:\n%s", output)
			}

			renderedContent, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", outputPath, err)
			}
			if !strings.Contains(string(renderedContent), "rendered-"+tc.format) {
				t.Fatalf("rendered output %q missing marker for format %q:\n%s", outputPath, tc.format, string(renderedContent))
			}

			sourcePath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".puml"
			sourceContent, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", sourcePath, err)
			}
			for _, marker := range []string{"@startuml", "@enduml"} {
				if !strings.Contains(string(sourceContent), marker) {
					t.Fatalf("generated source diagram missing marker %q:\n%s", marker, string(sourceContent))
				}
			}
		})
	}
}

func TestDiagramCommandReturnsClearErrorWhenPlantUMLMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	outputPath := filepath.Join(t.TempDir(), "generated", "diagram.png")
	_, err := executeRootCommand("diagram", "--format", "png", "--output", outputPath)
	if err == nil {
		t.Fatal("executeRootCommand(diagram --format png --output) error = nil, want missing plantuml error")
	}

	const want = "plantuml not found in PATH (install: brew install plantuml)"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func writeFakePlantUML(t *testing.T, binDir string) {
	t.Helper()

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", binDir, err)
	}

	scriptPath := filepath.Join(binDir, "plantuml")
	script := `#!/bin/sh
set -eu

format="${1:-}"
input="${2:-}"

case "$format" in
  -tpng) ext="png" ;;
  -tsvg) ext="svg" ;;
  *) echo "unsupported format: $format" >&2; exit 1 ;;
esac

output="${input%.puml}.$ext"
printf "rendered-%s\n" "$ext" > "$output"
`

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", scriptPath, err)
	}
}

func setWorkingDirectory(t *testing.T, directory string) {
	t.Helper()

	currentDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	if err := os.Chdir(directory); err != nil {
		t.Fatalf("Chdir(%q) error = %v", directory, err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(currentDirectory); err != nil {
			t.Fatalf("restore working directory to %q: %v", currentDirectory, err)
		}
	})
}
