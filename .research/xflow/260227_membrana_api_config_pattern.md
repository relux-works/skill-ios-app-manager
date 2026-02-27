# Membrana: AppConfigManager + ApiConfigurator Pattern

Research date: 2026-02-27
Source: `/Users/alexis/src/membrana/app/Application/Sources/`

---

## Architecture Summary

```
AppConfig.Business.Manager (SINGLETON, IoC .container)
    |
    +-- Stores env (prod/stage/dev) in Keychain
    +-- Implements IAppConfigManager (composed of 3 narrow protocols)
    |     IApiConfigManager   -> resolver() -> Configuration.Api
    |     ISSOConfigManager   -> sso
    |     ISupportChatsConfigManager -> supportChat
    |
    +-- DI: all modules receive narrow protocol via constructor injection
          |
          v
Per-Module Fetcher (actor)
    |
    +-- Receives: apiConfigManager: IApiConfigManager
    +-- Creates: Config(resolveConfig: apiConfigManager.resolver)
    |     Config inherits AppConfig.Data.ApiConfigurator (base class)
    |     Captures closure -> calls resolver() at call time, not init time
    |
    +-- Endpoint definitions are computed properties:
          var refresh: ApiEndpoint { .init(path: "\(baseUrl)/token/refresh", type: .post) }
          where baseUrl = "\(resolveConfig().mobileApiBackendUrl)/sso"
```

## Key Types

### 1. IAppConfigManager — Protocol Composition

```swift
// File: AppConfig+Business+Manager+Protocols.swift
protocol IApiConfigManager: Sendable {
    func resolver() -> AppConfig.Business.Model.Configuration.Api
    var api: AppConfig.Business.Model.Configuration.Api { get }
}
protocol ISSOConfigManager: Sendable {
    var sso: AppConfig.Business.Model.Configuration.SSO { get }
}
protocol ISupportChatsConfigManager: Sendable {
    var supportChat: AppConfig.Business.Model.Configuration.SupportChat { get }
}

protocol IAppConfigManager:
    IApiConfigManager, ISSOConfigManager, ISupportChatsConfigManager {
    var env: AppConfig.Business.Model.Env { get async }
    func updateEnvConfig(new config: AppConfig.Business.Model.Env) async
}
```

### 2. Configuration.Api — URL Building

```swift
// File: App+Config+Business+Model+Env+Configuration.swift
struct Api {
    private let baseUrl: String             // "gateway.klasta.me"
    private let publicFilesBaseUrl: String   // "files.klasta.me"
    private let mobileApiVersion: String     // "v3"
    private let signalsWsVersion: String     // "1.0"
    private let fileServiceVersion: String   // "v2"
    private let fssServiceVersion: String    // "1.0"

    var mobileApiBackendUrl: String { "https://\(baseUrl)" }
    var mobileApiBackendUnversionedUrl: String { "\(mobileApiBackendUrl)/api" }
    var mobileApiBackendVersionedUrl: String { "\(mobileApiBackendUrl)/api/\(mobileApiVersion)" }
    var signalsWsVersionedUrl: String { "wss://\(baseUrl)/ws/\(signalsWsVersion)/services" }
    var fileServiceVersionedUrl: String { "\(mobileApiBackendUrl)/api/\(fileServiceVersion)/files" }
    var fssServiceVersionedUrl: String { "\(mobileApiBackendUrl)/api/\(fssServiceVersion)/user-actions" }
}
```

### 3. Env Enum — Presets

```swift
// File: App+Config+Business+Model+Env.swift
enum Env: String { case prod, stage, dev }

extension Env {
    var config: Configuration {
        switch self {
        case .prod:  return .prodCfg
        case .stage: return .stageCfg
        case .dev:   return .devCfg
        }
    }
}
```

### 4. ApiConfigurator — Base Class for Module Configs

```swift
// File: AppConfig+Data+Configurator.swift
open class ApiConfigurator: @unchecked Sendable {
    let resolveConfig: @Sendable () -> Configuration.Api
    init(resolveConfig: @Sendable @escaping () -> Configuration.Api) {
        self.resolveConfig = resolveConfig
    }
}
```

### 5. UrlComponents — Shared Path Segments

```swift
// File: AppConfig+Data+ApiConfig.swift
static let profile = "profiles"
static let account = "account"
static let sso = "sso"
static let assist = "secretary"
// ... etc
```

## Module Consumption Pattern (Auth example)

### Module init — receives IApiConfigManager

```swift
// Auth+Module.swift
init(apiConfigManager: AppConfig.Business.IApiConfigManager, ...) async {
    let tokenFetcher = Auth.TokenProvider.Data.Api.Fetcher(
        client: rpcClient,
        apiConfigManager: apiConfigManager   // passed through
    )
}
```

### Fetcher — creates Config from resolver closure

