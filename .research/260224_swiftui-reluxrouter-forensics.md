# Product Forensics — `swiftui-reluxrouter`

- Task: `TASK-260224-2kpi7l` (forensics-swiftui-reluxrouter)
- Date: 2026-02-24
- Target: `.temp/repos/swiftui-reluxrouter/` @ `0865f7a` (branch `main`)
- Method: 4-layer MapReduce (L1→L4) in one document
- Constraint: Read-only analysis (no source changes)

## Highlights (Key takeaways)

- The package provides three router states under `Relux.Navigation`: stack routers for `NavigationStack`/`NavigationSplitView` (`ProjectingRouter`, `CodableProjectedRouter`) plus a modal sheet stack router (`ModalRouter`). (`Sources/Routers/...`)
- `ProjectingRouter` (iOS 16+) keeps a SwiftUI `NavigationPath` plus a parallel `pathProjection` that tags pages as either `.known(Page)` or `.external`, allowing detection of “system/back-button” navigation via path length diffs. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:5-119`)
- `CodableProjectedRouter` (iOS 17+) leverages `NavigationPath.codable` to maintain a decoded `[Page]` (`customPath`) for inspection, persistence (UserDefaults), and custom operations like “remove page before last”. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:14-337`, `Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter+Reducer.swift:11-58`)
- Deep-linking is not a URL router; instead, the package supports *navigation-state restore* by encoding/decoding `NavigationPath.CodableRepresentation` and storing it in UserDefaults. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:222-317`)
- Modal navigation is modeled as a stack of `Page` values and rendered via a fixed-depth nested `.sheet(item:)` chain (`sheetStack`) supporting up to 8 layers. (`Sources/Routers/ModalRouter/View+sheetsStack.swift:12-128`)
- Documentation appears stale: README refers to a `Relux.Navigation.Router` type and action labels that don’t exist/match this repo’s code (current type is `CodableProjectedRouter`). (`README.md:3,26-45,107-110,124-125` vs `Sources/Routers/CodableProjectedRouter/...` and `Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter+Action.swift:13`)

---

# L1 — Recon (surface scan)

## Package overview

- Swift tools version: `6.0`. (`Package.swift:1`)
- Product: `ReluxRouter` (library). (`Package.swift:13-17`)
- Targets: single target `ReluxRouter` with sources in `Sources/`. (`Package.swift:22-29`)
- Dependency: `swift-relux` from `9.0.0` (product `Relux`). (`Package.swift:19-27`)
- Platforms: iOS 16, macOS 13, watchOS 9, tvOS 16, macCatalyst 16. (`Package.swift:6-12`)

## Source organization (excluding `.git/`)

```
swiftui-reluxrouter/
  Package.swift
  README.md
  LICENSE
  Sources/
    Relux+Navigation+Namespace.swift
    InternalUtils/
      Collection+Subscript+Safe.swift
    Routers/
      ProjectedRouter/
        Relux+Navigation+ProjectingRouter.swift
        Relux+Navigation+ProjectingRouter+Action.swift
      CodableProjectedRouter/
        Relux+Navigation+CodableProjectedRouter.swift
        Relux+Navigation+CodableProjectedRouter+Action.swift
        Relux+Navigation+CodableProjectedRouter+Reducer.swift
      ModalRouter/
        Relux+Navigation+ModalRouter.swift
        View+sheetsStack.swift
```

File counts (excluding `.git/`):
- `10` × `.swift`, `1` × `README.md`, `1` × `.gitignore`, `1` × `LICENSE` (13 total files).

## Tests

- No `Tests/` directory and no test target declared in `Package.swift`. (`Package.swift:22-29`)

---

# L2 — Deep-dive (read all source files)

## Public API surface (catalog)

### `Relux.Navigation.ProjectingRouter<Page>` (iOS 16+)

- Declaration: `public final class ProjectingRouter<Page>: RouterProtocol, ObservableObject where Page: PathComponent`. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:20-23`)
- State:
  - `@Published public var path: NavigationPath` (SwiftUI-bound navigation stack). (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:31`)
  - `@Published public private(set) var pathProjection: [ProjectedPage]` where `ProjectedPage = .known(Page) | .external`. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:5-10,37`)
