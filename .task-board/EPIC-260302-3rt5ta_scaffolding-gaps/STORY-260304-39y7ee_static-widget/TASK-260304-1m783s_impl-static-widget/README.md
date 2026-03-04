# TASK-260304-1m783s: impl-static-widget

## Description
Implement static widget plugin scaffold.

## What to build

New package: `internal/staticwidget/` following the setup pattern.

### register.go
- init() → registry.Register() with ID=StaticWidget, Category=Extensions
- Dependencies: [WidgetBase]

### setup.go — Setup(input) creates:
1. StaticConfiguration widget struct
2. TimelineProvider with placeholder/snapshot/timeline stubs
3. TimelineEntry model (date + sample data)
4. Widget View stub (SwiftUI)
5. Registers widget into existing WidgetBundle from widget-base

### Templates (setup_templates/):
- static_widget.swift.tmpl — Widget struct with StaticConfiguration
- timeline_provider.swift.tmpl — TimelineProvider protocol conformance
- timeline_entry.swift.tmpl — Entry model
- widget_view.swift.tmpl — SwiftUI view stub

### WidgetBundle registration:
- Patch existing WidgetBundle to add the new static widget entry
- Pattern: find WidgetBundle body, add StaticWidget() entry

## CLI command
`ios-app-manager static-widget setup` with standard --dry-run/--yes flags

## Reference
- Widget research: `.research/260224_live-activities-widgets.md` section 2
- Setup pattern: `internal/foundationplus/setup.go`

## Tests
- Unit tests for Setup()
- Golden file tests for generated templates

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
