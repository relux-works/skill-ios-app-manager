# relux-sample — Product Forensics (TASK-260224-38fnpq)

- **Date:** 2026-02-24
- **Target:** `/Users/alexis/src/relux-works/tuist-starter/.temp/repos/relux-sample/`
- **Revision analyzed:** `9979d63ebaf5dc7628db0287fb929eebfe7c42ad` (`9979d63 Bump swift-relux to 9.0.0, swiftui-relux to 8.0.0`)
- **Deliverable:** Full L1–L4 forensics report (read-only; **no source modifications**)

## Scope

Analyze `relux-sample` as a reference SwiftUI app showcasing Relux architecture. Focus areas:

- How Relux packages wire together in a real app
- Module structure and naming conventions
- App bootstrap + registration order
- Feature modules (Auth, Notes, Navigation, ErrorHandling, Account, Logger)
- Dependency injection setup (SwiftIoC) and DI seams (router adapters, service factories)
- Testing strategy and implemented tests

Constraints:

- Read-only for `relux-sample` sources (analysis only).
- Integrations with `swiftui-relux` and `swift-ioc` are treated as external dependencies; only the integration touchpoints are described, with citations to the local dependency repos where needed.

---

## Highlights / Key Takeaways

1. **Two-tier DI is the core wiring pattern:** an app-level `SampleApp.Registry` (`SwiftIoC.IoC`) resolves `Relux` and top-level modules, while each module often builds a private IoC container for its own dependencies. Sources: `relux_sample/IoC/IoC.swift:10-37`, `relux_sample/Modules/Notes/Notes+Module.swift:11-21`, `Packages/Auth/Sources/AuthReluxImpl/Auth+Module.swift:21-35`.
2. **Bootstrap is async-first via `Relux.Resolver`:** the app shows a splash while resolving `Relux`, then injects store UI states into the SwiftUI environment. Sources: `relux_sample/App.swift:22-33`, `.temp/repos/swiftui-relux/Sources/View+ReluxResolver.swift:64-75`, `.temp/repos/swiftui-relux/Sources/EnvironmentObject+ViewState.swift:10-18`.
3. **Navigation is “Redux-style”:** `AppRouter` is a `Relux.Navigation.ProjectingRouter<AppPage>` registered as a `HybridState`, and navigation changes are dispatched as actions (`AppRouter.Action.*`). Sources: `relux_sample/Modules/Navigation/Navigation+Module.swift:7-12`, `relux_sample/Modules/Navigation/Navigation+Module.swift:49-51`, `relux_sample/Utils/UI/Relux+NavLink.swift:51-55`.
4. **Auth demonstrates the “6-products-per-domain” modular architecture** (Models / ReluxInt / ReluxImpl / ServiceInt / ServiceImpl / TestSupport) with **dynamic products** and the **self-reference trick** to keep dynamic linkage stable. Sources: `Packages/Auth/Package.swift:10-25`, `Docs/Patterns/RELUX_MODULAR.md:17-31`.
5. **Domain-to-app navigation is decoupled using a router protocol + adapter:** the Auth package defines `Auth.Business.IRouter`, and the app provides `AuthRouterAdapter` that returns `AppRouter.Action` values (no app imports inside the domain package). Sources: `Packages/Auth/Sources/AuthReluxInt/Business/Auth+Business+Protocols.swift:14-18`, `relux_sample/Adapters/AuthRouterAdapter.swift:4-11`.
6. **Cross-domain coordination is implemented in practice via effects:** Notes failures dispatch `ErrorHandling.Business.Effect.track`, and logout triggers app-wide state cleanup while excluding `AppRouter` to preserve navigation stack. Sources: `relux_sample/Modules/Notes/Business/Middleware/Notes+Business+Flow.swift:51-55`, `relux_sample/Modules/App/Business/SampleApp+Business+Saga.swift:41-46`.
7. **Container/View separation is consistently used** (Containers own dispatch; Views are pure props/actions). This follows the project guide pattern and is visible across Notes + Account + App UI. Sources: `PROJECT_GUIDE.md:130-142`, `relux_sample/Modules/Notes/UI/List/Notes+UI+List+Container.swift:21-24`, `relux_sample/Modules/Notes/UI/List/Page/Notes+UI+ List+Container+Page.swift:6-23`.
8. **Testing is layered and mostly follows the docs**: Notes has Swift Testing suites for reducer and flow logic + UIState pipelines; shared test helpers exist both in `Packages/TestInfrastructure` and in the app test target. Sources: `Docs/Patterns/TESTING_STRATEGY.md:1-33`, `relux_sampleTests/Notes/NotesTests+Namespace.swift:4-19`, `Packages/TestInfrastructure/Sources/TestInfrastructure/Helpers/ReluxTestingExtensions.swift:1-32`.
9. **There are a few “paper cuts” that a CLI could auto-detect/fix:** (a) root `Package.swift` appears unrelated to the sample and references a missing `Sources/` folder; (b) Auth package test imports look outdated; (c) a couple Notes UI page filenames contain spaces. Sources: `Package.swift:24-33`, `Packages/Auth/Tests/AuthTests/AuthBasicsTests.swift:1-4`, file list under `relux_sample/Modules/Notes/UI/**`.

---

# L1 — Recon (Surface Scan)

## Repo Structure (Top-Level)

```
relux-sample/
├── Docs/Patterns/                       # architecture pattern docs
├── Packages/
│   ├── Auth/                            # domain package (6 products)
│   ├── AuthUI/                          # UI package (API + impl)
│   └── TestInfrastructure/              # shared test utilities
├── relux_sample/                        # app target sources (modules, DI, adapters, utils)
├── relux_sampleTests/                   # Swift Testing-based tests for app modules
├── relux_sample.xcodeproj/              # Xcode project (links/embeds dynamic products)
├── TestsSupport/                        # test-support helpers (also referenced by root Package.swift)
├── PROJECT_GUIDE.md / README.md         # entry docs
└── Package.swift                        # (appears to be for `swift-httpclient`, see notes)
```

