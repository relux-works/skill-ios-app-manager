# swift-ioc — Product Forensics (TASK-260224-1l5ql3)

- Date: 2026-02-24
- Target: `.temp/repos/swift-ioc/`
- Revision analyzed: `26af13da762704b39fff140a72b290a610dfd008` (`git -C .temp/repos/swift-ioc rev-parse HEAD`)
- Deliverable: Full L1–L4 forensics report (read-only; no source modifications)

## Highlights / Key Takeaways

- **Async-first DI, but simple:** `IoC` is a tiny type-keyed container with 2 registration paths (sync + async) and 2 lifecycles (`.container`, `.transient`). Source: `.temp/repos/swift-ioc/Sources/IoC.swift:3`, `.temp/repos/swift-ioc/Sources/IoC.swift:36`, `.temp/repos/swift-ioc/Sources/IoC+Lifecycle.swift:2`.
- **No explicit “scoped” lifecycle:** Scoping is achieved by creating separate `IoC` instances (e.g., per app registry, per module). Source: `.temp/repos/swift-ioc/Sources/IoC+Lifecycle.swift:2`, plus Relux integration patterns below.
- **Thread-safety is “best-effort / unchecked”:** Container and maps are lock-protected, but the public container is `@unchecked Sendable`, and async resolvers rely on `nonisolated(unsafe)` to store state/closures. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:3`, `.temp/repos/swift-ioc/Sources/Internal/AtomicMap.swift:4`, `.temp/repos/swift-ioc/Sources/IoC+ResolversAsync.swift:4`.
- **Resolution behavior is asymmetric:** `get(by:)` crashes if the type was registered async-only; `getAsync(by:)` can resolve either async or sync registrations. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:83`.
- **Zero real tests today:** Test target exists but contains only a stub. Source: `.temp/repos/swift-ioc/Tests/swift-iocTests/SwiftIocTests.swift:1`.

---

# L1 — Recon

## Codebase Metrics (excluding `.git/`)

Computed with:
- `find Sources Tests -type f | wc -l`
- `find Sources -name '*.swift' | wc -l`
- `find Tests -name '*.swift' | wc -l`
- `find Sources Tests -name '*.swift' -print0 | xargs -0 wc -l`

| Metric | Value |
|---|---:|
| Swift source files | 9 |
| Swift test files | 1 |
| Swift LOC (Sources + Tests) | 411 |

Per-file LOC:
- `Sources/IoC.swift` — 148
- `Sources/IoC+ResolversAsync.swift` — 52
- `Sources/IoC+ResolversSync.swift` — 48
- `Sources/Internal/AsyncLock.swift` — 52
- `Sources/Internal/AtomicMap.swift` — 50
- `Sources/IoC+Logger.swift` — 23
- `Sources/Internal/Optional+IsNil.swift` — 16
- `Sources/IoC+IResolvers.swift` — 10
- `Sources/IoC+Lifecycle.swift` — 6
- `Tests/swift-iocTests/SwiftIocTests.swift` — 6

## Repo / Package Structure

```
.temp/repos/swift-ioc/
├── Package.swift
├── README.md
├── LICENSE
├── Sources/
│   ├── IoC.swift
│   ├── IoC+IResolvers.swift
│   ├── IoC+Lifecycle.swift
│   ├── IoC+Logger.swift
│   ├── IoC+ResolversAsync.swift
│   ├── IoC+ResolversSync.swift
│   └── Internal/
│       ├── AsyncLock.swift
│       ├── AtomicMap.swift
│       └── Optional+IsNil.swift
└── Tests/
    └── swift-iocTests/
        └── SwiftIocTests.swift
```

## Package.swift (SwiftPM)

