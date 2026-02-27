# STORY-260227-tuoxox: scaffold-at-init

## Description
Add Configuration utility package to the init scaffolding pipeline.

What init does for Configuration:
- Creates Packages/Configuration/ with Package.swift (utility, interface type)
- Creates Sources/Configuration/Configuration.swift with public enum Configuration {}
- Writes .module-type=utility
- Adds Configuration to Project.swift, root Package.swift, Workspace.swift

AC:
- ios-app-manager init creates Configuration package alongside other scaffolded files
- Package.swift is valid, namespace file exists
- All manifests reference Configuration
- Tests pass

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
