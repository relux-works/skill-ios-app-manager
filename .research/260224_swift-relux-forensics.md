# swift-relux forensics (Relux)

- **Task:** `TASK-260224-2h4953` (forensics-swift-relux)
- **Date:** 2026-02-24
- **Target:** `.temp/repos/swift-relux/`
- **Revision:** `f32cf5be0768bd029d6afaa06fb256a1bcab8ce9` (origin `relux-works/swift-relux`)
- **Method:** 4-layer MapReduce (L1→L4)
- **Read-only:** No source files modified

## Scope

Analyze the `swift-relux` SwiftPM package (“Relux”) as a Redux/Flux-inspired, async-first state management library:

- Package structure and build configuration
- Public API surface
- Architecture and runtime flow (Store, Reducer model, Actions, Effects, Sagas/Flows)
- “Middleware” / interception patterns (or absence thereof)
- State observation model (incl. SwiftUI fit)
- Async effects model and `ActionResult` propagation
- Threading/executor model and ordering guarantees
- Tests and provided test utilities

## Acceptance Criteria (this report)

- L1 recon + L2 deep-dive + L3 synthesis + L4 product synthesis in a single document
- Key takeaways highlighted
- Claims supported with file/line citations
- Covers package structure, public API, architecture, patterns, integration points

## Highlights / key takeaways

1. **Relux is a `@MainActor` singleton composition root** that wires a `Dispatcher` actor to exactly two subscribers: `Store` and `RootSaga`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux.swift#L1`–`L24`)
2. **Dispatch is actor-isolated (`Dispatcher`) and supports serial vs concurrent execution** of multiple actions; each action is fanned out concurrently to subscribers. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher.swift#L1`–`L83`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher+Interface.swift#L25`–`L86`)
3. **“Reducers” are `async` instance methods on state objects**, not standalone pure functions. `BusinessState.reduce(with:)` is the reducer hook; `cleanup()` is the lifecycle hook. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L1`–`L16`)
4. **Effects are just Actions (`Effect : Action`) routed to sagas**. `RootSaga` only reacts to actions that conform to `Effect`, and it notifies all registered sagas/flows concurrently. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Action/Relux.Effect.swift#L1`–`L3`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L11`–`L29`)
5. **`ActionResult` primarily reports saga/flow outcomes**: the `Store` subscriber always returns `nil`, while `RootSaga` returns a reduced `ActionResult` only for `Effect`s. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L33`–`L37`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L11`–`L29`)
6. **State observation is not implemented as a first-class API** in this package: `UIState` is a marker protocol, stored in `Store.uiStates`, but is never notified on dispatch; only `BusinessState` and “temporal” `HybridState` instances are reduced. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L16`–`L30`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L61`–`L97`)
7. **Swift version messaging mismatch:** `Package.swift` sets `swift-tools-version: 6.0` (SwiftPM requires Swift 6+ toolchain), while README claims Swift 5.10+. (Source: `.temp/repos/swift-relux/Package.swift#L1`, `.temp/repos/swift-relux/README.md`)
8. **Test utilities live inside the main target** (`Sources/TestUtils/**`) and are `public`, which likely ships them to production consumers unless split into a separate target/product. (Source: `.temp/repos/swift-relux/Package.swift#L21`–`L30`, `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing.swift#L1`–`L3`)

---

# L1 — Recon (surface scan)

## Repository layout

Top-level:

- `Package.swift`
- `README.md`
- `LICENSE` (MIT)
- `Sources/`
- `Tests/`

Sources (key subtrees):

```
Sources/
  Relux/
    Internal/
      Relux+Subscriber/
      Utils/
      Sequence+AsSet.swift
    Models/
    Relux/
      Relux.swift
      Relux+Action/
      Relux+Dispatcher/
      Relux+Logger/
      Relux+Module/
      Relux+Navigation/
      Relux+Saga/
      Relux+Store/
  TestUtils/
    Internal/LockedState.swift
    Relux+Testing*.swift
```

