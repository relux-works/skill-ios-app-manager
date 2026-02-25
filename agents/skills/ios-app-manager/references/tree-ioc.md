# Tree: swift-ioc (Relux Ecosystem)

## Purpose
Instruction tree for dependency injection in Relux-style iOS projects using `SwiftIoC`.

## Source Patterns (Relux)
- `swift-ioc`: `.container` and `.transient` lifecycles, `register/get/getAsync/waitForResolve`.
- `relux-sample`: `SampleApp.Registry` with global `IoC`, `configure()`, and typed resolve wrappers.
- `membrana-app`: large `Relux.Registry` with module registration and cross-module async resolution.
- `tuist-starter` templates: `composition_root.swift.tmpl`, `ioc_registration.swift.tmpl`, `ioc_resolver.swift.tmpl`, `module_registration.swift.tmpl`.

## Instruction Tree

### 1. Pick the DI boundary first
1. If wiring the main app graph:
   Use one app-level container: `static let ioc = IoC(logger: IoC.Logger(enabled: false))`.
2. If wiring a feature/module internals:
   Register into the passed container with `ModuleRegistration.register(into:ioc, ...)`.
3. If you need scoped lifetime:
   `SwiftIoC` has no `.scoped`; create a separate `IoC` instance per scope (module/session/flow).
4. If wiring App Intents / Widget / extensions:
   Use `AppDependencyManager` for extension boundary injection (see step 7).

### 2. Initialize container and composition root
1. Create `Registry`/`CompositionRoot` (`@MainActor`) with static `ioc`.
2. Expose `configure()` and call it exactly once at startup.
3. Keep registration functions pure and deterministic; avoid side effects in `configure()`.

### 3. Register services with correct lifecycle
1. Singleton/shared service:
   `ioc.register(Type.self, lifecycle: .container) { ... }`
2. Factory/new instance per resolve:
   `ioc.register(Type.self, lifecycle: .transient) { ... }`
3. Override existing binding (tests/dev toggles):
   `withReplacement: true`.
4. Register protocols, not concretes, for module boundaries (`any ServiceProtocol`).

### 4. Module-level injection rules
1. Each module owns its own registration function (`<Module>ModuleRegistration.register(...)`).
2. Register module service, middleware, and store in module registration.
3. Accept `serviceFactory` or interfaces for module-local customization.
4. Keep module package split:
   interface package exposes protocols, impl package binds concrete types.

### 5. Resolution patterns
1. Preferred explicit resolve wrappers:
   `resolve`, `resolveAsync`, `optionalResolve`, `optionalResolveAsync`, `waitForResolve`.
2. Use sync resolve only for sync registrations.
   `get(by:)` fatals if type was registered async-only.
3. Use async resolve for async module builders:
   `await ioc.getAsync(by:)`.
4. Property-wrapper pattern for extension surfaces:
   in App Intents use `@AppDependency var service: ServiceProtocol`.
5. Property-wrapper pattern inside app code:
   only as project-level adapter (custom wrapper around `Registry.resolve`), not a native `SwiftIoC` API.

### 6. Cross-module dependency resolution
1. Register dependencies before dependents.
   In `configure()`: core/shared modules first, feature modules after.
2. Resolve through interfaces (`any XService`) to prevent impl-to-impl coupling.
3. For async module dependencies, use `resolveAsync(...)` in builders.
4. Add dependency preflight where useful (template already supports `preflightDependencies`).

### 7. Extension target pattern (`AppDependencyManager`)
1. Keep intent definitions shared and protocol-driven.
2. In each bundle (app/widget/intents extension), register concrete dependencies separately:
   `AppDependencyManager.shared.add(dependency: LiveService(...))`.
3. In intents, consume by property wrapper:
   `@AppDependency var service: ServiceProtocol`.
4. Treat `AppDependencyManager` as cross-bundle seam; keep `SwiftIoC` inside bundle-local composition roots.

### 8. Testing and overrides
1. Create a fresh `IoC` per test.
2. Register mocks/fakes for protocol keys.
3. Override bindings via `withReplacement: true`.
4. For module tests, call module registration with mock `serviceFactory`.
5. Avoid `waitForResolve` in tests unless bounded; it loops forever if binding never appears.

### 9. Safety checks before merge
1. No sync `resolve` call for async-only bindings.
2. No hidden cross-scope dependency (wrong container instance).
3. `configure()` order guarantees all required modules/services are registered before use.

