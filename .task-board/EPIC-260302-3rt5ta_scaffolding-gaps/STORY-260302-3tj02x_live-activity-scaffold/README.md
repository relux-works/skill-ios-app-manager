# STORY-260302-3tj02x: live-activity-scaffold

## Description
Scaffold Live Activity as a pluggable widget type.

Live Activities = ActivityKit (app-side lifecycle) + WidgetKit (UI in widget extension). Push-to-update via APNs requires push token flow.

What to scaffold:
1. ActivityAttributes + ContentState in SharedKit (Codable + Hashable, small payload)
2. ActivityConfiguration in widget extension (Lock Screen UI + Dynamic Island: compact/expanded/minimal)
3. App-side manager: LiveActivityManager with start/update/end lifecycle
4. Push token flow: activity.pushTokenUpdates → send to server
5. NSSupportsLiveActivities + NSSupportsLiveActivitiesFrequentUpdates in Info.plist
6. Push-to-start token (iOS 17.2+) optional scaffold

Depends on: widget-scaffold (shares WidgetBundle), push-notification-setup (push-to-update), extension-base (shared module for ActivityAttributes)

Key constraint: ActivityAttributes MUST live in shared module — both app and widget extension compile against same type. Server payload must match ContentState Codable keys.

Source: .research/260224_live-activities-widgets.md sections 4-5, connect-ios audit A2-A3 deep dive (synthesis.md)

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