Sources: directory listing (see `PROJECT_GUIDE.md:58-84` for intended structure; note some planned packages are not present).

## File Counts (excluding `.git/`)

Computed from `find . -type f -not -path './.git/*'`:

| Metric | Value |
|---|---:|
| Total files | 164 |
| `swift` | 145 |
| `md` | 9 |
| `json` | 4 |
| `pbxproj` | 1 |
| `xcscheme` | 1 |
| `xcworkspacedata` | 1 |
| `entitlements` | 1 |

Swift LOC (all `.swift` files): **5147** lines (scripted scan; see Appendix).

## Project Configuration

### Xcode Project (primary “build system” here)

The sample is primarily an Xcode project (`relux_sample.xcodeproj`) with:

- **Local SwiftPM packages:** `Packages/Auth`, `Packages/AuthUI` (embedded via `XCLocalSwiftPackageReference`). Source: `relux_sample.xcodeproj/project.pbxproj:591-599`.
- **Remote SwiftPM packages:** `swift-log` and `swift-ioc` (remote package refs). Source: `relux_sample.xcodeproj/project.pbxproj:602-618`.
- **Local dev dependency:** `swiftui-relux` referenced as a file reference at `../swiftui-relux` (likely for local iteration). Source: `relux_sample.xcodeproj/project.pbxproj:66-70`.
- **Dynamic products are explicitly embedded:** Auth’s dynamic libraries are added to “Embed Frameworks”. Source: `relux_sample.xcodeproj/project.pbxproj:48-63`.

### SwiftPM (secondary, package-local)

The repo also contains multiple `Package.swift` files:

- Root `Package.swift` defines `swift-httpclient` products (`HttpClient`, `HttpClientTestSupport`), but the referenced `Sources/` folder is missing in this repo snapshot. Source: `Package.swift:4-33`.
- Domain package definitions:
  - `Packages/Auth/Package.swift` (6 products, dynamic linkage; depends on `swift-relux`, `swift-ioc`). Source: `Packages/Auth/Package.swift:10-25`.
  - `Packages/AuthUI/Package.swift` (depends on local Auth + remote `swiftui-relux`). Source: `Packages/AuthUI/Package.swift:14-17`.
  - `Packages/TestInfrastructure/Package.swift` (depends on `swift-relux`). Source: `Packages/TestInfrastructure/Package.swift:13-21`.

---

# L2 — Deep Dive (Read All Sources)

## Documentation & “Ground Truth” Patterns

The repo’s own docs define the intended architecture and conventions:

- `README.md` summarizes the modular products/layers and positions Relux as async-first UDF for SwiftUI. Source: `README.md:12-39`.
- `PROJECT_GUIDE.md` defines:
  - dependency direction rules (UI → interfaces → implementations → models),
  - state categories (BusinessState, UIState, HybridState),
  - Container/View separation rules (dispatch in containers only),
  - naming + file structure conventions. Source: `PROJECT_GUIDE.md:90-123`, `PROJECT_GUIDE.md:126-166`, `PROJECT_GUIDE.md:200-228`.
- `Docs/Patterns/*` provides detailed templates:
  - modular architecture and dynamic products (`RELUX_MODULAR.md`). Source: `Docs/Patterns/RELUX_MODULAR.md:17-31`.
  - orchestrators pattern (`RELUX_ORCHESTRATION.md`) — **described but not implemented as a package in this repo snapshot**.
  - flow vs saga (`RELUX_FLOW_VS_SAGA.md`) — reflected directly in Auth (Saga) vs Notes (Flow).
  - layered testing (`TESTING_STRATEGY.md`) + test infra (`TEST_INFRASTRUCTURE.md`) + per-domain test support (`DOMAIN_TEST_SUPPORT.md`).

## App Bootstrap & Wiring

### 1) Startup sequence (SwiftUI → Relux resolution)

1. `SampleApp.init()` calls `Registry.configure()` to register resolvers in the app IoC container. Source: `relux_sample/App.swift:17-20`, `relux_sample/IoC/IoC.swift:21-37`.
2. The root scene uses `Relux.Resolver(splash:content:resolver:)`:
   - shows splash while `resolver()` runs,
   - once resolved, calls `content(relux)` and injects UI states into SwiftUI env via `.passingObservableToEnvironment(fromStore:)`. Sources: `relux_sample/App.swift:22-33`, `.temp/repos/swiftui-relux/Sources/View+ReluxResolver.swift:64-75`, `.temp/repos/swiftui-relux/Sources/EnvironmentObject+ViewState.swift:10-18`.
3. After Relux is ready, the root container runs `setupAppContext()` to seed initial navigation + auth state. Source: `relux_sample/App.swift:41-58`.

### 2) Relux initialization + module registration order

`Registry.buildRelux()` creates the Relux instance and registers modules in a deliberate order. Source: `relux_sample/IoC/IoC.swift:42-55`.

Order:

1. `ErrorHandling.Module`
2. `Navigation.Module` (registers routers as UI state)
3. `SampleApp.Module` (orchestrator saga; no state)
4. `Auth.Module` (domain state + saga; from Packages/Auth)
5. `Notes.Module` (async init; registers BusinessState + UIState + Flow)

Sources: `relux_sample/IoC/IoC.swift:48-54`.

### 3) Initial app context

`setupAppContext()` dispatches concurrently:

- `SampleApp.Business.Effect.setAppContext` → sets `AppRouter` stack to auth local-auth page.
- `Auth.Business.Effect.obtainAvailableBiometryType` → populates auth state with available biometry type.

