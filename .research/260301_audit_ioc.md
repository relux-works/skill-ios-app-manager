# Module Audit: ioc

## Package
`internal/ioc/` — SwiftIoC container integration setup

## Setup() Flow (step by step)

1. **Validate input** — checks `ProjectRoot` (required) and `AppName` (required). `ModulesPath` is optional (defaults to `"Packages"`).

2. **Resolve modules path** — `ResolveModulesPath(projectRoot, modulesPath)` → resolves relative to project root, defaults to `<ProjectRoot>/Packages`.

3. **Add SwiftIoC to Package.swift** — calls `deps.AddExternalDep()` to add the `swift-ioc` git dependency (`from: 1.0.1`) to `<ModulesPath>/Package.swift`. Idempotent (skips if already present).

4. **Add SwiftIoC to Project.swift** — calls `tuistproj.ApplyManifestEditsToFile()` to add `.external(name: "SwiftIoC")` to `<ProjectRoot>/Project.swift`. Idempotent.

5. **Discover modules** — scans `<ModulesPath>/` for Interface/Impl directory pairs (e.g. `Auth/` + `AuthImpl/`). For each pair:
   - Checks for Relux imports in Swift source → sets `IsAsync`
   - Reads `.module-type` marker file → sets `Category`
   - Reads `.builder-config` marker file → sets `BuilderArgs`
   - Hidden dirs and non-split modules are ignored.

6. **Scaffold Registry.swift** — renders `registry.swift.tmpl` with discovered modules into `<ProjectRoot>/Targets/<AppName>/Sources/App/<AppName>.Registry.swift`. **Always overwrites** (not a patch — full file generation).

7. **Update App.swift** — edits `<ProjectRoot>/Targets/<AppName>/Sources/App.swift`:
   - Adds `import SwiftIoC` (only SwiftIoC; module imports live in Registry.swift)
   - Injects `init() { Registry.configure() }` into the App struct body
   - Idempotent (skips if `Registry.configure()` already present)

## Files Created

| Path | Description |
|------|-------------|
| `Targets/<AppName>/Sources/App/<AppName>.Registry.swift` | IoC registry — module registrations, resolve helpers, builder functions |

## Files Patched

| Path | How |
|------|-----|
| `Package.swift` (in modules root) | Adds `swift-ioc.git` external dependency via `deps.AddExternalDep()` |
| `Project.swift` (project root) | Adds `.external(name: "SwiftIoC")` target dependency via `tuistproj.ApplyManifestEditsToFile()` |
| `Targets/<AppName>/Sources/App.swift` | Adds `import SwiftIoC` + `init() { Registry.configure() }` via regex-based text injection |

## SetupInput Fields

```go
type SetupInput struct {
    ProjectRoot string   // required — absolute path to project root
    AppName     string   // required — from config.AppName
    ModulesPath string   // optional — from config.ModulesPath, defaults to "Packages"
}
```

All fields come from config (`ios-app-manager.json`). No CLI flags beyond `--config`.

## Prerequisites

- **Project initialized** — `Package.swift`, `Project.swift`, and `App.swift` must exist (typically from `init` command)
- **No other modules required** — works with zero modules (generates empty Registry)
- **No dependency on relux setup** — but if relux modules exist, they are detected via `hasReluxImport()` and template conditionally adds Relux infrastructure (Store, RootSaga builders)

## Special Flags/Params

**None.** The CLI command (`ioc setup`) takes no flags — all input comes from config file.

## Category

**infra** — IoC is core dependency injection infrastructure.

## Registry Fit Assessment

### Special Nature

The `ioc` package is **fundamentally different** from other modules. It is not a typical "scaffold some Swift files" module — it is the **registry generator itself**. It:

1. **Owns** `Registry.swift` — the file that all other modules' registrations go into
2. **Discovers** other modules (scans Packages/ for Interface/Impl pairs)
3. **Renders** the registry template with grouped module data
4. **Patches** App.swift and Package.swift/Project.swift

### Fit: NEEDS SPECIAL HANDLING

The `ioc` module **does not fit the standard Module struct cleanly** because:

1. **It generates the registry** — other modules register INTO it. Circular: the registry module cannot register itself via the same mechanism.
2. **Its Setup() has a side effect on every other module** — it scans and re-renders ALL module registrations.
3. **Plan() would need module discovery** — to show what will be registered, it needs `DiscoverModules()`.
4. **Dependencies are inverted** — every other module depends on IoC, but IoC's setup depends on having modules to discover.

### Recommended Approach

- Register `ioc` as a Module with `Category: "infra"` and `Dependencies: []` (no prerequisites beyond init)
- Its `Setup()` can wrap the existing `ioc.Setup()` directly — `SetupInput` maps 1:1
- Add a special flag (e.g. `IsRegistryGenerator: true`) or handle by convention: always run `ioc setup` last / re-run after adding modules
- The current pattern of always overwriting Registry.swift is fine — it is a generated file, not hand-edited

### Exported Utilities Used by Other Modules

The `ioc` package exports functions that OTHER modules call during their setup:

- `WriteModuleType(moduleDir, moduleType)` — writes `.module-type` marker
- `WriteBuilderConfig(moduleDir, args)` — writes `.builder-config` marker
- `EnsureImport(content, moduleName)` — adds import to Swift file
- `DiscoverModules(modulesRoot)` — scans for Interface/Impl pairs
- `RenderRegistry()` / `RenderRegistryWithData()` — renders Registry.swift
- `ScaffoldRegistryWithData()` — writes Registry.swift to disk
- `BuildModuleImports()` — builds sorted import list
- `GroupModulesByCategory()` — groups modules by category
- `MapModuleTypeToCategory()` — maps module type string to category
- `ResolveModulesPath()` — resolves modules root path

This means `ioc` is really a **shared library** for registry operations, not just a standalone setup module.

## Key Constants

```go
swiftIoCURL     = "https://github.com/relux-works/swift-ioc.git"
swiftIoCVersion = "from: 1.0.1"
swiftIoCPackage = "SwiftIoC"
ModuleTypeFile  = ".module-type"
BuilderConfigFile = ".builder-config"
```

## Test Coverage

- **Unit tests** (`setup_test.go`): DiscoverModules, RenderRegistry, EditAppSwift, EnsureImport, BuildModuleImports, GroupModulesByCategory, MapModuleTypeToCategory, validation
- **Integration tests** (`ioc_test.go` in cli/): Full setup with modules, setup without modules, idempotency, help output
- Coverage is solid — all major code paths tested

## Key Takeaways

1. **IoC is the only infra module** — it sets up the DI container that everything else registers into
2. **Simplest SetupInput** — just ProjectRoot + AppName + ModulesPath, no special flags
3. **Registry.swift is fully regenerated** on every `ioc setup` — not patched, overwritten from template
4. **Dual role**: standalone setup command AND shared library for other modules (WriteModuleType, WriteBuilderConfig, etc.)
5. **Must run after other modules** to pick up their registrations — or be re-run when new modules are added
6. **Template uses semantic MARK sections** with anchors (infra, foundation, features, network, utils) for organized code
7. **Module type → category mapping**: kit/shared → foundation, feature/relux-feature/ui → features, utility → utils
