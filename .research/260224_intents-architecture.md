# App Intents Architecture (Tuist + SPM, all-dynamic, Interface/Impl split)
  
- **Task:** `TASK-260224-znzotl` (research-intents-architecture)  
- **Date:** 2026-02-24  
- **Context:** Tuist-based iOS app with modular architecture where each product module is split into **two Swift packages**: `ModuleName` (**Interface**) + `ModuleNameImpl` (**Implementation**). All package products are integrated as **dynamic frameworks**.

## Highlights / Key Takeaways

1. **An App Intent must be present in every bundle that needs to execute it** (app, widget extension, App Intents extension). “Define once” means “put it in a shared module” and **link that module into each bundle**, not “ship it only once”. (Input research: `.research/260224_intents-siri-shortcuts.md`)
2. **iOS 17+ is the practical baseline for modular App Intents**: it adds official support for packaging App Intents in frameworks/dynamic libraries (`AppIntentsPackage`), and WidgetKit’s `AppIntentConfiguration` (App-Intent-based widget configuration). (WWDC23 + SDK headers)
3. **Extension entrypoints stay in targets; intent types live in shared frameworks**: keep `@main` for WidgetKit and `@main` for the App Intents extension in their respective targets, but put `AppIntent` / `AppEntity` / `EntityQuery` implementations into a shared module.
4. **Use `AppDependency` / `AppDependencyManager` as the cross-bundle DI seam**: intents can depend on feature protocols; each bundle registers concrete implementations (app vs widget vs App Intents extension) at startup.
5. **SPM dynamic boundaries require separate packages**: if you model Interface+Impl as targets inside one package, `Impl` will embed Interface symbols (no true dynamic link). Two packages per module is required to preserve a real dynamic boundary. (Input research: `.research/260224_spm-linking-tuist.md`)

---

## Scope / Assumptions

- **Primary target OS:** iOS 17+ (recommended).  
  - Needed for `WidgetKit.AppIntentConfiguration` (iOS 17+) and `AppIntents.AppIntentsPackage` (iOS 17+).
- **Legacy support (optional):** if you must support iOS 16 widgets with configuration, you’ll likely need SiriKit intent-based `IntentConfiguration` (`INIntent` + `.intentdefinition`). That’s a parallel track and is not designed into the “single source of truth App Intents” path here.
- **Non-goals:** full SiriKit migration plan, deep UX/HIG for Shortcuts, APNs payload design for Live Activity push updates (covered in other research).

---

## 0) Constraints that Drive the Architecture

### 0.1 Bundles & execution model

In practice you’ll have multiple bundles that can host/execute intent code:

- **Main app bundle** (may define App Shortcuts / donate intents / execute intents while foregrounded).
- **Widget extension** (executes widget actions; uses App Intents for interactive widgets; uses `WidgetConfigurationIntent` for configurable widgets).
- **App Intents extension** (dedicated “intent execution bundle”; useful to avoid background-launching the app and to isolate intent work).
- **Live Activities UI** lives in the widget extension, but **shared models** can (and should) live in shared modules.

**Implication:** plan for **one shared intent implementation module** linked into multiple bundles, plus **bundle-specific bootstrapping**.

**Sources**
- Input research: `.research/260224_intents-siri-shortcuts.md` (cross-target reuse + App Intents extension notes)
- Xcode template: `.../App Intents Extension.xctemplate/AppIntentsExtension.swift` (entrypoint is `@main` in the extension target)

### 0.2 “All-dynamic” + Interface/Impl split (SPM linking reality)

SwiftPM target-to-target dependencies inside one package don’t give you a true dynamic boundary between products. If you want:

`ModuleNameImpl.dylib` → dynamically links → `ModuleName.dylib`

…you must split them into **two separate packages**, not two targets in one package.

**Sources**
- Input research: `.research/260224_spm-linking-tuist.md` (Mach-O + `otool -L` verification)

---

## 1) Where do intent definitions live? (Q1)

### Option A — Dedicated `SharedIntents` product module (recommended)

Create a dedicated product module:

- `SharedIntents` (Interface package): public protocols/DTOs used by intents (and optionally stable `AppEntity` identifiers/value types).
- `SharedIntentsImpl` (Implementation package): actual `AppIntent`, `AppEntity`, `EntityQuery`, `DynamicOptionsProvider` implementations.

**Rationale**

