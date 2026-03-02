# STORY-260302-1hnx69: widget-scaffold

## Description
Scaffold WidgetKit widgets as a pluggable extension type.

Design: base widget scaffold + sub-types as registry plugins (same pattern as modules). Each widget type registers itself with metadata, templates, dependencies.

Widget types to support:
1. Static widget — StaticConfiguration + TimelineProvider + Entry + View
2. Configurable widget — AppIntentConfiguration + AppIntentTimelineProvider (depends on app-intents)
3. Future: additional widget types plug in via same registry

What to scaffold per widget:
- Widget struct (@main or WidgetBundle entry)
- TimelineProvider (placeholder/snapshot/timeline)
- TimelineEntry model
- Widget View stub
- WidgetBundle if multiple widgets
- Resources/ for widget assets
- Extension Project.swift (via extension-base makeAppExtensionProject)

Capabilities needed: appGroups (for shared data)
Depends on: extension-base, app-intents-scaffold
Source: tuist-akme AcmeWidget.swift, .research/260224_live-activities-widgets.md section 2

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
