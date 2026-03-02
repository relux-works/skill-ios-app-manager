# TASK-260302-2cwzaf: diagram-format-flag

## Description
Add --format flag to ios-app-manager diagram command. Supported formats: puml (default, current behavior), png, svg. For png/svg: write temp .puml, shell out to plantuml -t<format>, output rendered file. Default output path: diagrams/scaffolding-pipeline.<format>. If plantuml not in PATH for png/svg, return clear error with install hint (brew install plantuml). Update existing tests, add tests for format validation and png/svg render (mock or skip if plantuml not available).

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