- App Intents are a **system-integration surface** (Shortcuts/Siri/Spotlight/Widget actions), so grouping them avoids scattering “system contract” code across feature modules.
- Putting intent code in one place makes it easier to:
  - keep it **extension-safe** (no UIKit-only dependencies),
  - enforce **App Group / shared storage** usage,
  - avoid multiple `AppShortcutsProvider` definitions across bundles.

### Option B — Intents live in the feature module that owns the domain

Each feature package (`PaymentsImpl`, `ProjectsImpl`, …) defines its own intents.

**Pros:** feature ownership is clean; fewer “god modules”.  
**Cons:** every extension that uses any intent must depend on many feature modules; you need a “one-provider” policy for app shortcuts and a packaging story for registering all packages.

### Option C — Hybrid: per-feature intent packages + one aggregator

Per feature:

- `FeatureIntents` + `FeatureIntentsImpl` packages

Plus:

- `SharedIntentsImpl` that depends on all feature-intents packages and (optionally) defines the single `AppShortcutsProvider`.

**When to pick:** large app with many teams/features, where “system integrations” are still centralized for registration/bootstrapping.

**Recommendation:** start with **Option A**, move to **Option C** if `SharedIntentsImpl` grows too large.

**Sources**
- WWDC22: “Dive into App Intents” (App Intents as unified system action model): https://developer.apple.com/videos/play/wwdc2022/10032/
- Input research: `.research/260224_intents-siri-shortcuts.md` (Tuist organization patterns A/B/C)

---

## 2) How to avoid code duplication across app and extensions? (Q2)

### 2.1 Put these in shared frameworks (SPM packages)

**Shared intent layer**

- `AppIntent` types (`perform()` implementations)
- `AppEntity` types (+ identifiers)
- `EntityQuery` / `DynamicOptionsProvider` implementations
- shared parameter enums/value types

**Shared “capability” layers (used by intents + widgets + live activities)**

- **Domain protocols** (service interfaces, repositories)
- **Shared storage** (App Group container access)
- **Pure models** (Codable DTOs, Activity attributes, widget timeline models)

### 2.2 Keep these in the extension/app targets (thin wrappers)

- **Widget entrypoints**: `@main` `WidgetBundle` / widget definitions, SwiftUI widget UI.
- **App Intents extension entrypoint**: `@main struct MyIntentsExtension: AppIntentsExtension {}` (Xcode template).
- **Bundle-specific dependency registration**: register concrete services into `AppDependencyManager` (app vs widget vs intents extension).
- **Target-only entitlements & Info.plist**: App Groups, Siri/Shortcuts capabilities, background modes as needed.

### 2.3 DI seam: `AppDependency` / `AppDependencyManager`

Inside intents, depend on **protocols**:

```swift
import AppIntents

struct RefreshDataIntent: AppIntent {
  static var title: LocalizedStringResource { "Refresh data" }

  @AppDependency var service: RefreshServicing

  func perform() async throws -> some IntentResult {
    try await service.refresh()
    return .result()
  }
}
```

In each bundle’s “composition root”, register implementations:

```swift
import AppIntents

func registerIntentDependencies() {
  AppDependencyManager.shared.add(dependency: RefreshServiceLive(/* app group, clients */))
}
```

This keeps **intent definitions shared** while allowing **bundle-specific implementations** (e.g., read-only services in widgets, full services in app).

**Sources**
- SDK header: `AppDependency` + `AppDependencyManager` (`AppIntents.swiftinterface`):
  - `/Applications/Xcode.app/.../iPhoneSimulator26.2.sdk/.../AppIntents.swiftinterface` (search “`AppDependencyManager`”, “`AppDependency`”)
- Input research: `.research/260224_intents-siri-shortcuts.md` (module split: domain vs intents vs UI)

---

## 3) Module structure for Tuist (names, packages, dependency graph) (Q3)

### 3.1 Proposed product modules (concrete names)

Cross-cutting (“platform”) modules:

- `SharedModels` + `SharedModelsImpl` (value types, Codable DTOs used across surfaces)
- `SharedStorage` + `SharedStorageImpl` (App Group access, caching, persistence adapters)
- `SharedIntents` + `SharedIntentsImpl` (App Intents, entities, queries)
- `AppShortcuts` + `AppShortcutsImpl` (**only** the `AppShortcutsProvider`; link into exactly one bundle)

Feature modules (example):

- `Projects` + `ProjectsImpl`
- `Accounts` + `AccountsImpl`

Bundle composition (optional but recommended for clarity):

- `AppComposition` + `AppCompositionImpl` (register app dependencies, wires feature impls)
- `ExtensionsComposition` + `ExtensionsCompositionImpl` (register extension-safe dependencies; used by widget + App Intents extension)

