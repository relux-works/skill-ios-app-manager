# TASK-260302-2gjbtw: add-external-dep-struct

## Description
Add ExternalDep struct and ExternalDeps field to registry.Module. Populate ExternalDeps in all register.go files that have external Swift package dependencies.

ExternalDep struct:
- URL string (git repo URL)
- Version string (semver, e.g. "1.0.1")
- Product string (Swift product name for Project.swift, e.g. "SwiftIoC")
- Package string (optional, SPM package name if different from Product — used for Package.swift .package(name:) when needed)

Modules with external deps:
- IoC: swift-ioc 1.0.1, product SwiftIoC
- Relux: swiftui-relux 8.0.1, product SwiftUIRelux (NOTE: also needs swift-relux but that is added by module create --type relux-feature, not by relux setup)
- HttpClient: swift-httpclient 6.0.0, product HttpClient, package name "HttpClient"

Modules WITHOUT external deps: SecureStore, TokenProvider, AppConfig, Utilities

Also add ExternalDeps to registry_test.go coverage.

IMPORTANT: Do NOT remove anything from setup.go files yet — only ADD the ExternalDeps declarations to register.go. The next task handles the migration.

AC:
- ExternalDep struct exported from registry package
- ExternalDeps field on Module struct
- 3 modules have ExternalDeps populated in register.go
- 4 modules have empty ExternalDeps
- Tests pass, lint clean

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
