# STORY-260225-1nv0nz: relux-feature-module-type

## Description
Add new module type relux-feature to the CLI. When user runs module create Auth --type relux-feature, it generates Interface/Impl split where Interface protocol conforms to Relux.Module (exposing states/sagas) and Impl has scaffolded HybridState, Action, Effect, Flow with simple manual DI. Both packages get swift-relux as SPM dependency. Regular feature type stays unchanged.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