### 3.2 Dependency diagram (ASCII)

**Bundles**

```
 ┌───────────────────────────┐
 │           App             │
 │  (Tuist target: .app)     │
 └─────────────┬─────────────┘
               │ depends on
               ▼
     SharedIntentsImpl  +  AppCompositionImpl  +  SharedStorageImpl
               ▲
               │ depends on
               │
 ┌─────────────┴─────────────┐         ┌───────────────────────────┐
 │      Widgets Extension     │         │     App Intents Extension  │
 │ (Tuist target: .appex)     │         │ (Tuist target: extensionkit│
 └─────────────┬─────────────┘         └─────────────┬─────────────┘
               │ depends on                         │ depends on
               ▼                                    ▼
        SharedIntentsImpl                    SharedIntentsImpl
        SharedStorageImpl                    SharedStorageImpl
        ExtensionsCompositionImpl            ExtensionsCompositionImpl
                                             + AppShortcutsImpl (only here OR only in App)
```

**Core packages**

```
SharedIntentsImpl ──▶ SharedIntents (interface)
        │
        ├──▶ SharedStorage (interface)
        ├──▶ SharedModels (interface)
        └──▶ Feature interfaces (e.g., Projects, Accounts)

SharedStorageImpl ──▶ SharedStorage (interface)
```

### 3.3 Why `AppShortcutsProvider` is split out

`AppShortcutsProvider` is a “registration surface”. If you link the provider type into multiple bundles, you risk multiple shortcut sets being exposed (or non-deterministic “which bundle wins” behavior).

So: **put the provider in its own module** and link it into exactly one bundle:

- **either** the main App bundle  
- **or** the App Intents extension bundle (often preferred when you want to avoid background app launches)

**Sources**
- SDK header: `AppShortcutsProvider` (`AppIntents.swiftinterface`)
- Input research: `.research/260224_intents-siri-shortcuts.md` (note about single provider)
- WWDC23: “Explore enhancements to App Intents” (mentions defining providers in an App Intents extension): https://developer.apple.com/videos/play/wwdc2023/10103/

---

## 4) AppIntents framework specifics & constraints (Q4)

### 4.1 Availability checkpoints that matter

- `AppIntent` exists since iOS 16.0 (`AppIntents.swiftinterface`).
- `AppIntentsPackage` is iOS 17.0+ (`AppIntents.swiftinterface`).
- Widget configuration with App Intents:
  - `WidgetKit.AppIntentConfiguration` is iOS 17.0+ (`WidgetKit.swiftinterface`)
  - `WidgetConfigurationIntent` is iOS 17.0+ (`AppIntents.swiftinterface`)

**Implication:** if you want a single shared App Intents module used by widgets for configuration and actions, iOS 17+ is the clean baseline.

### 4.2 Can `AppIntent` types live in dynamic frameworks?

Yes — Apple explicitly introduced/expanded **packaging**:

- iOS 17 / Xcode 15: “App Intents can now be defined in frameworks … we’ve introduced `AppIntentsPackage` APIs.” (WWDC23)
- 2025: packaging expands to **Swift packages and static libraries** (WWDC25).

In this project’s architecture (SPM packages integrated as dynamic frameworks), that means:

- Put `AppIntent` conformances in `SharedIntentsImpl` (a dynamic framework product).
- Link `SharedIntentsImpl` into each bundle that needs to execute them.

### 4.3 What must remain in the app/extension target?

- The extension entrypoint must be in the target:
  - App Intents extension template uses `@main struct … : AppIntentsExtension {}`.
  - Its template `Info.plist` uses `EXExtensionPointIdentifier = com.apple.appintents-extension`.
- Widget entrypoints (`@main Widget` / `WidgetBundle`) must be in the widget extension target.

So: **types can live in frameworks; entrypoints cannot**.

### 4.4 Extension-safety constraints

Some intent APIs are unavailable in app extensions (example from SDK header: `ForegroundContinuableIntent` is unavailable for `iOSApplicationExtension`). Treat shared-intent code as “extension code” and keep it:

- UI-free (no UIKit assumptions)
- fast and resilient (Shortcuts + widget execution is system-budgeted)
- backed by shared storage (App Groups) rather than in-process state

**Sources**
- WWDC23 packaging statement: https://developer.apple.com/videos/play/wwdc2023/10103/
- WWDC25 packaging statement: https://developer.apple.com/videos/play/wwdc2025/275/
- SDK header: `AppIntentsPackage`, `AppIntent`, `ForegroundContinuableIntent`:
  - `/Applications/Xcode.app/.../iPhoneSimulator26.2.sdk/.../AppIntents.swiftinterface`
