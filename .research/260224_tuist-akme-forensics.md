# Product Forensics Report — `tuist-akme` (Tuist-based iOS/macOS starter)

- **Date:** 2026-02-24
- **Target repo:** `/Users/alexis/src/relux-works/tuist-akme/`
- **Repo revision analyzed:** `3ca4b9cc1de4feddc6c804a9257d54796a628c3f` (branch: `main`)
- **Goal:** Understand the Tuist structure/patterns so `tuist-starter` can replicate them.
- **Constraint:** Read-only on `tuist-akme` (no modifications).

---

## Executive Summary (L4)

`tuist-akme` is a small, Tuist-driven multi-platform app workspace (iOS + macOS) with a **modular architecture** and an explicit **composition root** pattern. It implements a “project DSL” as a local Tuist plugin (`ProjectInfraPlugin`) that standardizes:

- module identity (`ModuleID`), naming, paths, and bundle IDs
- dependency rules (Interface/Impl split, external dependency allow-list)
- app + extension project factories and entitlements generation from a `Capability` DSL
- automation scripts that keep module enums in sync and validate architectural edges using `tuist graph -f json`

The repo is closer to a **template** than a full app: only one feature module (`Auth`) and two composition roots exist, but the scaffolding supports growth into Core/Shared/Utility layers.

**Sources:** `README.md`, `Tuist.swift`, `Workspace.swift`, `Makefile`, `Apps/**/Project.swift`, `Modules/**/Project.swift`, `TuistPlugins/ProjectInfraPlugin/**`.

---

## Highlights / Key Takeaways

1. **Two-tier Tuist helper system**
   - **Generic DSL plugin:** `TuistPlugins/ProjectInfraPlugin` (reusable patterns and enforcement).
   - **Project-specific helpers:** `Tuist/ProjectDescriptionHelpers` (bundle IDs, composition roots, tags, external dep allow-list, etc.).

2. **Strong module boundary model**
   - Each “feature module” is always 4 targets: `Interface`, `Impl`, `Testing`, `Tests`. (`ProjectFactory.makeFeature` enforces shape.)
   - Only **composition roots** may link other modules’ **Impl** targets. Other modules should link only **Interface** targets. This is checked by a script reading `tuist graph` JSON + bundle ID conventions.

3. **Environment-driven identity + sandbox-compatible**
   - Tuist manifest sandbox is **enabled** and environment variables are mapped to `TUIST_*` via `Makefile` so manifests can read them.
   - Repo-tracked `.env.shared` defines canonical IDs/names; local `.env` supplies Team ID + bundle ID suffix for collision-free dev signing.

4. **Deterministic identifier scheme**
   - Bundle IDs are derived from a repo-wide `coreRoot`, plus `sharedRoot` for explicitly shared capability identifiers (cross-platform).
   - Local dev namespacing uses `TUIST_BUNDLE_ID_SUFFIX` inserted after `com.acme`-like prefix.

5. **Capabilities → entitlements DSL**
   - App/app-extension manifests declare `[Capability]`; `EntitlementsFactory` builds `.entitlements`.
   - Portal capability metadata is loaded from Xcode’s `DVTPortalCachedPortalCapabilities.json` for platform support validation.

**Sources:** `Tuist.swift`, `Makefile`, `.env.shared`, `Tuist/ProjectDescriptionHelpers/AppIdentifiers.swift`, `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/*`, `Scripts/check_tuist_graph_architecture.py`, `Docs/RFC-0001-Identifiers.md`.

---

## L1 Recon — Repo Structure & Inventory

### Top-level layout (as implemented)

```
Apps/
  iOSApp/
    Extensions/AcmeWidget/
    Sources/
  macOSApp/
    Sources/
Docs/
  RFC-0001-Identifiers.md
Modules/
  CompositionRoots/
    AppCompositionRoot/
    WidgetCompositionRoot/
  Features/
    Auth/
Scripts/
Tuist/
  ProjectDescriptionHelpers/
TuistPlugins/
  ProjectInfraPlugin/
Makefile
Package.swift
Package.resolved
Tuist.swift
Workspace.swift
.env.shared
```

Notably, the README documents additional layers (`Modules/Core`, `Modules/Shared`, `Modules/Utility`) but they are not present in the current commit; the tooling supports them (generated enums produce empty `allCases`). See “Gaps / Follow-ups” later.

