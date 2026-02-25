package relux

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// InitModuleInput captures parameters for the module-init command.
type InitModuleInput struct {
	ModuleName           string
	ModulePath           string
	InterfacePackageName string
	ImplPackageName      string
	TemplateSet          []string
}

// InitCommand bootstraps a full Relux module structure from templates.
type InitCommand struct {
	engine *TemplateEngine
}

// NewInitCommand creates a relux-init command.
func NewInitCommand(engine *TemplateEngine) (*InitCommand, error) {
	if engine == nil {
		return nil, fmt.Errorf("template engine is required")
	}

	return &InitCommand{engine: engine}, nil
}

type templateTarget struct {
	templateName string
	outputPath   string
}

var defaultTemplateSet = []string{
	"namespace",
	"module",
	"interface",
	"impl",
}

// Run executes relux-init and returns paths written to disk.
func (c *InitCommand) Run(ctx context.Context, input InitModuleInput) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	moduleName, err := normalizeModuleName(input.ModuleName)
	if err != nil {
		return nil, err
	}

	layout, err := resolveModuleLayout(moduleName, input.ModulePath)
	if err != nil {
		return nil, err
	}

	templateVars := TemplateVariables{
		ModuleName:           moduleName,
		ModuleNameLower:      lowerFirst(moduleName),
		InterfacePackageName: defaultValue(input.InterfacePackageName, moduleName),
		ImplPackageName:      defaultValue(input.ImplPackageName, moduleName+"Impl"),
	}

	targets, err := templateTargetsForSet(layout, input.TemplateSet)
	if err != nil {
		return nil, err
	}

	written := make([]string, 0, len(targets))
	for _, target := range targets {
		rendered, renderErr := c.engine.Render(target.templateName, templateVars)
		if renderErr != nil {
			return nil, fmt.Errorf("render %q: %w", target.templateName, renderErr)
		}

		if writeErr := writeFile(target.outputPath, rendered); writeErr != nil {
			return nil, writeErr
		}

		written = append(written, target.outputPath)
	}

	sort.Strings(written)
	return written, nil
}

func templateTargetsForSet(layout moduleLayout, templateSet []string) ([]templateTarget, error) {
	definitions := map[string]templateTarget{
		"namespace": {templateName: "namespace.swift.tmpl", outputPath: filepath.Join(layout.InterfaceSourcesDir, "Namespace.swift")},
		"module":    {templateName: "module.swift.tmpl", outputPath: filepath.Join(layout.InterfaceSourcesDir, "Module.swift")},
		"interface": {templateName: "interface.swift.tmpl", outputPath: filepath.Join(layout.InterfaceSourcesDir, "Module+Interface.swift")},
		"impl":      {templateName: "impl.swift.tmpl", outputPath: filepath.Join(layout.ImplSourcesDir, "Module+Impl.swift")},

		"relux_namespace": {templateName: "relux_namespace.swift.tmpl", outputPath: filepath.Join(layout.InterfaceSourcesDir, "Namespace.swift")},
		"relux_interface": {templateName: "relux_interface.swift.tmpl", outputPath: filepath.Join(layout.InterfaceSourcesDir, "Module+Interface.swift")},
		"relux_action":    {templateName: "relux_action.swift.tmpl", outputPath: filepath.Join(layout.InterfaceSourcesDir, "Business+Action.swift")},
		"relux_effect":    {templateName: "relux_effect.swift.tmpl", outputPath: filepath.Join(layout.InterfaceSourcesDir, "Business+Effect.swift")},
		"relux_impl":      {templateName: "relux_impl.swift.tmpl", outputPath: filepath.Join(layout.ImplSourcesDir, "Module+Impl.swift")},
		"relux_state":     {templateName: "relux_state.swift.tmpl", outputPath: filepath.Join(layout.ImplSourcesDir, "Business+State.swift")},
		"relux_flow":      {templateName: "relux_flow.swift.tmpl", outputPath: filepath.Join(layout.ImplSourcesDir, "Business+Flow.swift")},
	}

	selected := templateSet
	if len(selected) == 0 {
		selected = defaultTemplateSet
	}

	targets := make([]templateTarget, 0, len(selected))
	seen := make(map[string]struct{}, len(selected))
	for _, raw := range selected {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}

		target, ok := definitions[name]
		if !ok {
			return nil, fmt.Errorf("template %q is not supported", name)
		}

		targets = append(targets, target)
		seen[name] = struct{}{}
	}

	return targets, nil
}

func defaultValue(input string, fallback string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed != "" {
		return trimmed
	}
	return fallback
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
