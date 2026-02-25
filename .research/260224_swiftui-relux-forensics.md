# swiftui-relux — Product Forensics (TASK-260224-1ih0dr)

- **Date:** 2026-02-24
- **Target:** `/Users/alexis/src/relux-works/tuist-starter/.temp/repos/swiftui-relux/`
- **Revision:** `0e99b77cc482bad040256ebc62beae2dfdb76bfd` ("Bump swift-relux to 9.0.0, swiftui-reluxrouter to 11.0.0")

## Scope

Analyze the `swiftui-relux` Swift Package as SwiftUI bindings for Relux:

- View integration patterns
- State observation model
- Property wrappers (if any)
- Environment injection
- How views subscribe to store changes

**Constraint:** read-only for package sources (no modifications).

---

## Highlights / Key Takeaways

1. **SwiftUIRelux is primarily an integration/utility layer**, not a full binding framework: it injects Relux “UI states” into SwiftUI environment and provides protocols + lightweight callback wrappers.
2. **View updates rely on SwiftUI’s own observation** (`ObservableObject` for iOS 16; `Observation`/`Observable` for iOS 17+), not on a custom subscription mechanism in this package. (`Sources/EnvironmentObject+ViewState.swift:23-57`)
3. **`Relux.Resolver` is the root integration point**: it asynchronously resolves a `Relux` instance, then injects `store.uiStates` into the environment. (`Sources/View+ReluxResolver.swift:35-77`)
4. **Container/View separation is encouraged but not enforced by code**: `Relux.UI.Container` is a marker protocol; `Relux.UI.View` is an `Equatable` view keyed by `props`. (`Sources/Protocols/Relux+UI+Container.swift:4-6`, `Sources/Protocols/Relux+UI+View.swift:4-21`)
5. **Callback wrappers (`ViewCallback`) make async callbacks equatable** by hashing a call-site identity (`#fileID/#function/#line`), decoupling SwiftUI diffing from closure identity. (`Sources/Relux+UI+ViewCallback.swift:1-57`)

---

# L1 — Recon (Surface Scan)

## Directory Structure

`swiftui-relux/` contains only a SwiftPM package definition and `Sources/`.

- `Package.swift`
- `Sources/` (11 `.swift` files)
  - `Protocols/` (5 protocol definitions)
  - integration utilities (`Resolver`, environment injection, callbacks)

(See file list: `.gitignore`, `LICENSE`, `Package.swift`, and `Sources/**.swift`.)

## File Counts (excluding `.git/`)

- Total files: **14**
- By extension:
  - `swift`: **12** (includes `Package.swift`)
  - `gitignore`: **1**
  - none: **1** (`LICENSE`)

## SwiftPM / Package.swift

- Tools version: **Swift 6.0** (`Package.swift:1`)
- Platforms: **iOS 16**, **macOS 13** (`Package.swift:6-9`)
- Product:
  - `library(name: "SwiftUIRelux", targets: ["SwiftUIRelux"])` (`Package.swift:11-17`)
- Dependencies:
  - `swift-relux` `upToNextMajor(from: "9.0.0")`
  - `swiftui-reluxrouter` `upToNextMajor(from: "11.0.0")`
  (`Package.swift:19-22`)
- Target:
  - `target(name: "SwiftUIRelux", dependencies: [Relux, ReluxRouter], path: "Sources")` (`Package.swift:24-33`)

---

# L2 — Deep Dive (Read All Sources)

## Public API Surface (Catalog)

### Re-exports

- `@_exported import ReluxRouter` and `@_exported import Relux` (`Sources/Reexported+Modules.swift:1-2`)
  - Practical effect: `import SwiftUIRelux` implicitly imports the Relux core and router modules.

### Root integration & environment injection

- `SwiftUI.View.passingObservableToEnvironment(fromStore:)` (`Sources/EnvironmentObject+ViewState.swift:5-18`)
  - Collects all `store.uiStates.values` and injects them into SwiftUI environment (`Sources/EnvironmentObject+ViewState.swift:10-17`).
  - iOS 16/macOS 13 path: injects only objects that are `ObservableObject` via `.environmentObject(_)` (`Sources/EnvironmentObject+ViewState.swift:31-40`).
  - iOS 17/macOS 14 path: injects both:
    - `ObservableObject` via `.environmentObject(_)` (`Sources/EnvironmentObject+ViewState.swift:46-51`)
    - `Observable & AnyObject` via `.environment(_)` (`Sources/EnvironmentObject+ViewState.swift:52-55`)