**Sources:** `README.md`, `Modules/**`, `Tuist/ProjectDescriptionHelpers/Modules+Generated.swift`.

### File counts (git-tracked)

- Total tracked files: **61**
- By type: **48** `.swift`, **6** `.py`, **2** `.md`, plus `Makefile`, `.env.shared`, `Package.swift`, `Package.resolved`, etc.

**Sources:** `git ls-files` inventory (derived), `Makefile`, `Scripts/*.py`, `TuistPlugins/**`.

### Key configuration entry points

- Tuist config + plugin: `Tuist.swift`
- Workspace composition: `Workspace.swift`
- SPM dependencies (Tuist-integrated): `Package.swift` + `Package.resolved`
- Build/automation: `Makefile` + `Scripts/*`
- Project manifests: `Apps/**/Project.swift`, `Modules/**/Project.swift`
- Tuist DSL helpers:
  - project-specific: `Tuist/ProjectDescriptionHelpers/*`
  - generic plugin: `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/*`

**Sources:** `Tuist.swift`, `Workspace.swift`, `Package.swift`, `Makefile`, `Apps/**/Project.swift`, `Modules/**/Project.swift`.

---

## L2 Deep Dive — Key Files & Mechanisms

### 1) Tuist configuration (`Tuist.swift`, `Workspace.swift`)

Tuist uses a **local plugin** and sets an Xcode compatibility constraint:

```swift
let config = Tuist(
    project: .tuist(
        compatibleXcodeVersions: .upToNextMajor("26.0"),
        plugins: [
            .local(path: .relativeToRoot("TuistPlugins/ProjectInfraPlugin")),
        ],
        generationOptions: .options(
            disableSandbox: false
        )
    )
)
```

Workspace includes **all** app + module projects:

```swift
let workspace = Workspace(
    name: Environment.workspaceName.getString(default: "AcmeApp"),
    projects: [
        "Apps/**",
        "Modules/**",
    ]
)
```

**What this implies**
- Manifests can import `ProjectInfraPlugin` and `ProjectDescriptionHelpers`.
- Manifest sandboxing is **enabled**, so environment variables must be `TUIST_*` (addressed in `Makefile`).

**Sources:** `Tuist.swift`, `Workspace.swift`, `Makefile` (env mapping comments).

### 2) Environment model & local signing namespacing (`.env.shared`, `Makefile`, `ConfigurationHelper`)

Repo-tracked configuration lives in `.env.shared`:

```bash
WORKSPACE_NAME=AcmeApp
CORE_ROOT=com.acme.akmeapp
SHARED_ROOT=com.acme.akmeapp.shared
IOS_APP_PROJECT_NAME=iOSApp
IOS_APP_NAME=AcmeApp
IOS_MIN_VERSION=16.0
MACOS_APP_PROJECT_NAME=macOSApp
MACOS_APP_NAME=AcmeMacApp
MACOS_MIN_VERSION=13.0
```

Local-only `.env` (gitignored) provides:
- `DEVELOPMENT_TEAM_ID` (mapped to `TUIST_DEVELOPMENT_TEAM_ID`)
- `BUNDLE_ID_SUFFIX` (mapped to `TUIST_BUNDLE_ID_SUFFIX`)

Makefile maps user-facing variables → sandbox-visible `TUIST_*` variables.

Local namespacing insertion rule (preserves wildcard App IDs like `com.acme.*`):

```swift
public static func applySuffix(_ suffix: String?, to base: String, afterComponents: Int) -> String {
    // inserts suffix components after first `afterComponents` dot segments
}
```

**Sources:** `.env.shared`, `Makefile`, `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/ConfigurationHelper.swift`.

### 3) The core DSL: module identity, paths, and targets (`ModuleID`, `TargetFactory`, `BundleID`)

`ModuleID` is the central primitive:

```swift
public struct ModuleID: Hashable, Sendable {
    public let scope: ModuleScope   // common/darwin/ios/macos/...
    public let layer: ModuleLayer   // core/feature/shared/utility/compositionroot/app
    public let name: String         // folder/product name (PascalCase)
    var path: Path { ... }          // Modules/<Layer>/<Name>
    var interfaceTarget: String { "\(name)Interface" }
    var implTarget: String { name }
    var testingTarget: String { "\(name)Testing" }
    var testsTarget: String { "\(name)Tests" }
}
```

