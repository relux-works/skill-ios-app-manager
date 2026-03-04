# STORY-260304-12mkl3: configurable-widget

## Description
Scaffold a configurable (interactive) widget plugin that registers into widget-base WidgetBundle.

What it creates:
1. AppIntentConfiguration widget struct
2. AppIntentTimelineProvider
3. ConfigurationAppIntent (widget config intent)
4. ActionAppIntent stubs (widget interaction)
5. Shared state mutation pattern (App Group UserDefaults → intent mutates → WidgetCenter.reloadTimelines)
6. Registration into WidgetBundle from widget-base

Interactive widgets (iOS 17+) execute AppIntent on tap — this is the foundation for any widget interactivity.

Depends on: widget-base, app-intents-scaffold
Source: tuist-akme Widget-Configurable.swift, .research/260224_live-activities-widgets.md section 3

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