- Swift tools: `// swift-tools-version: 6.0`. Source: `.temp/repos/swift-ioc/Package.swift:1`.
- Package name: `swift-ioc`. Source: `.temp/repos/swift-ioc/Package.swift:5`.
- Library product: `SwiftIoC` from target `SwiftIoC`. Source: `.temp/repos/swift-ioc/Package.swift:15`.
- Platforms: iOS 13, macOS 10.15, tvOS 13, watchOS 6. Source: `.temp/repos/swift-ioc/Package.swift:8`.
- No external dependencies. Source: `.temp/repos/swift-ioc/Package.swift:21`.

---

# L2 — Deep Dive (All Source Files)

## 1) Public API Surface (Catalog)

### `IoC`

**Type**
- `public final class IoC: @unchecked Sendable`. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:3`.

**Lifecycle**
- `public enum Lifecycle { case container; case transient }`. Source: `.temp/repos/swift-ioc/Sources/IoC+Lifecycle.swift:2`.

**Registration**
- Sync:
  - `public func register<T>(_ type: T.Type, lifecycle: Lifecycle = .transient, withReplacement: Bool = false, resolver: @escaping () -> T)`
  - Source: `.temp/repos/swift-ioc/Sources/IoC.swift:36`.
- Async:
  - `public func register<T: Sendable>(_ type: T.Type, lifecycle: Lifecycle = .transient, withReplacement: Bool = false, resolver: @escaping () async -> T)`
  - Source: `.temp/repos/swift-ioc/Sources/IoC.swift:58`.

**Resolution**
- Sync getter:
  - `public func get<T>(by type: T.Type) -> T?`
  - Source: `.temp/repos/swift-ioc/Sources/IoC.swift:83`.
- Async getter:
  - `public func getAsync<T>(by type: T.Type) async -> T?`
  - Source: `.temp/repos/swift-ioc/Sources/IoC.swift:112`.
- Async “wait until available”:
  - `public func waitForResolve<T>(_ type: T.Type) async -> T`
  - Source: `.temp/repos/swift-ioc/Sources/IoC.swift:140`.

### Logging

- `public protocol ILogger { func send(_ msg: String) }`. Source: `.temp/repos/swift-ioc/Sources/IoC+Logger.swift:2`.
- Default logger:
  - `public struct Logger: ILogger` with `init(enabled: Bool = true)`
  - `send` prints to stdout as `SwiftIoC: ...` (when enabled).
  - Source: `.temp/repos/swift-ioc/Sources/IoC+Logger.swift:8`.

### Resolver Protocols + Implementations

- `public protocol SyncResolver { associatedtype T; func instance() -> T }`. Source: `.temp/repos/swift-ioc/Sources/IoC+IResolvers.swift:2`.
- `public protocol AsyncResolver: Sendable { associatedtype T: Sendable; func instance() async -> T }`. Source: `.temp/repos/swift-ioc/Sources/IoC+IResolvers.swift:6`.

Implementations (all `public`):
- `SyncContainerResolver<T>` — caches single instance; `NSLock` guarded. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversSync.swift:4`.
- `SyncTransientResolver<T>` — returns `build()` each time. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversSync.swift:36`.
- `AsyncContainerResolver<T: Sendable>` — caches single instance; guarded via `AsyncLock`. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversAsync.swift:4`.
- `AsyncTransientResolver<T: Sendable>` — returns `await build()` each time (no caching). Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversAsync.swift:33`.

## 2) Architecture & Data Model

## High-level design

`IoC` holds 2 independent registries:
- `mapForSync: AtomicMap<ObjectIdentifier, any SyncResolver>`
- `mapForAsync: AtomicMap<ObjectIdentifier, any AsyncResolver>`

Source: `.temp/repos/swift-ioc/Sources/IoC.swift:6`.

The container key is `ObjectIdentifier`, derived from the type passed into registration/getters. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:4`, `.temp/repos/swift-ioc/Sources/IoC.swift:25`.

### Keying: important nuance

`IoC` currently computes its key via:

- `func key(of obj: Any) -> ObjectIdentifier { .init(type(of: obj)) }`. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:27`.

And then uses it like:
- `let key = key(of: type)` where `type` is `T.Type`. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:42`.

This means the key is based on the **dynamic type of the metatype value** (i.e. `T.Type`), not directly `ObjectIdentifier(T.self)`.

This is *internally consistent* because `register` and `get` both use the same `key(of:)` implementation. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:36`.

However, the presence of an unused alternate implementation:
- `static func key(of type: Any.Type) -> ObjectIdentifier { .init(type) }`. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:30`.

…creates a footgun: if future code mixes the two key functions, lookups will silently fail for already-registered types.

## 3) Registration semantics

### Sync registration

Behavior:
- Registers a sync resolver into `mapForSync`.
- If `withReplacement == false` and there is already an entry for this key, it crashes with `fatalError(...)`.
- Lifecycle determines which resolver wrapper is stored:
  - `.container` → `SyncContainerResolver(build:)`
  - `.transient` → `SyncTransientResolver(build:)`

Source: `.temp/repos/swift-ioc/Sources/IoC.swift:36`.

### Async registration

Behavior:
- Registers an async resolver into `mapForAsync`.
- Duplicate registration policy matches sync (fatal unless `withReplacement`).
- Lifecycle determines:
  - `.container` → `AsyncContainerResolver(build:)`
  - `.transient` → `AsyncTransientResolver(build:)`

Source: `.temp/repos/swift-ioc/Sources/IoC.swift:58`.

### Mixed registration (sync + async for same type)

Notably:
- Sync registration only checks `mapForSync` for duplicates. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:44`.
- Async registration only checks `mapForAsync`. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:66`.

So you *can* register the same `T` both sync and async (with no replacement), producing 2 independent bindings. This may be intentional (allow separate async impl), but it can also create “two sources of truth” for a single type.

## 4) Resolution semantics

### `get(by:)` (sync)

Resolution order:
1. If there is a sync resolver for `T`, return `resolver.instance() as? T`.
2. Else, if there is an async resolver for `T`, crash with:
   - `fatalError("type ... is registered as async, but sync access is attempted")`
3. Else return `nil`.

Source: `.temp/repos/swift-ioc/Sources/IoC.swift:83`.

Implication:
- If a type is “async-only”, the API forces async resolution (or a crash).
- Callers that wrap `get(by:)!` (force unwrap) will crash both on “not registered” and on “async-only mismatch”.

### `getAsync(by:)` (async)

Resolution order:
1. If there is an async resolver, return `await resolver.instance() as? T`.
2. Else, if there is a sync resolver, return `resolver.instance() as? T`.
3. Else return `nil`.

Source: `.temp/repos/swift-ioc/Sources/IoC.swift:112`.

Implication:
- Async resolution is “superset”: it can fetch both async and sync registrations.
- This matches common Relux usage: prefer async `resolveAsync` and use sync `resolve` when guaranteed local/simple.

### `waitForResolve(_:)`

Implementation:
- Busy-waits forever until `getAsync(by:)` returns non-`nil`, yielding between attempts. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:140`.

Observations:
- There is no timeout, no cancellation check, and no notification mechanism (it’s polling).
- If a dependency is never registered, the task will loop forever.

## 5) Lifecycle / Scoping model

### `.container` (singleton per registration)

Sync:
- `SyncContainerResolver` caches `_instance` behind `NSLock`. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversSync.swift:4`.

Async:
- `AsyncContainerResolver` caches `_instance` behind `AsyncLock`. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversAsync.swift:4`, `.temp/repos/swift-ioc/Sources/Internal/AsyncLock.swift:5`.

This is a **per-resolver singleton**, effectively “singleton per IoC container instance per type”.

### `.transient` (factory)

Sync:
- `SyncTransientResolver.instance()` always calls `build()`. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversSync.swift:36`.