- `SwiftUI.View.resolvedRelux(content:resolver:)` (`Sources/View+ReluxResolver.swift:15-24`)
  - Wraps the caller view in `Relux.Resolver`.

- `Relux.Resolver<Splash, Content>: View` (`Sources/View+ReluxResolver.swift:35-79`)
  - Holds `@State private var resolved: Relux?` (`Sources/View+ReluxResolver.swift:36-37`).
  - While `resolved == nil`, shows splash and runs `resolver()` in a `.task` modifier (`Sources/View+ReluxResolver.swift:66-71`).
  - Once resolved, renders `content(relux)` and injects UI states: `.passingObservableToEnvironment(fromStore: relux.store)` (`Sources/View+ReluxResolver.swift:72-75`).

### UI namespace + protocols

- `Relux.UI` namespace: `extension Relux { public enum UI {} }` (`Sources/Relux+Namespace+UI.swift:3-4`)

- `Relux.UI.Container`: marker protocol for SwiftUI views (`Sources/Protocols/Relux+UI+Container.swift:4-6`)

- `Relux.UI.View`: protocol for “pure” presentational SwiftUI views (`Sources/Protocols/Relux+UI+View.swift:4-12`)
  - Requirements:
    - `SwiftUI.View & Equatable`
    - `associatedtype Props = Relux.UI.ViewProps`
    - `nonisolated var props: Props { get }`
  - Default `Equatable` implementation: if `Props: Equatable`, equality is `lhs.props == rhs.props` (`Sources/Protocols/Relux+UI+View.swift:16-21`).

- `Relux.UI.ViewProps`: `DynamicProperty & Equatable & Hashable & Sendable` (`Sources/Protocols/Relux+UI+ViewProps.swift:3-5`)

- `Relux.UI.ViewCallbacks`: `Equatable & Sendable` (`Sources/Protocols/Relux+UI+ViewCallbacks.swift:2-4`)

### Callback wrapper types

- `Relux.UI.ViewCallback<Input: Sendable>` (`Sources/Relux+UI+ViewCallback.swift:1-34`)
  - Stores a private async closure `@Sendable (Input) async -> Void` (`Sources/Relux+UI+ViewCallback.swift:14-15`).
  - Is `Equatable` / `Identifiable` by a `CallSite` key made from `#fileID/#function/#line` (`Sources/Relux+UI+ViewCallback.swift:4-13`, `Sources/Relux+UI+ViewCallback.swift:31-33`).
  - Convenience overloads:
    - `Input == Void` initializer and `callAsFunction()` (`Sources/Relux+UI+ViewCallback.swift:37-51`).
    - Parameter-pack call syntax when `Input` is a tuple (`Sources/Relux+UI+ViewCallback.swift:53-57`).

- `Relux.UI.ViewAction<each Input: Sendable>` (iOS 17+ only; **deprecated as experimental**) (`Sources/Relux+UI+ViewAction.swift:9-12`)
  - Similar “equatable callback by call-site” but modeled as a variadic generic action (`Sources/Relux+UI+ViewAction.swift:13-46`).

### Deprecated legacy protocols

- Global `ViewProps` and `ViewActions` protocols are deprecated in favor of `Relux.UI.*` (`Sources/Protocols/View+Props.swift:1-7`).

## How View Subscription Works (as implemented)

### 1) What gets injected

`passingObservableToEnvironment(fromStore:)` gathers `store.uiStates.values` (from `Relux.Store`) and tries to inject each value into the environment (`Sources/EnvironmentObject+ViewState.swift:10-17`).

In Relux core, `uiStates` is a dictionary of `any Relux.UIState` stored in the store (`swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift:4-13`).

### 2) Observation mechanics

This package does **not** implement its own publisher/subscriber for SwiftUI. Instead:

- On iOS 16/macOS 13: only `ObservableObject` values are injected as `@EnvironmentObject` (`Sources/EnvironmentObject+ViewState.swift:31-40`). SwiftUI invalidates dependent views when the `ObservableObject` announces changes.

- On iOS 17/macOS 14+: additionally supports Observation’s `Observable` types via `.environment(_)` (`Sources/EnvironmentObject+ViewState.swift:43-57`). SwiftUI invalidates dependent views when observable properties change.

### 3) How store changes reach SwiftUI

Because `Relux.UIState` is only a marker protocol (`swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift:9-16`), the **actual “subscription”** is achieved by convention:

- UI state objects stored in `uiStates` are expected to also conform to an observation protocol (`ObservableObject` or `Observable`) so SwiftUI can react to mutations.
- Many UI-facing objects are **Hybrid** states. In Relux core, `HybridState` is both `BusinessState` and `UIState` (`swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift:12-16`). When you connect a state to the store, it is inserted into both `businessStates` and `uiStates` if it conforms to both (`swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift:63-77`).
  - The store propagates actions by calling `reduce(with:)` on `businessStates` (which includes Hybrid states) (`swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift:16-30`).
  - Those states mutate their own observable properties (`@Published` / `@Observable`), and SwiftUI updates.

### 4) Router integration as a concrete example

Relux defines `Navigation.RouterProtocol` as `HybridState` (`swift-relux/Sources/Relux/Relux/Relux+Navigation/Relux+Navigation.swift:6-13`). In `swiftui-reluxrouter`, routers are observable UI states (e.g. `ModalRouter` is `ObservableObject` with `@Published` state and implements `reduce/cleanup`) and thus work naturally with `swiftui-relux` environment injection.

Example (Modal router): `ModalRouter` is `@MainActor`, `ObservableObject`, and implements `cleanup()` and `reduce(with:)` (`swiftui-reluxrouter/Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:7-107`).

## Tests / Documentation

- `swiftui-relux` includes **no `Tests/` target** and **no README** in the package itself (repo root contains only `Package.swift`, `LICENSE`, `.gitignore`, and `Sources/`).
- Usage and patterns are documented in the broader ecosystem:
  - `relux-sample/README.ru.md` describes `Relux.Resolver` and `.passingObservableToEnvironment(fromStore:)` (`relux-sample/README.ru.md:100-131`).
  - `relux-sample/PROJECT_GUIDE.md` outlines the intended Container/View split and use of `Relux.UI.ViewCallback` (`relux-sample/PROJECT_GUIDE.md:126-223`).
  - `relux-sample/relux_sample/App.swift` describes accessing state via `EnvironmentObject` (ObservableObject) or `Environment` (@Observable) (`relux-sample/relux_sample/App.swift:24-28`).

---

# L3 — Domain Synthesis

## Domain: Environment Injection & Observation

- **Injection point:** `Relux.Resolver` injects all `store.uiStates` into SwiftUI environment post-resolution (`Sources/View+ReluxResolver.swift:72-75`).
- **Observation support:**
  - iOS 16/macOS 13: `ObservableObject` only via `environmentObject` (`Sources/EnvironmentObject+ViewState.swift:31-40`).
  - iOS 17/macOS 14+: supports Observation `Observable` via `environment` (`Sources/EnvironmentObject+ViewState.swift:43-57`).
- **Key implicit contract:** a UI state must be observable to be useful in SwiftUI; this is not enforced by `Relux.UIState` protocol itself.

## Domain: View Integration Pattern (Container ↔ View)

- Package provides marker protocols (`Relux.UI.Container`, `Relux.UI.View`, `Relux.UI.ViewProps`, `Relux.UI.ViewCallbacks`) to support a “connected container / pure view” approach (`Sources/Protocols/*.swift`).
- Intended usage (from sample documentation):
  - Containers use `@EnvironmentObject` to read UI states and dispatch actions.
  - Views are pure and depend only on primitive props + callbacks (`relux-sample/PROJECT_GUIDE.md:126-223`).

## Domain: Callback Modeling

- `ViewCallback` makes callbacks stable under `Equatable` comparisons by using call-site identity (file/function/line) (`Sources/Relux+UI+ViewCallback.swift:4-34`).
- This supports the overall goal of minimizing SwiftUI invalidation by making view props equatable without capturing closure identity.

## Domain: API Stability / Compatibility

- `ViewAction` is explicitly marked “Highly Experimental” and deprecated in favor of `ViewCallback` (`Sources/Relux+UI+ViewAction.swift:9-12`).
- Legacy protocols exist (deprecated) to ease migration (`Sources/Protocols/View+Props.swift:1-7`).

## Domain: Integration Points

- With `swift-relux`:
  - `Relux.Store.uiStates` is the bridge to SwiftUI (`swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift:4-13`).
  - `HybridState` enables a single state object to both reduce actions and be injected into the UI (`swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift:12-16`).
- With `swiftui-reluxrouter`:
  - Router types are HybridStates and observable, making them ideal UI states for environment injection (`swift-relux/Sources/Relux/Relux/Relux+Navigation/Relux+Navigation.swift:6-13`, `swiftui-reluxrouter/Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:7-107`).

---

# L4 — Product Synthesis

## Executive Summary

