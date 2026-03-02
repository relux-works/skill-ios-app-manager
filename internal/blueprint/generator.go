package blueprint

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// renderTarget maps a template to an output file path.
type renderTarget struct {
	templateName string
	outputPath   string
	vars         TemplateVars
}

// Generator orchestrates blueprint template rendering and file writes.
type Generator struct {
	modulesRoot string
	engine      *TemplateEngine
}

// NewGenerator creates a blueprint generator targeting the given modules root directory.
func NewGenerator(modulesRoot string) *Generator {
	return &Generator{
		modulesRoot: modulesRoot,
		engine:      NewTemplateEngine(),
	}
}

// Generate renders all templates defined by the blueprint and writes files to disk.
// Returns the list of written file paths.
func (g *Generator) Generate(bp *Blueprint) ([]string, error) {
	if err := bp.Validate(); err != nil {
		return nil, err
	}

	vars := BuildVars(bp)
	targets := g.buildTargets(bp, vars)

	written := make([]string, 0, len(targets))
	for _, target := range targets {
		rendered, err := g.engine.Render(target.templateName, target.vars)
		if err != nil {
			return nil, fmt.Errorf("render %q: %w", target.templateName, err)
		}

		if err := writeFile(target.outputPath, rendered); err != nil {
			return nil, err
		}

		written = append(written, target.outputPath)
	}

	sort.Strings(written)
	return written, nil
}

func (g *Generator) buildTargets(bp *Blueprint, vars TemplateVars) []renderTarget {
	n := bp.Name
	ifacePkg := filepath.Join(g.modulesRoot, n)
	implPkg := filepath.Join(g.modulesRoot, n+"Impl")
	ifaceSrc := filepath.Join(ifacePkg, "Sources", n)
	implSrc := filepath.Join(implPkg, "Sources", n+"Impl")

	var targets []renderTarget

	// Always generated: module skeleton
	targets = append(targets,
		renderTarget{
			templateName: "bp_namespace.swift.tmpl",
			outputPath:   filepath.Join(ifaceSrc, n+".swift"),
			vars:         vars,
		},
		renderTarget{
			templateName: "bp_module.swift.tmpl",
			outputPath:   filepath.Join(ifaceSrc, "Module", n+".Module.swift"),
			vars:         vars,
		},
		renderTarget{
			templateName: "bp_interface.swift.tmpl",
			outputPath:   filepath.Join(ifaceSrc, "Module", n+".Module+Interface.swift"),
			vars:         vars,
		},
		renderTarget{
			templateName: "bp_impl.swift.tmpl",
			outputPath:   filepath.Join(implSrc, "Module", n+".Module+Impl.swift"),
			vars:         vars,
		},
	)

	// Always generated: business layer
	targets = append(targets,
		renderTarget{
			templateName: "bp_action.swift.tmpl",
			outputPath:   filepath.Join(ifaceSrc, "Business", n+".Business+Action.swift"),
			vars:         vars,
		},
		renderTarget{
			templateName: "bp_effect.swift.tmpl",
			outputPath:   filepath.Join(ifaceSrc, "Business", "Middleware", n+".Business+Effect.swift"),
			vars:         vars,
		},
		renderTarget{
			templateName: "bp_error.swift.tmpl",
			outputPath:   filepath.Join(ifaceSrc, "Business", n+".Business+Error.swift"),
			vars:         vars,
		},
		renderTarget{
			templateName: "bp_business_model.swift.tmpl",
			outputPath:   filepath.Join(ifaceSrc, "Business", "Models", n+".Business+Model+Scaffolded.swift"),
			vars:         vars,
		},
		renderTarget{
			templateName: "bp_state.swift.tmpl",
			outputPath:   filepath.Join(implSrc, "Business", n+".Business+State.swift"),
			vars:         vars,
		},
		renderTarget{
			templateName: "bp_reducer.swift.tmpl",
			outputPath:   filepath.Join(implSrc, "Business", n+".Business+Reducer.swift"),
			vars:         vars,
		},
		renderTarget{
			templateName: "bp_flow.swift.tmpl",
			outputPath:   filepath.Join(implSrc, "Business", "Middleware", n+".Business+Flow.swift"),
			vars:         vars,
		},
		renderTarget{
			templateName: "bp_service.swift.tmpl",
			outputPath:   filepath.Join(implSrc, "Business", "Middleware", n+".Business+Service.swift"),
			vars:         vars,
		},
	)

	// Conditional: data.http
	if bp.HasHTTP() {
		targets = append(targets,
			renderTarget{
				templateName: "bp_fetcher.swift.tmpl",
				outputPath:   filepath.Join(ifaceSrc, "Data", "Api", "Http", n+".Data+Api+Fetcher.swift"),
				vars:         vars,
			},
			renderTarget{
				templateName: "bp_fetcher_config.swift.tmpl",
				outputPath:   filepath.Join(ifaceSrc, "Data", "Api", "Http", n+".Data+Api+Fetcher+Config.swift"),
				vars:         vars,
			},
			renderTarget{
				templateName: "bp_dto_model.swift.tmpl",
				outputPath:   filepath.Join(ifaceSrc, "Data", "Api", "DTO", n+".Data+Api+DTO+ScaffoldedResponse.swift"),
				vars:         vars,
			},
		)
	}

	// Conditional: UI features
	if bp.HasFeatures() {
		// Once-per-module UI files
		targets = append(targets,
			renderTarget{
				templateName: "bp_viewstate.swift.tmpl",
				outputPath:   filepath.Join(implSrc, "UI", n+".UI+ViewState.swift"),
				vars:         vars,
			},
			renderTarget{
				templateName: "bp_router.swift.tmpl",
				outputPath:   filepath.Join(implSrc, "UI", n+".UI+Router.swift"),
				vars:         vars,
			},
			renderTarget{
				templateName: "bp_page_enum.swift.tmpl",
				outputPath:   filepath.Join(ifaceSrc, "UI", "Model", n+".UI+Model+Page.swift"),
				vars:         vars,
			},
		)

		// Per-feature UI files
		for _, feature := range bp.Features() {
			featureVars := vars.WithFeature(feature)
			targets = append(targets,
				renderTarget{
					templateName: "bp_container.swift.tmpl",
					outputPath:   filepath.Join(implSrc, "UI", feature.Name, n+".UI+"+feature.Name+"+Container.swift"),
					vars:         featureVars,
				},
				renderTarget{
					templateName: "bp_container_localstate.swift.tmpl",
					outputPath:   filepath.Join(implSrc, "UI", feature.Name, n+".UI+"+feature.Name+"+Container+LocalState.swift"),
					vars:         featureVars,
				},
				renderTarget{
					templateName: "bp_page.swift.tmpl",
					outputPath:   filepath.Join(implSrc, "UI", feature.Name, "Page", n+".UI+"+feature.Name+"+Page.swift"),
					vars:         featureVars,
				},
				renderTarget{
					templateName: "bp_page_props.swift.tmpl",
					outputPath:   filepath.Join(implSrc, "UI", feature.Name, "Page", n+".UI+"+feature.Name+"+Page+Props.swift"),
					vars:         featureVars,
				},
			)
		}
	}

	// Conditional: UI components
	if bp.HasComponents() {
		for _, component := range bp.Components() {
			compVars := vars.WithComponent(component)
			targets = append(targets,
				renderTarget{
					templateName: "bp_component.swift.tmpl",
					outputPath:   filepath.Join(ifaceSrc, "UI", "Components", n+".UI+"+component.Name+".swift"),
					vars:         compVars,
				},
			)
		}
	}

	return targets
}

func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory for %q: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}