Sources: `relux_sample/App.swift:55-58`, `relux_sample/Modules/App/Business/SampleApp+Business+Saga.swift:35-38`, `Packages/Auth/Sources/AuthReluxImpl/Business/Middleware/Auth+Business+Saga.swift:22-31`.

## Dependency Injection Pattern (SwiftIoC)

### App-level IoC (“composition root”)

`SampleApp.Registry.ioc` is a global container used to:

- build the `Relux` instance (`Relux.init(...).register { ... }`),
- provide the Relux `Store` and logger,
- provide modules, routers, and UI providers.

Sources: `relux_sample/IoC/IoC.swift:14-37`, `relux_sample/IoC/IoC.swift:42-101`.

Notable details:

- `SampleApp.Registry.resolveAsync` and friends use `ioc.getAsync(by:)` / `ioc.get(by:)`. Source: `relux_sample/IoC/IoC.swift:105-120`.
- `SampleApp.relux` uses `waitForResolve(Relux.self)` which is a SwiftIoC helper that spins until the async instance is available. Sources: `relux_sample/App.swift:13-15`, `.temp/repos/swift-ioc/Sources/IoC.swift:140-146`.

### Module-level IoC (“local composition”)

Each module typically defines its own `IoC` container, registers only what it needs, and then exposes:

- `states: [any Relux.AnyState]`
- `sagas: [any Relux.Saga]`

Examples:

- `Navigation.Module`: router + modal router are local IoC singletons. Source: `relux_sample/Modules/Navigation/Navigation+Module.swift:23-35`.
- `ErrorHandling.Module`: saga + service. Source: `relux_sample/Modules/ErrorHandling/ErrorHandling+Module.swift:23-41`.
- `Notes.Module`: business state, UI state, service, fetcher, and Flow (async). Source: `relux_sample/Modules/Notes/Notes+Module.swift:11-21`, `relux_sample/Modules/Notes/Notes+Module.swift:25-62`.
- `Auth.Module` (package): state + saga, router injected, service created via factory. Source: `Packages/Auth/Sources/AuthReluxImpl/Auth+Module.swift:21-35`, `Packages/Auth/Sources/AuthReluxImpl/Auth+Module.swift:39-67`.

## Navigation (Router-as-State)

### AppRouter (stack navigation)

`AppRouter` is a specialized Relux router state:

- `typealias AppRouter = Relux.Navigation.ProjectingRouter<AppPage>`
- registered as a `Navigation.Business.IRouter` (`HybridState`)
- initialized with `pages: [.splash]`.

Sources: `relux_sample/Modules/Navigation/Navigation+Module.swift:7-12`, `relux_sample/Modules/Navigation/Navigation+Module.swift:49-51`.

Routes are defined by `AppPage`:

- `.splash`
- `.auth(page: Auth.UI.Model.Page = .logoutFlow)`
- `.app(page: SampleApp.UI.Main.Model.Page = .main)`

Source: `relux_sample/Modules/Navigation/UI/Models/Navigation+UI+Model+AppPage.swift:4-9`.

### ModalRouter (sheet navigation)

`ModalRouter` is a simple `@Observable` `HybridState` with a single `modalSheet` slot, plus actions `.present`/`.dismiss`. Sources:

- Definition: `relux_sample/Modules/Navigation/Business/Navigation+Business+ModalRouter.swift:1-5`
- Actions: `relux_sample/Modules/Navigation/Business/Navigation+Business+ModalRouter+Action.swift`
- Reducer: `relux_sample/Modules/Navigation/Business/Navigation+Business+ModalRouter+Reducer.swift`

In the root container, modal presentation is centralized:

- `.sheet(item: modalRouter.binding.modalSheet, ...)`
- modal content explicitly reinjects store UI states into the environment (`passingObservableToEnvironment`) because modals are outside the main navigation stack hierarchy.

Source: `relux_sample/Modules/App/UI/Root/App+UI+Root+Container.swift:14-18`, `relux_sample/Modules/App/UI/Root/App+UI+Root+Container.swift:44-52`.

## UI Composition Pattern (Containers + Views)

The app follows the guide’s Container/View separation:

- Containers conform to `Relux.UI.Container` and are responsible for:
  - reading UI state from `@EnvironmentObject`/`@Environment`,
  - dispatching actions/effects,
  - mapping state → props,
  - building view callbacks.
- Views conform to `Relux.UI.View` and consume `Props` and `Actions` only.

Guide: `PROJECT_GUIDE.md:130-142`.

Example (Notes list):

- Container builds `Page(props:..., actions:...)`. Source: `relux_sample/Modules/Notes/UI/List/Notes+UI+List+Container.swift:17-23`.
- Page renders props and uses callbacks. Source: `relux_sample/Modules/Notes/UI/List/Page/Notes+UI+ List+Container+Page.swift:6-23`.

## Feature Modules / Packages

### 1) Auth Domain (Packages/Auth)

Auth is the “reference implementation” of the modular architecture described in docs.

#### Products and dependencies

`Packages/Auth/Package.swift` defines products:

- `AuthModels` (dynamic)
- `AuthReluxInt` (dynamic)
- `AuthReluxImpl` (dynamic)
- `AuthServiceInt` (dynamic)
- `AuthServiceImpl` (dynamic)
- `AuthTestSupport` (static)

Source: `Packages/Auth/Package.swift:10-18`.

Self-reference trick for dynamic linkage:

- `.package(name: "Auth-Self", path: ".")`

Source: `Packages/Auth/Package.swift:19-25`.

#### Public interfaces

