package relux

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// AddMiddlewareInput captures parameters for relux-add-middleware command.
type AddMiddlewareInput struct {
	ModuleName            string
	ModulePath            string
	MiddlewareName        string
	InterfacePackageName  string
	ImplPackageName       string
	ServiceProtocol       string
	ServiceImplementation string
}

// AddMiddlewareCommand generates a new middleware file from template.
type AddMiddlewareCommand struct {
	engine *TemplateEngine
}

// NewAddMiddlewareCommand creates relux-add-middleware command.
func NewAddMiddlewareCommand(engine *TemplateEngine) (*AddMiddlewareCommand, error) {
	if engine == nil {
		return nil, fmt.Errorf("template engine is required")
	}
	return &AddMiddlewareCommand{engine: engine}, nil
}

// Run executes relux-add-middleware and returns the generated file path.
func (c *AddMiddlewareCommand) Run(ctx context.Context, input AddMiddlewareInput) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	moduleName, err := normalizeModuleName(input.ModuleName)
	if err != nil {
		return "", err
	}

	layout, err := resolveModuleLayout(moduleName, input.ModulePath)
	if err != nil {
		return "", err
	}

	middlewareName, err := normalizeMiddlewareName(input.MiddlewareName)
	if err != nil {
		return "", err
	}

	middlewareStem := strings.TrimSuffix(middlewareName, "Middleware")
	if middlewareStem == "" {
		middlewareStem = middlewareName
	}

	fileStem := toSnakeCase(middlewareStem)
	fileName := fileStem + "_middleware.swift"
	if strings.HasSuffix(fileStem, "_middleware") {
		fileName = fileStem + ".swift"
	}
	targetPath := filepath.Join(layout.ImplSourcesDir, fileName)
	if isExistingFile(targetPath) {
		return "", fmt.Errorf("middleware file %q already exists", targetPath)
	}

	templateVars := TemplateVariables{
		ModuleName:           moduleName,
		ModuleNameLower:      lowerFirst(moduleName),
		InterfacePackageName: defaultValue(input.InterfacePackageName, moduleName),
		ImplPackageName:      defaultValue(input.ImplPackageName, moduleName+"Impl"),
	}

	rendered, err := c.engine.Render("middleware.swift.tmpl", templateVars)
	if err != nil {
		return "", fmt.Errorf("render middleware template: %w", err)
	}

	prefix := moduleName + middlewareStem
	content := string(rendered)
	content = strings.ReplaceAll(content, moduleName+"MiddlewareProtocol", prefix+"MiddlewareProtocol")
	content = strings.ReplaceAll(content, moduleName+"Middleware", prefix+"Middleware")

	if err := writeFile(targetPath, []byte(content)); err != nil {
		return "", err
	}

	return targetPath, nil
}
