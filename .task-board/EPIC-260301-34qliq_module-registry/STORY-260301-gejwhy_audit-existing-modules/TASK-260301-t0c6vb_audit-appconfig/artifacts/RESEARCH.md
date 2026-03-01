# Module Audit: appconfig

## Overview

The `appconfig` package scaffolds environment-switching configuration (prod/stage/dev) with an API configurator pattern. It generates 8 Swift files and patches `Registry.swift` with IoC registration.

**Package:** `internal/appconfig/`
**CLI command:** `app-config setup`
**Category:** foundation

---

## 1. Setup() Flow — Step by Step

1. **Validate input** — checks `ProjectRoot` and `AppName` are non-empty
2. **Compute Swift type name** — `scaffold.SwiftTypeName(appName)` (e.g. "XFlow" → "XFlow")
3. **Check Registry.swift exists** — at `Targets/<AppName>/Sources/App/<TypeName>.Registry.swift`. If missing → error: "run 'ioc setup' first"
4. **Check SecureStore is registered** — reads Registry.swift, looks for `SecureStore.Module.Interface.self` string. If missing → error: "run 'secure-store setup' first"
5. **Scaffold 8 Swift files** — into `Targets/<AppName>/Sources/AppConfig/`. Idempotent: skips files that already exist
6. **Patch Registry.swift** — inserts IoC registration line + builder function. Idempotent: skips if `IApiConfigManager.self` already present

---

## 2. Files Created

All under `Targets/<AppName>/Sources/AppConfig/`:

| File | Template | Purpose |
|------|----------|---------|
| `AppConfig.swift` | `namespace.swift.tmpl` | Root namespace enum with `Business` and `Data` sub-enums |
| `AppConfig.Env.swift` | `env.swift.tmpl` | `Env` enum: `.prod`, `.stage`, `.dev` |
| `AppConfig.Env+Configuration.swift` | `configuration.swift.tmpl` | `Configuration` struct with nested `Api` (baseURL, apiVersion) |
| `AppConfig.Env+Configuration+Presets.swift` | `presets.swift.tmpl` | Static presets for prod/stage/dev with placeholder URLs |
| `AppConfig.Manager+Protocols.swift` | `protocols.swift.tmpl` | `IApiConfigManager` protocol + global typealias |
| `AppConfig.Manager.swift` | `manager.swift.tmpl` | `Manager` class: thread-safe env state, SecureStore persistence |
| `AppConfig.ApiConfigurator.swift` | `api_configurator.swift.tmpl` | `ApiConfigurator` struct with closure-based config resolution |
| `AppConfig.UrlComponents.swift` | `url_components.swift.tmpl` | Static URL path components (auth, users, profile) |

**Note:** Templates are static (no Go template variables), copied verbatim. Not rendered via `text/template` — just `ReadFile` + `WriteFile`.

---

## 3. Files Patched

### Registry.swift (`Targets/<AppName>/Sources/App/<TypeName>.Registry.swift`)

Two insertions:

**a) Registration line** — inserted after `// MARK: - Foundation (scaffolding anchor: foundation)`:
```swift
ioc.register(IApiConfigManager.self, lifecycle: .container, resolver: Self.buildAppConfigManager)
```

**b) Builder function** — inserted before the closing brace of the `Foundation Builders` extension:
```swift
private static func buildAppConfigManager() -> IApiConfigManager {
    AppConfig.Business.Manager(secureStore: resolve(SecureStoring.self))
}
```

**Anchor dependencies:**
- `// MARK: - Foundation (scaffolding anchor: foundation)` — for registration
- `// MARK: - Foundation Builders (scaffolding anchor: foundation-builders)` — for builder function
- Both anchors must exist (created by `ioc setup`)

---

## 4. SetupInput Fields

```go
type SetupInput struct {
    ProjectRoot string   // Root directory of the iOS project
    AppName     string   // App name from config (e.g. "XFlow")
}
```

**Populated from:** `config.LoadConfig()` → `cfg.AppName`. `ProjectRoot` derived from `filepath.Dir(configPath)`.

**No additional flags/params.** CLI takes zero flags — everything comes from the config file.

---

## 5. Prerequisites

| Prerequisite | Check Method | Error Message |
|---|---|---|
| **ioc setup** | `os.Stat()` on Registry.swift path | "Registry.swift not found — run 'ioc setup' first" |
| **secure-store setup** | `strings.Contains(registry, "SecureStore.Module.Interface.self")` | "SecureStore not found in Registry.swift — run 'secure-store setup' first" |

**Runtime dependency chain:** `ioc setup` → `secure-store setup` → `app-config setup`

The Manager depends on `SecureStoring` protocol (from SecureStore module) for persisting the selected environment. The builder resolves `SecureStoring.self` from IoC.

---

## 6. Special Flags/Params

**None.** The `app-config setup` command takes no flags. All input comes from the project config file (`ios-app-manager.json`).

---

## 7. Category

**Foundation** — the module registers in the `foundation` section of Registry.swift and its builder goes into `foundation-builders`.

---

## 8. Registry Module Struct Fit Assessment

### Verdict: Fits cleanly

**Mapping to Module struct:**

```go
Module{
    ID:           "appconfig",
    Name:         "AppConfig",
    Description:  "Environment switching and API configuration manager",
    Dependencies: []ModuleID{"ioc", "securestore"},
    Category:     "foundation",
    Setup:        appconfig.Setup,  // signature matches with adapter
    Plan:         ...,              // straightforward: 8 files + registry patch
    UsageGuide:   "Configure API presets in AppConfig.Env+Configuration+Presets.swift",
}
```

**Observations:**

1. **SetupInput is minimal** — only `ProjectRoot` and `AppName`, which are standard fields that fit into the common `SetupInput` struct with no extras
2. **No special flags** — simplest module to unify
3. **Templates are static** — no Go template rendering needed, just file copies. This is unique among the modules — most others use `text/template`. Not a problem for the registry, just a simplification
4. **Idempotency is clean** — both file scaffolding (skip existing) and registry patching (check for marker string) are properly idempotent
5. **Prerequisite checking is explicit** — checks for ioc (Registry.swift existence) and SecureStore (registration string in Registry.swift). Maps cleanly to `Dependencies` field
6. **No package management** — unlike some modules, app-config does NOT add Swift package dependencies or modify Package.swift. It only creates files in the app target and patches Registry.swift
7. **Files go into app target, not Packages/** — output is `Targets/<AppName>/Sources/AppConfig/`, not `Packages/`. This is consistent with it being an app-level concern, not a standalone module package

### Potential adaptation notes:

- The `patchRegistry()` uses brace-matching logic (`findMatchingBrace`) to locate insertion points. This pattern is shared with other modules and could be extracted to a common utility during registry unification
- The hardcoded anchor constants (`foundationAnchor`, `foundationBuildersAnchor`) and registration/builder strings are module-specific but follow the same pattern as other foundation modules

---

## Key Takeaways

- **Simplest module to integrate** — no flags, no package deps, minimal SetupInput, static templates
- **Clean dependency chain** — explicitly checks for ioc and SecureStore before proceeding
- **Foundation category** — sits alongside SecureStore in the registry
- **Static templates** — no template rendering, just file copies (unique among modules)
- **App-target output** — files go to `Targets/<AppName>/Sources/AppConfig/`, not `Packages/`
- **Thread-safe Manager** — uses NSLock for concurrent access, SecureStore for env persistence
- **Placeholder URLs** — presets use example.com domains, user must customize after scaffolding