- Router protocol (app must implement): `Auth.Business.IRouter` with `setAuth(page:)` and `pushMain()`. Source: `Packages/Auth/Sources/AuthReluxInt/Business/Auth+Business+Protocols.swift:14-18`.
- Effects: `checkAuthContext`, `obtainAvailableBiometryType`, `authorizeWithBiometry`, `logout`, `runLogoutFlow`. Source: `Packages/Auth/Sources/AuthReluxInt/Business/Middleware/Auth+Business+Effect.swift`.
- Actions: `authSucceed`, `authFailed`, `obtainAvailableBiometryTypeSucceed`, `logOutSucceed`. Source: `Packages/Auth/Sources/AuthReluxInt/Business/Auth+Business+Action.swift`.
- UI page model: `Auth.UI.Model.Page` = `.logoutFlow | .localAuth`. Source: `Packages/Auth/Sources/AuthReluxInt/UI/Model/Auth+UI+Model+Page.swift:4-7`.
- Service protocol: `Auth.Business.IService` provides `availableBiometry`, `runLocalAuth`, `recreateLAContext`. Source: `Packages/Auth/Sources/AuthServiceInt/AuthServiceInterface.swift:4-14`.

#### Implementation wiring

`Auth.Module` composes:

- State: `Auth.Business.State` (`HybridState` via `IState`)
- Saga: `Auth.Business.Saga` (`ISaga`)
- Router: injected from app (`IRouter`)
- Service: created via **factory closure** passed from the app.

Source: `Packages/Auth/Sources/AuthReluxImpl/Auth+Module.swift:21-35`, `Packages/Auth/Sources/AuthReluxImpl/Auth+Module.swift:39-67`.

Notable design choice: `AuthReluxImpl` depends on `AuthServiceInt` (not `AuthServiceImpl`), and the app chooses the concrete service via `serviceFactory`. This increases testability and keeps the Relux implementation product less coupled. Source: `Packages/Auth/Package.swift:52-60`, `Packages/Auth/Sources/AuthReluxImpl/Auth+Module.swift:21-24`.

#### App adapter (integration point)

The app provides `AuthRouterAdapter` translating Auth navigation needs into `AppRouter.Action`:

- `setAuth(page:)` → `AppRouter.Action.set([.auth(page: page)])`
- `pushMain()` → `AppRouter.Action.push(.app(page: .main))`

Source: `relux_sample/Adapters/AuthRouterAdapter.swift:4-11`.

#### Auth UI integration

The app uses `AuthUIProviding` (from AuthUIAPI) so that the app’s root router can render auth pages without importing concrete views. Source: `relux_sample/Modules/App/UI/Root/App+UI+Root+Router.swift:10-15`.

#### Tests and test support

- `AuthTestSupport` provides a `ServiceMock` and testable action wrappers. Source: `Packages/Auth/Sources/AuthTestSupport/Mocks/Auth+ServiceMock.swift`, `Packages/Auth/Sources/AuthTestSupport/Assertions/Auth+TestHelpers.swift`.
- The `AuthTests` target currently uses XCTest and references an apparently outdated module name (`AuthImplementation`). Source: `Packages/Auth/Tests/AuthTests/AuthBasicsTests.swift:1-4`.

This mismatch is a good candidate for `relux-manager doctor` checks (see Recommendations).

---

### 2) Auth UI Package (Packages/AuthUI)

Auth UI is split into:

- `AuthUIAPI` product: `AuthUIProviding` protocol. Source: `Packages/AuthUI/Sources/AuthUIAPI/AuthUIProviding.swift:5-8`.
- `AuthUI` product: `AuthUIRouter` which maps `Auth.UI.Model.Page` to concrete containers (`Auth.UI.Initial.Container` / `Auth.UI.LocalAuth.Container`). Source: `Packages/AuthUI/Sources/AuthUI/UI/Auth+UI+Router.swift:5-16`.

The concrete containers dispatch domain effects:

- Logout flow container triggers `Auth.Business.Effect.runLogoutFlow` on task. Source: `Packages/AuthUI/Sources/AuthUI/UI/LogoutFlow/Auth+UI+LogoutFlow+Container.swift`.
- Local auth container triggers `Auth.Business.Effect.authorizeWithBiometry`. Source: `Packages/AuthUI/Sources/AuthUI/UI/LocalAuth/Auth+UI+LocalAuth+Container.swift`.

Dependencies:

- Depends on local Auth package and remote `swiftui-relux`. Source: `Packages/AuthUI/Package.swift:14-17`.

---

### 3) Notes Domain (in-app module)

Notes is implemented as an app module (not extracted into `Packages/` yet), but follows the same conceptual layering:

#### Module wiring

`Notes.Module` registers:

- `Notes.Business.State` (BusinessState)
- `Notes.UI.State` (UIState derived from BusinessState via Combine pipelines)
- `Notes.Business.Service` (uses `Notes.Data.Api.Fetcher`)
- `Notes.Business.Flow` (Relux.Flow) as the saga implementation

Source: `relux_sample/Modules/Notes/Notes+Module.swift:11-21`, `relux_sample/Modules/Notes/Notes+Module.swift:25-62`.

#### State model

- BusinessState: actor, owns truth as `@Published var notes: MaybeData<[Note], Err>`. Source: `relux_sample/Modules/Notes/Business/Notes+Business+State.swift`.
- UIState: `ObservableObject` mapping business notes into:
  - grouped-by-day list (`notesGroupedByDay`)
  - dict-by-id (`notes`)

Source: `relux_sample/Modules/Notes/UI/Notes+UI+State.swift:6-13`, `relux_sample/Modules/Notes/UI/Notes+UI+State.swift:28-38`.

#### Side effects: Flow (returns `Relux.Flow.Result`)

The Notes effect handler is a Flow:

- `obtainNotes`: on failure, dispatches both a Notes failure action and `ErrorHandling.Business.Effect.track`, but returns `.success` (the flow doesn’t fail in this branch). Source: `relux_sample/Modules/Notes/Business/Middleware/Notes+Business+Flow.swift:40-56`.
- `upsert` and `delete`: on failure, dispatch Notes failure + track error and return `.failure(err)`. Source: `relux_sample/Modules/Notes/Business/Middleware/Notes+Business+Flow.swift:59-90`.

