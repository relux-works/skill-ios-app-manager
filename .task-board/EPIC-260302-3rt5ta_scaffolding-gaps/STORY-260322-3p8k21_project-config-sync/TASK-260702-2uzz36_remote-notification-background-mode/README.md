# Remote notification background mode

## Description
Extend ios-app-manager background_modes support to include the iOS remote-notification UIBackgroundModes value for apps that implement APNs background notification callbacks.

## Scope
Update ios-app-manager config schema constants, validator messages/tests, background mode generator tests, README/SKILL/CLI docs, and setup/install verification so projects can declare remote-notification without hand-editing scaffold-owned Project.swift.

## Acceptance Criteria
background_modes accepts remote-notification in ios-app-manager config validation. generate background-modes-config/project-config emits .string("remote-notification") in host UIBackgroundModes alongside existing modes. Documentation lists audio, voip, and remote-notification with intended use. Relevant Go tests pass, setup/install refreshes the installed skill/tool, and downstream VideoCallDemo uses the corrected generator.