Async:
- `AsyncTransientResolver.instance()` always calls `await build()`. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversAsync.swift:33`.

### Scoped

There is **no explicit** `.scoped` lifecycle. Source: `.temp/repos/swift-ioc/Sources/IoC+Lifecycle.swift:2`.

In Relux codebases, “scope” is commonly achieved by **creating multiple IoC containers**:
- app-level registry: one `IoC` in `Relux.Registry` / `SampleApp.Registry` (container-wide singletons)
- module-level IoC: each module builds its own `IoC` to scope domain dependencies

Sources:
- `.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift:16`
- `.temp/repos/relux-sample/relux_sample/IoC/IoC.swift:14`
- `.temp/repos/relux-sample/Docs/Patterns/RELUX_MODULAR.md:173`

## 6) Thread-safety and Concurrency

### Map safety (`AtomicMap`)

- `AtomicMap` is a lock-protected dictionary (`NSLock`) and declared `@unchecked Sendable`. Source: `.temp/repos/swift-ioc/Sources/Internal/AtomicMap.swift:4`.
- All reads/writes are done inside `lock.withLock { ... }`. Source: `.temp/repos/swift-ioc/Sources/Internal/AtomicMap.swift:10`.

### Sync singleton safety

- `SyncContainerResolver` uses `NSLock` and lazily initializes `_instance`. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversSync.swift:6`.

### Async singleton safety

- `AsyncLock` is an `actor` implementing mutual exclusion using a continuation queue. Source: `.temp/repos/swift-ioc/Sources/Internal/AsyncLock.swift:5`.
- `AsyncContainerResolver.instance()` runs the build/cache path under `await lock.withLock { ... }`. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversAsync.swift:17`.

### Concurrency model caveats

1) `IoC` itself is `@unchecked Sendable`. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:3`.
   - This asserts thread-safety without compiler verification.
   - The internal design is largely lock-based, but *correctness depends on consistent locking across all future mutations*.

2) `AsyncContainerResolver` stores `_instance` and `build` as `nonisolated(unsafe)` while marking the class `Sendable`. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversAsync.swift:7`.
   - This bypasses Swift’s data-race checks for these properties.
   - The mutual exclusion comes from `AsyncLock`, but the compiler cannot help ensure the closure captures only safe state.

3) `AsyncTransientResolver` declares an `AsyncLock` but does not use it. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversAsync.swift:35`.
   - Probably leftover; not a correctness issue for transient resolution (it always builds fresh).

## 7) Logging / Observability