This directly demonstrates the “Flow vs Saga” doc: UI can await results when needed. Source: `Docs/Patterns/RELUX_FLOW_VS_SAGA.md:1-43`.

#### UI: routes and screens

Notes defines its own UI pages and routes:

- `Notes.UI.Model.Page` = list/details/create/edit. Source: `relux_sample/Modules/Notes/UI/Model/Notes+UI+Page.swift:1-7`.
- Router switches on page and instantiates containers. Source: `relux_sample/Modules/Notes/UI/Notes+UI+Router.swift:5-12`.

Notable flow integration: Create screen awaits the flow result and closes only on success. Source: `relux_sample/Modules/Notes/UI/Create/Notes+UI+Create+Container.swift:30-35`.

---

### 4) App “Orchestrator” Module (SampleApp)

`SampleApp.Module` contributes only sagas (no state). Source: `relux_sample/Modules/App/App+Module.swift:14-25`.

The saga:

- Sets initial navigation context (`AppRouter.Action.set([.auth(page: .localAuth)])`). Source: `relux_sample/Modules/App/Business/SampleApp+Business+Saga.swift:35-38`.
- Listens to Auth logout flow (`Auth.Business.Effect.runLogoutFlow`) and cleans up store state while excluding `AppRouter` so the nav stack survives logout. Source: `relux_sample/Modules/App/Business/SampleApp+Business+Saga.swift:27-30`, `relux_sample/Modules/App/Business/SampleApp+Business+Saga.swift:41-46`.

This is an “orchestrator saga” at app scope (cross-domain coordination). Source: `PROJECT_GUIDE.md:112-115`.

---

### 5) Error Handling Module

`ErrorHandling.Module` provides a saga that receives `ErrorHandling.Business.Effect.track(error:)` and delegates to a service (currently prints as placeholder). Sources:

- Effect: `relux_sample/Modules/ErrorHandling/Business/Middleware/ErrorHandling+Business+Effect.swift:1-4`
- Module wiring: `relux_sample/Modules/ErrorHandling/ErrorHandling+Module.swift:23-41`
- Saga: `relux_sample/Modules/ErrorHandling/Business/Middleware/ErrorHandling+Business+Saga.swift:17-29`
- Service: `relux_sample/Modules/ErrorHandling/Business/Middleware/ErrorHandling+Business+Service.swift`

This module is used by Notes Flow on failures. Source: `relux_sample/Modules/Notes/Business/Middleware/Notes+Business+Flow.swift:51-55`.

---

### 6) Account UI Module

Account screen is UI-only; it dispatches:

- Logout: `Auth.Business.Effect.logout`
- Debug: `ModalRouter.Action.present(page: .debug)`

Source: `relux_sample/Modules/Account/UI/Account+UI+Container.swift`.

---

### 7) Logger (Relux.Logger implementation)

The app provides a custom Relux logger using `swift-log` (`Logging.Logger`) for DEBUG builds. It logs:

- action name + associated values,
- call site (`fileID`, `functionName`, `lineNumber`),
- execution duration,
- action result.

Source: `relux_sample/Modules/Logger/Logger.swift:32-62`.

## Shared Utilities (Notable Patterns)

- `MaybeData<T,E>` is the “loading state” sum type used across Notes UI. Source: `relux_sample/Utils/MaybeData.swift`.
- `@retroactive @unchecked Sendable` is used to mark common Combine publishers as sendable (strict concurrency escape hatch). Source: `relux_sample/Utils/Combine+Sendable.swift:9-12`.
- `Relux.NavigationLink(page:)` is implemented as an `AsyncButton` that dispatches `AppRouter.Action.push(page)`, avoiding double-taps during animations. Source: `relux_sample/Utils/UI/Relux+NavLink.swift:16-57`.

## Testing (What’s Implemented)

### App tests (`relux_sampleTests/`)

- Uses the Swift Testing framework (`@Suite`, `@Test`). Source: `relux_sampleTests/Notes/NotesTests+Namespace.swift:4-19`.
- Notes coverage includes:
  - Flow behavior (dispatches actions + error tracking on failures). Source: `relux_sampleTests/Notes/Business/Saga/NotesTests+Business+Saga+Obtain.swift`.
  - BusinessState reducer updates. Source: `relux_sampleTests/Notes/Business/State/NotesTests+Business+State+Obtain.swift`.
  - UIState pipeline correctness with async publisher bridging and timeouts. Source: `relux_sampleTests/Notes/UI/State/NotesTests+UI+State.swift:24-104`.

### Shared test utilities (`Packages/TestInfrastructure/`)

`TestInfrastructure` includes:

- Logger find helpers for actions/effects. Source: `Packages/TestInfrastructure/Sources/TestInfrastructure/Helpers/ReluxTestingExtensions.swift:1-32`.
- Timeout helper. Source: `Packages/TestInfrastructure/Sources/TestInfrastructure/Helpers/AsyncTestHelpers.swift`.
- Optional HttpClient mocks behind `#if canImport(HttpClient)`. Source: `Packages/TestInfrastructure/Sources/TestInfrastructure/DomainMocks/PublishedWSClientMock.swift:1-4`.

### Domain-specific test support (`AuthTestSupport`)

Auth supplies:

- `ServiceMock` (call tracking + handlers). Source: `Packages/Auth/Sources/AuthTestSupport/Mocks/Auth+ServiceMock.swift`.
- testable action wrapper for stable assertions. Source: `Packages/Auth/Sources/AuthTestSupport/Assertions/Auth+TestHelpers.swift:6-29`.

---

## Public API Surface (Catalog)

This section lists the **intended consumption surface** between modules/packages.

### App target (`relux_sample`)

