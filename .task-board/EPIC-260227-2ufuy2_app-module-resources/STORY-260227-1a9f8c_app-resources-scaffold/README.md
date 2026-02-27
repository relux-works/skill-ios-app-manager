# STORY-260227-1a9f8c: app-resources-scaffold

## Description
When init scaffolds the project, the app module must have Sources/ and Resources/ directories.

Resources/ must contain:
- Assets.xcassets/ with AppIcon asset (default placeholder icon — Xcode hammer or Swift bird, downloaded from the internet during scaffold)
- Tuist manifest must reference the Resources/ directory so Xcode picks up assets

The icon is a placeholder — user replaces it later. But the project should build and show a real icon on the home screen from day one.

Implementation:
- Update internal/scaffold/ to create Resources/ dir alongside Sources/
- Embed or download a default AppIcon (1024x1024 PNG) during scaffold
- Generate proper Contents.json for the asset catalog
- Update Project.swift template to include resources: ["Resources/**"]

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
