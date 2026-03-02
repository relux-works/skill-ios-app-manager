# STORY-260302-667ruw: notification-service-ext

## Description
Scaffold Notification Service Extension.

Intercepts push notifications before display — modify content, download attachments, decrypt payload. Runs in background with ~30s budget.

What to scaffold:
1. Extension target: product .appExtension, NSExtensionPointIdentifier = com.apple.usernotifications.service
2. NotificationService class (UNNotificationServiceExtension subclass)
3. Stub: contentHandler + bestAttemptContent pattern
4. App Group for shared data with main app
5. Extension Project.swift via extension-base

Capabilities needed: push, appGroups
Depends on: extension-base, push-notification-setup

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