- Reexports for convenience:
  - `@_exported import Relux`
  - `@_exported import ReluxRouter`
  - `@_exported import SwiftUIRelux`
  Source: `relux_sample/App.swift:1-3`.

### `AuthModels` (Packages/Auth)

- `public enum Auth { ... }` namespace. Source: `Packages/Auth/Sources/AuthModels/Auth+Namespace.swift:2-11`.
- `Auth.Business.Err` (public). Source: `Packages/Auth/Sources/AuthModels/Auth+Business+Err.swift`.
- `Auth.Business.Model.BiometryType` (public). Source: `Packages/Auth/Sources/AuthModels/Auth+Business+Model+BiometryType.swift`.

### `AuthReluxInt` (Packages/Auth)

- `Auth.Business.Action` (public). Source: `Packages/Auth/Sources/AuthReluxInt/Business/Auth+Business+Action.swift`.
- `Auth.Business.Effect` (public). Source: `Packages/Auth/Sources/AuthReluxInt/Business/Middleware/Auth+Business+Effect.swift`.
- Protocols:
  - `Auth.Business.IState: Relux.HybridState`
  - `Auth.Business.ISaga: Relux.Saga`
  - `Auth.Business.IRouter` (domain navigation seam)
  Source: `Packages/Auth/Sources/AuthReluxInt/Business/Auth+Business+Protocols.swift:4-18`.
- `Auth.UI.Model.Page` (public). Source: `Packages/Auth/Sources/AuthReluxInt/UI/Model/Auth+UI+Model+Page.swift:4-7`.
- Reexport convenience: `@_exported import AuthModels`. Source: `Packages/Auth/Sources/AuthReluxInt/Reexports.swift:2-6`.

### `AuthServiceInt` (Packages/Auth)

- `Auth.Business.IService` protocol (public). Source: `Packages/Auth/Sources/AuthServiceInt/AuthServiceInterface.swift:4-14`.

### `AuthServiceImpl` (Packages/Auth)

- `Auth.Business.Service` actor (public). Source: `Packages/Auth/Sources/AuthServiceImpl/Auth+Business+Service.swift:5-13`.

### `AuthReluxImpl` (Packages/Auth)

- `Auth.Module` (public) composing state+saga using `SwiftIoC`. Source: `Packages/Auth/Sources/AuthReluxImpl/Auth+Module.swift:12-35`.
- `Auth.Business.State` (public) implements `IState`. Source: `Packages/Auth/Sources/AuthReluxImpl/Business/Auth+Business+State.swift`.
- `Auth.Business.Saga` (public) implements `ISaga`. Source: `Packages/Auth/Sources/AuthReluxImpl/Business/Middleware/Auth+Business+Saga.swift`.

### `AuthTestSupport` (Packages/Auth)

- `Auth.Business.ServiceMock` (public). Source: `Packages/Auth/Sources/AuthTestSupport/Mocks/Auth+ServiceMock.swift:5-33`.
- `Auth.Business.Action.TestableAction` + logger helper. Source: `Packages/Auth/Sources/AuthTestSupport/Assertions/Auth+TestHelpers.swift:6-39`.

### `AuthUIAPI` (Packages/AuthUI)

- `public protocol AuthUIProviding`. Source: `Packages/AuthUI/Sources/AuthUIAPI/AuthUIProviding.swift:5-8`.

### `AuthUI` (Packages/AuthUI)

- `public struct AuthUIRouter: AuthUIProviding`. Source: `Packages/AuthUI/Sources/AuthUI/UI/Auth+UI+Router.swift:5-16`.

### `TestInfrastructure` (Packages/TestInfrastructure)

- `withTimeout(seconds:operation:)` + `TimeoutError`. Source: `Packages/TestInfrastructure/Sources/TestInfrastructure/Helpers/AsyncTestHelpers.swift`.
- `Relux.Testing.Logger.findAction/findEffect/assertDispatched`. Source: `Packages/TestInfrastructure/Sources/TestInfrastructure/Helpers/ReluxTestingExtensions.swift:1-32`.
- `JSONFixtures` helper. Source: `Packages/TestInfrastructure/Sources/TestInfrastructure/Stubs/JSONFixtures.swift`.
- `StubError`. Source: `Packages/TestInfrastructure/Sources/TestInfrastructure/Stubs/StubError.swift`.
- Optional HttpClient mocks: `PublishedWSClientMock`, `WSClientMock`, `RpcAsyncClientMock` (only when `HttpClient` is available). Source: `Packages/TestInfrastructure/Sources/TestInfrastructure/DomainMocks/PublishedWSClientMock.swift:1-4`.

---

# L3 — Domain Synthesis (Reduce)

## Architectural Domains

### 1) Composition / Bootstrap Domain

Responsibilities:

- Provide a single source of truth (`Relux.Store`) and register modules in correct order.
- Use async resolution so SwiftUI can render splash while DI completes.

Key artifacts:

- `SampleApp.Registry` (app-level IoC). Source: `relux_sample/IoC/IoC.swift:14-37`.
- `Relux.Resolver` / `passingObservableToEnvironment`. Sources: `.temp/repos/swiftui-relux/Sources/View+ReluxResolver.swift:64-75`, `.temp/repos/swiftui-relux/Sources/EnvironmentObject+ViewState.swift:10-18`.

### 2) Navigation Domain

Responsibilities:

- Own the navigation stack state (`AppRouter`) and modal state (`ModalRouter`).
- Provide typed route enums (`AppPage`, plus nested routes like `Notes.UI.Model.Page`).
- Allow any module/UI to navigate by dispatching actions.

Key artifacts:

- `Navigation.Module` registers both router states. Source: `relux_sample/Modules/Navigation/Navigation+Module.swift:23-35`.
- `Relux.NavigationLink` wrapper to dispatch push actions. Source: `relux_sample/Utils/UI/Relux+NavLink.swift:51-55`.

### 3) Auth Domain (packaged)

