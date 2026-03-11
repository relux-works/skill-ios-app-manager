# TASK-260311-ts8woj: add-validation-migration-and-docs

## Description
Add validation, migration coverage, and documentation so version drift is detected automatically and the new workflow is clear.

## Scope
Add a lightweight validation check or test that fails when target manifests hardcode divergent version/build values, migrate xflow-ios to the new structure without changing shipped values, regenerate and verify app plus widget bundle versions, and document the single-file or single-input version bump workflow.

## Acceptance Criteria
Validation or tests fail on divergent hardcoded version/build values; existing xflow-ios migrates with unchanged shipped values; regenerated app and widget produce identical CFBundleShortVersionString and CFBundleVersion; documentation tells maintainers exactly where to update version/build going forward.
