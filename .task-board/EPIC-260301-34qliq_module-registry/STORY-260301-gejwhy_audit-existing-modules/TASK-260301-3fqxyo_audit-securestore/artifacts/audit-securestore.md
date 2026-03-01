# Module Audit: SecureStore

## Overview

The `securestore` package scaffolds a Keychain wrapper module with interface/implementation split. It creates `SecureStore` (interface) and `SecureStoreImpl` (implementation) packages that provide type-safe, access-group-scoped Keychain operations.

**CLI command**: `secure-store setup --access-group <group>`

---

## 1. Setup() Flow (step by step)

1. **Validate input** — requires `ProjectRoot`, `AppName`, `AccessGroup` (all non-empty)
2. **Resolve platform** — defaults to `iOS(.v17)` if not specified
3. **Resolve modules path** — via `ioc.ResolveModulesPath()`, defaults to `Packages/`
4. **Create interface package dir** (`Packages/SecureStore/`)
   - Creates dir + `Package.swift` via `tuistproj.GeneratePackageSwift()` (type=Interface)
   - Creates `Sources/SecureStore/` subdir
   - Idempotent: skips if dir exists
5. **Write `.module-type` marker** — value `kit` (drives IoC registry category grouping -> foundation section)
6. **Write `.builder-config` marker** — content: `serviceName: Configuration.Keychain.serviceName, accessGroup: Configuration.AppGroups.<ACCESS_GROUP_KEY>`
   - `<ACCESS_GROUP_KEY>` = access group string uppercased with dots replaced by underscores (e.g. `group.org.xflow.app` -> `GROUP_ORG_XFLOW_APP`)
   - This content is injected verbatim into `Impl()` constructor call in Registry.swift
7. **Create impl package dir** (`Packages/SecureStoreImpl/`)
   - Same as step 4 but with type=Impl
   - Creates `Sources/SecureStoreImpl/` subdir
8. **Scaffold interface Swift files** (3 files from templates):
   - `SecureStore.swift` — namespace enum
   - `Module/SecureStore.Module.swift` — module definition
   - `Module/SecureStore.Module+Interface.swift` — protocol + typealias
9. **Scaffold impl Swift files** (1 file from template):
   - `Module/SecureStore.Module+Impl.swift` — actor implementation with Keychain ops
