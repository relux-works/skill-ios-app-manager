# http-client setup.go + CLI + Configuration.HttpClient

## Description
Create internal/httpclient/setup.go and internal/cli/http_client.go:

setup.go:
- Adds swift-httpclient external dep to Package.swift
- Adds SwiftHTTPClient to Project.swift
- Creates Targets/<AppName>/Sources/Configuration/HttpClient/Configuration+HttpClient.swift
  with timeout constants (timeoutForResponse, timeoutResourceInterval)
- Patches Registry.swift: adds direct registration for IRpcAsyncClient with buildHttpClient() builder
  that uses Configuration.HttpClient params

cli:
- newHttpClientCommand() with setup subcommand, wired into root.go

AC: http-client setup runs, manifests updated, Configuration+HttpClient.swift created, Registry patched with builder.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