Responsibilities:

- Own auth business state, effects, and side effects (biometry auth).
- Declare navigation needs via `IRouter` protocol (no app deps).
- Allow UI implementations to be swapped via AuthUIAPI + AuthUI.

Key integration points:

- Router adapter in app. Source: `relux_sample/Adapters/AuthRouterAdapter.swift:4-11`.
- AuthUIProviding resolved from app IoC. Source: `relux_sample/Modules/App/UI/Root/App+UI+Root+Router.swift:10-15`.

### 4) Notes Domain (in-app)

Responsibilities:

- Demonstrate business state + UI aggregation + Flow result patterns.
- Demonstrate error tracking side effects (`ErrorHandling`).

Key integration points:

- Error tracking dispatch on failure. Source: `relux_sample/Modules/Notes/Business/Middleware/Notes+Business+Flow.swift:51-55`.
- UI uses flow result to decide navigation (create). Source: `relux_sample/Modules/Notes/UI/Create/Notes+UI+Create+Container.swift:30-35`.

### 5) Cross-cutting Concerns Domain

- ErrorHandling: `Effect.track(error:)` + saga + service. Source: `relux_sample/Modules/ErrorHandling/Business/Middleware/ErrorHandling+Business+Effect.swift:1-4`, `relux_sample/Modules/ErrorHandling/Business/Middleware/ErrorHandling+Business+Saga.swift:17-29`.
- Logging: Relux.Logger implementation. Source: `relux_sample/Modules/Logger/Logger.swift:32-62`.
- Testing infra: shared helper package + test target helpers. Sources: `Packages/TestInfrastructure/**`, `relux_sampleTests/**`.

## Cross-domain Interaction Graph (Observed)

```
Notes.Flow (failure) ──dispatches──▶ ErrorHandling.Effect.track(error:)

Auth.Saga .runLogoutFlow ──(effect observed by)──▶ SampleApp.Saga
SampleApp.Saga ──calls──▶ store.cleanup(excluding: [AppRouter])
```

Sources: `relux_sample/Modules/Notes/Business/Middleware/Notes+Business+Flow.swift:51-55`, `relux_sample/Modules/App/Business/SampleApp+Business+Saga.swift:27-30`, `relux_sample/Modules/App/Business/SampleApp+Business+Saga.swift:41-46`.

---

# L4 — Product Synthesis (Product-level)

## Executive Summary

`relux-sample` is a reference SwiftUI app meant to demonstrate:

- **Relux-based unidirectional data flow** using Swift Concurrency (actors for state/services and async effect handlers),
- **navigation as state** (router states manipulated via actions),
- **strict modularization by domain** (Auth is fully extracted and follows a multi-product template),
- **async-first DI** (SwiftIoC), and
- **layered testing** (flow/reducer/UIState tested in isolation).

Key “how it all wires together” answer:

- Xcode project links local domain packages (Auth/AuthUI/TestInfrastructure) + external Relux/SwiftUIRelux/SwiftIoC/logging.
- `SampleApp.Registry` composes everything, builds `Relux`, and registers modules.
- Modules provide state objects and saga/flow handlers; SwiftUI gets states via environment injection, and screens are composed by Container/View boundaries.

## Architecture Overview (Wiring Diagram)

```
SwiftUI App
  SampleApp (App.swift)
    └─ Relux.Resolver(splash/content/resolver)
         └─ resolves Relux via SwiftIoC (SampleApp.Registry)

Relux instance
  Store + Logger
  Registered Modules (order)
    1) ErrorHandling.Module  → Saga(track error) + Service
    2) Navigation.Module     → States: AppRouter, ModalRouter
    3) SampleApp.Module      → Saga(set context, cleanup on logout)
    4) Auth.Module (package) → State + Saga + injected Router + ServiceFactory
    5) Notes.Module          → BusinessState + UIState + Flow + Service + Fetcher

SwiftUI environment
  .passingObservableToEnvironment(fromStore:)
    → provides UI states to Containers via @EnvironmentObject/@Environment
```

Sources: `relux_sample/App.swift:22-33`, `relux_sample/IoC/IoC.swift:42-55`, `.temp/repos/swiftui-relux/Sources/EnvironmentObject+ViewState.swift:10-18`.

## Key Patterns & Conventions (as Implemented)

1. **Namespacing via nested `enum` trees** (`SampleApp.UI.Root`, `Notes.UI.List`, `Auth.Business`). Source: `PROJECT_GUIDE.md:215-228`, `relux_sample/Modules/Notes/Notes+Namespace.swift:1-25`.
2. **Container/View split** with Props + Actions and `ViewCallback` wrappers. Source: `PROJECT_GUIDE.md:130-142`, `relux_sample/Modules/App/Relux+UI.swift`.
3. **Router protocol pattern** (domain declares protocol; app adapts to its own router). Source: `Packages/Auth/Sources/AuthReluxInt/Business/Auth+Business+Protocols.swift:14-18`, `relux_sample/Adapters/AuthRouterAdapter.swift:4-11`.
4. **Flow vs Saga** used as intended:
   - Auth uses Saga (fire-and-forget).
   - Notes uses Flow (returns Result; UI can await it). Sources: `Packages/Auth/Sources/AuthReluxImpl/Business/Middleware/Auth+Business+Saga.swift:21-27`, `relux_sample/Modules/Notes/Business/Middleware/Notes+Business+Flow.swift:28-36`, `relux_sample/Modules/Notes/UI/Create/Notes+UI+Create+Container.swift:30-35`.
5. **State classification**:
   - BusinessState = actor + reducer + cleanup.
   - UIState = ObservableObject mapping pipelines from BusinessState. Sources: `PROJECT_GUIDE.md:104-109`, `relux_sample/Modules/Notes/Business/Notes+Business+State.swift`, `relux_sample/Modules/Notes/UI/Notes+UI+State.swift:28-38`.

