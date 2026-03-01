# Module Registry — Final Design (Post-Audit)

## Package: `internal/registry/`

### ModuleID Constants

```go
package registry

type ModuleID string

const (
    IoC           ModuleID = "ioc"
    Relux         ModuleID = "relux"
    SecureStore   ModuleID = "secure-store"
    TokenProvider ModuleID = "token-provider"
    HttpClient    ModuleID = "http-client"
    AppConfig     ModuleID = "app-config"
    Utilities     ModuleID = "utilities"
    DefaultsStore ModuleID = "defaults-store"
)
```

### SetupInput (unified)

```go
// SetupInput is the unified input for all module Setup() functions.
type SetupInput struct {
    ProjectRoot string            // required — absolute path to iOS project root
    AppName     string            // required — from config (e.g. "XFlow")
    ModulesPath string            // optional — defaults to "Packages"
    ExtraArgs   map[string]string // module-specific params
}
```

- `Platform` dropped (dead field, always defaults to iOS v17)
- `AccessGroup` for SecureStore → `ExtraArgs["access-group"]`
- Future module-specific flags → ExtraArgs (scales without struct changes)

### Module struct

```go
type Module struct {
    ID           ModuleID
    Name         string            // human-readable: "SecureStore"
    Description  string            // one-liner: "Keychain wrapper with interface/impl split"
    Category     Category          // infra, foundation, network, utils
    Dependencies []ModuleID        // must be set up before this module

    // Two-phase setup
    Plan         func(SetupInput) (string, error) // returns plan text (what will be created/patched)
    Setup        func(SetupInput) error           // actual scaffolding
    UsageGuide   string                           // printed after setup

    // Metadata
    CLIUse       string            // cobra Use field: "secure-store"
    CLIShort     string            // cobra Short: "Manage SecureStore module"
    SetupShort   string            // setup subcommand Short: "Create SecureStore kit module"
    ExtraFlags   []ExtraFlag       // module-specific CLI flags
}

type ExtraFlag struct {
    Name     string // "access-group"
    Usage    string // "app group for shared keychain access"
    Required bool
    ArgKey   string // maps to ExtraArgs key
}

type Category string

const (
    Infra      Category = "infra"
    Foundation Category = "foundation"
    Network    Category = "network"
    Utils      Category = "utils"
)
```

### Global Registry

```go
var modules = make(map[ModuleID]*Module)

func Register(m *Module) {
    if _, exists := modules[m.ID]; exists {
        panic(fmt.Sprintf("module %s already registered", m.ID))
    }
    modules[m.ID] = m
}

func Get(id ModuleID) *Module {
    return modules[id]
}

func All() map[ModuleID]*Module {
    return modules
}

// AllSorted returns modules in dependency order (topological sort).
func AllSorted() []*Module { ... }

// CheckDependencies verifies all dependencies of module are already set up
// by checking Registry.swift for their registration markers.
func CheckDependencies(id ModuleID, registryPath string) error { ... }
```

### IoC Special Handling

IoC is the only outlier — it generates Registry.swift that others register into.

**Decision**: IoC registers as a normal Module with `Category: Infra`. No special `IsRegistryGenerator` flag. Instead:
- `Dependencies: []` (no deps, always first)
- Its shared library functions (`DiscoverModules`, `ScaffoldRegistryWithData`, etc.) remain as package-level exports in `internal/ioc/`
- Other modules import `internal/ioc/` directly for registry patching (as they already do)
- The `internal/registry/` package does NOT import `internal/ioc/` — no circular dependency

### Registry Patching: Standardize on Anchor Pattern

**Current state**: 2 patterns (full regen vs anchor patch).

**Decision**: Keep both, formalize them:
- `PatchMode: "regenerate"` — full Registry.swift rebuild via `ioc.ScaffoldRegistryWithData()`. Used by modules that create new Packages/ (securestore, tokenprovider, utilities) because they need `.module-type` markers discovered.
- `PatchMode: "anchor"` — targeted insert at MARK anchors. Used by modules that only add to existing sections (httpclient, appconfig).

No forced migration — each module keeps its working pattern. Both are valid.

### Dependency Graph (from audit)

```
ioc (infra, no deps)
 ├── relux (infra, deps: ioc)
 ├── httpclient (network, deps: ioc)
 ├── securestore (foundation, no hard dep)
 │   └── appconfig (foundation, deps: ioc, securestore)
 ├── tokenprovider (foundation, no hard dep)
 └── utilities (utils, no hard dep)
```

SecureStore/TokenProvider/Utilities technically work without IoC (they create packages, not patch registry). But if IoC exists, they regenerate Registry to include themselves. Decision: list IoC as optional dep (soft), not blocking.

