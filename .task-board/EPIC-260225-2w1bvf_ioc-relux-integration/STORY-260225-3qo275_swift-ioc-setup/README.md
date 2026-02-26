# STORY-260225-3qo275: swift-ioc-setup

## Description
CLI command `ioc setup` that integrates SwiftIoC into a generated Tuist project. Based on relux-sample-app pattern:

1. Adds SwiftIoC SPM dependency (github.com/relux-works/swift-ioc from: 1.0.1) to root Package.swift
2. Adds SwiftIoC as dependency to the main app target in Project.swift (.external(name: SwiftIoC))
3. Creates IoC registry file in app Sources (Registry.swift) with:
   - Static IoC container instance
   - configure() method that registers all existing modules
   - resolve/resolveAsync helper methods
4. Updates App.swift entry point to call Registry.configure() in init
5. Each module Interface gets a Module protocol conformance that IoC can register

AC:
- `ios-app-manager ioc setup` adds SwiftIoC dep and creates registry scaffold
- Existing modules are auto-discovered and wired into registry
- Project builds after running the command
- Go tests cover the new command

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
