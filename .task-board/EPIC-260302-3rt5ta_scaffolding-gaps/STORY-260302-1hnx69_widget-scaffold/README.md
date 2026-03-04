# widget-base

## Description
Base widget extension scaffold — the foundation target that all widget types plug into.

What it creates:
1. Widget extension target (via extension-base makeAppExtensionProject)
2. Empty WidgetBundle (@main entry point)
3. App Groups capability for shared data between app and extension
4. Shared types module stub (for ActivityAttributes and other cross-process types)
5. Extension Info.plist with NSExtensionPointIdentifier = com.apple.widgetkit-extension

This is JUST the base container. Individual widget types (static, configurable, live-activity) are separate plugins that register into this WidgetBundle.

Depends on: extension-base (app-extensions), app-groups (init/secure-store)
Blocks: all widget type plugins

Source: .research/260224_live-activities-widgets.md section 2.1

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
