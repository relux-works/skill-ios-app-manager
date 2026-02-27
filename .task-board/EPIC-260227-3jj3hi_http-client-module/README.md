# EPIC-260227-3jj3hi: http-client-module

## Description
HttpClient — base HTTP transport module. Kit type (business logic, no UI) with interface/impl split. Pure transport layer, no auth, no API config.

Interface package (HttpClient):
- HttpClientProtocol — base HTTP transport interface
- Methods: request(url:method:headers:body:) -> Response, download, upload, etc.
- No knowledge of API configuration or authentication

Impl package (HttpClientImpl):
- HttpClient.Impl — wraps swift-httpclient library
- Configured with basic defaults (timeouts, retry policy, logging)
- Registered in IoC as HttpClientProtocol

Dependencies:
- swift-httpclient (external Swift package)
- IoC (registration)

NOT included (moved to separate modules):
- ApiConfigurator — separate foundation module for base URLs, API versioning, headers
- TokenProvider — separate foundation module for auth tokens

Feature modules (relux + backend) assemble their own API client from three foundation ingredients injected via protocols on init:
1. HttpClientProtocol (transport, from this module)
2. TokenProviding (auth, from TokenProvider module)
3. ApiConfiguring (URLs/config, from ApiConfigurator module)

CLI command: http-client setup — scaffolds module + IoC registration.

## Scope
(define epic scope)

## Acceptance Criteria
(define acceptance criteria)
