# STORY-260224-1alzh5: linting-swiftlint

## Description
SwiftLint integration via make lint. Config based on membrana (see .research/ref_membrana-swiftlint.yml), adapted for Tuist module structure. Excludes: DerivedData, Packages, external deps, generated code. Integrated into make validate pipeline. Config templated by CLI on project init — excluded paths auto-populated based on module structure.

IMPORTANT: NEVER run as Xcode pre-build phase / build tool plugin. Linting is a standalone make target, invoked explicitly by dev or CI. Running in pre-build kills build performance for zero benefit — lint errors dont block compilation anyway.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