Targets get deterministic bundle IDs that encode kind:

```swift
// <coreRoot>.mod.<scope>.<layer>.<module>.<kind> (+ optional env suffix)
public static func module(_ module: ModuleID, kind: Kind) -> String { ... }
```

This bundle ID format is later exploited by the architecture checker script to infer “interface vs impl vs tests” from `tuist graph` JSON.

**Sources:** `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/ModuleID.swift`, `BundleID.swift`, `TargetFactory.swift`, `Scripts/check_tuist_graph_architecture.py`.

### 4) Dependency DSL + enforcement (`Dependency`, `CompositionRootDependency`, external allow-list)

Non-composition modules should depend on **Interface** targets:

```swift
public struct Dependency {
    public static func interface(_ module: ModuleID) -> Dependency
    public static func testing(_ module: ModuleID) -> Dependency
    public static func external(dependencyDescriptor: some ExternalDependencyDescriptor) -> Dependency
}
```

Composition roots can link **Interface + Impl** together:

```swift
public struct CompositionRootDependency {
    public static func module(_ module: ModuleID) -> CompositionRootDependency
}
```

Raw-string external dependencies are compile-time forbidden:

```swift
@available(*, unavailable, message: "Don't use raw strings. Define a project-level allow-list ...")
public static func external(_ name: String) -> Dependency
```

Project-specific allow-list is defined in `Tuist/ProjectDescriptionHelpers/ExternalDependency.swift` (with layer permissions), then used via `Dependency.external(dependency:)`.

**Sources:** `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/DependencyDSL.swift`, `Tuist/ProjectDescriptionHelpers/ExternalDependency.swift`, `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/ProjectFactory.swift`.

### 5) Project factories (standard shapes)

#### Feature modules (`makeFeature`)

`ProjectFactory.makeFeature` always creates **4 targets** and enforces:
- Interface target has **no external dependencies**
- external dependencies must be allowed for the module’s layer

```swift
public static func makeFeature(
  module: ModuleID,
  dependencies: [Dependency],
  testDependencies: [Dependency] = [],
  ...
) -> Project
```

#### Composition roots (`makeCompositionRoot`)

Composition roots are single-target frameworks that may link implementation targets directly:

```swift
public static func makeCompositionRoot(
  module: CompositionRoot,
  dependencies: [CompositionRootDependency],
  ...
) -> Project
```

#### Host apps + extensions

Apps depend on a single `CompositionRoot` and optionally embed extensions. Entitlements are generated from `Capability` declarations.

**Sources:** `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/ProjectFactory.swift`, `TargetFactory.swift`, `CompositionRoot.swift`, `Apps/**/Project.swift`, `Modules/**/Project.swift`.

### 6) App & module manifests in this repo (concrete examples)

#### iOS host app

```swift
let project = ProjectFactory.makeHostApp(
    projectName: projectName,
    appName: appName,
    bundleId: AppIdentifiers.iOSApp.bundleId,
    destinations: .iOS,
    deploymentTargets: .iOS(iosMinVersion),
    compositionRoot: .app,
    embeddedExtensions: [
        AppProjects.iOS.acmeWidget,
    ],
    capabilities: .iOSPlusAppex + [
        .iCloudCloudKitContainer(container: .shared),
    ],
    developmentTeamId: developmentTeamId
)
```

#### Feature module (`Auth`)

```swift
let project = ProjectFactory.makeFeature(
    module: .feature(.auth, scope: .darwin),
    destinations: Destinations.iOS.union(Destinations.macOS),
    dependencies: [
        .external(dependency: .algorithms),
        // ... plus other modules' Interface deps
    ],
    tags: [
        .owner(.identity),
        .area(.auth),
        .layer(.feature),
    ]
)
```

#### App composition root (`AppCompositionRoot`)

Composition root depends on the implementation of every module enumerated per layer:

```swift
let featureImplementations: [CompositionRootDependency] = FeatureLayer.allCases
    .map { .implementation(.feature($0)) }

let project = ProjectFactory.makeCompositionRoot(
    module: .app,
    dependencies: featureImplementations /* + core/shared/utility */
)
```

