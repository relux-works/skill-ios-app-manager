# Module Audit: relux

## Package
`internal/relux/` — setup path (`Setup()` function in `setup.go`)

## CLI Command
`ios-app-manager relux setup` — defined in `internal/cli/relux.go`

---

## 1. Setup() Flow (step by step)

1. **Validate input** — checks `ProjectRoot` and `AppName` are non-empty
2. **Derive paths** — computes `appTypeName` (PascalCase), `modulesRoot`, `sourcesDir`, `appDir`
3. **Check prerequisite** — verifies `Registry.swift` exists at `Targets/<AppName>/Sources/App/<AppTypeName>.Registry.swift`; fails with "run 'ioc setup' first" if missing
4. **Add swiftui-relux dependency** — calls `deps.AddExternalDep()` to add `swiftui-relux` (from: 8.0.1) to `Packages/Package.swift`; idempotent (skips if already present)
5. **Add SwiftUIRelux to Project.swift** — calls `tuistproj.ApplyManifestEditsToFile()` to add `.external(name: "SwiftUIRelux")` dependency; idempotent
6. **Discover modules** — calls `ioc.DiscoverModules()` to scan `modulesRoot` for existing module packages
7. **Regenerate Registry.swift** — calls `ioc.ScaffoldRegistryWithData()` with `HasRelux: true` flag, rebuilding the full Registry.swift with all discovered module imports
8. **Update App.swift for Relux** — adds `@_exported import Relux`, `import SwiftUIRelux`; replaces `WindowGroup { ... }` body with `Relux.Resolver(splash:content:resolver:)` pattern
9. **Scaffold Splash.swift** — renders `setup_templates/splash.swift.tmpl` → `Targets/<AppName>/Sources/App/<AppTypeName>.Splash.swift`
10. **Scaffold Content.swift** — renders `setup_templates/content.swift.tmpl` → `Targets/<AppName>/Sources/App/<AppTypeName>.Content.swift`
11. **Scaffold ReluxLogger.swift** — renders `setup_templates/relux_logger.swift.tmpl` → `Targets/<AppName>/Sources/App/<AppTypeName>.ReluxLogger.swift`
12. **Patch Package.swift for Tuist** — appends `#if TUIST / PackageSettings(productTypes: ["Relux": .framework])` block to `Packages/Package.swift`; idempotent

---

## 2. Files Created

| File | Path (relative to project root) | Source |
|------|------|--------|
| Splash.swift | `Targets/<AppName>/Sources/App/<AppTypeName>.Splash.swift` | `setup_templates/splash.swift.tmpl` |
| Content.swift | `Targets/<AppName>/Sources/App/<AppTypeName>.Content.swift` | `setup_templates/content.swift.tmpl` |
| ReluxLogger.swift | `Targets/<AppName>/Sources/App/<AppTypeName>.ReluxLogger.swift` | `setup_templates/relux_logger.swift.tmpl` |

Template data: `setupTemplateData{ AppTypeName string }` — only one field.

---

## 3. Files Patched

| File | Path | Modification |
|------|------|-------------|
| Package.swift (SPM) | `Packages/Package.swift` | Adds `swiftui-relux` as external dependency via `deps.AddExternalDep()` |
| Package.swift (SPM) | `Packages/Package.swift` | Appends `#if TUIST / PackageSettings(productTypes: ["Relux": .framework])` block |
| Project.swift (Tuist) | `Project.swift` (project root) | Adds `.external(name: "SwiftUIRelux")` dependency via manifest edit |
| Registry.swift | `Targets/<AppName>/Sources/App/<AppTypeName>.Registry.swift` | Full regeneration via `ioc.ScaffoldRegistryWithData()` with `HasRelux: true` |
| App.swift | `Targets/<AppName>/Sources/App.swift` | Adds `@_exported import Relux`, `import SwiftUIRelux`; replaces `WindowGroup` body with `Relux.Resolver(...)` |

---

## 4. SetupInput Fields