## Integration Points with Other Relux Packages

- **`swift-relux`** (Relux core): provides `Relux`, `Relux.Store`, `Relux.Saga`, `Relux.Flow`, router primitives, and testing logger used in app tests. Used across nearly all modules (see import distribution in Appendix).
- **`swiftui-relux`** (SwiftUI integration): provides `Relux.Resolver` and environment injection helpers. Sources: `.temp/repos/swiftui-relux/Sources/View+ReluxResolver.swift:35-77`, `.temp/repos/swiftui-relux/Sources/EnvironmentObject+ViewState.swift:5-18`.
- **`swift-ioc`** (DI): provides `IoC` container and async resolution helpers (`getAsync`, `waitForResolve`). Sources: `.temp/repos/swift-ioc/Sources/IoC.swift:140-146`, plus app usage in `relux_sample/IoC/IoC.swift:14-37`.
- **`swift-log`**: app logger uses `Logging.Logger` behind `Relux.Logger` implementation. Sources: `relux_sample.xcodeproj/project.pbxproj:602-610`, `relux_sample/Modules/Logger/Logger.swift:1-3`.
- **`HttpClient`**: referenced only in test mocks; TestInfrastructure provides optional mocks guarded by `#if canImport(HttpClient)`. Source: `Packages/TestInfrastructure/Sources/TestInfrastructure/DomainMocks/PublishedWSClientMock.swift:1-4`.

## Recommendations for `relux-manager` CLI

Based on the patterns/gaps observed, a CLI component would be high leverage if it can:

1. **Scaffold a new domain package** using the 6-product template (Models / ReluxInt / ReluxImpl / ServiceInt / ServiceImpl / TestSupport), including:
   - self-reference trick (`<Domain>-Self`),
   - dynamic products configuration,
   - initial router protocol and adapter template,
   - default module IoC wiring.
   Sources/templates: `Docs/Patterns/RELUX_MODULAR.md:17-31`, `Packages/Auth/Package.swift:10-25`, `Packages/Auth/Sources/AuthReluxImpl/Auth+Module.swift:39-67`.
2. **Scaffold a UI package (`<Domain>UI`)** split into API + implementation (`<Domain>UIAPI` + `<Domain>UI`), mirroring AuthUI. Sources: `Packages/AuthUI/Package.swift:10-33`, `Packages/AuthUI/Sources/AuthUIAPI/AuthUIProviding.swift:5-8`.
3. **Generate “module registration” snippets** for app composition root, with ordering guidance (infra → navigation → app/orchestrators → domains → orchestrators). Source: `relux_sample/IoC/IoC.swift:48-54`, plus guidance in `Docs/Patterns/RELUX_ORCHESTRATION.md:170-186`.
4. **Add a `doctor` command** that validates:
   - all dynamic products are linked and embedded in the Xcode project (Embed Frameworks) — Auth does this. Source: `relux_sample.xcodeproj/project.pbxproj:48-63`.
   - `Package.swift` targets exist on disk (flag the root `Package.swift` referencing missing `Sources/`). Source: `Package.swift:24-33`.
   - file naming issues (e.g., accidental spaces in filenames) that can break tooling.
   - test targets import valid module/product names (flag `AuthImplementation`). Source: `Packages/Auth/Tests/AuthTests/AuthBasicsTests.swift:1-4`.
5. **Generate layered test skeletons** matching `TESTING_STRATEGY.md`:
   - Flow/Saga tests that assert dispatched actions/effects via logger,
   - Reducer tests,
   - UIState pipeline tests with async publisher helpers,
   - optional smoke test wiring.
   Sources: `Docs/Patterns/TESTING_STRATEGY.md:1-33`, `relux_sampleTests/Notes/Business/Saga/NotesTests+Business+Saga+Obtain.swift`, `relux_sampleTests/Notes/UI/State/NotesTests+UI+State.swift:24-104`.
6. **Codify “router protocol + adapter”** as a first-class concept (generate both sides):
   - `Domain.Business.IRouter` protocol in `*ReluxInt`
   - app adapter translating to `AppRouter.Action.*`
   Sources: `Packages/Auth/Sources/AuthReluxInt/Business/Auth+Business+Protocols.swift:14-18`, `relux_sample/Adapters/AuthRouterAdapter.swift:4-11`.

## Risks / Gaps / Cleanup Candidates (Observed)

- **Root `Package.swift` likely does not belong to this repo snapshot** (or is incomplete): `HttpClient` target path `Sources` does not exist. Source: `Package.swift:24-28`.
- **Auth package tests appear stale** (XCTest + wrong module import). Source: `Packages/Auth/Tests/AuthTests/AuthBasicsTests.swift:1-4`.
- **Notes UI has a couple filenames with spaces**, which may break scripts or conventions. (Observed in file paths under `relux_sample/Modules/Notes/UI/**`.)
- Docs mention orchestrator packages (`SessionOrchestration`, `DataOrchestration`) in `PROJECT_GUIDE.md`, but they are not present in `Packages/` in this snapshot. Source: `PROJECT_GUIDE.md:63-66`.

---

# Appendix

## Full Source Scan Evidence

- All Swift source files were read via scripted scan (`145` files; `5147` lines).
- Script output captured to `/tmp/relux-sample-scan.json` (generated locally in this environment).

## Import Distribution (Top)

From scan:

- `import SwiftUI` (37)
- `import Relux` (19)
- `import AuthReluxInt` (17)
- `import AuthModels` (16)
- `import SwiftUIRelux` (14)
- `import Testing` (13)
- `import SwiftIoC` (6)

This aligns with the “Relux + SwiftUIRelux + SwiftIoC” center of gravity described in docs. Sources: `PROJECT_GUIDE.md:30-39`, `relux_sample.xcodeproj/project.pbxproj:602-659`.