- Actions: `push(_ page: Page, allowingDuplicates: Bool = false)`, `set(_ pages: [Page])`, `removeLast(Int = 1)`. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter+Action.swift:6-25`)
- RouterProtocol entry points:
  - `public func reduce(with action: any Relux.Action) async` casts to `ProjectingRouter.Action` and delegates. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:66-78`)
  - `public func cleanup() async` clears `path` and `pathProjection`. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:58-64`)

#### How it synchronizes `path` and `pathProjection`

- One-way Combine pipeline: observes `$path`, computes `pagesDiff = path.count - pathProjection.count`, and:
  - if diff < 0: removes from projection
  - if diff > 0: appends `.external` placeholders
  - if diff == 0: no-op  
  (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:98-121`)

Interpretation:
- When navigation changes via SwiftUI/system (e.g., back-swipe), `path` changes first, and the router updates `pathProjection` using only length diffs. New pushes from “outside” are represented as `.external` because `NavigationPath` does not expose typed elements on iOS 16.

#### Internal reduction model

- For internal (Relux) actions, it mutates both `pathProjection` and `path` directly:
  - `.push`: appends `.known(page)` and `path.append(page)`. Duplicate prevention checks if *any* `.known(page)` exists in `pathProjection` when `allowingDuplicates == false`. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:135-154`)
  - `.set`: sets `pathProjection = pages.map(.known)` and `path = NavigationPath(pages)` (guarded by projection equality). (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:156-168`)
  - `.removeLast`: removes up to `min(count, pathProjection.count)` from both. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:170-177`)

#### Notable issue (init pages)

- The initializer only applies `.set(pages)` when `pages.isEmpty`, which means non-empty `pages:` are ignored. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:42-46`)
  - Likely intended: `if !pages.isEmpty { internalReduce(with: .set(pages)) }`

### `Relux.Navigation.CodableProjectedRouter<Page>` (iOS 17+)

- Declaration: `@Observable @MainActor public final class CodableProjectedRouter<Page>: RouterProtocol, Observable where Page: PathCodableComponent`. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:14-17`)
- State:
  - `public var path: NavigationPath` with `didSet` side effects (internal/external change detection, persistence, callback, and `customPath` sync). (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:19-41`)
  - `public var customPath: [Page]` = decoded typed path for inspection / custom ops. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:43-44`)
- Actions:
  - `.push(page:disableAnimation:)`
  - `.set(pages:)`
  - `.removeLast(count:)`
  - `.removeBeforeLast`  
  (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter+Action.swift:6-26`)
- RouterProtocol entry points:
  - `public func reduce(with action: any Relux.Action) async` casts to `CodableProjectedRouter.Action` and delegates. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:322-337`)
  - `public func cleanup() async` resets `path`. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:218-220`)

#### Internal vs external navigation change detection

- `path.didSet` distinguishes changes initiated by router actions via `_isInternalChange` (set in `internalReduce`) vs system/UI changes:
  - Internal: updates `previousPathCount`, optionally persists. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:23-29`)
  - External (count change only): computes `changeAmount = previousPathCount - path.count`, persists, and calls `onSystemNavigationChange?(changeAmount)`. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:29-38`)
  - Always updates `customPath`. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:39-40`)

Notes:
- External change detection is based on `count` diffs; if `path` changes without changing count, the callback won’t fire (but `customPath` still updates).

#### Typed decoding: `NavigationPath` → `[Page]` (`customPath`)

- `updateCustomPath()`:
  - Requires `path.codable != nil`, then encodes `NavigationPath.CodableRepresentation` and decodes it as `[String]` pairs `[typeName, jsonString, ...]`. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:89-106`)
  - Enforces `typeName == String(reflecting: Page.self)` for each pair. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:107-124`)
  - Decodes each JSON string into `Page` and appends to `pages`. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:126-138`)
  - Reverses the decoded list before assigning to `customPath` (root→top order). (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:140-142`)

#### Deep-linking / restoration primitives (encode/decode + UserDefaults)

The router exposes “navigation state” serialization and persistence:
- Encode/decode helpers for `NavigationPath.CodableRepresentation`: (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:222-260`)
  - `public static func encodePath(_ path: NavigationPath, ...) -> Data?`
  - `public static func decodePath(from data: Data, ...) -> NavigationPath?`
- UserDefaults helpers: (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:263-278`)
  - `loadDataFromUserDefaults(forKey:)`
  - `saveDataToUserDefaults(_:forKey:)`
- Convenience persistence:
  - `saveNavigationPathToUserDefaults(forKey:)` and `restoreNavigationPathFromUserDefaults(forKey:)`. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:294-317`)
