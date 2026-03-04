# TASK-260304-hr7frj: impl-live-activity

## Description
Implement Live Activity widget plugin scaffold.

## What to build

New package: `internal/liveactivity/` following the setup pattern.

### register.go
- init() → registry.Register() with ID=LiveActivity, Category=Extensions
- Dependencies: [WidgetBase, PushNotification]

### setup.go — Setup(input) creates:
1. ActivityAttributes + ContentState in SharedKit (Codable + Hashable)
2. ActivityConfiguration in widget extension (Lock Screen UI + Dynamic Island: compact/expanded/minimal)
3. App-side LiveActivityManager: start/update/end lifecycle
4. Push token flow stub: activity.pushTokenUpdates → send to server
5. NSSupportsLiveActivities in Info.plist
6. Registers ActivityConfiguration into existing WidgetBundle

### Templates:
- activity_attributes.swift.tmpl — ActivityAttributes + ContentState (goes to SharedKit)
- activity_configuration.swift.tmpl — ActivityConfiguration + DynamicIsland views
- live_activity_manager.swift.tmpl — app-side lifecycle manager

### Key constraint
ActivityAttributes MUST live in shared module — both app and widget extension compile against same type.

## Reference
- Widget research: `.research/260224_live-activities-widgets.md` sections 4-5
- Setup pattern: `internal/foundationplus/setup.go`

## Tests
- Unit tests for Setup()
- Golden file tests for generated templates

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