```go
type SetupInput struct {
    ProjectRoot string   // from config file directory (filepath.Dir of config path)
    AppName     string   // from config.AppName
    ModulesPath string   // from config.ModulesPath (normalized)
}
```

All three fields are derived from the config file (`ios-app-manager.json`). No CLI flags — `relux setup` takes zero arguments.

---

## 5. Prerequisites

| Prerequisite | How Checked | Error Message |
|-------------|-------------|---------------|
| **ioc setup** must have run first | `os.Stat()` on `Registry.swift` path | "Registry not found at <path> — run 'ioc setup' first" |
| **Config file** must exist | `config.LoadConfig()` in CLI command | "load config: ..." |
| **AppName** must be non-empty | `validateSetupInput()` | "app name is required" |
| **Packages/ dir** must exist | Implicit via `deps.AddExternalDep()` and module discovery | Various file I/O errors |

Hard dependency: **ioc setup** — the relux setup explicitly checks for Registry.swift and calls `ioc.ScaffoldRegistryWithData()` / `ioc.DiscoverModules()` / `ioc.EnsureImport()` from the ioc package.

---

## 6. Special Flags / Parameters

**None.** The `relux setup` command takes zero CLI flags. Everything comes from config.

Constants hardcoded in `setup.go`:
- `swiftUIReluxURL = "https://github.com/relux-works/swiftui-relux.git"`
- `swiftUIReluxVersion = "from: 8.0.1"`
- `swiftUIReluxPackage = "swiftui-relux"`
- `swiftUIReluxProduct = "SwiftUIRelux"`

---

## 7. Category

**infra** — Relux is core infrastructure. It sets up the state management framework that all feature modules depend on. It patches App.swift, Registry.swift, and Package.swift — all app-level infrastructure files.

---

## 8. Registry Module Struct Fit Assessment

### Fits cleanly? **Mostly — with one notable complexity.**

**What fits well:**
- `ID`: `"relux"`
- `Name`: `"Relux"`
- `Description`: `"SwiftUI Relux state management framework integration"`
- `Dependencies`: `[]ModuleID{"ioc"}` — clear, explicit prerequisite
- `Category`: `"infra"`
- `Setup(SetupInput) error`: maps directly to `relux.Setup(relux.SetupInput{...})`
- `Plan(SetupInput) string`: straightforward — list the files/patches
- `UsageGuide`: can describe the Relux.Resolver pattern, module init, etc.

**Complexity / adaptation needed:**

1. **SetupInput mapping**: The current `relux.SetupInput` has `ProjectRoot`, `AppName`, `ModulesPath` — all three are in the common `SetupInput` struct proposal. Clean fit.

2. **Heavy cross-package coupling**: Setup calls into `ioc.ScaffoldRegistryWithData()`, `ioc.DiscoverModules()`, `ioc.EnsureImport()`, `deps.AddExternalDep()`, `tuistproj.ApplyManifestEditsToFile()`. This is fine for a `Setup()` function, but the registry must ensure the `ioc` module is set up first (which the `Dependencies` field handles).

3. **Registry regeneration side effect**: Relux setup triggers a full Registry.swift regeneration (not just patching). This means running `relux setup` will re-discover all modules and rebuild Registry.swift. If another module was set up but its packages were removed, this could cause data loss. The registry system should be aware that some modules trigger full registry rebuilds.

4. **No special input fields needed**: Unlike `secure-store` (which needs `--access-group`), relux setup requires zero extra parameters. The common `SetupInput` is sufficient.

---

## Key Takeaways

- **Relux setup is a 12-step process** that touches 5 existing files and creates 3 new ones
- **Hard dependency on ioc** — must run `ioc setup` first; explicitly checked via Registry.swift existence
- **Zero CLI flags** — everything from config file
- **Idempotent** — all patch operations check for existing content before modifying
- **Registry.swift full regeneration** is the most notable side effect — it rebuilds the entire file, not a targeted patch
- **Clean fit for Module struct** — no special input fields needed, dependency on `ioc` maps directly to `Dependencies` field
- Template data is minimal: only `AppTypeName` (PascalCase of AppName)
