# BUG-260518-vqrag4: swift-6-strict-memory-safety-missing-from-app-build-settings

## Description
Repro: generated Swift 6 app/extension manifests include SWIFT_STRICT_CONCURRENCY=complete, but Xcode Strict Memory Safety remains NO because the strict baseline does not emit SWIFT_STRICT_MEMORY_SAFETY

## Scope
(define bug scope / affected area)

## Acceptance Criteria
(define fix acceptance criteria)
