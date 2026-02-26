# EPIC-260227-3jj3hi: http-client-module

## Description
Base HttpClient feature module (not relux). Shared dependency for all feature modules to provide unified HTTP configuration.

Module structure (interface/impl split):

Interface package (HttpClient):
- HttpClient protocol — base API client interface
- ApiConfigurator protocol — HTTP configuration

Impl package (HttpClientImpl):
- HttpClient.Impl — concrete implementation, depends on ApiConfigurator and TokenProvider (both via protocols/interfaces)
- Default ApiConfigurator implementation lives here — must be instantiated externally and injected as a shared dependency

IoC resolution chain:
1. ApiConfigurator resolved → shared instance configured with env/baseURL/headers
2. TokenProvider resolved → provides access tokens (from TokenProvider epic)
3. HttpClient resolved → HttpClient.Impl(apiConfigurator: ApiConfigurator, tokenProvider: TokenProvider)
4. Feature modules resolved → FeatureModule.Impl(httpClient: HttpClient)

All dependencies through interfaces. HttpClient is injected into every feature module that needs API access.

CLI command to scaffold this module + IoC registration. Depends on TokenProvider epic being done first.

Reference implementation: membrana-ios-app (/Users/alexis/src/membrana/app/)

Key patterns from reference:
- ApiConfigurator stores a CLOSURE resolveConfig: () -> Configuration.Api, not the config itself. This allows runtime env switching without rebuilding fetchers.
- Configuration.Api is a value type with private base fields (baseUrl, apiVersion, etc.) and computed URL properties (mobileApiBackendVersionedUrl, signalsWsVersionedUrl, etc.)
- Env is persisted in Keychain, cached in memory with NSLock
- Manager singleton implements IAppConfigManager (full) and IApiConfigManager (subset) — registered separately in IoC for ISP
- Fetcher gets three deps: IRpcAsyncClient (http), IAuthTokenProvider (tokens), Config (URLs via closure)
- Each feature Fetcher.Config inherits from base ApiConfigurator and builds its own ApiEndpoints using resolveConfig() closure
- resolver() method returns the closure itself (not the value) for passing by reference into Fetcher.Config constructors

## Scope
(define epic scope)

## Acceptance Criteria
(define acceptance criteria)
