# STORY-260302-16jsfr: registry-external-deps

## Description
Add ExternalDeps field to registry.Module so each module declares its external Swift package dependencies (URL, version, product name) alongside internal deps. setup_command.go automatically applies them to Package.swift and Project.swift during setup — no more hardcoded constants in each setup.go.

Also: auto-generate diagrams/scaffolding-pipeline.puml from registry data (internal deps + external deps + categories) so the diagram never drifts from code.

Scope:
1. Add ExternalDep struct and ExternalDeps []ExternalDep to registry.Module
2. Migrate hardcoded constants from ioc/setup.go, relux/setup.go, httpclient/setup.go into register.go ExternalDeps
3. setup_command.go applies ExternalDeps to Package.swift + Project.swift before calling Setup()
4. Each module setup.go no longer manages its own external deps
5. New CLI command: ios-app-manager diagram — generates .puml from registry.AllSorted() deps/categories
6. Update diagrams/scaffolding-pipeline.puml to be auto-generated output

AC:
- All external deps declared in register.go, not setup.go
- setup_command handles Package.swift + Project.swift patching generically
- ios-app-manager diagram produces valid PlantUML matching current architecture
- Demo app builds clean after migration
- Existing tests pass

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