`swiftui-relux` is a small SwiftPM package that bridges Relux state management into SwiftUI primarily through **environment injection** of `store.uiStates` and a root **async resolver view**. It encourages a Container/View separation by providing lightweight protocols and an equatable async callback wrapper.

## Architecture Overview

High-level flow:

1. App/root view uses `Relux.Resolver` (or `.resolvedRelux`) to asynchronously obtain a `Relux` instance (`Sources/View+ReluxResolver.swift:15-24`, `Sources/View+ReluxResolver.swift:35-79`).
2. After resolution, the content view is wrapped with `.passingObservableToEnvironment(fromStore: relux.store)` (`Sources/View+ReluxResolver.swift:72-75`).
3. `passingObservableToEnvironment` iterates over `relux.store.uiStates.values` and injects each UI state into SwiftUI environment as:
   - `@EnvironmentObject` when it’s `ObservableObject`
   - `@Environment(T.self)` (via `.environment(_)`) when it’s `Observable` (iOS 17+)
   (`Sources/EnvironmentObject+ViewState.swift:31-57`).
4. Containers can read UI states from environment; UI states update as actions are reduced (typically via `HybridState`), triggering SwiftUI updates.

## Public API (Practical Catalog)

- `View.passingObservableToEnvironment(fromStore: Relux.Store) -> some View`
- `View.resolvedRelux(content: (Relux) -> Content, resolver: () async -> Relux) -> some View`
- `Relux.Resolver<Splash, Content>: View`
- Namespace + protocols:
  - `Relux.UI`
  - `Relux.UI.Container`, `Relux.UI.View`, `Relux.UI.ViewProps`, `Relux.UI.ViewCallbacks`
- Callback wrapper:
  - `Relux.UI.ViewCallback<Input>` (recommended)
  - `Relux.UI.ViewAction<each Input>` (deprecated, iOS 17+)
- Re-exports:
  - `Relux`, `ReluxRouter`

## Key Patterns & Conventions

- **Environment-driven state access**: states must be stored in `store.uiStates` and must be observable for SwiftUI consumption.
- **Equatable Views via Props**: `Relux.UI.View` encourages `props`-based equality so SwiftUI can avoid unnecessary invalidations (`Sources/Protocols/Relux+UI+View.swift:16-21`).
- **Callbacks over bindings**: callback wrappers model async actions without leaking closure identity into equality checks (`Sources/Relux+UI+ViewCallback.swift:1-57`).

## Recommendations for `relux-manager` (CLI) Component

If `relux-manager` scaffolds or audits Relux+SwiftUI apps, it should encode the implicit contracts observed here:

1. **Scaffold a Root Resolver**
   - Generate a `Root` that uses `Relux.Resolver` and ensures `.passingObservableToEnvironment(fromStore:)` is applied.

2. **Enforce “UIState must be observable”**
   - Provide templates for UI states that conform to `ObservableObject` (iOS 16+) or `@Observable` (iOS 17+).
   - Add a static check (or docs) warning that non-observable `UIState`s will not be injected (and thus can’t drive UI updates).

3. **Generate Container/View skeletons**
   - Container: `Relux.UI.Container` using `@EnvironmentObject` / `@Environment(T.self)` to read state.
   - View: `Relux.UI.View` with `Props: Relux.UI.ViewProps` and `Actions: Relux.UI.ViewCallbacks`.

4. **Prefer `ViewCallback` over `ViewAction`**
   - `ViewAction` is marked deprecated/experimental; CLI templates should use `ViewCallback` (`Sources/Relux+UI+ViewAction.swift:9-12`).

5. **Document the “Equatable by call-site” trade-off**
   - `ViewCallback` equality uses only call-site, not captured values. In generated code, keep callback instantiations stable and avoid constructing “different semantic callbacks” on the same line.

---

## Appendix: Source Index (swiftui-relux)

- `Package.swift`
- `Sources/Reexported+Modules.swift`
- `Sources/Relux+Namespace+UI.swift`
- `Sources/EnvironmentObject+ViewState.swift`
- `Sources/View+ReluxResolver.swift`
- `Sources/Relux+UI+ViewCallback.swift`
- `Sources/Relux+UI+ViewAction.swift`
- `Sources/Protocols/Relux+UI+Container.swift`
- `Sources/Protocols/Relux+UI+View.swift`
- `Sources/Protocols/Relux+UI+ViewCallbacks.swift`
- `Sources/Protocols/Relux+UI+ViewProps.swift`
- `Sources/Protocols/View+Props.swift`
