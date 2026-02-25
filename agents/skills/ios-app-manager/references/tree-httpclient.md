# Tree: swift-httpclient (Relux Ecosystem)

## Purpose
Instruction tree for HTTP and WebSocket networking with `HttpClient` (`swift-httpclient` / `darwin-httpclient`) in Relux-style apps.

## Source Patterns (Relux)
- `swift-httpclient`: `RpcClient` actor, `IRpcAsyncClient`, `ApiEndpoint`, `ApiResponse`, `ApiError`, `RetryParams`.
- `swift-httpclient`: decorator-style interception via `RpcAsyncClientStubbable` and `PublishedStubbableWSClient` (no built-in middleware pipeline).
- `membrana-app`: IoC wiring of `IRpcAsyncClient` + `IRpcAsyncClientStubbable` + `IPublishedWSClient`.
- `membrana-app`: endpoint config objects, auth headers utility, and token refresh flow in token provider.
- `swift-httpclient` tests: `MockURLProtocol` + `URLSession.mockedWithProtocol(...)`.

## Instruction Tree

### 1. Choose client surface
1. For app/business logic:
   use `IRpcAsyncClient` (`async/await`) as the default abstraction.
2. For WebSocket with Combine consumers:
   use `IPublishedWSClient`; for stubbing, wrap with `IPublishedStubbableWSClient`.
3. For testability and module boundaries:
   inject protocols (`IRpcAsyncClient`, `IRpcAsyncClientStubbable`, `IPublishedWSClient`) via IoC.

### 2. Setup and configure HTTP client
1. Build base client:
   `RpcClient(sessionConfig: ApiSessionConfigBuilder.buildConfig(...), logger: ...)`.
2. Register in IoC as `.container` singleton.
3. If stubbing/interception is needed:
   wrap base client with `RpcAsyncClientStubbable(client: ...)` and inject the wrapper where appropriate.

### 3. Define endpoints
1. Use endpoint config structs/classes per feature:
   `ApiEndpoint(path: "\(baseUrl)/resource", type: .get/.post/...)`.
2. Keep endpoint definitions centralized in config, not scattered in fetchers.
3. Use `ApiFullEndpoint` only when needed; main ecosystem usage is direct `ApiEndpoint(path:type:)`.

### 4. Request/response mapping
1. Perform request:
   `client.performAsync(endpoint:headers:queryParams:bodyData:...)`.
2. Map `ApiResponse.data` to typed model via `JSONDecoder`.
3. Handle empty success explicitly:
   `204` can return `ApiResponse(data: nil, ...)`.
4. Map failures from `ApiError` using `responseCode` and payload context.

### 5. Middleware chain pattern (decorators)
1. There is no native request middleware chain API.
2. Build chain with wrappers in this order:
   auth header injection -> retry policy -> logging/metrics -> transport client.
3. Use existing wrappers where available:
   `RpcAsyncClientStubbable` and `PublishedStubbableWSClient`.
4. Keep wrappers protocol-based so they compose through IoC like any other dependency.

### 6. Error handling pattern
1. Treat non-2xx as `ApiError` and branch by `responseCode`.
2. Use `ApiError` metadata (`rawData`, `responseHeaders`, `requestType`, `url`) for diagnostics and mapping.
3. Convert low-level transport errors into domain errors in each fetcher/service.
4. Known library quirk:
   async `head(...)` currently calls `.delete`; avoid relying on `head` without validating behavior.

### 7. Authentication flows
1. Bearer token:
   set `HeaderKey.authorization` with `HeaderValue.bearer(token:)` (or equivalent app header utility).
2. API key:
   set `HeaderKey.apiKey` (`X-API-Key`) in request headers.
3. Token refresh flow (Relux/Membrana pattern):
   token provider checks token freshness -> calls refresh endpoint -> persists new token -> retries/continues request path.
4. Use per-request dynamic headers for app ID, request ID, language, and auth context as needed.

### 8. Retry strategy
1. For async client, use `RetryParams(count:delay:condition:)` or tuple `RequestRetrys`.
2. Retry only on explicit conditions (for example network/transient server errors).
3. Keep retry bounded; avoid unbounded loops.

### 9. Testing: mock transport layer
1. Unit-test transport behavior with `MockURLProtocol` and `URLSession.mockedWithProtocol(...)`.
2. Validate method, headers, body, and URL encoding through request handlers.
3. For feature-level tests, use `RpcAsyncClientStubbable` endpoint rules instead of real network.
4. For WebSocket flows, use `PublishedStubbableWSClient` or `WebSocketTasking` mocks.

### 10. Safety checks before merge
1. Every fetcher maps `ApiError.responseCode` to domain errors intentionally.
2. Auth headers are injected consistently for protected endpoints.
3. Retry conditions do not hide permanent failures.
4. Tests cover both success payload mapping and key failure branches.

