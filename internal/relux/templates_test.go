package relux

import (
	"testing"

	"github.com/relux-works/ios-app-manager/internal/testutil"
)

func TestTemplates(t *testing.T) {
	engine, err := NewTemplateEngine()
	if err != nil {
		t.Fatalf("failed to create template engine: %v", err)
	}

	tests := []struct {
		name         string
		templateName string
		goldenName   string
		vars         TemplateVariables
	}{
		{
			name:         "namespace",
			templateName: "namespace.swift.tmpl",
			goldenName:   "relux/namespace",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
		{
			name:         "module",
			templateName: "module.swift.tmpl",
			goldenName:   "relux/module",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
		{
			name:         "interface",
			templateName: "interface.swift.tmpl",
			goldenName:   "relux/interface",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
		{
			name:         "impl",
			templateName: "impl.swift.tmpl",
			goldenName:   "relux/impl",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
		{
			name:         "relux_namespace",
			templateName: "relux_namespace.swift.tmpl",
			goldenName:   "relux/relux_namespace",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
		{
			name:         "relux_interface",
			templateName: "relux_interface.swift.tmpl",
			goldenName:   "relux/relux_interface",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
		{
			name:         "relux_action",
			templateName: "relux_action.swift.tmpl",
			goldenName:   "relux/relux_action",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
		{
			name:         "relux_effect",
			templateName: "relux_effect.swift.tmpl",
			goldenName:   "relux/relux_effect",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
		{
			name:         "relux_impl",
			templateName: "relux_impl.swift.tmpl",
			goldenName:   "relux/relux_impl",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
		{
			name:         "relux_state",
			templateName: "relux_state.swift.tmpl",
			goldenName:   "relux/relux_state",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
		{
			name:         "relux_flow",
			templateName: "relux_flow.swift.tmpl",
			goldenName:   "relux/relux_flow",
			vars: TemplateVariables{
				ModuleName:           "Auth",
				ModuleNameLower:      "auth",
				InterfacePackageName: "Auth",
				ImplPackageName:      "AuthImpl",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rendered, err := engine.Render(tc.templateName, tc.vars)
			if err != nil {
				t.Fatalf("failed to render template %q: %v", tc.templateName, err)
			}

			testutil.AssertGoldenFile(t, tc.goldenName, string(rendered))
		})
	}
}
