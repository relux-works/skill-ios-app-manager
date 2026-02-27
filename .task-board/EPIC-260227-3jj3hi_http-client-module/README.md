# HttpClient IoC registration

## Description
HttpClient — NOT a separate module. IoC registration of swift-httpclient RpcClient type with config params from Configuration.HttpClient.

What http-client setup does:
1. Adds swift-httpclient as external dep to root Package.swift
2. Adds .external(name: "SwiftHTTPClient") to Project.swift
3. Adds Configuration.HttpClient extension with timeout params to Packages/Configuration/
4. Patches Registry.swift: import SwiftHTTPClient + import Configuration, builder buildHttpClient() using Configuration.HttpClient params, registration in Foundation section

Dependencies: IoC (for registration), Configuration (for params), swift-httpclient (external).
No Packages/HttpClient/, no Module.Interface/Impl, no .module-type.

## Scope
http-client setup CLI command: add external dep + IoC registration in Registry

## Acceptance Criteria
1. http-client setup adds swift-httpclient to Package.swift
2. SwiftHTTPClient added to Project.swift dependencies
3. Registry.swift gets HTTPClient registration in Foundation section
4. make test green, demo app builds
5. Idempotent (re-running is safe)