**Sources:** `Apps/iOSApp/Project.swift`, `Modules/Features/Auth/Project.swift`, `Modules/CompositionRoots/AppCompositionRoot/Project.swift`, `Tuist/ProjectDescriptionHelpers/Modules+Generated.swift`.

### 7) Automation scripts (Makefile + Python)

Key commands:
- `make` → bootstrap (once) → sync modules → generate workspace
- `make module layer=feature name=Payment` → create module skeleton + resync enums
- `make check-graph` → runs a graph-based dependency rule check
- `make check-docs` → enforces doc comment coverage for DSL layer (helpers/plugins)

Important scripts:
- `Scripts/sync_modules.py` scans `Modules/*` and regenerates `Tuist/ProjectDescriptionHelpers/Modules+Generated.swift`.
- `Scripts/check_tuist_graph_architecture.py` parses `tuist graph -f json` and forbids non-composition-root Impl→Impl edges.
- `Scripts/sync_portal_capabilities.py` generates the `Capability.PortalCapability` enum from Xcode metadata.
- `Scripts/tuist_generate.py` wraps `tuist generate` with phase-based logging.

**Sources:** `Makefile`, `Scripts/*.py`, `Tuist/ProjectDescriptionHelpers/Modules+Generated.swift`.

---

## L3 Domain Synthesis

### A) Project Configuration domain (Tuist + workspace + environment)

**Patterns**
- Single workspace enumerating `Apps/**` + `Modules/**`.
- Local Tuist plugin provides the “standard library” for manifests.
- Sandboxing enabled → env var mapping via Makefile.
- Multi-platform settings centralized in `BuildEnvironment` (debug/release settings dictionary).

**Replication notes for `tuist-starter`**
- Keep the Makefile env mapping pattern if sandbox stays enabled.
- Port `BuildEnvironment` and settings merge logic as-is, then adjust versions to your baseline.

**Sources:** `Tuist.swift`, `Workspace.swift`, `Makefile`, `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/BuildEnvironment.swift`.

### B) Module Architecture domain (layers, targets, boundaries, tags)

**Module shape (enforced)**
- Feature module = 4 targets: `*Interface`, `*` (impl), `*Testing`, `*Tests`.
- Composition root = single framework target (impl only), allowed to link module impl targets.

**Dependency rules**
- Interface target has **no dependencies** (factory hardcodes dependencies to `[]`).
- Impl depends on its interface + other modules’ interfaces + allowed external libs.
- Tests depend on impl + module testing helpers + optional testing deps from other modules.
- Architectural check script enforces “no impl→impl (unless composition root)”.

**Tagging for CI/focus**
- Tag keys: `owner`, `area`, `layer`, `platform`
- Tags are serialized as `"<key>:<value>"` and attached via `Target.metadata.tags` (impl + tests targets).

**Sources:** `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/ProjectFactory.swift`, `TargetFactory.swift`, `DependencyDSL.swift`, `Tuist/ProjectDescriptionHelpers/Tags/*`, `Scripts/check_tuist_graph_architecture.py`.

### C) Build & Automation domain (Makefile + setup)

**Automation intent**
- Provide a “one command bootstrap” (`make`) that installs tools, fetches deps (`tuist install`), generates workspace, and opens Xcode (unless in CI).
- Keep developer setup friction low via `Scripts/setup-tools.sh` and `.env` prompting.

**Sources:** `Makefile`, `Scripts/setup-tools.sh`, `Scripts/tuist_generate.py`.

### D) External Dependencies domain (SPM + allow-listing)

**SPM model**
- Root `Package.swift` declares external dependencies (currently `swift-algorithms`).
- Tuist-specific `#if TUIST` section configures `PackageSettings`:
  - `productTypes` mapping (e.g. `"Algorithms": .framework`)
  - default settings for external dependency targets

**Allow-list enforcement**
- External deps must be declared in `Tuist/ProjectDescriptionHelpers/ExternalDependency.swift` with allowed layers.
- Manifest uses `.external(dependency: .algorithms)` (typed) instead of raw strings.

**Sources:** `Package.swift`, `Package.resolved`, `Tuist/ProjectDescriptionHelpers/ExternalDependency.swift`, `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/DependencyDSL.swift`.

