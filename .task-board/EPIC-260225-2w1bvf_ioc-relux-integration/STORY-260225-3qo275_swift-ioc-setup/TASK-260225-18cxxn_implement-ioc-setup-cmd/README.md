# TASK-260225-18cxxn: implement-ioc-setup-cmd

## Description
Implement `ioc setup` CLI command that integrates SwiftIoC into a generated Tuist project.

The command:
1. Adds SwiftIoC SPM dependency to root Package.swift
2. Adds .external(name: SwiftIoC) to Project.swift app target dependencies
3. Scaffolds Registry.swift in Targets/{AppName}/Sources/ with IoC container + configure() + resolve helpers
4. Updates App.swift init to call Registry.configure()
5. Auto-discovers existing modules and wires them into Registry.configure()

AC:
- `ios-app-manager ioc setup --config <path>` works on a generated project
- SwiftIoC dependency added to Package.swift and Project.swift
- Registry.swift created with working IoC setup
- App.swift updated with Registry.configure() call
- Existing modules auto-registered
- Project builds after the command (tuist install + generate + xcodebuild)
- Go tests cover the new command
- See precondition resource impl-guide.md for implementation details

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
