# DefaultsStore — Design Document

## API

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

## Primitives: set<T> type switch

```swift
func set<T>(_ value: T?, for key: String) {
    switch value {
    case let v as Int?:    defaults.set(v, forKey: key)
    case let v as String?: defaults.set(v, forKey: key)
    case let v as Float?:  defaults.set(v, forKey: key)
    case let v as Double?: defaults.set(v, forKey: key)
    case let v as Bool?:   defaults.set(v, forKey: key)
    case let v as URL?:    defaults.set(v, forKey: key)
    case let v as Data?:   defaults.set(v, forKey: key)
    case let v as Date?:   defaults.set(v, forKey: key)
    default:               defaults.set(value, forKey: key)
    }
}

func get<T>(for key: String) -> T? {
    defaults.object(forKey: key) as? T
}
```

## Codable support

```swift
func setCodable<T: Encodable>(_ value: T?, for key: String) {
    guard let value else {
        defaults.removeObject(forKey: key)
        return
    }
    let data = try? JSONEncoder().encode(value)
    defaults.set(data, forKey: key)
}

func getCodable<T: Decodable>(for key: String) -> T? {
    guard let data = defaults.data(forKey: key) else { return nil }
    return try? JSONDecoder().decode(T.self, from: data)
}
```

Separate methods — explicit API, not runtime Encodable check.

## Multi-protocol branching pattern (KEY ARCHITECTURE)

DefaultsStore is the universal implementation. For specific use cases, define NARROW protocols:

```swift
// Feature-specific — app-only storage
protocol IOnboardingDefaults: Sendable {
    var hasCompletedOnboarding: Bool { get set }
    var lastOnboardingStep: String? { get set }
}

// Extension sharing — App Group suite
protocol ISharedExtensionDefaults: Sendable {
    var currentUserId: String? { get set }
    var currentOrganizationId: String? { get set }
    var shouldLoadNextContact: Bool { get set }
}
```

Conformance via extension on DefaultsStore (or thin wrapper):

```swift
extension DefaultsStore: IOnboardingDefaults {
    var hasCompletedOnboarding: Bool {
        get { get(for: Keys.hasCompletedOnboarding) ?? false }
        set { set(newValue, for: Keys.hasCompletedOnboarding) }
    }
    // ...
}
```

IoC registration — multiple instances:

```swift
// App-only store (standard UserDefaults)
ioc.register(IOnboardingDefaults.self, lifecycle: .container) {
    DefaultsStore() // no suite = .standard
}

// Shared with extensions (App Group suite)
ioc.register(ISharedExtensionDefaults.self, lifecycle: .container) {
    DefaultsStore(suite: Configuration.DefaultsStore.sharedSuite)
}
```

Features depend on narrow protocol:

```swift
actor Onboarding.Flow {
    private let defaults: IOnboardingDefaults // not IDefaultsStore
    init(defaults: IOnboardingDefaults) { ... }
}
```

Benefits:
- Different instances (standard vs App Group) serve different protocols
- Features declare minimal dependency surface
- Easy to mock in tests (small protocol, not the whole store)
- Multiple DefaultsStore instances coexist

## Swift 6

```swift
@preconcurrency import class Foundation.NSUserDefaults
```

## User's previous impl (reference)

```swift
struct DefaultsStore: IDefaultsStore {
    private let defaults: UserDefaults
    private let suiteName: String?
    init(suite: String? = .none) {
        self.suiteName = suite
        switch suite {
            case .none: self.defaults = .standard
            case let .some(name): self.defaults = UserDefaults(suiteName: name) ?? .standard
        }
    }
}
```

## Scaffolding

- Interface: Packages/DefaultsStore/Sources/DefaultsStore/IDefaultsStore.swift
- Impl: Packages/DefaultsStoreImpl/Sources/DefaultsStoreImpl/DefaultsStore.swift
- Config: Targets/<App>/Sources/Configuration/Configuration+DefaultsStore.swift
- Registry: foundation section, buildDefaultsStore()
- .module-type: shared

## Dependencies

- Requires: ioc setup
- Optional: app_groups in config (for suite name constant)

## References

- SecureStore (internal/securestore/) — same scaffold pattern
- Membrana App Group UserDefaults — .research/xflow/membrana-fetcher-pattern.md
- connect-ios data sharing — .research/xflow/connect-ios-audit/synthesis.md (A2/A3 deep dive)
