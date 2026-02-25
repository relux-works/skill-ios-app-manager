package relux

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/components"
	"github.com/relux-works/ios-app-manager/internal/modules"
)

// ReluxManager implements Relux workflows via file-template commands.
type ReluxManager struct {
	modulesRoot          string
	initCommand          *InitCommand
	addActionCommand     *AddActionCommand
	addMiddlewareCommand *AddMiddlewareCommand
}

var _ components.ReluxManager = (*ReluxManager)(nil)

// NewReluxManager creates a ReluxManager facade.
func NewReluxManager(modulesRoot string) (*ReluxManager, error) {
	engine, err := NewTemplateEngine()
	if err != nil {
		return nil, err
	}

	initCommand, err := NewInitCommand(engine)
	if err != nil {
		return nil, err
	}

	addMiddlewareCommand, err := NewAddMiddlewareCommand(engine)
	if err != nil {
		return nil, err
	}

	root := strings.TrimSpace(modulesRoot)
	if root == "" {
		root = "."
	}

	return &ReluxManager{
		modulesRoot:          root,
		initCommand:          initCommand,
		addActionCommand:     NewAddActionCommand(),
		addMiddlewareCommand: addMiddlewareCommand,
	}, nil
}

// InitModule creates initial Relux files for a module.
func (m *ReluxManager) InitModule(ctx context.Context, moduleName string, moduleType string) error {
	descriptor, err := modules.GetModuleType(moduleType)
	if err != nil {
		return err
	}
	if !descriptor.HasRelux {
		return nil
	}

	_, err = m.initCommand.Run(ctx, InitModuleInput{
		ModuleName:  moduleName,
		ModulePath:  m.modulePath(moduleName),
		TemplateSet: descriptor.TemplateSet,
	})
	return err
}

// AddAction adds action case and reducer stub for an existing module.
func (m *ReluxManager) AddAction(ctx context.Context, moduleName string, actionName string) error {
	_, err := m.addActionCommand.Run(ctx, AddActionInput{
		ModuleName: moduleName,
		ModulePath: m.modulePath(moduleName),
		ActionName: actionName,
	})
	return err
}

// AddActionWithParams adds action case and reducer stub with action parameters.
func (m *ReluxManager) AddActionWithParams(ctx context.Context, moduleName string, actionName string, params []ActionParameter) error {
	_, err := m.addActionCommand.Run(ctx, AddActionInput{
		ModuleName:   moduleName,
		ModulePath:   m.modulePath(moduleName),
		ActionName:   actionName,
		ActionParams: params,
	})
	return err
}

// AddReducer is reserved for future reducer generation workflow.
func (m *ReluxManager) AddReducer(ctx context.Context, moduleName string, reducerName string) error {
	return fmt.Errorf("add reducer is not implemented for module %q reducer %q", moduleName, reducerName)
}

// AddMiddleware generates an additional middleware file in module implementation package.
func (m *ReluxManager) AddMiddleware(ctx context.Context, moduleName string, middlewareName string) error {
	_, err := m.addMiddlewareCommand.Run(ctx, AddMiddlewareInput{
		ModuleName:     moduleName,
		ModulePath:     m.modulePath(moduleName),
		MiddlewareName: middlewareName,
	})
	return err
}

func (m *ReluxManager) modulePath(moduleName string) string {
	return filepath.Join(m.modulesRoot, moduleName)
}
