# Notification Service Extension scaffold

## Description
Scaffold Notification Service Extension as a concrete app-extension plugin using the extension Core package contract. The generated .appex target contains only UNNotificationServiceExtension runtime glue; notification mutation/parsing/attachment/decryption hooks and other testable internals live in <ExtensionName>Core SwiftPM package.

## Scope
Generate extension target with product .appExtension and NSExtensionPointIdentifier com.apple.usernotifications.service; generate <ExtensionName>Core package with testable service logic interfaces/stubs; link the Core product into the extension target; wire app groups and push-related capability assumptions through existing capability/config plugins; ensure project-config metadata sync updates bundle id, version, min target, team id, and build flags for the extension.

## Acceptance Criteria
Command creates a Notification Service Extension target with caller-provided extension name and deterministic bundle id suffix; command creates and links <ExtensionName>Core package; Core package has tests or is compatible with generated package/unit test target flow; extension wrapper compiles with UNNotificationServiceExtension subclass and delegates to Core logic; rerunning the command does not duplicate targets, packages, files, dependencies, entitlements, or Info.plist keys; generated extension metadata updates when project-config values change; docs mention that push-specific behavior is implemented and tested in Core, not directly in the .appex target.