## File counts

- Total Swift files: **36** (`Sources`: 33 incl. `TestUtils`, `Tests`: 2, plus `Package.swift`). (Derived from repo scan; see file list in task output)
- Rough size: **~1,677 LOC** across `.swift` files. (Source: `wc -l` aggregate)

## SwiftPM configuration

- Package name: `swift-relux`. (Source: `.temp/repos/swift-relux/Package.swift#L4`–`L6`)
- Library product: `Relux` targeting `Relux`. (Source: `.temp/repos/swift-relux/Package.swift#L12`–`L16`)
- Platforms declared: iOS 13, macOS 10.15, tvOS 13, watchOS 6. (Source: `.temp/repos/swift-relux/Package.swift#L6`–`L11`)
- Targets:
  - `Relux` target uses **`path: "Sources"`** (so all Swift files under `Sources/**` are compiled into module `Relux`, including `Sources/TestUtils/**` unless conditionally excluded). (Source: `.temp/repos/swift-relux/Package.swift#L21`–`L25`)
  - `ReluxTests` test target depends on `Relux`. (Source: `.temp/repos/swift-relux/Package.swift#L26`–`L30`)
- Dependencies: none. (Source: `.temp/repos/swift-relux/Package.swift#L18`–`L19`)
- Tools version: `// swift-tools-version: 6.0`. (Source: `.temp/repos/swift-relux/Package.swift#L1`)

## Documentation snapshot

README positioning:

- “Redux’s Swift-y cousin … async” + unidirectional flow, actor-based, modular, sagas/flows, logging reflection, reducers inside state. (Source: `.temp/repos/swift-relux/README.md`)

---

# L2 — Deep-dive (read all source files)

## 1) Composition root: `Relux` singleton

### What it is

`Relux` is a `@MainActor` final class that:

