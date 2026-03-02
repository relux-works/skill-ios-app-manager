# TASK-260302-1r8xb1: diagram-generation-cmd

## Description
Add ios-app-manager diagram command that auto-generates diagrams/scaffolding-pipeline.puml from registry data.

The command reads registry.AllSorted() and generates a PlantUML component diagram showing:
- All modules as components, colored by Category (infra=yellow, foundation=blue, network=orange, utils=purple, feature=beige)
- Internal dependencies as solid arrows (from Module.Dependencies)
- External dependencies as cloud nodes (from Module.ExternalDeps) with arrows from modules that use them
- Grouped by category in packages
- Legend with color meanings
- init as root node

Implementation:
1. New package internal/diagram/ with GeneratePlantUML(modules []registry.Module) string
2. New CLI command in internal/cli/ — ios-app-manager diagram [--output path] (default: diagrams/scaffolding-pipeline.puml)
3. The command imports all modules via blank imports (same as root.go) to ensure registry is populated
4. Template-based generation (embed a .tmpl or build string directly)

The output should match the style of the existing diagrams/scaffolding-pipeline.puml (plain theme, ortho lines, component rectangles).

AC:
- ios-app-manager diagram produces valid .puml file
- Output matches actual registry dependencies (no manual sync needed)
- plantuml renders it without errors
- Command is registered in root.go

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