```swift
// Auth+TokenProvider+Data+Api+Fetcher.swift
actor Fetcher {
    private let config: Config
    init(apiConfigManager: AppConfig.Business.IApiConfigManager, ...) {
        self.config = .init(resolveConfig: apiConfigManager.resolver)
    }
    func getAuthSettings(with token: String) async -> ... {
        let result = await client.performAsync(endpoint: config.authSettings, ...)
    }
}
```

### Config — inherits ApiConfigurator, defines endpoints

```swift
// Auth+TokenProvider+Data+Api+Fetcher+Config.swift
final class Config: AppConfig.Data.ApiConfigurator, @unchecked Sendable {
    private var baseUrl: String { "\(resolveConfig().mobileApiBackendUrl)/\(UrlComponents.sso)" }
    var refresh: ApiEndpoint { .init(path: "\(baseUrl)/token/refresh", type: .post) }
    var logout: ApiEndpoint { .init(path: "\(baseUrl)/logout", type: .post) }
    var authSettings: ApiEndpoint { .init(path: "\(baseUrl)/authentication-info", type: .get) }
}
```

## Same Pattern in Other Modules

| Module | Config class | Base URL source |
|--------|-------------|----------------|
| Auth.TokenProvider | `Auth.TokenProvider.Data.Api.Fetcher.Config` | `mobileApiBackendUrl + /sso` |
| Auth.PasswordFetcher | `Auth.Data.Api.PasswordFetcher.Config` | `mobileApiBackendVersionedUrl + /account` |
| Assist | `Assist.Data.Api.Fetcher.Config` | `mobileApiBackendVersionedUrl + /secretary` |
| Socket | `Socket.Data.Api.WSClient.Config` | `signalsWsVersionedUrl` |
| Tariff | `Tariff.Data.Api.Fetcher.Config` | `mobileApiBackendVersionedUrl + /...` |
| TempMailbox | `TempMailbox.Data.Api.Fetcher.Config` | `mobileApiBackendVersionedUrl + /temporary-mails` |

## IoC Registration

```swift
// Relux+Registry.swift
ioc.register(IAppConfigManager.self, lifecycle: .container, resolver: Self.buildAppConfigManager)
ioc.register(IApiConfigManager.self, lifecycle: .container, resolver: Self.buildApiConfigManager)

private static func buildAppConfigManager() -> IAppConfigManager {
    AppConfig.Business.Manager(keychain: sharedKeychain)
}
private static func buildApiConfigManager() -> IApiConfigManager {
    resolve(IAppConfigManager.self)  // same instance, narrower type
}
```

## Env Switching (Saga)

```swift
// App+Config+Business+Saga.swift
func updateEnvConfig(new env: Env) async -> Relux.Flow.Result {
    await appConfigManager.updateEnvConfig(new: env)
    await actions { Auth.Business.Effect.resetSsoConfig; ... }
    return .success
}
```

All Config instances that captured `resolver` closure automatically get new URLs on next call — no notification needed, pure lazy resolution.

## Design Principles

1. **Closure injection** — `resolver()` returns live config, not cached snapshot
2. **Protocol narrowing** — modules get only what they need (`IApiConfigManager`, not full `IAppConfigManager`)
3. **Inheritance for configs** — `ApiConfigurator` base class, module configs override with endpoints
4. **Actor isolation** — Fetchers are actors, Config is `@unchecked Sendable`
5. **Keychain persistence** — env survives app restart
6. **Computed endpoints** — every URL built at call time from current env

## Key File Paths

```
Modules/App/AppConfig/
  AppConfig+Namespace.swift
  Business/Middleware/AppConfig+Business+Manager.swift
  Business/Middleware/AppConfig+Business+Manager+Protocols.swift
  Business/Models/App+Config+Business+Model+Env.swift
  Business/Models/App+Config+Business+Model+Env+Configuration.swift
  Business/Models/App+Config+Business+Model+Env+Configuration+Presets.swift
  Data/AppConfig+Data+ApiConfig.swift                (UrlComponents)
  Data/AppConfig+Data+Configurator.swift             (ApiConfigurator base)

Modules/Auth/
  Auth+Module.swift
  TokenProvider/Data/Api/Auth+TokenProvider+Data+Api+Fetcher.swift
  TokenProvider/Data/Api/Auth+TokenProvider+Data+Api+Fetcher+Config.swift

Modules/Assist/
  Assist+Module.swift
  Data/Api/RPC/Assist+Data+Api+Fetcher.swift
  Data/Api/RPC/Assist+Data+Api+Fetcher+Config.swift

Modules/Socket/
  Socket+Module.swift
  Data/Socket+Data+Api+WSClient.swift
  Data/Socket+Data+Api+WSClient+Config.swift

IoC/Relux+Registry.swift
```
