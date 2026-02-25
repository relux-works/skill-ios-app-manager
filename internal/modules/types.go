package modules

// ModuleType identifies a module flavor used by scaffolding.
type ModuleType string

const (
	// ModuleTypeFeature is a full Relux + SwiftUI module with interface/impl split.
	ModuleTypeFeature ModuleType = "feature"
	// ModuleTypeKit is a Relux logic module without UI, with interface/impl split.
	ModuleTypeKit ModuleType = "kit"
	// ModuleTypeShared is for shared state/services, with interface/impl split.
	ModuleTypeShared ModuleType = "shared"
	// ModuleTypeUI is a SwiftUI-only module with interface/impl split.
	ModuleTypeUI ModuleType = "ui"
	// ModuleTypeUtility is a single-package utility module without Relux or split.
	ModuleTypeUtility ModuleType = "utility"
)

// ModuleTypeDescriptor describes scaffolding behavior for one module type.
type ModuleTypeDescriptor struct {
	Type                  ModuleType
	HasInterfaceImplSplit bool
	HasRelux              bool
	HasUI                 bool
	TemplateSet           []string
	Description           string
}