- Xcode template: App Intents extension Info.plist snippet:
  - `/Applications/Xcode.app/.../App Intents Extension.xctemplate/TemplateInfo.plist`

---

## 5) Data sharing between app ↔ widgets ↔ intents (Q5)

### 5.1 Baseline: App Groups

Use App Groups to share:

- `UserDefaults(suiteName: "group.<id>")`
- a shared file container via `FileManager.default.containerURL(forSecurityApplicationGroupIdentifier:)`

This enables `EntityQuery` / `DynamicOptionsProvider` to read user data when running out-of-process (widget or App Intents extension).

### 5.2 Practical guidance for intent execution

- Design queries to be **read-only**, fast, and cache-friendly.
- Prefer a local read model in the App Group container (e.g., a lightweight JSON/SQLite “projection”) rather than forcing network calls in every query.
- For write operations:
  - either write directly to the shared store (if safe),
  - or write an intent “command” to shared storage and let the app reconcile when it next runs (useful when business rules require full app context).

### 5.3 Live Activities crossover

Live Activities require shared types across targets:

- Put `ActivityAttributes` + shared `ContentState` models in `SharedModels` (or a dedicated `LiveActivitiesModels` module).
- The widget extension uses those types for `ActivityConfiguration` UI.
- Intents that affect a Live Activity should ideally **mutate shared state** and let the app perform `Activity.update(_:)` (to keep ActivityKit calls centralized), unless you intentionally allow ActivityKit usage in the intent host.

**Sources**
- Apple docs: Configuring app groups: https://developer.apple.com/documentation/xcode/configuring-app-groups/
- Input research: `.research/260224_live-activities-widgets.md` (App Groups + Live Activities shared models guidance)

---

## 6) Concrete Tuist + SPM layout (with `Package.swift`) (Q6)

### 6.1 Proposed `Packages/` folder layout (example)

```
Packages/
  SharedIntents/
    Package.swift              # product: SharedIntents (dynamic)
    Sources/SharedIntents/...

  SharedIntentsImpl/
    Package.swift              # product: SharedIntentsImpl (dynamic)
    Sources/SharedIntentsImpl/...

  AppShortcuts/
    Package.swift              # product: AppShortcuts (dynamic)
  AppShortcutsImpl/
    Package.swift              # product: AppShortcutsImpl (dynamic; contains AppShortcutsProvider)

  SharedStorage/
  SharedStorageImpl/
  SharedModels/
  SharedModelsImpl/
  Projects/
  ProjectsImpl/
  ...
```

### 6.2 `SharedIntents/Package.swift` (Interface package)

```swift
// Packages/SharedIntents/Package.swift
import PackageDescription

let package = Package(
  name: "SharedIntents",
  platforms: [.iOS(.v17)],
  products: [
    .library(name: "SharedIntents", type: .dynamic, targets: ["SharedIntents"]),
  ],
  dependencies: [
    // Interfaces only:
    // .package(path: "../SharedModels"),
    // .package(path: "../Projects"),
  ],
  targets: [
    .target(
      name: "SharedIntents",
      dependencies: [
        // .product(name: "SharedModels", package: "SharedModels"),
        // .product(name: "Projects", package: "Projects"),
      ]
    ),
  ]
)
```

### 6.3 `SharedIntentsImpl/Package.swift` (Implementation package)

```swift
// Packages/SharedIntentsImpl/Package.swift
import PackageDescription

let package = Package(
  name: "SharedIntentsImpl",
  platforms: [.iOS(.v17)],
  products: [
    .library(name: "SharedIntentsImpl", type: .dynamic, targets: ["SharedIntentsImpl"]),
  ],
  dependencies: [
    .package(path: "../SharedIntents"),
    .package(path: "../SharedStorage"),
    .package(path: "../SharedModels"),

    // Feature interfaces the intents depend on:
    .package(path: "../Projects"),
    .package(path: "../Accounts"),
  ],
  targets: [
    .target(
      name: "SharedIntentsImpl",
      dependencies: [
        .product(name: "SharedIntents", package: "SharedIntents"),
        .product(name: "SharedStorage", package: "SharedStorage"),
        .product(name: "SharedModels", package: "SharedModels"),
        .product(name: "Projects", package: "Projects"),
        .product(name: "Accounts", package: "Accounts"),
      ]
    ),
  ]
)
```

