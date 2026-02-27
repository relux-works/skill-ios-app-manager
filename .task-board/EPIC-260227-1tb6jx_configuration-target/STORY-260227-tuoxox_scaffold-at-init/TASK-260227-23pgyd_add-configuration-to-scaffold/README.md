# TASK-260227-23pgyd: add-configuration-to-scaffold

## Description
Update init scaffolding to create Configuration folder:
- Create Targets/<AppName>/Sources/Configuration/ directory
- Create Configuration.swift with:
  /// Shared configuration constants across app targets.
  enum Configuration {}

No Package.swift, no manifest changes — it is part of the app target sources, picked up automatically.

AC: init creates Configuration folder with namespace file.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
