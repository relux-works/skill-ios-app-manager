# TASK-260304-3bg21p: impl-extension-registry

## Description
Implement extension scaffold infrastructure in the Go CLI.

## What to build

1. **Extension type registry** in `internal/registry/`:
   - Add ExtensionType enum: widget, notification-service, notification-content
   - Extensions register via init() same pattern as modules
   - Each extension type has: ID, templates, dependencies, scaffold function

2. **makeAppExtensionProject()** in `internal/tuistproj/`:
   - Generate Extension Project.swift with:
     - Correct bundleIdType for extensions
     - NSExtensionPointIdentifier in Info.plist config
     - Proper embedding in host app target
   - Template: `extension_project.swift.tmpl`

3. **Host app Project.swift patching**:
   - Add embeddedExtensions entry to existing Project.swift
   - Pattern: `.appExtension(target: "WidgetExtension")`

4. **SharedKit module scaffold**:
   - Create Packages/SharedKit/ with Package.swift (utility type)
   - For types shared between app and extensions (ActivityAttributes, shared models, App Group constants)
   - Write .module-type = utility

## Reference
- Existing module scaffold pattern: `internal/utilities/setup.go`, `internal/foundationplus/setup.go`
- tuistproj package generation: `internal/tuistproj/manager.go`
- Package.swift template: `internal/tuistproj/templates/package.swift.tmpl`
- Extension point IDs: com.apple.widgetkit-extension, com.apple.usernotifications.service, com.apple.usernotifications.content-extension

## Tests
- Unit tests for extension Project.swift generation
- Unit tests for host app patching
- Golden file tests for generated manifests

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