### 6.4 Tuist dependency wiring (conceptual)

In Tuist targets, depend on the **implementation products** where you need the concrete intent types:

- App target depends on `SharedIntentsImpl`
- Widget extension depends on `SharedIntentsImpl`
- App Intents extension depends on `SharedIntentsImpl`
- Only **one** bundle depends on `AppShortcutsImpl`

Conceptually (pseudo-Tuist):

```swift
TargetDependency.package(product: "SharedIntentsImpl")
TargetDependency.package(product: "SharedStorageImpl")
```

### 6.5 Bootstrapping checklist (per bundle)

Each bundle should call a bootstrap function early:

- register `AppDependencyManager` dependencies (implementations)
- ensure App Group identifier is configured and consistent
- keep the bootstrap minimal and synchronous where possible

**Sources**
- SDK header: `WidgetKit.AppIntentConfiguration`:
  - `/Applications/Xcode.app/.../iPhoneSimulator26.2.sdk/.../WidgetKit.swiftinterface`
- Xcode template: App Intents extension (`@main`, extension point id):
  - `/Applications/Xcode.app/.../App Intents Extension.xctemplate/*`
- Input research: `.research/260224_spm-linking-tuist.md` (two-package dynamic boundary requirement)

---

## Fact-checking (what was verified)

Verified against local SDK headers / templates shipped with **Xcode 26.2**:

- `AppIntentsPackage` availability is **iOS 17.0+** (`AppIntents.swiftinterface`).
- `WidgetConfigurationIntent` availability is **iOS 17.0+** (`AppIntents.swiftinterface`).
- `WidgetKit.AppIntentConfiguration` availability is **iOS 17.0+** (`WidgetKit.swiftinterface`).
- App Intents extension template uses:
  - `@main struct … : AppIntentsExtension {}` (`AppIntentsExtension.swift`)
  - `EXExtensionPointIdentifier = com.apple.appintents-extension` (`TemplateInfo.plist`)

---

## Recommendation (summary)

1. Create a dedicated **shared intents implementation module**: `SharedIntentsImpl` and link it into **App + Widgets + App Intents extension**.
2. Keep `SharedIntentsImpl` extension-safe by depending only on:
   - feature **interfaces** (protocols/DTOs),
   - shared storage/models,
   - and AppIntents/WidgetKit APIs.
3. Use `AppDependencyManager` as the main seam: register concrete services per bundle.
4. Split `AppShortcutsProvider` into its own module and link it into exactly one bundle.
5. Keep App Group storage as the primary shared data plane for queries and intent execution.

---

## Source Index

### Input research (this repo)

- `.research/260224_intents-siri-shortcuts.md`
- `.research/260224_live-activities-widgets.md`
- `.research/260224_spm-linking-tuist.md`

### WWDC / Apple videos

- WWDC22 — Dive into App Intents: https://developer.apple.com/videos/play/wwdc2022/10032/
- WWDC23 — Explore enhancements to App Intents (framework packaging): https://developer.apple.com/videos/play/wwdc2023/10103/
- WWDC25 — Explore new advances in App Intents (Swift package/static lib packaging): https://developer.apple.com/videos/play/wwdc2025/275/

### Apple docs

- WidgetKit — Making a configurable widget: https://developer.apple.com/documentation/widgetkit/making-a-configurable-widget
- Xcode — Configuring app groups: https://developer.apple.com/documentation/xcode/configuring-app-groups/
- AppIntents API collection: https://developer.apple.com/documentation/appintents

### Framework headers / templates (local)

- AppIntents SDK interface:
  - `/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneSimulator.platform/Developer/SDKs/iPhoneSimulator26.2.sdk/System/Library/Frameworks/AppIntents.framework/Modules/AppIntents.swiftmodule/arm64-apple-ios-simulator.swiftinterface`
- WidgetKit SDK interface:
  - `/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneSimulator.platform/Developer/SDKs/iPhoneSimulator26.2.sdk/System/Library/Frameworks/WidgetKit.framework/Modules/WidgetKit.swiftmodule/arm64-apple-ios-simulator.swiftinterface`
- Xcode App Intents extension template:
  - `/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/Library/Xcode/Templates/Project Templates/iOS/Application Extension/App Intents Extension.xctemplate/TemplateInfo.plist`
  - `/Applications/Xcode.app/Contents/Developer/Platforms/iPhoneOS.platform/Developer/Library/Xcode/Templates/Project Templates/iOS/Application Extension/App Intents Extension.xctemplate/AppIntentsExtension.swift`