- On registration success, the container logs the type being registered (sync/async). Source: `.temp/repos/swift-ioc/Sources/IoC.swift:54`, `.temp/repos/swift-ioc/Sources/IoC.swift:76`.
- On resolution, it logs “no instance” or “resolved successfully”, and for reference types it logs `ObjectIdentifier(obj)`. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:95`.
- Logging is pluggable via `ILogger`; default `Logger` prints to stdout and can be disabled. Source: `.temp/repos/swift-ioc/Sources/IoC+Logger.swift:2`.

## 8) Documentation & Examples

The README documents:
- App-level `Relux.Registry` hosting a static `IoC`. Source: `.temp/repos/swift-ioc/README.md:16`.
- Logger configuration examples. Source: `.temp/repos/swift-ioc/README.md:32`.
- Resolver wrapper methods (sync/async/optional/wait). Source: `.temp/repos/swift-ioc/README.md:50`.
- “Wait for resolve” is described as continuously checking/yielding until available. Source: `.temp/repos/swift-ioc/README.md:150`.

## 9) Tests

- Test target exists (`SwiftIoCTests`), using Swift’s `Testing` library, but contains only a placeholder test. Source: `.temp/repos/swift-ioc/Tests/swift-iocTests/SwiftIocTests.swift:1`.

---

# L3 — Domain Synthesis (Grouped Findings)

## API / UX (Developer experience)

- Extremely small surface area: 1 container type, 2 register overloads, 2 getters + wait. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:36`.
- Strongly type-based lookup (`T.Type`), no strings/tags/names. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:36`.
- Fatal errors are used for programmer mistakes (duplicate registration; sync access to async-only binding). Source: `.temp/repos/swift-ioc/Sources/IoC.swift:44`, `.temp/repos/swift-ioc/Sources/IoC.swift:86`.

## Lifecycle / Scoping

- Only `.container` vs `.transient`. Source: `.temp/repos/swift-ioc/Sources/IoC+Lifecycle.swift:2`.
- “Scoped” behavior is achieved by container instances, not lifecycle. Source: `.temp/repos/relux-sample/Docs/Patterns/RELUX_MODULAR.md:195`.

## Internals / Patterns

- **Lock-protected registries** via `AtomicMap` (`NSLock`). Source: `.temp/repos/swift-ioc/Sources/Internal/AtomicMap.swift:6`.
- **Lazy singleton resolvers** for `.container` using locks and cached `_instance`. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversSync.swift:19`.
- **Async mutual exclusion** with an actor-based lock rather than `NSLock`. Source: `.temp/repos/swift-ioc/Sources/Internal/AsyncLock.swift:5`.
- Many methods are annotated with `@inlinable` + `@inline(__always)` for cross-module inlining/perf. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:17`.

## Thread-safety assessment (fact-based)

What is clearly protected:
- Map reads/writes. Source: `.temp/repos/swift-ioc/Sources/Internal/AtomicMap.swift:38`.
- Sync singleton initialization. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversSync.swift:20`.
- Async singleton initialization. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversAsync.swift:17`.

What is “unchecked / on the user”:
- Container overall Sendable correctness. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:3`.
- Async resolver internal state/closure safety. Source: `.temp/repos/swift-ioc/Sources/IoC+ResolversAsync.swift:7`.

## Gaps / Risks

