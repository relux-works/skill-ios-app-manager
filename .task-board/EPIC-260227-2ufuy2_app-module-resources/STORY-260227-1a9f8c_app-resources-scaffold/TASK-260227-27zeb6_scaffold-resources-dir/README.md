# TASK-260227-27zeb6: scaffold-resources-dir

## Description
Update scaffold pipeline to create app module Resources/ directory with Assets.xcassets and default AppIcon.

1. Add Resources/Assets.xcassets/AppIcon.appiconset/ to scaffold output
2. Embed a default 1024x1024 PNG icon (Xcode hammer or Swift bird — download and embed via Go embed.FS)
3. Generate Contents.json for AppIcon.appiconset with proper platform sizes
4. Update Project.swift template: add resources parameter pointing to Resources/**
5. Tests: golden files for the new directory structure + asset catalog contents

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
