# TASK-260302-evoxt1: generic-dep-application

## Description
Make setup_command.go automatically apply ExternalDeps from registry.Module BEFORE calling mod.Setup(). Then remove all hardcoded external dep logic from individual setup.go files.

In setup_command.go (RunE, after config loading, before Plan):
1. Read mod.ExternalDeps
2. For each ExternalDep: call deps.AddExternalDep(dep.URL, dep.Version, dep.Package, dep.Product, modulesRoot)
3. For each ExternalDep: call tuistproj.ApplyManifestEditsToFile(projectSwiftPath, AddDependency with .external(name: dep.Product))
4. Idempotent — skip if already contains

Remove from setup.go files:
- ioc/setup.go: remove swiftIoCURL/swiftIoCVersion/swiftIoCPackage constants, remove addSwiftIoCToPackageSwift(), remove addSwiftIoCToProjectSwift(), remove their calls from Setup()
- relux/setup.go: remove swiftUIReluxURL/swiftUIReluxVersion/swiftUIReluxPackage/swiftUIReluxProduct constants, remove addSwiftUIReluxDep(), remove addSwiftUIReluxToProjectSwift(), remove their calls from Setup()
- httpclient/setup.go: remove swift-httpclient URL/version constants, remove addHttpClientToPackageSwift() and addHttpClientToProjectSwift() (or equivalent), remove their calls from Setup()

NOTE: relux/setup.go also patches Package.swift with #if TUIST PackageSettings block for Relux product type — this is NOT an external dep, keep it in relux setup.go.

NOTE: Some modules (SecureStore, TokenProvider, Utilities) add LOCAL package paths to Package.swift/Project.swift (not external deps). Those are different — they add .package(path:) not .package(url:). Leave those in their setup.go files.

AC:
- setup_command.go has generic ExternalDeps application loop
- ioc/setup.go, relux/setup.go, httpclient/setup.go no longer contain external dep constants or functions
- Full pipeline works: init → ioc setup → relux setup → secure-store setup → token-provider setup → utilities setup → module create Auth --type relux-feature → ioc setup → http-client setup → app-config setup
- All tests pass, lint clean
- Demo app builds (make build, regenerate demo, xcodebuild)

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
