# STORY-260302-3tj02x: live-activity-scaffold

## Description
Scaffold Live Activity as a widget plugin that registers into widget-base WidgetBundle.

Live Activities = ActivityKit (app-side lifecycle) + WidgetKit (UI in widget extension). Push-to-update via APNs requires push token flow.

What it creates:
1. ActivityAttributes + ContentState in shared types module (Codable + Hashable, small payload)
2. ActivityConfiguration in widget extension (Lock Screen UI + Dynamic Island: compact/expanded/minimal)
3. App-side LiveActivityManager: start/update/end lifecycle
4. Push token flow: activity.pushTokenUpdates → send to server
5. NSSupportsLiveActivities + NSSupportsLiveActivitiesFrequentUpdates in Info.plist
6. Push-to-start token (iOS 17.2+) optional scaffold
7. Registration into WidgetBundle from widget-base

Key constraint: ActivityAttributes MUST live in shared module — both app and widget extension compile against same type.

Depends on: widget-base (shares WidgetBundle), push-notification-setup (push-to-update)
Source: .research/260224_live-activities-widgets.md sections 4-5

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
