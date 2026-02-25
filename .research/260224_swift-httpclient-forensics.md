# swift-httpclient — Product Forensics (TASK-260224-26011j)

- Date: 2026-02-24
- Target: `.temp/repos/swift-httpclient/` (git SHA `d0eb811fa67b3b474b11196787e67b00c8ca4703`)
- Package name / product: `swift-httpclient` → library product `HttpClient`. (`Package.swift#L1`–`Package.swift#L30`)
- Goal: Map the full API surface and internal architecture with emphasis on request/response handling, stubbing, error model, WebSockets, async/await, Combine, and TLS pinning.

---

## Highlights / Key Takeaways

1. **One module, multiple client “front-ends”:** `RpcClient` is a `public actor` that exposes *async/await* APIs (actor-isolated), plus *Combine*, *blocking sync*, and *callback* APIs via `nonisolated` methods/protocol conformances. (`Sources/RpcClient/RpcClient.swift#L4`–`Sources/RpcClient/RpcClient.swift#L25`, `Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L4`, `Sources/RpcClient/Sync/RpcClient+Sync.swift#L3`, `Sources/RpcClient/Callback/RpcClient+AsyncCallback.swift#L3`)
2. **No middleware chain abstraction:** There’s no explicit request/response middleware pipeline; instead, the project uses **decorator wrappers** (e.g., `RpcAsyncClientStubbable`, `PublishedStubbableWSClient`) as interception points. (`Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L4`–`Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L7`, `Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient.swift#L11`–`Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient.swift#L27`)
3. **Request/response model is intentionally low-level:** Responses are `ApiResponse(data: Data?, headers: ResponseHeaders, code: Int)` and errors are `ApiError` with `rawData`, `responseHeaders`, etc. There are **no Codable-based typed request/response helpers** in the public surface. (`Sources/ApiResponse.swift#L3`–`Sources/ApiResponse.swift#L23`, `Sources/ApiError.swift#L4`–`Sources/ApiError.swift#L99`)
4. **Retry support exists only for async/await** (and only on selected overloads), implemented via recursion + `Task.sleep` and an optional `condition(ApiError) -> Bool`. (`Sources/RetryParams.swift#L3`–`Sources/RetryParams.swift#L16`, `Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L47`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L78`, `Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L256`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L283`)
5. **WebSockets are split into two implementations:**
   - `WSClient` (experimental) returns `AsyncStream` of message results. (`Sources/WSClient/IWSClient.swift#L3`–`Sources/WSClient/IWSClient.swift#L103`)
   - `PublishedWSClient` (preferred) publishes messages + connection state via Combine and includes keep-alive + reconnect logic. (`README.md#L30`–`README.md#L32`, `Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L15`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L225`)
6. **Notable quirks/bugs baked into behavior/tests:**
   - `head(...)` in async & publisher clients uses `.delete` as the HTTP method. (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L128`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L137`, `Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L62`–`Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L68`, and tests assert this: `Tests/HttpClientTests/RpcClientAsyncTests.swift#L121`–`Tests/HttpClientTests/RpcClientAsyncTests.swift#L143`)
   - `PublishedWSClient.reconnect(with:)` blocks with `sleep(interval)` inside an actor, rather than `Task.sleep`. (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L126`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L143`)

---

## L1 — Recon (Surface Scan)

### Directory tree (high-level)

```
swift-httpclient/
├── Package.swift
├── README.md
├── Sources/
│   ├── (core types): ApiEndpoint, ApiError, ApiRequestType, ApiResponse, RetryParams, ApiSessionConfigBuilder
│   ├── Internal/ (URL building + helpers)
│   ├── Logging/
│   ├── RpcClient/ (Async, Sync, Combine, Callback, Curl)
│   ├── SSLPinning/
│   ├── TestUtils/
│   ├── Utils/
│   └── WSClient/ (WSClient + PublishedWSClient + stubbing)
└── Tests/HttpClientTests/ (Swift Testing framework)
```

### File counts (by folder)

- `Sources/`: 42 files (all Swift)
- `Tests/`: 19 files (all Swift)
- Total Swift files: 62

### Package configuration

- Swift tools: **6.0**. (`Package.swift#L1`)
- Platforms: iOS 13 / watchOS 6 / macOS 11 / tvOS 13. (`Package.swift#L6`–`Package.swift#L11`)
- Products: single library product **`HttpClient`**. (`Package.swift#L12`–`Package.swift#L16`)
- Dependencies: none. (`Package.swift#L19`–`Package.swift#L23`)

### README claims (validated against code)

- Supports async/await + Combine + callbacks + WebSockets + stubbing. (`README.md#L3`, validated throughout `Sources/RpcClient/*` and `Sources/WSClient/*`)
- SSL pinning helpers exist in `Sources/SSLPinning/`. (`README.md#L40`–`README.md#L42`, `Sources/SSLPinning/CertVerificationChallenge.swift#L12`)

---

## L2 — Deep Dive (All Source + Tests)

### 1) Public API surface (catalog)

This section lists the major publicly accessible types/protocols that define how consumers integrate.

#### Core HTTP model types

- `ApiRequestType`: HTTP method enum with padded `description`. (`Sources/ApiRequestType.swift#L1`–`Sources/ApiRequestType.swift#L24`)
- `ApiEndpoint`: `(path: String, type: ApiRequestType)`; `Hashable` so it can be used as a dictionary key for stubbing. (`Sources/ApiEndpoint.swift#L3`–`Sources/ApiEndpoint.swift#L18`)
- `ApiFullEndpoint`: `(baseUrl: URL, path: String, type: ApiRequestType)` with computed `url`. Marked with a `#warning` about endpoint semantics. (`Sources/ApiEndpoint.swift#L21`–`Sources/ApiEndpoint.swift#L40`)
- `ApiResponse`: `(data: Data?, headers: ResponseHeaders, code: Int)` and a case-insensitive `headerValue(forKey:)`. (`Sources/ApiResponse.swift#L3`–`Sources/ApiResponse.swift#L23`)
- `ApiError`: `Error` with `violation`, `message`, `rawData`, `responseCode`, `requestType`, request `headers/params`, and `responseHeaders`. (`Sources/ApiError.swift#L4`–`Sources/ApiError.swift#L16`, `Sources/ApiError.swift#L30`–`Sources/ApiError.swift#L99`)
  - `ApiError.ErrorViolation`: severity-ish enum including `authProblem`, `silent`, `warning`, `error`, `fatal`. (`Sources/ApiError.swift#L102`–`Sources/ApiError.swift#L119`)
- `RetryParams`: sendable retry configuration with `count`, `delay()`, and `condition(ApiError) -> Bool`. (`Sources/RetryParams.swift#L3`–`Sources/RetryParams.swift#L16`)

#### Typealiases + header helpers

The library models headers and query params as plain `[String: String]` with convenience constants/functions:

- Typealiases: `Headers`, `QueryParams`, `ResponseHeaders`, `ResponseCode`, plus `RequestRetrys` (tuple form of retries). (`Sources/Client+Typealiases.swift#L3`–`Sources/Client+Typealiases.swift#L12`)
- `ResponseCode` convenience flags and range checks. (`Sources/Client+Typealiases.swift#L13`–`Sources/Client+Typealiases.swift#L30`)
- Header keys (e.g. `.authorization`) and header values including `.bearer(token:)`. (`Sources/Client+Typealiases.swift#L42`–`Sources/Client+Typealiases.swift#L115`)

#### Request building

- `IRequestBuilder`: builds `URLRequest` for RPC and WS, and builds request URLs with query params. (`Sources/Internal/Request+Builder.swift#L3`–`Sources/Internal/Request+Builder.swift#L7`)
- `RequestBuilder`: trivial conformer; most consumers use the protocol defaults. (`Sources/Internal/Request+Builder.swift#L9`)

#### Logging

- `HttpClientLogging`: minimal logger protocol. (`Sources/Logging/Logger.swift#L2`–`Sources/Logging/Logger.swift#L4`)
- `DefaultLogger.shared`: prints to stdout. (`Sources/Logging/DefaultLogger.swift#L1`–`Sources/Logging/DefaultLogger.swift#L6`)

#### RPC clients (HTTP)

- `RpcClient`: `public actor` that holds `URLSession` and a logger and conforms to `IRequestBuilder`. (`Sources/RpcClient/RpcClient.swift#L4`–`Sources/RpcClient/RpcClient.swift#L25`, `Sources/RpcClient/RpcClient.swift#L43`)
- Protocols defining supported paradigms:
  - `IRpcAsyncClient`: async/await returning `Result<ApiResponse, ApiError>` and includes file/function/line metadata in the protocol. (`Sources/RpcClient/Async/IRpcClient+WithAsyncAwait.swift#L3`–`Sources/RpcClient/Async/IRpcClient+WithAsyncAwait.swift#L60`)
  - `IRpcPublisherClient`: Combine `AnyPublisher<ApiResponse, ApiError>`. (`Sources/RpcClient/Publisher/IRpcClient+WithPublisher.swift#L4`–`Sources/RpcClient/Publisher/IRpcClient+WithPublisher.swift#L42`)
  - `IRpcSyncClient`: blocking `Result<ApiResponse, ApiError>`. (`Sources/RpcClient/Sync/IRpcClient+Sync.swift#L3`–`Sources/RpcClient/Sync/IRpcClient+Sync.swift#L28`)
  - `IRpcCompletionClient`: callback style for GET/POST plus sync `delete/head` returning `Result`. (`Sources/RpcClient/Callback/IRpcClient+WithCallback.swift#L3`–`Sources/RpcClient/Callback/IRpcClient+WithCallback.swift#L33`)
- cURL helper:
  - `RpcClient.create_cURL(...)` stringifies a request as a `curl -vX ...` command. (`Sources/RpcClient/RpcClient+Curl.swift#L4`–`Sources/RpcClient/RpcClient+Curl.swift#L12`)

#### Stubbing (HTTP)

- `IRpcAsyncClientStubbable`: async client with stub rule management. (`Sources/RpcClient/AsyncStubable/IRpcAsyncClientStubbable.swift#L1`–`Sources/RpcClient/AsyncStubable/IRpcAsyncClientStubbable.swift#L6`)
- `RpcAsyncClientStubbable`: `public actor` decorator around `IRpcAsyncClient` that serves stubbed `ApiResponse` for matching `ApiEndpoint` keys. (`Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L4`–`Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L15`, `Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L18`–`Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L110`, `Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L168`–`Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L183`)
- “TestUtils” shipped in the main target:
  - `ApiResponse.stub(...)` and `ApiError.stub(...)`. (`Sources/TestUtils/ApiResponse+Stub.swift#L3`–`Sources/TestUtils/ApiResponse+Stub.swift#L10`, `Sources/TestUtils/ApiError+Stub.swift#L2`–`Sources/TestUtils/ApiError+Stub.swift#L14`)
  - `RpcClientWithAsyncAwaitMock`: actor mock for `IRpcAsyncClient` (records calls and returns configured results). (`Sources/TestUtils/AsyncRpcClient+Mock.swift#L3`–`Sources/TestUtils/AsyncRpcClient+Mock.swift#L78`)

#### WebSockets

- `WSClientError`: error enum shared by WS implementations. (`Sources/WSClient/WSClient+Error.swift#L3`–`Sources/WSClient/WSClient+Error.swift#L13`)
- `IWSClient` + `WSClient` (experimental):
  - `connect(...)` returns `AsyncStream<Result<Data, WSClientError>>`. (`Sources/WSClient/IWSClient.swift#L5`–`Sources/WSClient/IWSClient.swift#L10`)
  - `WSClient` is an actor using a `WebSocketTasking` factory for testability. (`Sources/WSClient/IWSClient.swift#L12`–`Sources/WSClient/IWSClient.swift#L41`)
- `WebSocketTasking`: protocol abstraction over `URLSessionWebSocketTask` (used by both WS clients). (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L4`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L13`)
- `IPublishedWSClient` + `PublishedWSClient` (preferred):
  - Publishes incoming messages via `msgPublisher` and connection state via `connectionPublisher`. (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L15`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L28`, `Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L78`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L85`)
  - Configuration via `PublishedWSClient.Config` with keep-alive and reconnect intervals and an async `headers` resolver closure. (`Sources/WSClient/PublishedWSClient/PublishedWSClient+Model+Config.swift#L8`–`Sources/WSClient/PublishedWSClient/PublishedWSClient+Model+Config.swift#L31`)
  - Delegate publishes status changes. (`Sources/WSClient/PublishedWSClient/PublishedWSClient+UrlSessionDelegate.swift#L4`–`Sources/WSClient/PublishedWSClient/PublishedWSClient+UrlSessionDelegate.swift#L23`, `Sources/WSClient/PublishedWSClient/PublishedWSClient+UrlSessionDelegate.swift#L27`–`Sources/WSClient/PublishedWSClient/PublishedWSClient+UrlSessionDelegate.swift#L45`)
- `PublishedStubbableWSClient`: decorator that stubs responses for outgoing messages by matching normalized JSON keys; merges its own subject with the base client’s `msgPublisher`. (`Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient.swift#L11`–`Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient.swift#L27`, `Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient.swift#L47`–`Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient.swift#L73`, `Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient.swift#L96`–`Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient.swift#L111`)
  - Stub model: `PublishedStubbableWSClient.Stub` and `Stub.OutgoingMsg`. (`Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient+Stub.swift#L4`–`Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient+Stub.swift#L47`)

#### TLS pinning (URLSessionDelegate helper)

- `ICertVerificationChallenge`: protocol alias of `URLSessionDelegate`. (`Sources/SSLPinning/CertVerificationChallenge.swift#L3`)
- `CertVerificationChallenge`: open class that validates server trust against local pinned certs by intersection/subset rules, only implemented for iOS 15+/macOS 12+. (`Sources/SSLPinning/CertVerificationChallenge.swift#L12`–`Sources/SSLPinning/CertVerificationChallenge.swift#L75`)
- `CertPublicKeyVerificationChallenge`: pins on public keys derived from certs. (`Sources/SSLPinning/CertVerificationChallenge.swift#L78`–`Sources/SSLPinning/CertVerificationChallenge.swift#L133`)

#### Concurrency helpers / utilities

- `UncheckedSendableWrapper<T>`: wraps non-Sendable payloads (used by `ResponseHeaders`). (`Sources/Utils/UncheckedSendableWrapper.swift#L1`–`Sources/Utils/UncheckedSendableWrapper.swift#L6`)
- Combine sendability shims: retroactive `@unchecked Sendable` for `AnyPublisher`, `Published.Publisher`, `PassthroughSubject`. (`Sources/Utils/Publisher+Sendable.swift#L3`–`Sources/Utils/Publisher+Sendable.swift#L5`)
- Async `sink` helper to allow `await` inside Combine’s `sink`. (`Sources/Utils/Publisher+AsyncSink.swift#L3`–`Sources/Utils/Publisher+AsyncSink.swift#L10`)
- JSON normalization for deterministic comparisons & stubbing key generation. (`Sources/Utils/Data+NormalizedJson.swift#L3`–`Sources/Utils/Data+NormalizedJson.swift#L39`)

---

### 2) HTTP request/response handling (RpcClient)

#### URL building

- `buildRequestUrl(path:queryParams:)`:
  - Percent-encodes the full input `path` using a custom `CharacterSet.rfc3986Unreserved`. (`Sources/Internal/Request+Builder.swift#L41`–`Sources/Internal/Request+Builder.swift#L47`, `Sources/Internal/Charset+RFC3986supportedChars.swift#L3`–`Sources/Internal/Charset+RFC3986supportedChars.swift#L6`)
  - Sorts query params by key, and replaces `+` with `%2B` in the percent-encoded query string. (`Sources/Internal/Request+Builder.swift#L49`–`Sources/Internal/Request+Builder.swift#L55`)
  - This yields deterministic URL strings (stable ordering) — useful for caching and tests.

#### URLRequest building

- `buildRpcRequest(url:type:headers:bodyData:)`:
  - Sets `httpMethod`.
  - Sets headers.
  - Assigns `httpBody` for `.post/.put/.delete/.patch`. (`Sources/Internal/Request+Builder.swift#L12`–`Sources/Internal/Request+Builder.swift#L26`)

#### Response parsing rules

In async/await and Combine implementations, the logic is broadly:

- Non-HTTP response → `ApiError(responseCode: 0, message: "no response: ...")`. (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L316`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L327`, similar mapping in publisher: `Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L111`–`Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L122`)
- Status code not in `[200, 300)` → `ApiError(message: "bad response: ...", responseHeaders: ...)`. (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L329`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L340`, publisher: `Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L124`–`Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L135`)
- `204 No Content` → success with `ApiResponse(data: nil, ...)`. (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L341`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L345`, publisher: `Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L136`–`Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L140`, sync: `Sources/RpcClient/Sync/RpcClient+Sync.swift#L179`–`Sources/RpcClient/Sync/RpcClient+Sync.swift#L183`)
- Otherwise: success with `ApiResponse(data: data, ...)`. (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L347`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L349`)

#### Error reporting payload

`ApiError` carries:

- “Who sent it”: `sender: Any.Type?` derived via `Mirror`. (`Sources/ApiError.swift#L70`)
- Request metadata: `url`, `requestType`, `headers`, `params`. (`Sources/ApiError.swift#L11`–`Sources/ApiError.swift#L15`)
- Response metadata: `responseCode`, `rawData`, and `responseHeaders`. (`Sources/ApiError.swift#L8`, `Sources/ApiError.swift#L12`, `Sources/ApiError.swift#L16`)

#### Logging and observability

- Logs include a cURL representation plus the response (including decoded UTF-8 body when possible). (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L311`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L313`, `Sources/RpcClient/RpcClient+Curl.swift#L4`–`Sources/RpcClient/RpcClient+Curl.swift#L12`)
- `stringifyData` strips a leading `<!doctype html>` prefix (best-effort). (`Sources/RpcClient/RpcClient.swift#L29`–`Sources/RpcClient/RpcClient.swift#L40`)

**Risk note:** cURL logging includes headers and body; if `Authorization` or PII is present, logs can leak secrets unless the injected logger redacts.

---

### 3) Retry behavior (async/await)

Retry logic exists as overloads that accept either:

- `RequestRetrys` tuple `(count, delay)` → converted into `RetryParams`. (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L25`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L45`, `Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L238`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L254`)
- `RetryParams` which also includes a retry `condition(ApiError)`. (`Sources/RetryParams.swift#L3`–`Sources/RetryParams.swift#L16`)

Implementation details:

- Executes request once; on `.failure(err)`, it checks `retrys.count > 0` and `retrys.condition(err)`. (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L59`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L62`, `Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L266`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L269`)
- Sleeps using `Task.sleep` with `delay()` seconds converted to nanoseconds. (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L63`, `Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L270`)
- Recurses with `count - 1`. (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L69`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L77`, `Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L274`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L282`)

**Behavioral consequence:** Retries re-run the full request builder and logging. There is no jitter/backoff policy beyond `delay()`, and no built-in 429/5xx specific strategies (left to `condition`).

---

### 4) “Middleware chain” analysis

There is no explicit middleware protocol like `RequestMiddleware`/`ResponseMiddleware`.

Instead, the library uses **wrappers/decorators**:

- `RpcAsyncClientStubbable` wraps `IRpcAsyncClient`, intercepting `get(...)` / `performAsync(...)` to serve stub responses for specific `ApiEndpoint` keys. (`Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L4`–`Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L39`, `Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L85`–`Sources/RpcClient/AsyncStubable/RpcAsyncClientStubbable.swift#L160`)
- `PublishedStubbableWSClient` wraps `IPublishedWSClient`, intercepting `send(...)` to inject stubbed inbound messages. (`Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient.swift#L47`–`Sources/WSClient/PublishedStubbableWSClient/PublishedStubbableWSClient.swift#L63`)

This is effectively a “middleware chain” *pattern* (decorator composition), but it’s not generalized (no standard hook points for auth, tracing, metrics, caching, etc.).

---

### 5) Auth handling

No auth module exists; auth is header-driven:

- `HeaderKey.authorization` constant. (`Sources/Client+Typealiases.swift#L46`)
- `HeaderValue.bearer(token:)` and `.basic(token:)` helpers. (`Sources/Client+Typealiases.swift#L84`–`Sources/Client+Typealiases.swift#L90`)

In practice, consumers provide auth headers per call or via their own wrapper.

---

### 6) Codable integration

The module does not expose typed `Codable` APIs:

- Requests accept `bodyData: Data?` (async/publisher) or `Data` (sync post/put). (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L4`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L11`, `Sources/RpcClient/Sync/RpcClient+Sync.swift#L17`–`Sources/RpcClient/Sync/RpcClient+Sync.swift#L45`)
- Responses return `Data?` via `ApiResponse.data`. (`Sources/ApiResponse.swift#L3`–`Sources/ApiResponse.swift#L6`)
- JSON normalization utilities use `JSONSerialization`, not `Codable`. (`Sources/Utils/Data+NormalizedJson.swift#L5`, `Sources/Utils/Data+NormalizedJson.swift#L22`–`Sources/Utils/Data+NormalizedJson.swift#L29`)

**Implication:** Consumers must own JSON encoding/decoding (e.g. `JSONEncoder`/`JSONDecoder`) and error payload decoding conventions.

---

### 7) WebSocket architecture details

#### A) `WSClient` (experimental)

- Designed as an actor around a `WebSocketTasking` abstraction. (`Sources/WSClient/IWSClient.swift#L12`–`Sources/WSClient/IWSClient.swift#L18`)
- `connect(...)` builds a request, creates a task via a factory, resumes, and returns an `AsyncStream` that loops calling `receive()`. (`Sources/WSClient/IWSClient.swift#L43`–`Sources/WSClient/IWSClient.swift#L83`, `Sources/WSClient/IWSClient.swift#L85`–`Sources/WSClient/IWSClient.swift#L103`)
- On receive error, it disconnects and yields `.failure(.disconnected)`. (`Sources/WSClient/IWSClient.swift#L99`–`Sources/WSClient/IWSClient.swift#L103`)

#### B) `PublishedWSClient` (preferred)

Core ideas:

- Configuration step creates a new `URLSession` using the provided `sessionConfig` and a custom `UrlSessionDelegate`. (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L100`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L105`, `Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L243`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L246`)
- Connection publishes:
  - messages via `PassthroughSubject`. (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L78`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L81`)
  - connection status via `delegate.$status`. (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L83`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L85`)
- Keep-alive uses a publisher factory (default: `Timer.publish` on `.main`) and the custom async `sink`. (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L165`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L175`, `Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L248`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L250`, `Sources/Utils/Publisher+AsyncSink.swift#L3`–`Sources/Utils/Publisher+AsyncSink.swift#L10`)
- Reconnect is triggered after `.failure(.transportError(...))`, using `Task.detached` to call `reconnect(with:)`. (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L197`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L203`, `Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L208`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L225`)

Key concern:

- `reconnect(with:)` uses `sleep(interval)` (blocking). For an async actor method, replacing with `try? await Task.sleep(...)` would avoid blocking a thread. (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L126`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L143`)

---

### 8) TLS pinning design

- Loads pinned certificates from file URLs into a `Set<SecCertificate>`. (`Sources/SSLPinning/CertVerificationChallenge.swift#L27`–`Sources/SSLPinning/CertVerificationChallenge.swift#L33`)
- On iOS 15+/macOS 12+, it compares the server’s certificate chain to pinned certs either by intersection (“any”) or by subset (“all”). (`Sources/SSLPinning/CertVerificationChallenge.swift#L52`–`Sources/SSLPinning/CertVerificationChallenge.swift#L69`)
- Public-key variant extracts `SecKey` and compares key blobs. (`Sources/SSLPinning/CertVerificationChallenge.swift#L78`–`Sources/SSLPinning/CertVerificationChallenge.swift#L127`)

---

### 9) Tests overview (what’s tested and how)

Tests use the Swift 6 `Testing` framework (`@Suite`, `@Test`, `#expect`, `#require`). Example: `ApiSessionConfigBuilderTests`. (`Tests/HttpClientTests/ApiSessionConfigBuilderTests.swift#L1`–`Tests/HttpClientTests/ApiSessionConfigBuilderTests.swift#L26`)

Coverage highlights:

- URL building encodes plus signs and sorts query items. (`Tests/HttpClientTests/RequestBuilderTests.swift#L6`–`Tests/HttpClientTests/RequestBuilderTests.swift#L20`)
- RPC (async/Combine/sync/callback) behaviors are tested using a `MockURLProtocol` URLProtocol-based interceptor. (`Tests/HttpClientTests/Helpers/MockURLProtocol.swift#L1`–`Tests/HttpClientTests/Helpers/MockURLProtocol.swift#L61`, `Tests/HttpClientTests/Helpers/URLSession+Mock.swift#L3`–`Tests/HttpClientTests/Helpers/URLSession+Mock.swift#L8`)
- WebSockets are tested via `MockWebSocketTask` implementing `WebSocketTasking`. (`Tests/HttpClientTests/PublishedWSClientTests.swift#L140`–`Tests/HttpClientTests/PublishedWSClientTests.swift#L250`)
- SSL pinning logic is tested by generating in-memory cert data and constructing `SecTrust`. (`Tests/HttpClientTests/SSLPinningTests.swift#L23`–`Tests/HttpClientTests/SSLPinningTests.swift#L207`)

---

## L3 — Domain Synthesis (Grouped Findings)

### Domain: HTTP RPC

- **Interfaces:** `IRpcAsyncClient`, `IRpcPublisherClient`, `IRpcSyncClient`, `IRpcCompletionClient`. (protocol sources above)
- **Primitive exchange:** `Data?` + headers + status code.
- **Error model:** `ApiError` is the central carrier; includes both request and response metadata.
- **Retry:** async only; simple recursive retry.
- **Observability:** default logger prints all details (incl. cURL + body).

### Domain: WebSockets

- `WSClient` gives an async stream interface (low-level, experimental).
- `PublishedWSClient` gives Combine-based message and connection streams, with keep-alive and reconnect.
- `PublishedStubbableWSClient` acts as a stubbing adapter for `PublishedWSClient`-style clients by normalizing outgoing JSON to a canonical key.

### Domain: Testing & Stubbing

- Library bakes in “test util” APIs as `public` inside the main target (`ApiResponse.stub`, `ApiError.stub`, mock async client actor). This is convenient but expands the public API and may be undesirable in production distributions. (`Sources/TestUtils/ApiResponse+Stub.swift#L3`–`Sources/TestUtils/ApiResponse+Stub.swift#L10`, `Sources/TestUtils/AsyncRpcClient+Mock.swift#L3`)
- URLProtocol-based mocking provides deterministic tests for request method / status handling without network calls.

### Domain: Security (TLS pinning)

- Pinning helpers are present and test-covered, but only use modern APIs (iOS 15+/macOS 12+) and fall back to default handling for older OS versions. (`Sources/SSLPinning/CertVerificationChallenge.swift#L44`–`Sources/SSLPinning/CertVerificationChallenge.swift#L73`)

### Domain: Concurrency & correctness

- Actors are used for shared mutable state in stubs and websocket clients.
- Some APIs are declared `nonisolated` to provide sync / publisher interfaces; this weakens the “actor serializes access” mental model but can be acceptable given `URLSession` thread-safety and immutable stored properties.
- `PublishedWSClient.reconnect` uses `sleep` (blocking) and should be treated as a potential responsiveness/perf footgun.

---

## L4 — Product Synthesis (Product-Level Understanding)

### Executive summary

`swift-httpclient` is a small, dependency-free Swift networking toolkit that offers:

- HTTP requests via `RpcClient` in three paradigms: async/await, Combine, and blocking sync; plus a callback-style helper. (`README.md#L3`, `Sources/RpcClient/RpcClient.swift#L4`)
- A pragmatic, low-level data model (`ApiResponse`/`ApiError`) rather than typed `Codable` models. (`Sources/ApiResponse.swift#L3`, `Sources/ApiError.swift#L4`)
- WebSocket support in both async-stream and Combine-published styles, with testable abstractions and a stubbing decorator. (`README.md#L30`–`README.md#L35`)
- TLS pinning delegates for URLSession. (`README.md#L40`–`README.md#L42`, `Sources/SSLPinning/CertVerificationChallenge.swift#L12`)

### Architecture overview

```
                    ┌──────────────────────────────────────────────┐
                    │                  HttpClient                  │
                    │          (SPM module / library target)        │
                    └──────────────────────────────────────────────┘

HTTP:
  ApiEndpoint + ApiRequestType + Headers/QueryParams
           │
           ▼
  IRequestBuilder (URL + URLRequest building) ──────► URLSession
           │                                            │
           ▼                                            ▼
       RpcClient (actor) ───────────► ApiResponse / ApiError (+ logging)
          │     │     │
          │     │     └─ Callbacks (IRpcCompletionClient)
          │     └─────── Combine (IRpcPublisherClient)
          └───────────── Blocking sync (IRpcSyncClient)

Stubbing:
  RpcAsyncClientStubbable (actor) wraps any IRpcAsyncClient

WebSockets:
  WSClient (actor, experimental) → AsyncStream of message results
  PublishedWSClient (actor) → Combine publishers (msgs + connection status)
  PublishedStubbableWSClient (actor) wraps IPublishedWSClient, injects stubbed msgs

Security:
  CertVerificationChallenge / CertPublicKeyVerificationChallenge (URLSessionDelegate)
```

### Integration points with other relux repos (observed)

`membrana-app` registers and composes HttpClient components in IoC:

- Registers `IRpcAsyncClient` as `RpcClient(sessionConfig:..., logger: ...)`. (`.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift#L322`–`.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift#L331`)
- Wraps it into `IRpcAsyncClientStubbable` via `RpcAsyncClientStubbable(client: ...)`. (`.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift#L316`–`.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift#L320`)
- Registers WebSockets as `PublishedWSClient` and wraps into `PublishedStubbableWSClient`. (`.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift#L333`–`.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift#L343`)

`relux-sample` provides mocks conforming to `IRpcAsyncClient`, `IWSClient`, and `IPublishedWSClient`, suggesting these protocols are the intended seam for dependency injection. (`.temp/repos/relux-sample/TestsSupport/RpcAsyncClientMock.swift#L5`, `.temp/repos/relux-sample/TestsSupport/WSClientMock.swift#L5`, `.temp/repos/relux-sample/TestsSupport/PublishedWSClientMock.swift#L6`)

### Recommendations (for `relux-manager` CLI component)

If `relux-manager` is a CLI that needs HTTP and/or WebSockets:

1. **Prefer `IRpcAsyncClient` + `RpcClient` for HTTP** (async/await fits CLI concurrency best). Use a custom `HttpClientLogging` implementation that supports log levels and redacts secrets.
2. **Add a thin, CLI-focused typed layer** on top of `ApiResponse/ApiError`:
   - `performJSON<T: Decodable>(...) -> Result<T, ApiError>` (decode from `ApiResponse.data`)
   - `encodeBody(_ value: some Encodable) -> Data` helpers
   - unified error decoding (e.g. decode JSON error payloads into structured CLI messages)
3. **Implement middleware as decorators** (consistent with existing patterns):
   - `AuthHeaderClient`: wraps `IRpcAsyncClient`, injects auth headers (bearer token, api key).
   - `RetryingClient`: wraps and provides exponential backoff / jitter keyed off `ApiError.responseCode`.
   - `TracingClient`: adds `X-Request-ID` etc (leveraging `HeaderKey` constants). (`Sources/Client+Typealiases.swift#L68`–`Sources/Client+Typealiases.swift#L71`)
4. **If WebSockets are required:** evaluate whether Combine is acceptable in the CLI runtime.
   - If not, prefer `WSClient` (AsyncStream) or create an adapter that converts `PublishedWSClient.msgPublisher` into `AsyncStream`.
5. **Fix/avoid known quirks** for CLI correctness:
   - Avoid `RpcClient.head(...)` due to DELETE behavior (or patch it in the CLI wrapper). (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L128`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L137`)
   - Avoid `PublishedWSClient.sleep` in reconnect if running in latency-sensitive CLI; consider patching upstream.

### Suggested improvements to the library (backlog)

These are concrete opportunities surfaced by the forensics pass:

- **Correctness:**
  - Fix `.head` methods in async/publisher to use `.head` instead of `.delete` (update tests accordingly). (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L128`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L137`, `Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L62`–`Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L68`)
  - Pass `bodyData` through `patch(...)` in publisher client; consider adding patch body support to sync client. (`Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L70`–`Sources/RpcClient/Publisher/RpcClient+Publisher.swift#L80`, `Sources/RpcClient/Sync/RpcClient+Sync.swift#L115`–`Sources/RpcClient/Sync/RpcClient+Sync.swift#L120`)
  - Callback client should accept `2xx` and handle `204`. (`Sources/RpcClient/Callback/RpcClient+AsyncCallback.swift#L126`–`Sources/RpcClient/Callback/RpcClient+AsyncCallback.swift#L139`)
- **Concurrency/perf:**
  - Replace `sleep(interval)` with `Task.sleep` in `PublishedWSClient.reconnect`. (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L126`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L143`)
- **API clarity:**
  - Resolve the `#warning` about endpoint semantics: split base URL + path cleanly (today `ApiEndpoint.path` often contains a full URL). (`Sources/ApiEndpoint.swift#L21`–`Sources/ApiEndpoint.swift#L23`)
  - Consider whether “TestUtils” should be public in the shipping module; if not, move to a separate test-support target.
- **Developer experience:**
  - Add (optional) typed `Codable` helpers without losing the low-level `Data` API (keep both).

---

## Fact-checking notes / verification checklist

- Package has **no external dependencies** and exposes a single library target. (`Package.swift#L12`–`Package.swift#L28`)
- HTTP async/await uses `URLSession.data(for:)` and treats non-2xx as `ApiError`. (`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L314`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L340`)
- Retry uses `RetryParams.count/delay/condition` and `Task.sleep`. (`Sources/RetryParams.swift#L3`–`Sources/RetryParams.swift#L16`, `Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L59`–`Sources/RpcClient/Async/RpcClient+AsyncAwait.swift#L77`)
- WebSocket keep-alive uses Combine Timer + async sink helper. (`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L165`–`Sources/WSClient/PublishedWSClient/PublishedWSClient.swift#L175`, `Sources/Utils/Publisher+AsyncSink.swift#L3`–`Sources/Utils/Publisher+AsyncSink.swift#L10`)
- Integration in `membrana-app` matches the intended seams: `IRpcAsyncClient` + `IRpcAsyncClientStubbable` + `IPublishedWSClient` + stubbing wrapper. (`.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift#L50`–`.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift#L57`, `.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift#L316`–`.temp/repos/membrana-app/Application/Sources/IoC/Relux+Registry.swift#L343`)

