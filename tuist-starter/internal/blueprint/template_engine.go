package blueprint

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

const templatesDir = "templates"

//go:embed templates/*.tmpl
var templatesFS embed.FS

// TemplateVars holds all variables available to blueprint templates.
type TemplateVars struct {
	ModuleName           string
	ModuleNameLower      string
	InterfacePackageName string
	ImplPackageName      string
	HasHTTP              bool
	HasWS                bool
	HasLocal             bool
	HasUI                bool
	EntryPointName       string
	Features             []Feature
	Components           []Component

	// Per-feature rendering (populated when rendering feature-specific templates).
	FeatureName      string
	FeatureNameLower string
	IsEntryPoint     bool

	// Per-component rendering (populated when rendering component-specific templates).
	ComponentName      string
	ComponentNameLower string
}

// WithFeature returns a copy of vars with FeatureName set for per-feature rendering.
func (v TemplateVars) WithFeature(f Feature) TemplateVars {
	v.FeatureName = f.Name
	v.FeatureNameLower = f.NameLower
	v.IsEntryPoint = v.EntryPointName == f.Name
	return v
}

// WithComponent returns a copy of vars with ComponentName set for per-component rendering.
func (v TemplateVars) WithComponent(c Component) TemplateVars {
	v.ComponentName = c.Name
	v.ComponentNameLower = c.NameLower
	return v
}

// TemplateEngine renders embedded blueprint Swift templates.
type TemplateEngine struct {
	templates fs.FS
}

// NewTemplateEngine creates a blueprint template engine.
func NewTemplateEngine() *TemplateEngine {
	return newTemplateEngine(templatesFS)
}

func newTemplateEngine(fsys fs.FS) *TemplateEngine {
	return &TemplateEngine{templates: fsys}
}

// BuildVars constructs TemplateVars from a Blueprint.
func BuildVars(bp *Blueprint) TemplateVars {
	return TemplateVars{
		ModuleName:           bp.Name,
		ModuleNameLower:      lowerFirst(bp.Name),
		InterfacePackageName: bp.Name,
		ImplPackageName:      bp.Name + "Impl",
		HasHTTP:              bp.HasHTTP(),
		HasWS:                bp.HasWS(),
		HasLocal:             bp.HasLocal(),
		HasUI:                bp.HasUI(),
		EntryPointName:       bp.EntryPoint(),
		Features:             bp.Features(),
		Components:           bp.Components(),
	}
}

// Render renders a single template by name with the given variables.
func (e *TemplateEngine) Render(templateName string, vars TemplateVars) ([]byte, error) {
	templatePath := filepath.ToSlash(filepath.Join(templatesDir, templateName))
	contents, err := fs.ReadFile(e.templates, templatePath)
	if err != nil {
		return nil, fmt.Errorf("read template %q: %w", templateName, err)
	}

	funcMap := template.FuncMap{
		"lower": strings.ToLower,
	}

	tpl, err := template.New(templateName).Funcs(funcMap).Option("missingkey=error").Parse(string(contents))
	if err != nil {
		return nil, fmt.Errorf("parse template %q: %w", templateName, err)
	}

	var rendered strings.Builder
	if err := tpl.Execute(&rendered, vars); err != nil {
		return nil, fmt.Errorf("render template %q: %w", templateName, err)
	}

	return []byte(rendered.String()), nil
}

// HasTemplate checks if a template file exists.
func (e *TemplateEngine) HasTemplate(templateName string) bool {
	templatePath := filepath.ToSlash(filepath.Join(templatesDir, templateName))
	_, err := fs.Stat(e.templates, templatePath)
	return err == nil
}
