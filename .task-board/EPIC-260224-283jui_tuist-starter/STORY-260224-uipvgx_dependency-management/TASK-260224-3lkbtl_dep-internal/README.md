# TASK-260224-3lkbtl: dep-internal

## Description
dep add/remove for inter-module dependencies. Wire ModuleA depends on ModuleB interface. Updates ModuleA Package.swift (adds .package path + .product dependency). Validates: no circular deps, only depends on interfaces (never impl directly unless same module).

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