### E) Capabilities & Entitlements domain (capability DSL + Xcode metadata)

**Model**
- App manifests declare `[Capability]` (e.g. `.appGroups()`, `.iCloud(services:)`, `.healthKit([...])`, etc.).
- `EntitlementsFactory` generates a single `.entitlements` dictionary and aborts if a multiplatform target would produce different entitlements per platform.

**Key convention**
- `sharedRoot` is **stable** and intentionally **not** namespaced by local suffix; local namespacing is opt-in for `.custom(..., namespacing: .environmentSuffix)`.

**Xcode alignment**
- Portal capability support/platform mapping is loaded from Xcode’s `DVTPortalCachedPortalCapabilities.json` at manifest evaluation time.
- Portal capability ID enum is generated by `Scripts/sync_portal_capabilities.py`.

**Sources:** `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/Capability.swift`, `EntitlementsFactory.swift`, `Capability+PortalCapability.swift`, `Scripts/sync_portal_capabilities.py`, `.env.shared`.

---

## L4 Product Synthesis — Architecture Overview & Replication Guidance

### Architecture overview (what to copy)

If you want `tuist-starter` to follow `tuist-akme` patterns, the “core” is:

1. **Local Tuist plugin** (`ProjectInfraPlugin`)
   - `ModuleID` + layer/scope enums
   - `ProjectFactory`/`TargetFactory` standard shapes
   - `Dependency` DSL + architecture validation hooks
   - `Capability` DSL + `EntitlementsFactory` (optional but valuable)

2. **ProjectDescriptionHelpers (project-specific)**
   - `AppIdentifiers` (coreRoot/sharedRoot + suffix application)
   - `ExternalDependency` allow-list
   - `CompositionRoot` shortcuts (`.app`, `.widget`, …)
   - `Modules+Generated` + `ModuleID` convenience constructors
   - Tags (owner/area/layer/platform) for focus generation

3. **Automation scripts**
   - keep module enums in sync
   - validate graph rules
   - generate portal capability IDs from Xcode
   - wrap Tuist generation output for readability

**Sources:** `TuistPlugins/ProjectInfraPlugin/**`, `Tuist/ProjectDescriptionHelpers/**`, `Scripts/**`, `Makefile`.

### Module catalog (as of analyzed commit)

**Apps**
- `Apps/iOSApp` — iOS host app (SwiftUI) with embedded widget extension.
- `Apps/macOSApp` — macOS host app (SwiftUI).
- `Apps/iOSApp/Extensions/AcmeWidget` — widget extension project.

**Composition roots**
- `Modules/CompositionRoots/AppCompositionRoot` — shared composition root for iOS+macOS.
- `Modules/CompositionRoots/WidgetCompositionRoot` — widget composition root (iOS-only).

**Features**
- `Modules/Features/Auth` — sample feature module with Interface/Impl/Testing/Tests targets.

**Sources:** directory structure + `Modules+Generated.swift` + per-module `Project.swift`.

### Dependency map (intended shape)

Legend:
- `→` target dependency
- `[Interface|Impl|Testing|Tests]` = per-module targets

```
iOS host app target (AcmeApp)
  → AppCompositionRoot (framework)
  → AcmeWidget (app extension target)

macOS host app target (AcmeMacApp)
  → AppCompositionRoot (framework)

AcmeWidget (app extension)
  → WidgetCompositionRoot (framework)

AppCompositionRoot (compositionroot impl)
  → AuthInterface
  → Auth
  → ... all other modules’ interface+impl via generated layer enums

WidgetCompositionRoot (compositionroot impl)
  → AuthInterface
  → Auth

Auth (impl)
  → AuthInterface
  → external: Algorithms (allow-listed)

AuthTesting
  → AuthInterface

AuthTests
  → Auth
  → AuthTesting
```

**Sources:** `Apps/iOSApp/Project.swift`, `Apps/macOSApp/Project.swift`, `Apps/iOSApp/Extensions/AcmeWidget/Project.swift`, `Modules/CompositionRoots/*/Project.swift`, `Modules/Features/Auth/Project.swift`, `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/ProjectFactory.swift`.

### Conventions worth replicating verbatim

