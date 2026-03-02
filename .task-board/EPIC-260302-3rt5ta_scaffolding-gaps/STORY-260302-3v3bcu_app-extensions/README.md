# Extension Base Scaffold — infrastructure for all app extensions

## Description
[A2-A3, Tier 3] Base infrastructure for app extensions. NOT individual extension types — those are separate stories.

What this covers:
1. Extension registry pattern — extensions register via init() same as modules (registry.ExtensionType: widget, live-activity, notification-service, notification-content)
2. Base scaffold: makeAppExtensionProject() equivalent in our CLI — generates Extension/Name/Project.swift with proper bundleIdType, NSExtensionPointIdentifier, embedding in host app
3. Shared module scaffold — SharedKit-style package for types shared between app and extensions (ActivityAttributes, shared models, App Group constants)
4. WidgetCompositionRoot pattern — separate minimal composition root for extension targets
5. ProjectTargetRef + AppProjects.swift generation — typed references for embedding
6. Host app Project.swift patching — add embeddedExtensions entry

Source: tuist-akme Apps/iOSApp/Extensions/AcmeWidget/, ProjectFactory.makeAppExtensionProject(), ExtensionSpec.swift, WidgetCompositionRoot

This is the foundation all extension types build on.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
