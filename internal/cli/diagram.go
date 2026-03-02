package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/diagram"
	"github.com/relux-works/ios-app-manager/internal/registry"
	"github.com/spf13/cobra"

	// Import module packages for init() registration.
	_ "github.com/relux-works/ios-app-manager/internal/appconfig"
	_ "github.com/relux-works/ios-app-manager/internal/httpclient"
	_ "github.com/relux-works/ios-app-manager/internal/ioc"
	_ "github.com/relux-works/ios-app-manager/internal/relux"
	_ "github.com/relux-works/ios-app-manager/internal/scaffold"
	_ "github.com/relux-works/ios-app-manager/internal/securestore"
	_ "github.com/relux-works/ios-app-manager/internal/tokenprovider"
	_ "github.com/relux-works/ios-app-manager/internal/utilities"
)

const (
	defaultDiagramFormat         = "puml"
	defaultDiagramOutputPathBase = "diagrams/scaffolding-pipeline"
	defaultDiagramOutputPath     = defaultDiagramOutputPathBase + ".puml"
)

var supportedDiagramFormats = []string{defaultDiagramFormat, "png", "svg"}

func newDiagramCommand(_ *RootOptions) *cobra.Command {
	outputPath := defaultDiagramOutputPath
	outputFormat := defaultDiagramFormat

	cmd := &cobra.Command{
		Use:   "diagram",
		Short: "Generate PlantUML module dependency diagram",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			format, err := normalizeDiagramFormat(outputFormat)
			if err != nil {
				return err
			}

			targetOutputPath := strings.TrimSpace(outputPath)
			if targetOutputPath == "" || !cmd.Flags().Changed("output") {
				targetOutputPath = defaultDiagramOutputPathForFormat(format)
			}

			targetOutputPath = filepath.Clean(targetOutputPath)
			targetDirectory := filepath.Dir(targetOutputPath)
			if targetDirectory != "." {
				if err := os.MkdirAll(targetDirectory, 0o755); err != nil {
					return fmt.Errorf("create output directory %q: %w", targetDirectory, err)
				}
			}

			modules := registry.AllSorted()
			contents := diagram.GeneratePlantUML(modules)

			if format == defaultDiagramFormat {
				if err := os.WriteFile(targetOutputPath, []byte(contents), 0o644); err != nil {
					return fmt.Errorf("write diagram %q: %w", targetOutputPath, err)
				}
			} else {
				pumlPath := plantUMLSourcePath(targetOutputPath)
				if err := os.WriteFile(pumlPath, []byte(contents), 0o644); err != nil {
					return fmt.Errorf("write diagram %q: %w", pumlPath, err)
				}

				if err := renderDiagramWithPlantUML(pumlPath, format); err != nil {
					return err
				}

				renderedPath := renderedDiagramPath(pumlPath, format)
				if renderedPath != targetOutputPath {
					if err := os.Rename(renderedPath, targetOutputPath); err != nil {
						return fmt.Errorf("move rendered diagram %q to %q: %w", renderedPath, targetOutputPath, err)
					}
				}
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Diagram written to %s\n", targetOutputPath)
			return err
		},
	}

	cmd.Flags().StringVar(
		&outputPath,
		"output",
		defaultDiagramOutputPath,
		"Path to output PlantUML file",
	)
	cmd.Flags().StringVar(
		&outputFormat,
		"format",
		defaultDiagramFormat,
		"Output format: puml, png, svg",
	)

	return cmd
}

func normalizeDiagramFormat(raw string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(raw))
	if format == "" {
		format = defaultDiagramFormat
	}

	for _, supported := range supportedDiagramFormats {
		if format == supported {
			return format, nil
		}
	}

	return "", fmt.Errorf(
		"unsupported format %q (supported: %s)",
		raw,
		strings.Join(supportedDiagramFormats, ", "),
	)
}

func defaultDiagramOutputPathForFormat(format string) string {
	return defaultDiagramOutputPathBase + "." + format
}

func plantUMLSourcePath(outputPath string) string {
	extension := filepath.Ext(outputPath)
	if extension == "" {
		return outputPath + ".puml"
	}

	return strings.TrimSuffix(outputPath, extension) + ".puml"
}

func renderedDiagramPath(pumlPath, format string) string {
	base := strings.TrimSuffix(pumlPath, filepath.Ext(pumlPath))
	return base + "." + format
}

func renderDiagramWithPlantUML(pumlPath, format string) error {
	if _, err := exec.LookPath("plantuml"); err != nil {
		return errors.New("plantuml not found in PATH (install: brew install plantuml)")
	}

	cmd := exec.Command("plantuml", "-t"+format, pumlPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			return fmt.Errorf("render diagram with plantuml: %w", err)
		}
		return fmt.Errorf("render diagram with plantuml: %w: %s", err, msg)
	}

	return nil
}