- Holds `store: Store`, `rootSaga: RootSaga`, and `dispatcher: Dispatcher`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux.swift#L1`–`L6`)
- Sets a single global instance `Relux.shared: Relux!` and forbids multiple instances (`fatalError` if already set). (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux.swift#L7`–`L23`)
- Initializes its dispatcher with exactly two subscribers: `[appStore, rootSaga]`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux.swift#L14`–`L19`)

### Module lifecycle

- `register(_ module:)` connects:
  - `module.states` into `Store.connect(state:)`
  - `module.sagas` into `RootSaga.connectSaga(saga:)`
  (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux.swift#L27`–`L40`)
- `unregister(_ module:)`:
  - `await store.disconnect(state:)` for each state
  - `rootSaga.disconnect(saga:)` for each saga
  (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux.swift#L51`–`L68`)

**Important nuance:** `Store.disconnect(state:)` only removes `BusinessState` entries (see Store section). If a module includes `UIState` (or a `HybridState` that was connected as both), `unregister` will not remove it from `uiStates`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L99`–`L109`)

## 2) Dispatch pipeline: `Relux.Dispatcher` actor

### Public entrypoints

You can dispatch actions in three ways:

1. `Relux.Dispatcher.actions(...) async -> ActionResult` and `action(...) async -> ActionResult` (methods on the dispatcher). (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher+Interface.swift#L25`–`L67`)
2. Global functions `actions(...)` / `action(...)` which call `Relux.shared.dispatcher`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher+Interface.swift#L90`–`L131`)
3. `performAsync(...)` for fire-and-forget dispatch in a detached `Task`, with optional completion. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher+Interface.swift#L133`–`L157`)

### Execution modes (serial vs concurrent)

- `ExecutionType.serially`: actions are processed in order using `sequentialPerform` (per-action fanout to subscribers is still concurrent). (Source: `.temp/repos/swift-relux/Sources/Relux/Models/Relux+ExecutionType.swift#L3`–`L7`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher+Interface.swift#L80`–`L85`)
- `ExecutionType.concurrently`: actions are processed concurrently using `concurrentPerform`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher+Interface.swift#L80`–`L85`)

### Subscriber fanout and logging

For each action:

- The dispatcher collects non-nil results from subscribers using `concurrentCompactMap { await $0.perform(action) }`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher.swift#L36`–`L47`, `.temp/repos/swift-relux/Sources/Relux/Internal/Utils/AsyncUtils.swift#L97`–`L127`)
- It then logs the action along with the reduced result using `logger.logAction(...)`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher.swift#L43`–`L45`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Logger/Relux+Logger.swift#L1`–`L12`)

### Middleware pattern assessment

There is **no public middleware/interceptor API**:

- Subscriber protocol (`Relux.Subscriber`) is `internal`, and the dispatcher’s initializer that accepts subscribers is `internal`. (Source: `.temp/repos/swift-relux/Sources/Relux/Internal/Relux+Subscriber/Relux+Subscriber.swift#L1`–`L14`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher.swift#L6`–`L12`)
- A protocol for dispatch abstraction (`IDispatcher`) exists but is fully commented out. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher+Interface.swift#L1`–`L23`)

**Implication:** Extending dispatch behavior (analytics, persistence, devtools, etc.) currently requires modifying Relux itself (or implementing that logic inside states/sagas/loggers), rather than composing middleware externally.

## 3) Actions vs Effects

### Action type

- `Relux.Action` requires `Sendable` + `Relux.EnumReflectable`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Action/Relux+Action.swift#L1`–`L3`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Logger/Relux+Logger+EnumReflectible.swift#L1`–`L6`)

### Effect type

- `Relux.Effect` is a marker protocol: `Effect : Action`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Action/Relux.Effect.swift#L1`–`L3`)

### Semantics

- The `Store` reduces **all Actions** (effects included), because it is subscribed to every dispatched action. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L33`–`L37`)
- `RootSaga` only reacts to actions that are `Effect`s. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L11`–`L18`)

## 4) Store + state “reducer” model

### State protocols (public surface)

- `AnyState`: class-bound + `TypeKeyable` + `Sendable`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L1`–`L2`)
- `BusinessState`: defines the reducer (`reduce(with:) async`) and lifecycle cleanup (`cleanup() async`). (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L4`–`L7`)
- `UIState`: `@MainActor` marker protocol. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L9`–`L10`)
- `HybridState`: `@MainActor` + `BusinessState` + `UIState` (meant for SwiftUI-friendly state that also reduces actions). (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L12`–`L16`)

**Fact-check note:** README states “every `BusinessState` … is an actor”; the protocol does not enforce `Actor`, but test utilities implement `BusinessState` as an `actor`, suggesting that’s the intended usage. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L4`–`L7`, `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+MockModule+State.swift#L7`–`L21`)

### Storage model

`Store` is a `@MainActor` class maintaining registries keyed by `TypeKeyable.Key`:

- `businessStates: [Key: any BusinessState]`
- `uiStates: [Key: any UIState]`
- `tempStates: [Key: StateRef]` where `StateRef` holds a **weak** `HybridState` reference. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L3`–`L13`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L19`–`L22`)

### Reducer execution

On dispatch, `Store.notify(_:)` reduces actions:

- Concurrently across all connected business states: `businessStates.concurrentForEach { await state.reduce(with: action) }`
- Concurrently across temporal hybrid states: `tempStates.concurrentForEach { await objectRef?.reduce(with: action) }`
  (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L16`–`L30`)

**Ordering/consistency implications**

- Within a single action: state reducers are invoked concurrently; there is no defined order between different states.
- Across actions:
  - If you dispatch **serially**, `Dispatcher` awaits subscriber completion per action, so actions are processed in order.
  - If you dispatch **concurrently**, multiple actions’ reducer invocations can overlap; per-state ordering relies on the state’s own isolation (e.g., being an `actor`).
  (Sources: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher.swift#L20`–`L83`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L16`–`L30`)

### Connect / disconnect

- `connect(state:)` tries to register an `AnyState` as a `BusinessState` and/or `UIState` (so a `HybridState` will be inserted into **both** dictionaries). (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L61`–`L77`)
- `connectTemporally(state:)` registers a `HybridState` weakly, for ephemeral view-owned state. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L79`–`L82`)
- `disconnect(state:)` only removes connected **business states** (calls `cleanup()` then removes from `businessStates`). It does not remove from `uiStates` nor from `tempStates`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L99`–`L109`)

