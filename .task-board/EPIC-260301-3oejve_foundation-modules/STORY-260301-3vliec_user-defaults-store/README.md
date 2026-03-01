# UserDefaults Store — foundation module with multi-protocol branching

## Description
Scaffold a UserDefaults wrapper as foundation module. Protocol-first, interface/impl split, IoC in foundation section.

## API Design

```swift
protocol IDefaultsStore: Sendable {
    func set<T>(_ value: T?, for key: String)
    func get<T>(for key: String) -> T?
    func setCodable<T: Encodable>(_ value: T?, for key: String)
    func getCodable<T: Decodable>(for key: String) -> T?
    func remove(for key: String)
}

struct DefaultsStore: IDefaultsStore, @unchecked Sendable {
    private let defaults: UserDefaults
    init(suite: String? = .none) {
        switch suite {
        case .none: self.defaults = .standard
        case let .some(name): self.defaults = UserDefaults(suiteName: name) ?? .standard
        }
    }
}
```

## Primitives handling in set<T>
Type switch for: String, Int, Float, Double, Bool, URL, Data, Date. Default branch calls defaults.set(value, forKey:).

## Codable support
setCodable: JSONEncoder → Data → defaults.set(data, forKey:)
getCodable: defaults.data(forKey:) → JSONDecoder → T
Separate methods (not runtime Encodable check) for explicit API.

## Multi-protocol branching pattern (KEY DECISION)
DefaultsStore is the universal impl. For specific use cases — define NARROW protocols that describe WHAT is stored:

```swift
// Feature-specific
protocol IOnboardingDefaults: Sendable {
    var hasCompletedOnboarding: Bool { get set }
    var lastOnboardingStep: String? { get set }
}

// Extension sharing
protocol ISharedExtensionDefaults: Sendable {
    var currentUserId: String? { get set }
    var currentOrganizationId: String? { get set }
    var shouldLoadNextContact: Bool { get set }
}
```

Each specialized protocol gets its own conformance extension on DefaultsStore (or a thin wrapper). Register specialized protocols in IoC — features depend on their narrow protocol, not full IDefaultsStore.

Benefits:
- Different instances (standard vs App Group suite) serve different protocols
- Features declare minimal dependency surface
- Easy to mock in tests (small protocol)
- Multiple DefaultsStore instances coexist (one per suite)

## Swift 6 compat
@preconcurrency import class Foundation.NSUserDefaults

## Scaffolding (via registry after module-registry epic)
- CLI: defaults-store setup (registered in module registry)
- Interface: Packages/DefaultsStore/ with IDefaultsStore protocol
- Impl: Packages/DefaultsStoreImpl/ with DefaultsStore struct
- Configuration+DefaultsStore.swift in app target (suite name from config app_groups)
- Registry patch: foundation section, buildDefaultsStore()
- .module-type marker: shared
- Templates via embed.FS
- Two-phase setup: plan shows files + usage guide with multi-protocol example → confirm → scaffold

## Dependencies
- Requires: ioc setup (Registry.swift)
- Optional: app_groups in config (for suite name)

## References
- SecureStore pattern (internal/securestore/)
- Membrana App Group UserDefaults usage
- connect-ios 5-mechanism data sharing (synthesis.md)
- User previous DefaultsStore impl with generic set/get

## Scope
(define story scope)

## Acceptance Criteria
- IDefaultsStore protocol with set/get for primitives + setCodable/getCodable for Codable types
- DefaultsStore struct with optional suite name init
- Interface/impl package split
- IoC registration in foundation section
- Configuration+DefaultsStore.swift generated
- Two-phase setup (plan+confirm) via module registry
- Usage guide includes multi-protocol branching pattern example
- Tests: unit + integration (same pattern as SecureStore)
- Demo app compiles with defaults-store setup in pipeline
