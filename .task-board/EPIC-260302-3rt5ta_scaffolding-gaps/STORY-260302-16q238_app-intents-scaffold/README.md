# STORY-260302-16q238: app-intents-scaffold

## Description
Scaffold App Intents infrastructure for interactive widgets and Shortcuts.

Interactive widgets (iOS 17+) execute AppIntent on tap, not arbitrary SwiftUI code. This is the foundation for configurable widgets and any widget interactivity.

What to scaffold:
1. AppIntent protocol conformance stubs
2. IntentConfiguration vs AppIntentConfiguration patterns
3. Shared state mutation pattern (App Group UserDefaults → intent mutates → WidgetCenter.reloadTimelines)
4. Basic intent types: ConfigurationAppIntent (widget config), ActionAppIntent (widget interaction)

Depends on: extension-base (needs shared module for cross-process state)
Blocks: widget-scaffold (interactive widgets need intents)

Source: tuist-akme Xcode template Widget-Configurable.swift, .research/260224_live-activities-widgets.md section 3

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
