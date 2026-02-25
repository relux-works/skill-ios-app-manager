# STORY-260224-2zw4a1: project-init

## Description
ios-app-manager init command. Two-layer config-driven project creation:

1. User provides a PROJECT CONFIG (JSON/YAML) with: APP_NAME, PROJECT_BUNDLE_ID, DEVELOPMENT_TEAM, APP_URL_SCHEME, app groups, MIN_TARGET_VERSION, SWIFT_VERSION, MARKETING_VERSION, etc.
2. CLI templates Tuist files (Tuist.swift, Workspace.swift, Project.swift, Package.swift) from the config — all ${VARIABLE} placeholders get substituted.
3. Then tuist generate builds the Xcode project.

Reference: membrana-ios-app config pattern (see .research/ref_membrana-config-example.json and ref_membrana-manifest-template.json).

The config is the single source of truth. Supports multiple configs per project (like membrana has proj_config_mts vs proj_config_fc for different build flavors).

Generated structure: base module layout, .gitignore, Makefile, entitlements stubs. Project must build out of the box with tuist generate + xcodebuild.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
