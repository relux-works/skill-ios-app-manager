# TASK-260304-psg4qf: impl-widget-base

## Description
Implement widget-base scaffold — the base widget extension that all widget types plug into.

## What to build

New package: `internal/widgetbase/` following the setup pattern (register.go + setup.go + setup_templates/).

### register.go
- init() → registry.Register() with ID=WidgetBase, Category=Extensions
- Dependencies: [AppExtensions] (needs extension-base infrastructure)
- Plan/Setup/UsageGuide functions

### setup.go — Setup(input) creates:
1. Widget extension target using extension-base makeAppExtensionProject():
   - Extension name: <AppName>Widget
   - NSExtensionPointIdentifier = com.apple.widgetkit-extension
   - Embedded in host app
2. WidgetBundle (@main entry point) — empty bundle that widget plugins will register into
3. App Groups entitlement for shared data between app and extension
4. Add extension to Project.swift and Workspace.swift

### Templates (setup_templates/):
- widget_bundle.swift.tmpl — @main WidgetBundle with placeholder
- widget_extension_project.swift.tmpl — if needed beyond extension-base

## CLI command
`ios-app-manager widget-base setup` with standard --dry-run/--yes flags

## Reference
- Setup pattern: `internal/foundationplus/setup.go`, `internal/utilities/setup.go`
- Registry pattern: `internal/foundationplus/register.go`
- CLI setup command: `internal/cli/setup_command.go`
- Widget research: `.research/260224_live-activities-widgets.md` section 2.1

## Tests
- Unit tests for Setup()
- Golden file tests for generated templates
- E2E: widget-base setup → verify extension target created, WidgetBundle generated

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