- **No test coverage** → regressions in resolution order, lifecycle caching, or concurrency won’t be caught automatically. Source: `.temp/repos/swift-ioc/Tests/swift-iocTests/SwiftIocTests.swift:1`.
- **Polling `waitForResolve`** can hide wiring bugs by “hanging forever” instead of surfacing a clear failure. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:140`.
- **No introspection API** (list, dump, remove registrations) exposed publicly; diagnosing wiring requires logs or manual reasoning. (`mapForSync/mapForAsync` are internal-only.) Source: `.temp/repos/swift-ioc/Sources/IoC.swift:7`.
- **Unused code** (`Optional.isNil/isNotNil`) suggests some cleanup opportunity. Source: `.temp/repos/swift-ioc/Sources/Internal/Optional+IsNil.swift:1`.

---

# L4 — Product Synthesis

## Executive Summary

`swift-ioc` is a minimal DI container tailored to Relux-style, async-heavy app wiring:
- It supports both sync and async constructors.
- It provides only two lifecycles: “singleton per container” (`.container`) and “factory” (`.transient`).
- It is intentionally small, but uses fatal errors and `@unchecked Sendable` / `nonisolated(unsafe)` escape hatches, which shifts safety to usage patterns and conventions.

Primary sources:
- Core API + semantics: `.temp/repos/swift-ioc/Sources/IoC.swift:3`
- Lifecycle model: `.temp/repos/swift-ioc/Sources/IoC+Lifecycle.swift:2`
- Concurrency primitives: `.temp/repos/swift-ioc/Sources/Internal/AsyncLock.swift:5`

## Architecture Overview

### Runtime flow (registration → resolution)

```
register(T, lifecycle, build)
  -> key(ObjectIdentifier of passed metatype's dynamic type)
  -> mapForSync[key] or mapForAsync[key]

get(T)
  -> mapForSync[key] else (if async-only) fatalError else nil

getAsync(T)
  -> mapForAsync[key] else mapForSync[key] else nil

waitForResolve(T)
  -> loop { if getAsync(T) != nil return; Task.yield() }
```

Sources:
- Keying + maps: `.temp/repos/swift-ioc/Sources/IoC.swift:6`
- get/getAsync/wait: `.temp/repos/swift-ioc/Sources/IoC.swift:83`

## Integration Points with Relux (Observed in Relux codebases)

### 1) App-level registry (“global container”)

Pattern:
- Define `Relux.Registry` / `SampleApp.Registry` as a namespace with a static `ioc`.
- Provide `configure()` that registers all dependencies.
- Add wrapper functions `resolve(...)`, `resolveAsync(...)`, optional variants, and sometimes `waitForResolve`.

Sources:
- Membrana app registry container + configure: `.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift:14`
- Membrana resolve wrappers: `.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift:409`
- Relux sample app registry: `.temp/repos/relux-sample/relux_sample/IoC/IoC.swift:10`

### 2) Module-level IoC (“scoped container”)

Pattern:
- Each domain/module builds its own `IoC` inside the module initializer to wire domain-specific dependencies.
- This effectively provides “scope” without a `.scoped` lifecycle.

Source:
- `.temp/repos/relux-sample/Docs/Patterns/RELUX_MODULAR.md:169`

### 3) Possible doc/API mismatch in the ecosystem

One repo’s README references `IoC.get(type: ...)`, which is **not** the API of `SwiftIoC.IoC` (it uses instance method `get(by:)`). This may be legacy documentation or a different IoC type in another module.

Source:
- `.temp/repos/swiftui-reluxrouter/README.md:101`

## Recommendations (for `relux-manager` CLI component)

Goal: make DI wiring safer, observable, and repeatable across Relux apps/modules.

### A) Static analysis / linting (repo-wide)

- Detect **async-only registration** of type `T` and any sync `get(by: T.self)` / `resolve(T.self)` call sites that could crash.
  - Fact basis: `get(by:)` fatals when only async binding exists. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:88`.
- Detect **duplicate registration** patterns (same type registered multiple times without `withReplacement: true`).
  - Fact basis: `register` fatals on duplicates. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:44`.
- Detect **`waitForResolve` usage without a bounded lifecycle** (no explicit configure call before awaiting), because it can hang forever.
  - Fact basis: `waitForResolve` is an infinite loop without timeout. Source: `.temp/repos/swift-ioc/Sources/IoC.swift:140`.

### B) Code generation helpers (templates)

- Generate a standard `Registry` template:
  - `static let ioc = IoC(logger: IoC.Logger(enabled: false))`
  - `configure()`
  - `resolve/resolveAsync/optional...` wrappers
  - Fact basis: repeated manually in Membrana + relux-sample. Source: `.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift:16`, `.temp/repos/relux-sample/relux_sample/IoC/IoC.swift:14`.
- Generate module-level “scoped IoC” templates (`buildIoC(router:)`) consistent with `RELUX_MODULAR.md`.
  - Source: `.temp/repos/relux-sample/Docs/Patterns/RELUX_MODULAR.md:194`.

### C) Runtime diagnostics (debug builds)

Even without changing `swift-ioc`, the CLI can:
- Add a build step to **enable IoC logs** (e.g., ensure `Logger(enabled: true)` for debug configurations) and collect logs to detect missing registrations early.
  - Fact basis: logging is optional. Source: `.temp/repos/swift-ioc/Sources/IoC+Logger.swift:13`.

If `swift-ioc` is allowed to evolve in the future (outside this task’s “read-only” scope), consider:
- A public `dump()` / `registeredTypes()` API to expose `mapForSync`/`mapForAsync` keys for debugging.
  - Fact basis: `AtomicMap` already exposes `keys` internally. Source: `.temp/repos/swift-ioc/Sources/Internal/AtomicMap.swift:24`.

---

## Appendix: License

MIT License. Source: `.temp/repos/swift-ioc/LICENSE:1`.

