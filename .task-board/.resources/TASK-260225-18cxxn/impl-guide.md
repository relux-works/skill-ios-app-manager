# Implementation Guide: `ioc setup` Command

## Overview

Add a new CLI command `ioc setup` to ios-app-manager that integrates SwiftIoC into a generated Tuist project.

**SwiftIoC:** https://github.com/relux-works/swift-ioc (lightweight DI container, version 1.0.1+)

## What the Command Does

`ios-app-manager ioc setup --config <path>` performs these steps:

### 1. Add SwiftIoC dependency to root Package.swift

Use existing `deps.AddExternalDep()` from `internal/deps/external.go`:
- URL: `https://github.com/relux-works/swift-ioc.git`
- Version: `from: "1.0.1"`

This adds `.package(url: "https://github.com/relux-works/swift-ioc.git", from: "1.0.1")` to the root Package.swift dependencies section.

### 2. Add SwiftIoC to Project.swift app target dependencies

Add `.external(name: "SwiftIoC")` to the app target dependencies in Project.swift.
Use `tuistproj.ApplyManifestEditsToFile()` with `AddDependency` edit type.

### 3. Scaffold Registry.swift

Create `Targets/{AppName}/Sources/Registry.swift` with:

```swift
import SwiftIoC

extension {AppTypeName} {
    @MainActor
    enum Registry {
        static let ioc = IoC()

        static func configure() {
            // Module registrations will be added here
            {moduleRegistrations}
        }

        static func resolve<T>(_ type: T.Type) -> T {
            ioc.get(by: type)!
        }

        static func resolveAsync<T>(_ type: T.Type) async -> T {
            await ioc.getAsync(by: type)!
        }
    }
}
```

Where `{moduleRegistrations}` auto-discovers existing modules (by scanning Packages/ for modules with Interface/Impl split) and generates:
```swift
ioc.register(TodoList.Module.Interface.self, lifecycle: .container) {
    TodoList.Module.Impl()
}
```

### 4. Update App.swift

Add `init() { Registry.configure() }` to the App struct. Parse the existing App.swift, find the struct declaration, inject the init block.

**Before:**
```swift
@main
struct DemoApp: App {
    var body: some Scene {
```

**After:**
```swift
import SwiftIoC
import TodoList
import TodoListImpl
import Settings
import SettingsImpl

@main
struct DemoApp: App {
    init() {
        Registry.configure()
    }

    var body: some Scene {
```

## Architecture

Follow existing patterns in the codebase:

### CLI Layer: `internal/cli/ioc.go`
- New file with `newIocCommand(opts *RootOptions) *cobra.Command`
- Subcommand `setup` (no args)
- Loads config, resolves paths
- Register in `root.go` via `cmd.AddCommand(newIocCommand(opts))`

### Business Logic: `internal/ioc/setup.go` (NEW package)
- `Setup(input SetupInput) error`
- `SetupInput` struct: `ProjectRoot string, AppName string, ModulesPath string`
- Orchestrates: add dep -> edit Project.swift -> scaffold Registry.swift -> update App.swift
- Uses `deps.AddExternalDep()` for Package.swift
- Uses `tuistproj.ApplyManifestEditsToFile()` for Project.swift
- Generates Registry.swift from template
- Edits App.swift with simple string manipulation

### Module Discovery
- Scan `{ModulesPath}/` for directories
- For each dir, check if `{dir}Impl/` sibling exists -> it is a split module
- Extract module names for registration

### Templates
- Registry.swift template in `internal/ioc/templates/registry.swift.tmpl`
- Use Go embed + text/template (same pattern as relux templates)

## Testing

- Unit test for module discovery (mock filesystem)
- Unit test for Registry.swift generation (golden file)
- Unit test for App.swift editing (before/after)
- CLI integration test (similar to TestModuleCreateFeature in cli/module_test.go)
- E2E: init project -> create modules -> ioc setup -> verify files exist and content correct

## Key Files to Reference

- `internal/cli/module.go` — CLI command pattern
- `internal/cli/root.go` — command registration
- `internal/deps/external.go` — AddExternalDep (reuse for Package.swift)
- `internal/tuistproj/manifest_edit.go` — ApplyManifestEditsToFile (reuse for Project.swift)
- `internal/scaffold/app_stub.go` — swiftTypeName helper (reuse for App type name)
- `internal/relux/template_engine.go` — template rendering pattern (follow for Registry.swift)
- `internal/config/schema.go` — ProjectConfig structure

## Constraints

- Do NOT modify existing module creation flow
- SwiftIoC version: `from: "1.0.1"`
- Registry pattern follows relux-sample-app (see .research/260224_swift-ioc-forensics.md)
- Must work idempotently (running twice should not break anything)
- Must work on a fresh project with 0 modules (empty configure())
- Must work on a project with existing modules (auto-discover and register)