- **Environment mapping for sandboxed manifests** (Makefile `TUIST_*` pattern).
- **Module enums generation** (`sync_modules.py` + `Modules+Generated.swift`).
- **Typed allow-list for external dependencies** (`ExternalDependencyDescriptor` + project enum).
- **Bundle ID encoding of module kind** (enables static graph checks).
- **Composition root dependency exception** (only place wiring can see impl targets).
- **Tagging system** for ownership/area/layer/platform focus generation.
- **Entitlements DSL** with strong validation (optional but robust).

**Sources:** `Makefile`, `Scripts/sync_modules.py`, `Tuist/ProjectDescriptionHelpers/ExternalDependency.swift`, `BundleID.swift`, `Scripts/check_tuist_graph_architecture.py`, `Tuist/ProjectDescriptionHelpers/Tags/*`, `EntitlementsFactory.swift`.

### Gaps / Follow-ups (observations + fact-checked)

These are not “bugs” necessarily (repo looks like a template), but are relevant when cloning patterns:

1. **README describes layers not present in repo**
   - README mentions `Modules/Core`, `Modules/Shared`, `Modules/Utility`; current repo only has `Features` and `CompositionRoots`.
   - Tooling handles empty layers via generated enums returning `allCases == []`.
   - **Sources:** `README.md`, `Tuist/ProjectDescriptionHelpers/Modules+Generated.swift`, filesystem tree.

2. **Graph checker layer name mismatch for composition roots**
   - Bundle IDs use `ModuleLayer.compositionRoot` raw value: `"compositionroot"`.
   - `Scripts/check_tuist_graph_architecture.py` looks for `"compositionRoot"` (camel case) in bundle ID segments.
   - Outcome: composition root targets may not be classified as modules by the checker (likely benign because rule is about non-composition-root impl→impl edges, but worth aligning if you extend checks).
   - **Sources:** `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/ModuleID.swift`, `BundleID.swift`, `Scripts/check_tuist_graph_architecture.py`.

3. **Swift version settings are inconsistent**
   - `BuildEnvironment` sets `SWIFT_VERSION = "6.2"` for targets.
   - `Package.swift`’s `PackageSettings` sets `SWIFT_VERSION = "5.9"` for external dependency targets.
   - Decide whether this split is intentional; for a starter, consider aligning.
   - **Sources:** `TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/BuildEnvironment.swift`, `Package.swift`.

4. **Sample app code imports a feature module directly**
   - `Apps/iOSApp/Sources/AcmeAppApp.swift` imports `Auth`.
   - If you want strict “app imports only composition root”, you may want to refactor sample app code accordingly in the starter.
   - **Sources:** `Apps/iOSApp/Sources/AcmeAppApp.swift`, `Apps/iOSApp/Project.swift`.

### Recommendations for bootstrapping `tuist-starter`

**Minimum replication set**
1. Copy `TuistPlugins/ProjectInfraPlugin` and keep it local-plugin based.
2. Copy `Tuist/ProjectDescriptionHelpers` helpers and adjust:
   - app names
   - coreRoot/sharedRoot defaults
   - composition roots you need (`.app`, `.widget`, etc.)
   - external dependency allow-list
3. Copy scripts + Makefile targets (`sync-modules`, `module`, `check-graph`, `check-docs`, `tuist-generate`).
4. Create module folder layout (even if empty): `Modules/Core`, `Modules/Shared`, `Modules/Utility`, `Modules/Features`, `Modules/CompositionRoots`.

**Starter “golden path”**
- keep `make` as the single entry point
- ensure `make module layer=... name=...` works from day 1
- keep architecture guardrails (graph check) and doc enforcement (DSL docs) in CI

**Sources:** `Makefile`, `Scripts/*`, `Tuist/ProjectDescriptionHelpers/*`, `TuistPlugins/ProjectInfraPlugin/*`.

---

## Fact-checking & Non-modification Statement

- `tuist-akme` was analyzed at commit `3ca4b9cc1de4feddc6c804a9257d54796a628c3f` with a clean working tree at the time of inspection.
- No commands that generate or write artifacts inside `tuist-akme` (e.g. `tuist generate`, `tuist graph`) were executed as part of this research; only file reads and listing commands were used.

**Sources:** `git status -sb` (observed clean), plus the file references embedded throughout this report.