- Init can restore automatically when `userDefaultsKey` is provided. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:53-82`)

Interpretation:
- This is “deep linking” in the sense of *restoring a navigation stack from serialized state* (Data/JSON), not mapping URLs to pages. URL→`[Page]` mapping is left to the app.

#### Reducer behavior

- `internalReduce(with:)` sets `_isInternalChange = true`, mutates `path`, then resets the flag. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter+Reducer.swift:11-58`)
- `.push(page:disableAnimation:)` optionally disables animations via `Transaction(animation: .none)` + `withTransaction`. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter+Reducer.swift:16-26`)
- `.removeBeforeLast` works on `customPath` (typed), then serializes and reconstructs a new `NavigationPath` via `NavigationPath.CodableRepresentation`. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter+Reducer.swift:37-54`, `Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:155-203`)

### `Relux.Navigation.ModalRouter<Page>` (iOS 16+)

- Declaration: `@MainActor public final class ModalRouter<Page>: RouterProtocol, ObservableObject where Page: ModalCodableComponent`. (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:4-9`)
- State:
  - `public let maxDepth: Int` (named `maxPages` in init parameter). (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:13-15,69-72`)
  - `@Published public var modalPage: [Page]` (modal stack). (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:21-30`)
  - Optional persistence via `userDefaultsKey`; `modalPage` and `previouslyReplacedPage` changes trigger save. (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:16-39`)
- Actions: `.pushModal(page:)`, `.popModal`, `.removePages(fromIndex:)`, `.dismissModal`. (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:112-130`)
- SwiftUI integration helper:
  - `public subscript(page index: Int) -> Binding<Page?>`:
    - `get`: returns `modalPage[safe: index]`
    - `set`: dispatches `.removePages(fromIndex: index)` (supports swipe-to-dismiss).  
    (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:47-56`, plus safe subscript in `Sources/InternalUtils/Collection+Subscript+Safe.swift:1-6`)
- RouterProtocol entry points:
  - `public func reduce(with action: any Relux.Action) async` casts to `ModalRouter.Action`. (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:101-106`)
  - `public func cleanup() async` clears stack and replaced page. (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:94-99`)

#### Stack semantics / max depth behavior

- Pushing beyond maxDepth:
  - Pops and stores the previous top into `previouslyReplacedPage`
  - Appends the new page asynchronously on MainActor (likely to allow SwiftUI sheet transition).  
  (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:144-155`)
- Removing pages from an index:
  - Removes suffix from index onward
  - If `previouslyReplacedPage` exists, it gets appended back and cleared.  
  (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:164-177`)
- `dismissModal` clears all and restores replaced page if present. (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:179-187`)

#### State restoration

- Encodes `ModalRouterState(modalStack: [Page], replacedPage: Page?)` to JSON and saves to UserDefaults. (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:193-213`)
- Restores state in init when key is provided. (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:77-85,215-234`)

### `View.sheetStack(router:content:)` (modal presentation helper)

- Public API: `View.sheetStack(router: ModalRouter<Page>, content: (Page) -> some View)`. (`Sources/Routers/ModalRouter/View+sheetsStack.swift:5-23`)
- Implementation: fixed nested `.sheet(item:)` chain for indices `0...7`. (`Sources/Routers/ModalRouter/View+sheetsStack.swift:32-128`)
- Notable issue: `depth` is computed and stored, but never used to gate modifier nesting. (`Sources/Routers/ModalRouter/View+sheetsStack.swift:19-30`)

## Documentation alignment check

The README does not match the current code surface in multiple places:
- Mentions `Router` and uses `Relux.Navigation.Router<...>()` instead of `CodableProjectedRouter`. (`README.md:3,26-45,107-110`)
- Example action call uses `.push(page:)` label, but `ProjectingRouter.Action` uses an unlabeled first parameter (`push(_:)`). (`README.md:124-125` vs `Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter+Action.swift:13`)

## Tests

- None in repo (no test target, no test sources). (`Package.swift:22-29`)

---

# L3 — Domain synthesis (grouped findings)

## Domain: Route definitions (type-safe “Page” model)

- Routes are defined by the consumer as `Page` types:
  - Stack navigation: `Page: PathComponent` (Equatable, Hashable, Sendable). (`.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Navigation/Relux+Navigation.swift:8`)
  - Codable stack navigation: `Page: PathCodableComponent` (adds `Codable`). (`.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Navigation/Relux+Navigation.swift:9`)
  - Modal navigation: `Page: ModalCodableComponent` (Codable + Identifiable; default `id = hashValue`). (`.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Navigation/Relux+Navigation.swift:11-22`)
- This package does **not** include view-to-route mapping. Consumers still define SwiftUI destinations (e.g., `.navigationDestination(for: Page.self)`) and sheet content builders; the routers only manage state.

## Domain: Stack navigation state management

- iOS 16+: `ProjectingRouter` manages a `NavigationPath` and a separate projection list:
  - Internal actions write both in lockstep. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:133-178`)
  - External changes are inferred by path length deltas and represented as `.external`. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:98-119`)
- iOS 17+: `CodableProjectedRouter` treats `NavigationPath` as the source of truth and maintains a typed mirror (`customPath`) by decoding the codable representation. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:19-150`)