### State observation

There is no observation/notification mechanism in the store beyond calling `reduce(with:)`:

- `uiStates` are stored and retrievable via `getState(UIState.self)` but are never notified on dispatch. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L16`–`L30`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L48`–`L52`)

**Practical consequence:** Any UI reactivity (SwiftUI `ObservableObject`, async streams, etc.) must be implemented by the concrete state types themselves or in another package layer not present here.

## 5) Side effects: `RootSaga`, `Saga`, and `Flow`

### RootSaga = effect router

- `RootSaga` is a `@MainActor` class that stores sagas by `TypeKeyable.Key`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L1`–`L8`)
- As a subscriber, it only handles actions that conform to `Effect`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L11`–`L18`)
- When an effect arrives, it concurrently applies it to all sagas:
  - If a saga also conforms to `Flow`, it uses `flow.apply(effect) -> ActionResult`
  - Otherwise it uses an internal helper that wraps `Saga.apply(effect) -> ActionResult.success`
  (Sources: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L20`–`L29`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Saga.swift#L8`–`L12`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Flow.swift#L1`–`L13`)

### Saga protocol

- `Saga` is explicitly `Actor`-constrained and `TypeKeyable`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Saga.swift#L1`–`L5`)
- Sagas get a dispatcher from `Relux.shared.dispatcher` and can dispatch new actions/effects via `action(...)` / `actions(...)`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Saga.swift#L15`–`L65`)

### Flow protocol = saga with result

- `Flow` refines `Saga` to allow `apply(effect) -> ActionResult`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Flow.swift#L1`–`L7`)
- Default `Flow.apply(effect) async` bridges to the result-returning overload and discards it. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Flow.swift#L10`–`L13`)

**Implication:** If you want effect handling to contribute failures/payloads to the dispatch result, implement `Flow` rather than plain `Saga`.

## 6) `ActionResult` model

### Structure

- `ActionResult` is `success(payload:)` or `failure(payload:)`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Action/Relux+ActionResult.swift#L1`–`L5`)
- Payloads are key-value maps:
  - `Payload.data: [AnyHashable: Sendable]`
  - `ErrPayload.data: [AnyHashable: Error]`
  (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Action/Relux+ActionResult.swift#L28`–`L54`)

### Reduction/merging

- A sequence of `ActionResult` reduces to:
  - `.success(merge(all success payloads))` if there are no failures
  - `.failure(merge(all error payloads))` if any failure exists
  (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Action/Relux+ActionResult+Reduced.swift#L1`–`L15`)

Merging uses `Dictionary.merging(..., uniquingKeysWith: { _, new in new })` (new wins). (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Action/Relux+ActionResult+Reduced.swift#L18`–`L35`)

## 7) Logging + enum reflection

### Logger interface

- `Relux.Logger` is `Sendable` and provides `logAction(_ action: EnumReflectable, result: ActionResult?, startTimeInMillis:, privacy:, fileID:, functionName:, lineNumber:)`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Logger/Relux+Logger.swift#L1`–`L12`)

### Enum reflection utilities

- `EnumReflectable` provides `caseName` and `associatedValues` via `Mirror`, plus default `subsystem`/`category`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Logger/Relux+Logger+EnumReflectible.swift#L1`–`L65`)

**Observation:** Because `Action` requires `EnumReflectable`, actions are designed to be enums (or at least Mirror-friendly), enabling logging without manual formatting.

## 8) Navigation helpers

Relux includes a small `Navigation` namespace:

- `RouterProtocol : HybridState` (so routers are main-actor reducer states). (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Navigation/Relux+Navigation.swift#L6`–`L13`)
- Path/modal component protocols plus default `id` via `hashValue` for modal components. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Navigation/Relux+Navigation.swift#L6`–`L22`)

## 9) Core utilities and primitives

- `TypeKeyable` gives every class/actor a stable key based on `ObjectIdentifier(type)`, used to enforce “one instance per type” in registries. (Source: `.temp/repos/swift-relux/Sources/Relux/Internal/Utils/TypeKeyable.swift#L2`–`L11`)
- Concurrency helpers (`asyncForEach`, `concurrentMap`, etc.) are vendored from John Sundell’s CollectionConcurrencyKit. (Source: `.temp/repos/swift-relux/Sources/Relux/Internal/Utils/AsyncUtils.swift#L3`–`L76`)
- `timestamp` provides cross-platform wall-clock time with seconds/millis/micros. (Source: `.temp/repos/swift-relux/Sources/Relux/Internal/Utils/timestamp/timestamp.swift#L17`–`L50`)
- `AsyncLock` exists as an internal actor-based lock but is unused in the package. (Source: `.temp/repos/swift-relux/Sources/Relux/Internal/Utils/AsyncLock.swift#L1`–`L55`)

## 10) Test utilities shipped with Relux

Because `Relux` target includes `Sources/TestUtils/**`, the following are public APIs in the `Relux` module:

- `Relux.Testing` namespace. (Source: `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing.swift#L1`–`L3`)
- `Relux.Testing.Logger` capturing actions/effects for assertions. (Source: `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+Logger.swift#L2`–`L32`)
- `Relux.Testing.MockModule` providing a module with an action-logging `BusinessState` and an effect-logging `Saga`. (Source: `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+MockModule.swift#L1`–`L18`, `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+MockModule+State.swift#L7`–`L21`, `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+MockModule+Saga.swift#L2`–`L9`)

Notable detail: `Relux+Testing+MockModule+State.swift` uses `@_exported import Foundation(…Essentials)`, which re-exports Foundation to downstream importers of Relux (potentially undesirable for a production library surface). (Source: `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+MockModule+State.swift#L1`–`L5`)

## 11) Tests

`ReluxTests` uses Swift’s `Testing` library and currently covers only:

- `LockedState` semantics (vendored from Swift Foundation). (Source: `.temp/repos/swift-relux/Tests/LockedStateTests.swift`)
- `timestamp` invariants. (Source: `.temp/repos/swift-relux/Tests/TimestampTests.swift`)

There are **no tests** for the core dispatch/store/saga behavior.

---

# L3 — Domain synthesis (grouped findings)

## API domain (what consumers touch)

- Initialize Relux once on main actor with a logger, then register modules:
  - `await Relux(logger: ..., appStore: ..., rootSaga: ...)`
  - `relux.register(module)`
  (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux.swift#L1`–`L48`)
- Dispatch via global `actions/action` helpers or via dispatcher/saga helpers. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher+Interface.swift#L25`–`L157`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Saga.swift#L24`–`L65`)

## State/reducer domain

- Reducers are `async` and live on the state instance (`BusinessState.reduce(with:)`). (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L4`–`L7`)
- Store fans out reducers concurrently across all connected states, plus optionally “temporal” hybrid states held weakly. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L16`–`L30`)
- Keyed-by-type registries enforce at most one instance per concrete state type (per dictionary). (Source: `.temp/repos/swift-relux/Sources/Relux/Internal/Utils/TypeKeyable.swift#L2`–`L11`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L84`–`L96`)

## Effects/side-effects domain

- Side effects are modeled as `Effect` actions and routed to registered sagas. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L11`–`L29`)
- `Saga` is actor-isolated and can dispatch more actions/effects. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Saga.swift#L1`–`L65`)
- `Flow` is the mechanism for effect handlers to return `ActionResult` (including failures). (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Flow.swift#L1`–`L13`)

## Observability / UI integration domain

- The package defines UI-typed state protocols (`UIState`, `HybridState`) but provides no subscription/streaming mechanism for UI observation. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L9`–`L16`)
- `Store.uiStates` is a registry only; these states do not receive actions via the store. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L16`–`L30`)
- The likely intended pattern is:
  - `HybridState` for simple SwiftUI screens (main-actor `ObservableObject` that also reduces).
  - `BusinessState` actor as the true source of data + a UI wrapper (`UIState`) that you wire up manually (not implemented here).
  (Inference based on README + available protocols; README source: `.temp/repos/swift-relux/README.md`)

## Extensibility / “middleware” domain

- The dispatch graph is currently fixed to two subscribers, and subscriber/middleware composition is not exposed publicly. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux.swift#L14`–`L19`, `.temp/repos/swift-relux/Sources/Relux/Internal/Relux+Subscriber/Relux+Subscriber.swift#L1`–`L4`)

## Testing domain

- Utility module (`Relux.Testing`) ships as public code in the main target. (Source: `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing.swift#L1`–`L3`)
- Core behavior lacks direct tests; only time/lock primitives are tested. (Source: `.temp/repos/swift-relux/Tests/LockedStateTests.swift`, `.temp/repos/swift-relux/Tests/TimestampTests.swift`)

---

# L4 — Product synthesis (what this package “is”)

## Executive summary

Relux is a small Swift Concurrency-first state management runtime that:

- Centralizes dispatch inside an actor (`Dispatcher`).
- Applies every dispatched `Action` to all registered `BusinessState` reducers (and optionally weak “temporal” `HybridState`s) via the `Store`.
- Routes only `Effect` actions to saga/flow actors via `RootSaga`.
- Aggregates and returns `ActionResult` primarily from flow-based effect handlers.

It is minimal by design: no external dependencies, a small API surface, and a strong “types-as-keys” registry model.

## Architecture overview (runtime flow)

```
Caller (any context)
  └─ actions()/action() (or saga.action/saga.actions)
      └─ Dispatcher (actor)
          ├─ fanout to Store.perform(...)           (Store is @MainActor Subscriber)
          │    └─ Store.notify(action)
          │         ├─ concurrent reduce() across BusinessState entries
          │         └─ concurrent reduce() across weak HybridState temporals
          └─ fanout to RootSaga.perform(...)        (RootSaga is @MainActor Subscriber)
               └─ if action is Effect:
                    concurrent apply(effect) across Saga/Flow entries
                    └─ Flows contribute ActionResult failures/payloads
```

Sources:

- Wiring + singleton: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux.swift#L1`–`L24`
- Dispatcher fanout + reduction: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher.swift#L20`–`L83`
- Store notify: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L16`–`L30`
- Effect routing: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L11`–`L29`

## Public API catalog (high level)

> This is an inventory of the major public surface; see sources for exact signatures.

**Core runtime**

- `@MainActor public final class Relux`
  - `public static var shared: Relux!`
  - `public let store: Relux.Store`
  - `public let rootSaga: Relux.RootSaga`
  - `public let dispatcher: Relux.Dispatcher`
  - `public init(logger: any Relux.Logger, appStore: Store = .init(), rootSaga: RootSaga = .init()) async`
  - `public func register(_ module: Relux.Module) -> Relux`
  - `public func register(@Relux.ModuleResultBuilder ...) async -> Relux`
  - `public func unregister(_ module: Relux.Module) async -> Relux`
  (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux.swift#L1`–`L82`)

- `@MainActor public final class Relux.Store`
  - `public func getState<T: BusinessState>(_: T.Type) -> T` (force-cast)
  - `public func connect(state: some AnyState)`
  - `public func connectTemporally<TS: HybridState>(state: TS) -> TS`
  - `public func disconnect(state: some AnyState) async`
  - `public func cleanup(exclusions: [BusinessState.Type] = []) async`
  (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L3`–`L127`)

- `@MainActor public final class Relux.RootSaga`
  - `public func connectSaga(saga: any Relux.Saga)`
  - `public func disconnect(saga: any Relux.Saga)`
  (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L1`–`L43`)

- `public actor Relux.Dispatcher`
  - `public init(logger: any Relux.Logger)`
  - `public func actions(...) async -> ActionResult`
  - `public func action(...) async -> ActionResult`
  (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher.swift#L1`–`L18`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher+Interface.swift#L25`–`L67`)

**Protocols**

- `public protocol Relux.Action : Sendable, Relux.EnumReflectable` (via `public extension Relux`) (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Action/Relux+Action.swift#L1`–`L3`)
- `public protocol Relux.Effect : Relux.Action` (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Action/Relux.Effect.swift#L1`–`L3`)
- `public protocol Relux.AnyState : AnyObject, TypeKeyable, Sendable` (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L1`–`L2`)
- `public protocol Relux.BusinessState : AnyState` with `reduce` + `cleanup` (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L4`–`L7`)
- `@MainActor public protocol Relux.UIState : AnyState` (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L9`–`L10`)
- `@MainActor public protocol Relux.HybridState : BusinessState, UIState` (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L12`–`L16`)
- `public protocol Relux.Saga : Actor, TypeKeyable` with `apply(effect)` (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Saga.swift#L1`–`L5`)
- `public protocol Relux.Flow : Relux.Saga` with result-returning apply (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+Flow.swift#L1`–`L7`)
- `public protocol Relux.Module : Sendable` with `[AnyState]` and `[Saga]` (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Module/Relux+Module.swift#L1`–`L6`)

**Action results**

- `public enum Relux.ActionResult` + `Payload` / `ErrPayload` helpers (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Action/Relux+ActionResult.swift#L1`–`L59`)

**Logging**

- `public protocol Relux.Logger` and `public protocol Relux.EnumReflectable` (Sources: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Logger/Relux+Logger.swift#L1`–`L20`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Logger/Relux+Logger+EnumReflectible.swift#L1`–`L65`)

**Convenience globals**

- Global `actions(...)` / `action(...)` / `performAsync(...)` using `Relux.shared.dispatcher`. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Dispatcher/Relux+Dispatcher+Interface.swift#L90`–`L157`)

**Testing**

- `Relux.Testing` namespace, logger, mock module. (Source: `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing.swift#L1`–`L3`, `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+Logger.swift#L2`–`L32`, `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+MockModule.swift#L1`–`L18`)

## Key patterns and conventions

- **Namespace-as-class:** Most public types are nested under `Relux` (the class), e.g. `Relux.Action`, `Relux.Store`, `Relux.Saga`.
- **Type-keyed registries:** Connected states/sagas are keyed by type (`ObjectIdentifier(Self)`), enforcing “one per type”. (Source: `.temp/repos/swift-relux/Sources/Relux/Internal/Utils/TypeKeyable.swift#L2`–`L11`)
- **Async reducers:** Reducers are `async`, and store executes them concurrently across states for a given action. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L16`–`L30`)
- **Effect gating:** Only `Effect` actions hit sagas; ordinary actions only reduce state. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L11`–`L18`)
- **Results via flows:** Only flows can return non-trivial `ActionResult` outcomes; plain sagas are wrapped as success. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Saga/Relux+RootSaga.swift#L20`–`L29`)

## Integration points (current + implied)

- **SwiftUI:** `@MainActor HybridState` is positioned as the simplest “SwiftUI-friendly state” type; you would typically add `ObservableObject`/`@Published` at the concrete type level. (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+State.swift#L12`–`L16`, README: `.temp/repos/swift-relux/README.md`)
- **Logging:** Provide a `Relux.Logger` implementation; enum reflection makes action/effect logging low-effort. (Sources: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Logger/Relux+Logger.swift#L1`–`L12`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Logger/Relux+Logger+EnumReflectible.swift#L21`–`L65`)
- **Testing:** `Relux.Testing.MockModule` can be used for black-box dispatch tests of actions/effects once a `Relux` singleton is initialized in the test harness. (Sources: `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+MockModule.swift#L1`–`L18`, `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+MockModule+State.swift#L7`–`L21`)

## Recommendations (for `relux-manager` CLI)

Concrete, code-aligned suggestions for a CLI component that manages Relux usage across a multi-module app:

1. **Scaffold generators**
   - Generate templates for:
     - `BusinessState` actor with `reduce(with:)` switch and `cleanup()`
     - `Saga`/`Flow` actor with `apply(effect:)` switch
     - `Module` that wires `states` + `sagas`
     - app composition (`Relux(logger:)` + `register { ... }`)
   - Ensure all generated actions conform to `Relux.Action` (i.e., `Sendable` + `EnumReflectable`).

2. **Static checks / lint rules (Relux-specific)**
   - Verify modules don’t register multiple instances of the same state/saga type (type-key collisions).
   - Detect `UIState` instances in `Module.states` and warn that `disconnect` doesn’t remove `uiStates` (potential leak/inconsistency).
   - Detect usage of `_exported import` in production-facing targets (currently present in `TestUtils`).

3. **Runtime diagnostics hooks**
   - Provide a standard `Logger` implementation that records:
     - timing (`startTimeInMillis`) and durations (requires logger to compute `now - startTime`)
     - action/effect names (`caseName`) and associated values for debugging
   - Optionally integrate a “devtools” subscriber concept (would require library change; see gaps below).

4. **Test harness helpers**
   - Generate a standard test bootstrap that:
     - sets `Relux.shared = nil` before each test suite (since singleton enforces one instance)
     - initializes `Relux` with `Relux.Testing.Logger`
     - registers a `Relux.Testing.MockModule`

## Gaps / risks surfaced

1. **UIState not reduced + disconnect asymmetry**
   - `UIState` is stored but never reduced; `disconnect` removes only business states. This is either an incomplete feature or a footgun (module unregister may leak UI states). (Source: `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L16`–`L30`, `.temp/repos/swift-relux/Sources/Relux/Relux/Relux+Store/Relux+Store.swift#L99`–`L109`)
2. **No middleware composition**
   - Subscriber protocol is internal and subscribers can’t be extended from outside; devtools/analytics/persistence hooks are limited to logger or modifying the library. (Source: `.temp/repos/swift-relux/Sources/Relux/Internal/Relux+Subscriber/Relux+Subscriber.swift#L1`–`L4`)
3. **Low test coverage on core behavior**
   - Dispatch ordering, store fanout, saga routing, and `ActionResult` propagation are untested. (Source: `.temp/repos/swift-relux/Tests/*`)
4. **Swift tools version / README mismatch**
   - `swift-tools-version: 6.0` conflicts with README’s “Swift 5.10+” claim; consumers on 5.10 cannot build the package via SwiftPM as-is. (Source: `.temp/repos/swift-relux/Package.swift#L1`, `.temp/repos/swift-relux/README.md`)
5. **TestUtils in production module**
   - Public testing utilities (and `_exported import`) ship as part of `Relux` module due to target path settings; consider separating into a dedicated test-support product. (Source: `.temp/repos/swift-relux/Package.swift#L21`–`L25`, `.temp/repos/swift-relux/Sources/TestUtils/Relux+Testing+MockModule+State.swift#L1`–`L5`)

---

## Appendix: license

Relux is MIT licensed. (Source: `.temp/repos/swift-relux/LICENSE`)