### Generic CLI Command Generation

```go
// In internal/cli/setup_command.go — generates cobra commands from registry

func NewSetupCommand(mod *registry.Module, opts *RootOptions) *cobra.Command {
    cmd := &cobra.Command{
        Use:   mod.CLIUse,
        Short: mod.CLIShort,
    }

    setupCmd := &cobra.Command{
        Use:   "setup",
        Short: mod.SetupShort,
        RunE: func(cmd *cobra.Command, args []string) error {
            input := buildSetupInput(opts, cmd)

            // Phase 1: Plan
            plan, err := mod.Plan(input)
            if err != nil { return err }
            fmt.Fprintln(cmd.OutOrStdout(), plan)
            fmt.Fprintln(cmd.OutOrStdout(), mod.UsageGuide)

            // Phase 2: Confirm (unless --yes)
            if !yes {
                if !confirm("Proceed?") { return nil }
            }

            // Phase 3: Check deps
            if err := registry.CheckDependencies(mod.ID, registryPath); err != nil {
                return err
            }

            // Phase 4: Setup
            if err := mod.Setup(input); err != nil { return err }

            fmt.Fprintf(cmd.OutOrStdout(), "%s setup complete\n", mod.Name)
            return nil
        },
    }

    // Add --yes and --dry-run flags
    setupCmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
    setupCmd.Flags().BoolVar(&dryRun, "dry-run", false, "print plan only")

    // Add module-specific flags from ExtraFlags
    for _, f := range mod.ExtraFlags {
        setupCmd.Flags().String(f.Name, "", f.Usage)
        if f.Required {
            setupCmd.MarkFlagRequired(f.Name)
        }
    }

    cmd.AddCommand(setupCmd)
    return cmd
}
```

### root.go becomes

```go
func NewRootCommandWithAppManager(...) *cobra.Command {
    // ... existing code ...

    // Auto-register all setup modules
    for _, mod := range registry.All() {
        cmd.AddCommand(NewSetupCommand(mod, opts))
    }

    // Non-setup commands remain manual
    cmd.AddCommand(
        newInitCommand(opts),      // init is special (not a "module")
        newModuleCommand(opts),
        newDependencyCommand(opts),
        // ...
    )
}
```

### Module Registration Example (SecureStore)

```go
// internal/securestore/register.go
package securestore

import "github.com/relux-works/ios-app-manager/internal/registry"

func init() {
    registry.Register(&registry.Module{
        ID:           registry.SecureStore,
        Name:         "SecureStore",
        Description:  "Keychain wrapper with interface/impl split",
        Category:     registry.Foundation,
        Dependencies: []registry.ModuleID{}, // works without IoC, regenerates Registry if present

        Plan:  Plan,
        Setup: SetupFromRegistry,

        UsageGuide: `## Usage
  let store = Registry.resolve(SecureStoring.self)
  store.set("token", forKey: "accessToken")
  store.get(forKey: "accessToken")

  TODO: expand with full usage instructions`,

        CLIUse:     "secure-store",
        CLIShort:   "Manage SecureStore module",
        SetupShort: "Create SecureStore kit module with Keychain wrapper",
        ExtraFlags: []registry.ExtraFlag{
            {Name: "access-group", Usage: "app group for shared keychain access", Required: true, ArgKey: "access-group"},
        },
    })
}

// SetupFromRegistry adapts the existing Setup() to work with registry.SetupInput.
func SetupFromRegistry(input registry.SetupInput) error {
    return Setup(SetupInput{
        ProjectRoot: input.ProjectRoot,
        AppName:     input.AppName,
        ModulesPath: input.ModulesPath,
        AccessGroup: input.ExtraArgs["access-group"],
    })
}

// Plan returns what will be created/patched.
func Plan(input registry.SetupInput) (string, error) {
    // TODO: implement
    return "TODO: SecureStore plan", nil
}
```

### Migration Strategy

1. Create `internal/registry/` with types + Register/Get/All
2. Add `register.go` to each module package with `init()` + Register call
3. Existing `setup.go` files UNCHANGED — adapter functions bridge old SetupInput → new
4. CLI: create generic `NewSetupCommand()`, replace per-module CLI files
5. Add `--yes` / `--dry-run` flags
6. Add stub Plan() and UsageGuide to each module
7. Test: all existing tests pass, demo pipeline works

### What's NOT changing

- `internal/ioc/` shared functions (DiscoverModules, etc.) — stay as-is
- Template files — stay embedded in their packages
- Module creation (`module create`) — separate from setup, not in registry
- `init` command — not a module, stays manual
- Existing SetupInput structs in each package — kept, adapter bridges to registry.SetupInput