## Domain: Persistence / “deep linking”

- `CodableProjectedRouter` provides primitives for:
  - encoding `NavigationPath` to Data (`encodePath`)
  - decoding Data to `NavigationPath` (`decodePath`)
  - storing/restoring Data in UserDefaults (`saveNavigationPathToUserDefaults`, `restoreNavigationPathFromUserDefaults`)  
  (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:222-317`)
- Format implications:
  - Serialization is tightly coupled to `String(reflecting: Page.self)` (fully-qualified type name) and a JSON string for each page (`[typeName, json, ...]`). (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:107-138,155-175`)
  - Refactors that rename modules/types can break restoration unless a migration step is introduced.

## Domain: Modal navigation patterns

- Modal routing is modeled as a stack of pages, with presentation driven by `.sheet(item:)` bindings:
  - `ModalRouter` exposes binding per index. (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:47-56`)
  - `sheetStack` composes nested sheet modifiers for up to 8 layers. (`Sources/Routers/ModalRouter/View+sheetsStack.swift:32-128`)
- Max depth is enforced at the state layer (`maxDepth`) with a “swap top page” strategy when over capacity. (`Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:144-155`)

## Domain: Relux integration pattern

- In Relux, routers are “UI + business” hybrid states:
  - `RouterProtocol` extends `Relux.HybridState`. (`.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Navigation/Relux+Navigation.swift:7`)
  - `HybridState` requires `reduce(with:) async` and `cleanup() async` and is `@MainActor`. (`.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift:4-16`)
- Package pattern:
  - Each router’s `reduce(with:)` casts `any Relux.Action` to its own nested `Action` enum and ignores other actions. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:72-78`, `Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:330-336`, `Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift:103-106`)

---

# L4 — Product synthesis (what it is, how to use it, recommendations)

## Executive summary

`swiftui-reluxrouter` is a small Swift package that adds routing state objects compatible with:
- SwiftUI navigation stacks (`NavigationPath`)
- modal sheet stacks (`.sheet(item:)`)

It’s designed to plug into a Relux store as additional `HybridState` instances and be injected into SwiftUI views via environment patterns used elsewhere in the Relux ecosystem (not shipped here, but shown in README).

## Architecture overview

- **Core abstraction**: “Router == Relux state” (`RouterProtocol` → `HybridState`). Routers receive global actions and selectively handle only their own action enum.
- **Stack navigation**:
  - iOS 16: `ProjectingRouter` uses Combine + `@Published` to drive SwiftUI and compensates for lack of typed `NavigationPath` inspection by keeping a side-channel projection.
  - iOS 17: `CodableProjectedRouter` uses Observation + `NavigationPath.codable` to decode actual `Page` values and offers serialization/persistence.
- **Modal navigation**:
  - `ModalRouter` manages a `[Page]` and provides index bindings used by the `sheetStack` view helper.

## Public API (quick reference)

- Stack (iOS 16+): `Relux.Navigation.ProjectingRouter<Page: PathComponent>`
  - `path: NavigationPath`, `pathProjection: [ProjectedPage]`
  - `Action`: `push`, `set`, `removeLast`
- Stack + codable restore (iOS 17+): `Relux.Navigation.CodableProjectedRouter<Page: PathCodableComponent>`
  - `path: NavigationPath`, `customPath: [Page]`
  - `Action`: `push(disableAnimation:)`, `set`, `removeLast`, `removeBeforeLast`
  - Persistence: encode/decode + UserDefaults helpers
- Modal (iOS 16+): `Relux.Navigation.ModalRouter<Page: ModalCodableComponent>`
  - `modalPage: [Page]`, `maxDepth`
  - `Action`: `pushModal`, `popModal`, `removePages`, `dismissModal`
  - View helper: `View.sheetStack(router:content:)`

## Key patterns and conventions

- **Action casting**: routers ignore unrelated actions by `as?` casting to their nested `Action`.
- **MainActor state**: all routers are `@MainActor`, matching Relux `UIState/HybridState`.
- **Typed pages as value enums**:
  - Path pages are hashable and sendable; modal pages are identifiable (with default `id = hashValue`), making `.sheet(item:)` workable without extra boilerplate.