10. **Patch Project.swift** — adds `.external(name: "SecureStore")` and `.external(name: "SecureStoreImpl")` to dependencies
11. **Patch root Package.swift** — adds `.package(path: "Packages/SecureStore")` and `.package(path: "Packages/SecureStoreImpl")`
12. **Patch Workspace.swift** — same as Package.swift (skips if file doesn't exist)
13. **Re-scaffold Registry.swift** (conditional) — if Registry.swift exists (IoC already set up):
    - Discovers all modules via `ioc.DiscoverModules()`
    - Checks if existing Registry has Relux imports
    - Regenerates Registry.swift with all discovered modules (including new SecureStore)

---

## 2. Files Created

| File | Path (relative to project root) |
|------|--------------------------------|
| Package.swift (interface) | `Packages/SecureStore/Package.swift` |
| Package.swift (impl) | `Packages/SecureStoreImpl/Package.swift` |
| Namespace | `Packages/SecureStore/Sources/SecureStore/SecureStore.swift` |
| Module definition | `Packages/SecureStore/Sources/SecureStore/Module/SecureStore.Module.swift` |
| Interface protocol | `Packages/SecureStore/Sources/SecureStore/Module/SecureStore.Module+Interface.swift` |
| Implementation | `Packages/SecureStoreImpl/Sources/SecureStoreImpl/Module/SecureStore.Module+Impl.swift` |
| Module type marker | `Packages/SecureStore/.module-type` (content: `kit`) |
| Builder config marker | `Packages/SecureStore/.builder-config` (content: constructor args) |

---

## 3. Files Patched

| File | How |
|------|-----|
| `Project.swift` | Adds `.external(name: "SecureStore")` and `.external(name: "SecureStoreImpl")` to dependencies array |
| `Package.swift` (root) | Adds `.package(path: "Packages/SecureStore")` and `.package(path: "Packages/SecureStoreImpl")` |
| `Workspace.swift` | Same as root Package.swift (skipped if file doesn't exist) |
| `Targets/<AppName>/Sources/App/<AppName>.Registry.swift` | **Regenerated entirely** if exists — discovers all modules, rebuilds imports + registrations |

---

## 4. SetupInput Fields

```go
type SetupInput struct {
    ProjectRoot string   // Required. Project root directory
    AppName     string   // Required. From config. Used for Registry.swift path
    ModulesPath string   // Optional. Defaults to "Packages"
    Platform    string   // Optional. Defaults to "iOS(.v17)"
    AccessGroup string   // Required. Must match a value from config app_groups
}
```

**CLI-level fields** (from config, not in SetupInput):
- `cfg.AppGroups []string` — used for `--access-group` validation (flag value must exist in this list)

---

## 5. Prerequisites

| Prerequisite | Required? | Why |
|-------------|-----------|-----|
| `ios-app-manager init` | **Yes** | Needs `Project.swift`, `Package.swift` to exist for patching |
| Config with `app_groups` | **Yes** | `--access-group` is validated against config's `app_groups` list |
| `ioc setup` | **No** (but recommended) | If Registry.swift exists, it gets regenerated to include SecureStore. If not, SecureStore is still created but won't be auto-registered in IoC |

**No hard dependency on any other module.** Imports only standard Apple frameworks (`Foundation`, `Security`) and the interface package (`SecureStore`).

---

## 6. Special Flags/Params

| Flag | Required | Description |
|------|----------|-------------|
| `--access-group` | **Yes** | App group identifier for shared Keychain access (e.g. `group.org.xflow.app`). Must exist in config `app_groups`. |

**Validation behavior (3 error paths):**
1. `--access-group` omitted + no `app_groups` in config -> error with guidance to add groups to config
2. `--access-group` omitted + `app_groups` present -> error listing available groups
3. `--access-group` value not in `app_groups` -> error with "not found" + available groups

The access group value flows into:
- `.builder-config` marker -> `Configuration.AppGroups.<KEY>` reference in Registry.swift constructor
- Does NOT appear directly in Swift template files — it's an IoC registry concern

---

## 7. Category

**`foundation`** — Module type is `kit`, which maps to `foundation` section in Registry.swift (per the category mapping: kit/shared -> foundation).

---

## 8. Registry Fit Assessment

### Fits cleanly: YES

The SecureStore module maps well to the Module struct:

```go
Module{
    ID:           "securestore",
    Name:         "SecureStore",
    Description:  "Keychain wrapper with interface/impl split for secure data storage",
    Dependencies: []ModuleID{},  // no module dependencies
    Category:     "foundation",
    Setup:        func(input SetupInput) error { ... },
    Plan:         func(input SetupInput) string { ... },
    UsageGuide:   "SecureStore setup complete",
}
```

**Special considerations:**

1. **Extra field: `AccessGroup`** — This is the only module (so far) that requires a mandatory flag beyond the common SetupInput fields. The registry's `SetupInput` must accommodate this. Options:
   - Add `AccessGroup string` to the common `SetupInput` (simple, but pollutes other modules)
   - Add `ExtraArgs map[string]string` to `SetupInput` for module-specific params (more generic)
   - Module reads it from config directly (breaks current CLI-level validation pattern)

2. **`.builder-config` marker** — This module writes a `.builder-config` file that influences how IoC Registry generates the constructor call. This is a pattern shared with other modules that need constructor params. No issue for registry fit — it's file-based, not code-based coupling.

3. **Idempotent** — Setup is fully idempotent (can run multiple times safely). Package creation skips if dir exists; manifest patches check for duplicates. Good fit for registry pattern where setup might be re-run.

4. **No external Swift dependencies** — Unlike relux-feature modules that add `swift-relux`, SecureStore uses only Apple's Security framework. No SPM dependency management needed.

---

## Key Takeaways

- **Clean, self-contained module** — no dependencies on other scaffolded modules
- **Hardcoded module name** (`SecureStore` / `SecureStoreImpl`) — not parameterized like generic module creation
- **Access group is the key differentiator** — the mandatory `--access-group` flag with config validation is unique to this module
- **Registry regeneration** is triggered as part of setup (step 13) — this is a side effect that the registry system should be aware of
- **Templates are simple** — no Go template variables, just static Swift code (templates render with `nil` data)
- **Actor-based impl** — the implementation uses Swift `actor` for thread safety, with `nonisolated` methods for synchronous Keychain API calls
