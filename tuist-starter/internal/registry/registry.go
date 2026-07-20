package registry

import (
	"fmt"
	"sort"
	"strings"
)

// ModuleID uniquely identifies a foundation module.
type ModuleID string

const (
	Init                ModuleID = "init"
	IoC                 ModuleID = "ioc"
	Relux               ModuleID = "relux"
	SecureStore         ModuleID = "secure-store"
	TokenProvider       ModuleID = "token-provider"
	FireAuthRelux       ModuleID = "fireauth-relux"
	HttpClient          ModuleID = "http-client"
	AppConfig           ModuleID = "app-config"
	AppExtensions       ModuleID = "app-extensions"
	WidgetBase          ModuleID = "widget-base"
	StaticWidget        ModuleID = "static-widget"
	LiveActivity        ModuleID = "live-activity"
	AppIntents          ModuleID = "app-intents"
	NotificationService ModuleID = "notification-service"
	TestTargets         ModuleID = "test-targets"
	Utilities           ModuleID = "utilities"
	FoundationPlus      ModuleID = "foundation-plus"
	SwiftUIPlus         ModuleID = "swiftui-plus"
	DefaultsStore       ModuleID = "defaults-store"
)

// Category groups modules for Registry.swift section placement.
type Category string

const (
	Infra      Category = "infra"
	Foundation Category = "foundation"
	Network    Category = "network"
	Utils      Category = "utils"
)

// SetupInput is the unified input for all module Setup() functions.
type SetupInput struct {
	ProjectRoot string            // required — absolute path to iOS project root
	AppName     string            // required — from config (e.g. "XFlow")
	ModulesPath string            // optional — defaults to "Packages"
	ConfigPath  string            // optional — selected ios-app-manager config path
	ExtraArgs   map[string]string // module-specific params
}

// ExtraFlag describes a module-specific CLI flag.
type ExtraFlag struct {
	Name     string // "access-group"
	Usage    string // "app group for shared keychain access"
	Required bool
	ArgKey   string // maps to ExtraArgs key
}

// ExternalDep describes an external Swift package dependency.
type ExternalDep struct {
	URL     string // git repo URL
	Version string // semver without "from:", e.g. "1.0.1"
	Product string // Swift product name for Project.swift .external(name:)
	Package string // SPM package name for Package.swift .package(name:), empty = use Product
}

// Capability describes a Tuist capability declared by a module.
type Capability struct {
	Type string            // keychainSharing, appGroups, pushNotifications, etc.
	Args map[string]string // optional params: e.g. {"group": "group.xxx"} for appGroups
}

// Module describes a foundation module that can be set up via the CLI.
type Module struct {
	ID           ModuleID
	Name         string   // human-readable: "SecureStore"
	Description  string   // one-liner: "Keychain wrapper with interface/impl split"
	Category     Category // infra, foundation, network, utils
	Dependencies []ModuleID
	ExternalDeps []ExternalDep
	// AdditionalFrameworkProducts are extra Swift package products that should be
	// forced to frameworks in root Package.swift PackageSettings.
	// Use this for runtime/transitive products that are not declared as direct
	// ExternalDeps but must not stay static under Tuist.
	AdditionalFrameworkProducts []string
	Capabilities                []Capability

	// Two-phase setup
	Plan       func(SetupInput) (string, error) // returns plan text
	Setup      func(SetupInput) error           // actual scaffolding
	UsageGuide string                           // printed after setup

	// CLI metadata
	CLIUse     string      // cobra Use field: "secure-store"
	CLIShort   string      // cobra Short: "Manage SecureStore module"
	SetupShort string      // setup subcommand Short
	ExtraFlags []ExtraFlag // module-specific CLI flags
}

var modules = make(map[ModuleID]*Module)

// Register adds a module to the global registry. Panics on duplicate ID.
func Register(m *Module) {
	if _, exists := modules[m.ID]; exists {
		panic(fmt.Sprintf("module %s already registered", m.ID))
	}
	modules[m.ID] = m
}

// Get returns the module with the given ID, or nil if not found.
func Get(id ModuleID) *Module {
	return modules[id]
}

// All returns all registered modules.
func All() map[ModuleID]*Module {
	return modules
}

// AllSorted returns modules in dependency order (topological sort).
// Modules with no dependencies come first.
func AllSorted() []*Module {
	result := make([]*Module, 0, len(modules))
	for _, m := range modules {
		result = append(result, m)
	}

	// Kahn's algorithm
	inDegree := make(map[ModuleID]int)
	for _, m := range result {
		if _, ok := inDegree[m.ID]; !ok {
			inDegree[m.ID] = 0
		}
		for _, dep := range m.Dependencies {
			inDegree[dep] += 0 // ensure dep exists in map
			inDegree[m.ID]++
		}
	}

	// Seed queue with zero-degree nodes, sorted by ID for determinism
	var queue []ModuleID
	for _, m := range result {
		if inDegree[m.ID] == 0 {
			queue = append(queue, m.ID)
		}
	}
	sort.Slice(queue, func(i, j int) bool { return queue[i] < queue[j] })

	sorted := make([]*Module, 0, len(result))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		sorted = append(sorted, modules[id])

		// Find modules that depend on this one and decrement
		for _, m := range result {
			for _, dep := range m.Dependencies {
				if dep == id {
					inDegree[m.ID]--
					if inDegree[m.ID] == 0 {
						queue = append(queue, m.ID)
						sort.Slice(queue, func(i, j int) bool { return queue[i] < queue[j] })
					}
				}
			}
		}
	}

	return sorted
}

// CheckDependencies verifies all dependencies of the given module are set up
// by checking registryContent (Registry.swift text) for their registration markers.
// A marker is the module name string (e.g. "SecureStore") appearing in the content.
func CheckDependencies(id ModuleID, registryContent string) error {
	m := Get(id)
	if m == nil {
		return fmt.Errorf("unknown module: %s", id)
	}

	var missing []string
	for _, depID := range m.Dependencies {
		dep := Get(depID)
		if dep == nil {
			missing = append(missing, string(depID))
			continue
		}
		// Skip modules without Setup — they are structural deps (e.g. Init)
		// and don't write to Registry.swift.
		if dep.Setup == nil {
			continue
		}
		// Skip infra/utils modules — they scaffold files but don't register
		// themselves in Registry.swift (no IoC registration).
		if dep.Category == Infra || dep.Category == Utils {
			continue
		}
		if !strings.Contains(registryContent, dep.Name) {
			missing = append(missing, dep.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing dependencies for %s: %s — run their setup first", m.Name, strings.Join(missing, ", "))
	}
	return nil
}

// HasRegistryDeps returns true if the module has at least one dependency
// that writes to Registry.swift (i.e. has a Setup function).
func HasRegistryDeps(id ModuleID) bool {
	m := Get(id)
	if m == nil {
		return false
	}
	for _, depID := range m.Dependencies {
		dep := Get(depID)
		if dep != nil && dep.Setup != nil && dep.Category != Infra && dep.Category != Utils {
			return true
		}
	}
	return false
}

// Reset clears the global registry. For testing only.
func Reset() {
	modules = make(map[ModuleID]*Module)
}
