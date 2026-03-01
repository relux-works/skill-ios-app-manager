# Module Audit: utilities

## Package
`internal/utilities/`

## CLI Command
`ios-app-manager utilities setup` — defined in `internal/cli/utilities.go`

---

## 1. Setup() Flow (step by step)

1. **Validate input** — requires non-empty `ProjectRoot` and `AppName`
2. **Resolve platform** — defaults to `iOS(.v17)` if not provided
3. **Resolve modules path** — via `ioc.ResolveModulesPath()`, defaults to `Packages`
4. **Create package directory** — `<ModulesPath>/Utilities/` with a `Package.swift` (type: `interface`, i.e. single package, no impl split). Idempotent: skips if dir already exists
5. **Write `.module-type` marker** — writes `utility` to `<ModulesPath>/Utilities/.module-type` for IoC registry category grouping
6. **Scaffold HttpClientUtils Swift files** — renders 3 templates into `Sources/Utilities/HttpClientUtils/`:
   - `HeaderMaps.swift` — static HTTP header dictionaries (JSON, form-urlencoded, auth bearer)
   - `BaseEncoder.swift` — `JSONEncoder.base` extension (snake_case keys, ISO8601 dates)
   - `BaseDecoder.swift` — `JSONDecoder.base` extension (snake_case keys, ISO8601 dates)
7. **Patch Project.swift** — adds `.external(name: "Utilities")` to the app target's dependencies
8. **Patch root Package.swift** — adds `.package(path: "Packages/Utilities")` to root dependencies
9. **Patch Workspace.swift** — adds `.package(path: "Packages/Utilities")` (skips if file doesn't exist)
10. **Re-scaffold Registry.swift** — if IoC is set up (Registry.swift exists), discovers all modules and regenerates the registry, preserving Relux import status

---

## 2. Files Created

| Path (relative to project root) | Description |
|---|---|
| `<ModulesPath>/Utilities/Package.swift` | SPM manifest (interface type, single target) |
| `<ModulesPath>/Utilities/.module-type` | Contains `utility` — for registry grouping |
| `<ModulesPath>/Utilities/Sources/Utilities/HttpClientUtils/HeaderMaps.swift` | HTTP header maps |
| `<ModulesPath>/Utilities/Sources/Utilities/HttpClientUtils/BaseEncoder.swift` | JSONEncoder extension |
| `<ModulesPath>/Utilities/Sources/Utilities/HttpClientUtils/BaseDecoder.swift` | JSONDecoder extension |

---

## 3. Files Patched

| File | How |
|---|---|
| `Project.swift` | Adds `.external(name: "Utilities")` to target dependencies |
| `Package.swift` (root) | Adds `.package(path: "<ModulesPath>/Utilities")` |
| `Workspace.swift` | Adds `.package(path: "<ModulesPath>/Utilities")` (only if file exists) |
| `Targets/<AppName>/Sources/App/<AppTypeName>.Registry.swift` | Fully re-scaffolded via `ioc.ScaffoldRegistryWithData()` if exists |

All patches are idempotent (skip if entry already present).

---

## 4. SetupInput Fields

```go
type SetupInput struct {
    ProjectRoot string   // Required. Resolved from config file location in CLI
    AppName     string   // Required. From config `app_name`
    ModulesPath string   // Optional. From config `modules_path`, defaults to "Packages"
    Platform    string   // Optional. SwiftPM platform, defaults to "iOS(.v17)". NOT exposed via CLI flag — unused by CLI command
}
```

Note: `Platform` field exists in `SetupInput` but the CLI command **never sets it** — always uses the default `iOS(.v17)`.

---

## 5. Prerequisites

- **Project.swift** must exist (will error if missing when patching)
- **Root Package.swift** must exist (same)
- **Workspace.swift** — optional (gracefully skipped if absent)
- **IoC setup** — optional but recommended. If `Registry.swift` exists, it's regenerated to include Utilities. If not, no registry integration happens
- **No other module dependencies** — Utilities is a leaf module with no imports of other project modules. Templates only import `Foundation`

---

## 6. Special Flags/Params

**None.** The CLI command has zero flags — everything comes from the config file:
- `app_name` -> `AppName`
- `modules_path` -> `ModulesPath`

No `--access-group`, no `--platform`, no extra params.

---

## 7. Category

**`utils`** — module type written to `.module-type` is `utility`, which maps to the `utils` registry section.

---

## 8. Registry Fit Assessment

**Fits cleanly.**

| Module struct field | Value | Notes |
|---|---|---|
| `ID` | `"utilities"` | Straightforward |
| `Name` | `"Utilities"` | Hardcoded constant |
| `Description` | `"HTTP client utilities (headers, encoder, decoder)"` | |
| `Dependencies` | `[]` (empty) | No prerequisites — truly independent |
| `Category` | `"utils"` | |
| `Setup` | `func(SetupInput) error` | Signature matches common struct |
| `Plan` | Trivial | Lists 3 files to create + manifest patches |
| `UsageGuide` | Short | "Import Utilities to access HeaderMaps, JSONEncoder.base, JSONDecoder.base" |

**No special needs.** This is the simplest module in the registry:
- No external Swift package dependencies (just Foundation)
- No special CLI flags
- No interface/impl split (utility type = single package)
- No dependencies on other modules
- Idempotent — safe to run multiple times
- Templates take zero data (no template variables, just static content)

The only minor note: the `Platform` field in `SetupInput` is unused by CLI but present in the struct — could be dropped or unified into a common `SetupInput`.

---

## Key Takeaways

1. **Simplest module** — no deps, no flags, no external packages, no interface/impl split
2. **Perfect registry candidate** — clean fit, no adaptation needed
3. **Templates are static** — no Go template variables at all, just `embed.FS` passthrough
4. **Idempotent** — package dir creation and manifest patching both handle "already exists" gracefully
5. **Registry integration built-in** — already calls `ioc.ScaffoldRegistryWithData()` to regenerate Registry.swift
6. **Dead field** — `Platform` in `SetupInput` is never set by CLI, always defaults to `iOS(.v17)`
