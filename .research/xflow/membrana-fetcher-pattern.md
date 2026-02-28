# Membrana Fetcher Pattern Analysis

Research date: 2026-03-01
Source: `/Users/alexis/src/membrana/app/Application/Sources/Modules/`

## Architecture Overview

Each feature module has its own fetcher in `Data/Api/Rpc/`:
```
Packages/<Name>/Sources/<Name>/
    Data/Api/Rpc/
        <Name>+Data+Api+Fetcher.swift       ← IFetcher protocol + Fetcher actor
        <Name>+Data+Api+Fetcher+Config.swift ← endpoint definitions
```

## Three Components

### 1. IFetcher Protocol
```swift
extension Tariff.Data.Api {
    protocol IFetcher: Sendable {
        func readBillingInfoB2B() async -> Result<Model.BillingInfoB2B, Tariff.Business.Err>
        func readBillingInfoB2CTariff() async -> Result<Model.BillingInfoB2CTariff, Tariff.Business.Err>
        func deleteAcc() async -> Result<Void, Tariff.Business.Err>
    }
}
```
- Async/await, returns `Result<T, ModuleErr>`
- Named for business intent, not HTTP method

### 2. Fetcher Actor
```swift
extension Tariff.Data.Api {
    actor Fetcher: IFetcher {
        private let client: IRpcAsyncClient
        private let tokenProvider: Auth.TokenProvider.Business.IAuthTokenProvider
        private let decoder: JSONDecoder = .init()
        private let config: Config

        init(
            client: IRpcAsyncClient,
            tokenProvider: Auth.TokenProvider.Business.IAuthTokenProvider,
            apiConfigManager: AppConfig.Business.IApiConfigManager
        ) {
            self.client = client
            self.tokenProvider = tokenProvider
            self.config = .init(resolveConfig: apiConfigManager.resolver)
        }
    }
}
```

Dependencies: HTTP transport + token provider + API config manager.

### 3. Endpoint Config
```swift
extension Tariff.Data.Api.Fetcher {
    final class Config: AppConfig.Data.ApiConfigurator, @unchecked Sendable {
        private var baseUrl: String { "\(resolveConfig().mobileApiBackendVersionedUrl)/\(UrlComponents.account)" }
        var getCurrentBillingInfoB2B: ApiEndpoint { .init(path: "\(baseUrl)/tariff-packages", type: .get) }
        var removeAcc: ApiEndpoint { .init(path: baseUrl, type: .delete) }
    }
}
```
- Inherits `AppConfig.Data.ApiConfigurator` → gets `resolveConfig: () -> Api.Configuration`
- Base URL dynamic (resolved at call time, supports env switching)
- Each endpoint = property returning `ApiEndpoint(path, type)`

## Endpoint Implementation Pattern

```swift
func readBillingInfoB2B() async -> Result<Model.BillingInfoB2B, Err> {
    // 1. Get token
    switch await tokenProvider.getAccessToken() {
    case let .failure(err):
        return .failure(.unauthorized(cause: err))
    case let .success(token):
        // 2. Call HTTP client
        let result = await client.performAsync(
            endpoint: config.getCurrentBillingInfoB2B,
            headers: ApiHeadersUtil.apiHeadersWithAuthorization(token: token, appId: await tokenProvider.appId),
            queryParams: [:],
            bodyData: nil,
            fileID: #fileID, functionName: #function, lineNumber: #line
        )
        // 3. Map response
        switch result {
        case let .success(response):
            do {
                let result = try decoder.decode(Model.BillingInfoB2B.self, from: response.data ?? .init())
                return .success(result)
            } catch {
                return .failure(.failedToObtainBillingInfoB2B(cause: error))
            }
        case let .failure(err):
            switch err.responseCode {
            case 401: return .failure(.unauthorized(cause: err))
            case 410: return .failure(.appVersionNotAllowed(cause: err))
            default: return .failure(.failedToObtainBillingInfoB2B(cause: err))
            }
        }
    }
}
```

Flow: token → HTTP call → deserialize → map errors.

## Per-Module Error Enums

```swift
extension Tariff.Business {
    enum Err: Error {
        case unauthorized(cause: Error)
        case appVersionNotAllowed(cause: Error)
        case failedToObtainBillingInfoB2C(cause: Error)
        case failedToDeleteAcc(cause: Error)
    }
}
```
Specific per-feature, not generic HTTP errors.

## Advanced Patterns

**Request body**: `["email": email].asJsonData` (dictionary → JSON via extension)

**Query params**: `queryParams: ["email": email]`

**Business rules in status codes**: 404 → empty list (not error), 429 → extract Retry-After header

**No service/repository layer on top**: Fetcher is directly used by Business/Flow layer.

## Key Takeaways for Scaffolding

1. **File structure**: `Data/Api/Rpc/` subdirectory inside feature namespace
2. **Actor-based**: Thread-safe by default
3. **3 injected deps**: `IRpcAsyncClient`, `IAuthTokenProvider`, `IApiConfigManager`
4. **Config inherits ApiConfigurator**: Gets dynamic base URL resolution
5. **Per-module error enum**: In `Business/` namespace
6. **No middleware**: Auth token handling is explicit in each endpoint call
7. **Testable**: All deps are protocols, mocked in tests

## Scaffolding Implications

What `module create --type relux-feature` should generate:
- `<Name>+Data+Api+Fetcher.swift` — IFetcher protocol + Fetcher actor stub
- `<Name>+Data+Api+Fetcher+Config.swift` — Config class inheriting ApiConfigurator
- `<Name>+Business+Err.swift` — Module error enum with common cases (unauthorized, appVersionNotAllowed)

What must exist before (prerequisites):
- `http-client setup` (IRpcAsyncClient registered in IoC)
- `token-provider setup` (IAuthTokenProvider registered)
- `app-config setup` (IApiConfigManager + ApiConfigurator registered)
