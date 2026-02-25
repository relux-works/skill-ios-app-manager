package modules

import (
	"fmt"
	"strings"
)

var moduleTypeOrder = []ModuleType{
	ModuleTypeFeature,
	ModuleTypeReluxFeature,
	ModuleTypeKit,
	ModuleTypeShared,
	ModuleTypeUI,
	ModuleTypeUtility,
}

var moduleTemplateSet = []string{"namespace", "module", "interface", "impl"}

var reluxFeatureTemplateSet = []string{
	"relux_namespace", "module", "relux_interface",
	"relux_action", "relux_effect",
	"relux_impl", "relux_state", "relux_flow",
}

var swiftReluxDep = ExternalDep{
	PackageName: "swift-relux",
	ProductName: "Relux",
	URL:         "https://github.com/relux-works/swift-relux.git",
	Version:     `from: "9.0.1"`,
}

var moduleTypeRegistry = map[ModuleType]ModuleTypeDescriptor{
	ModuleTypeFeature: {
		Type:                  ModuleTypeFeature,
		HasInterfaceImplSplit: true,
		HasRelux:              true,
		HasUI:                 true,
		TemplateSet:           moduleTemplateSet,
		Description:           "Full module with UI (interface + implementation split).",
	},
	ModuleTypeKit: {
		Type:                  ModuleTypeKit,
		HasInterfaceImplSplit: true,
		HasRelux:              true,
		HasUI:                 false,
		TemplateSet:           moduleTemplateSet,
		Description:           "Logic module without UI (interface + implementation split).",
	},
	ModuleTypeShared: {
		Type:                  ModuleTypeShared,
		HasInterfaceImplSplit: true,
		HasRelux:              true,
		HasUI:                 false,
		TemplateSet:           moduleTemplateSet,
		Description:           "Shared services module (interface + implementation split).",
	},
	ModuleTypeUI: {
		Type:                  ModuleTypeUI,
		HasInterfaceImplSplit: true,
		HasRelux:              true,
		HasUI:                 true,
		TemplateSet:           moduleTemplateSet,
		Description:           "SwiftUI component module (interface + implementation split).",
	},
	ModuleTypeReluxFeature: {
		Type:                  ModuleTypeReluxFeature,
		HasInterfaceImplSplit: true,
		HasRelux:              true,
		HasUI:                 false,
		TemplateSet:           reluxFeatureTemplateSet,
		ExternalDeps:          []ExternalDep{swiftReluxDep},
		Description:           "Relux feature module with state management (interface + implementation split).",
	},
	ModuleTypeUtility: {
		Type:                  ModuleTypeUtility,
		HasInterfaceImplSplit: false,
		HasRelux:              false,
		HasUI:                 false,
		TemplateSet:           []string{},
		Description:           "Utility-only module (single package, no templates).",
	},
}

// GetModuleType resolves a module type by name.
func GetModuleType(name string) (ModuleTypeDescriptor, error) {
	normalized := normalizeModuleType(name)
	if normalized == "" {
		return ModuleTypeDescriptor{}, fmt.Errorf("module type is required")
	}

	descriptor, ok := moduleTypeRegistry[ModuleType(normalized)]
	if !ok {
		return ModuleTypeDescriptor{}, fmt.Errorf(
			"unknown module type %q (supported: %s)",
			normalized,
			strings.Join(supportedModuleTypeStrings(), ", "),
		)
	}

	return cloneDescriptor(descriptor), nil
}

// AllModuleTypes lists all known module type descriptors in stable order.
func AllModuleTypes() []ModuleTypeDescriptor {
	descriptors := make([]ModuleTypeDescriptor, 0, len(moduleTypeOrder))
	for _, moduleType := range moduleTypeOrder {
		descriptor := moduleTypeRegistry[moduleType]
		descriptors = append(descriptors, cloneDescriptor(descriptor))
	}
	return descriptors
}

// ValidateModuleType validates that a module type name is supported.
func ValidateModuleType(name string) error {
	_, err := GetModuleType(name)
	return err
}

func normalizeModuleType(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func supportedModuleTypeStrings() []string {
	out := make([]string, 0, len(moduleTypeOrder))
	for _, moduleType := range moduleTypeOrder {
		out = append(out, string(moduleType))
	}
	return out
}

func cloneDescriptor(descriptor ModuleTypeDescriptor) ModuleTypeDescriptor {
	clone := descriptor
	if descriptor.TemplateSet != nil {
		clone.TemplateSet = make([]string, len(descriptor.TemplateSet))
		copy(clone.TemplateSet, descriptor.TemplateSet)
	}
	if descriptor.ExternalDeps != nil {
		clone.ExternalDeps = make([]ExternalDep, len(descriptor.ExternalDeps))
		copy(clone.ExternalDeps, descriptor.ExternalDeps)
	}
	return clone
}