- **External navigation detection**:
  - `ProjectingRouter`: `NavigationPath` length diff → `.external` placeholders.
  - `CodableProjectedRouter`: `previousPathCount` diff triggers `onSystemNavigationChange(changeAmount)`.
- **Persistence model**:
  - Uses UserDefaults by key, with JSON encoding of `NavigationPath.CodableRepresentation` or a custom `ModalRouterState`.
  - Strongly tied to fully-qualified type names.

## Integration points with other Relux packages

- Depends on `swift-relux` for:
  - router and page protocols (`RouterProtocol`, `PathComponent`, `PathCodableComponent`, `ModalCodableComponent`)
  - state lifecycle (`HybridState`: `reduce` + `cleanup`)  
  (See `.temp/repos/swift-relux/...` citations in L3.)
- The README’s SwiftUI environment wiring (`passingObservableToEnvironment(fromStore:)`) likely comes from `swiftui-relux` / app-level helpers, not from this package. (`README.md:57-72`)

## Recommendations for a `relux-manager` CLI component

High-leverage CLI support for this package would focus on reducing integration friction and preventing persistence breakage:

1. **Codegen templates**
   - Generate `Page` enums conforming to `PathCodableComponent` / `ModalCodableComponent` with stable `Codable` payload patterns.
   - Generate router registration boilerplate (one router per domain/tab) and SwiftUI destination stubs.

2. **Navigation state inspection + migration tooling**
   - Add a command to decode `NavigationPath.CodableRepresentation` JSON into a human-readable list of pages (and re-encode).
   - Support type-name migration maps (old fully-qualified type → new type) to keep persisted paths compatible across refactors.

3. **Static checks / lint**
   - Detect README/API mismatches (e.g., referenced type names that don’t exist).
   - Flag suspicious conditions (e.g., `ProjectingRouter.init(pages:)` ignoring non-empty `pages`). (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:42-46`)
   - Flag unused parameters that indicate behavior gaps (e.g., `sheetStack`’s unused `depth`). (`Sources/Routers/ModalRouter/View+sheetsStack.swift:19-30`)

4. **Test scaffolding**
   - Generate unit-test skeletons that dispatch router actions and assert `path.count`, `customPath`, and modal stack behavior (especially persistence encode/decode).

## Risks / gaps

- **Stale docs**: README examples won’t compile against current API without adjustments. (`README.md:3,26-45,124-125`)
- **No tests**: reducers and serialization formats are unverified in-repo. (`Package.swift:22-29`)
- **Potential bug**: `ProjectingRouter.init(pages:)` appears inverted and may ignore initial page stacks. (`Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift:42-46`)
- **Persistence fragility**: fully-qualified type name checks can break restoration across module/type renames. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:107-124`)
- **Modal presentation helper rigidity**: `sheetStack` is hard-coded to 8 levels and currently ignores computed `depth`. (`Sources/Routers/ModalRouter/View+sheetsStack.swift:19-30,32-128`)

---

# Fact-checks (verification notes)

## `NavigationPath.CodableRepresentation` JSON format + ordering

`CodableProjectedRouter.updateCustomPath()` assumes the encoded `NavigationPath.CodableRepresentation` JSON decodes as `[String]` pairs and that elements are in reverse push order (top→root), hence the explicit `pages.reversed()` assignment. (`Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift:89-142`)

Verified locally with Swift 6.2.3 by encoding a `NavigationPath` containing `a → b → c`:

- JSON produced: `["<TypeName>","\"c\"","<TypeName>","\"b\"","<TypeName>","\"a\""]` (reverse order).
- Decoding in-order yields `[c, b, a]`, requiring a reverse to recover root→top.

(Repro script was executed in this environment; sandbox required passing module-cache flags.)

## Read-only guarantee

`git status --porcelain` in `.temp/repos/swiftui-reluxrouter` was clean after analysis.

---

# Files read (swiftui-reluxrouter)

- `Package.swift`
- `README.md`
- `Sources/Relux+Navigation+Namespace.swift`
- `Sources/InternalUtils/Collection+Subscript+Safe.swift`
- `Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter.swift`
- `Sources/Routers/ProjectedRouter/Relux+Navigation+ProjectingRouter+Action.swift`
- `Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter.swift`
- `Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter+Action.swift`
- `Sources/Routers/CodableProjectedRouter/Relux+Navigation+CodableProjectedRouter+Reducer.swift`
- `Sources/Routers/ModalRouter/Relux+Navigation+ModalRouter.swift`
- `Sources/Routers/ModalRouter/View+sheetsStack.swift`

