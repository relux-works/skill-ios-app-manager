package relux

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

const templatesDir = "templates"

//go:embed templates/*.tmpl
var templatesFS embed.FS

var requiredTemplateNames = []string{
	"namespace.swift.tmpl",
	"module.swift.tmpl",
	"interface.swift.tmpl",
	"impl.swift.tmpl",
	"relux_namespace.swift.tmpl",
	"relux_interface.swift.tmpl",
	"relux_action.swift.tmpl",
	"relux_effect.swift.tmpl",
	"relux_impl.swift.tmpl",
	"relux_state.swift.tmpl",
	"relux_flow.swift.tmpl",
}

// TemplateVariables is the parameter set used by all module templates.
type TemplateVariables struct {
	ModuleName           string
	ModuleNameLower      string
	InterfacePackageName string
	ImplPackageName      string
}

func (v TemplateVariables) validate() error {
	if strings.TrimSpace(v.ModuleName) == "" {
		return errors.New("ModuleName is required")
	}
	if strings.TrimSpace(v.ModuleNameLower) == "" {
		return errors.New("ModuleNameLower is required")
	}
	if strings.TrimSpace(v.InterfacePackageName) == "" {
		return errors.New("InterfacePackageName is required")
	}
	if strings.TrimSpace(v.ImplPackageName) == "" {
		return errors.New("ImplPackageName is required")
	}
	return nil
}

// TemplateEngine renders and writes embedded Swift templates.
type TemplateEngine struct {
	templates fs.FS
	names     []string
}

// NewTemplateEngine loads all required Relux templates.
func NewTemplateEngine() (*TemplateEngine, error) {
	return newTemplateEngine(templatesFS)
}

func newTemplateEngine(embeddedTemplates fs.FS) (*TemplateEngine, error) {
	if len(requiredTemplateNames) == 0 {
		return nil, errors.New("required template list is empty")
	}

	names := make([]string, len(requiredTemplateNames))
	copy(names, requiredTemplateNames)
	for _, name := range names {
		templatePath := filepath.ToSlash(filepath.Join(templatesDir, name))
		if _, err := fs.ReadFile(embeddedTemplates, templatePath); err != nil {
			return nil, fmt.Errorf("load required template %q: %w", name, err)
		}
	}

	return &TemplateEngine{
		templates: embeddedTemplates,
		names:     names,
	}, nil
}

// TemplateNames returns a stable list of required template file names.
func (e *TemplateEngine) TemplateNames() []string {
	out := make([]string, len(e.names))
	copy(out, e.names)
	return out
}

// Render renders one template using the provided variables.
func (e *TemplateEngine) Render(templateName string, vars TemplateVariables) ([]byte, error) {
	if err := vars.validate(); err != nil {
		return nil, err
	}
	if !e.hasTemplate(templateName) {
		return nil, fmt.Errorf("template %q is not supported", templateName)
	}

	templatePath := filepath.ToSlash(filepath.Join(templatesDir, templateName))
	contents, err := fs.ReadFile(e.templates, templatePath)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", templateName, err)
	}

	tpl, err := template.New(templateName).Option("missingkey=error").Parse(string(contents))
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", templateName, err)
	}

	var rendered strings.Builder
	if err := tpl.Execute(&rendered, vars); err != nil {
		return nil, fmt.Errorf("render template %q: %w", templateName, err)
	}

	return []byte(rendered.String()), nil
}

// RenderAll renders all required templates. Keys are output file names (.tmpl removed).
func (e *TemplateEngine) RenderAll(vars TemplateVariables) (map[string][]byte, error) {
	if err := vars.validate(); err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(e.names))
	for _, templateName := range e.names {
		rendered, err := e.Render(templateName, vars)
		if err != nil {
			return nil, err
		}

		outputName := strings.TrimSuffix(templateName, ".tmpl")
		result[outputName] = rendered
	}
	return result, nil
}

// WriteAll renders all templates and writes them into targetDir.
func (e *TemplateEngine) WriteAll(targetDir string, vars TemplateVariables) ([]string, error) {
	rendered, err := e.RenderAll(vars)
	if err != nil {
		return nil, err
	}

	outputNames := make([]string, 0, len(rendered))
	for name := range rendered {
		outputNames = append(outputNames, name)
	}
	sort.Strings(outputNames)

	written := make([]string, 0, len(outputNames))
	for _, outputName := range outputNames {
		outputPath := filepath.Join(targetDir, outputName)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return nil, fmt.Errorf("create directory for %q: %w", outputPath, err)
		}

		if err := os.WriteFile(outputPath, rendered[outputName], 0o644); err != nil {
			return nil, fmt.Errorf("write rendered template %q: %w", outputPath, err)
		}

		written = append(written, outputPath)
	}

	return written, nil
}

func (e *TemplateEngine) hasTemplate(templateName string) bool {
	templatePath := filepath.ToSlash(filepath.Join(templatesDir, templateName))
	_, err := fs.Stat(e.templates, templatePath)
	return err == nil
}
