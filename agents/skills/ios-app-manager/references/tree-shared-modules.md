# Shared Relux Modules Catalog

This catalog defines baseline shared modules for Relux projects. Each module is a `shared` type with interface/implementation split.

## SharedIntents
- Purpose: Reusable App Intents surface for app, widget, and App Intents extension (iOS 17+).
- Type: shared (interface + impl)
- Dependencies: `SharedAuth`, `SharedStorage`, `SharedAnalytics`
- Setup:
  1. `ios-app-manager module create SharedIntents --type shared`
  2. `ios-app-manager dep add SharedIntents --depends-on SharedAuth`
  3. `ios-app-manager dep add SharedIntents --depends-on SharedStorage`
  4. `ios-app-manager dep add SharedIntents --depends-on SharedAnalytics`
- Integration: Keep `AppIntent`/`AppEntity` implementations in `SharedIntentsImpl`; link that package to each execution bundle (app, widget extension, App Intents extension) and register concrete services via `AppDependencyManager` during bundle startup.
- Testing: Define protocol-first intent services in `SharedIntents`; test `perform()` with test doubles for each dependency and verify fallback behavior when dependencies are not registered.

## SharedAnalytics
- Purpose: Analytics abstraction with provider registration and environment-specific routing.
- Type: shared (interface + impl)
- Dependencies: `SharedAuth`, `SharedNetworking`
- Setup:
  1. `ios-app-manager module create SharedAnalytics --type shared`
  2. `ios-app-manager dep add SharedAnalytics --depends-on SharedAuth`
  3. `ios-app-manager dep add SharedAnalytics --depends-on SharedNetworking`
  4. `ios-app-manager dep list SharedAnalytics`
- Integration: Expose a provider-agnostic analytics protocol in `SharedAnalytics`; register providers (Firebase, internal endpoint, noop) in composition root and inject the selected implementation into features.
- Testing: Use spy providers to assert event name/properties and ensure provider fan-out order is deterministic when multiple providers are registered.

## SharedAuth
- Purpose: Authentication state orchestration, secure token lifecycle, and session refresh contracts.
- Type: shared (interface + impl)
- Dependencies: `SharedStorage`
- Setup:
  1. `ios-app-manager module create SharedAuth --type shared`
  2. `ios-app-manager dep add SharedAuth --depends-on SharedStorage`
  3. `ios-app-manager dep list SharedAuth`
- Integration: Keep auth/session protocols in `SharedAuth`; implement token refresh and credential state machine in `SharedAuthImpl`; inject current auth context into networking and feature modules through composition root.
- Testing: Use in-memory token storage + fake clock to validate expiration, refresh, logout, and revoked-token flows without external services.

## SharedNetworking
- Purpose: Central HTTP client policy, request pipeline, and base URL/environment management.
- Type: shared (interface + impl)
- Dependencies: `SharedAuth`
- Setup:
  1. `ios-app-manager module create SharedNetworking --type shared`
  2. `ios-app-manager dep add SharedNetworking --depends-on SharedAuth`
  3. `ios-app-manager dep list SharedNetworking`
- Integration: Publish a minimal network client protocol in `SharedNetworking`; implement base URL resolution, interceptors, and auth header adapter in `SharedNetworkingImpl`; consume via interface-only dependency from feature/shared modules.
- Testing: Use custom `URLProtocol` stubs or mock transport adapters to verify request composition, retries, auth header injection, and error mapping.

## SharedStorage
- Purpose: Unified storage abstraction over UserDefaults/Keychain (including App Group variants).
- Type: shared (interface + impl)
- Dependencies: None
- Setup:
  1. `ios-app-manager module create SharedStorage --type shared`
  2. `ios-app-manager dep list SharedStorage`
- Integration: Keep storage keys and repository protocols in `SharedStorage`; implement UserDefaults/Keychain adapters in `SharedStorageImpl`; wire concrete stores once in composition root and inject typed storage interfaces downstream.
- Testing: Use in-memory key-value + keychain fakes to test namespace isolation, migration helpers, and read/write error handling.

## Extensibility: Adding New Shared Modules
- Follow the same catalog shape for each new module:
  1. Add section header: `## ModuleName`
  2. Fill fields in order: Purpose, Type, Dependencies, Setup, Integration, Testing
  3. Keep `Type` fixed to `shared (interface + impl)`
  4. Include exact `ios-app-manager module create <ModuleName> --type shared` command
  5. Add all dependency wiring commands via `ios-app-manager dep add ...`
  6. Document at least one integration rule (composition root, DI registration, or bundle linkage)
  7. Document at least one concrete mock/test-double pattern
- When adding a new module, update dependency chains in related module sections to keep the catalog graph accurate.
