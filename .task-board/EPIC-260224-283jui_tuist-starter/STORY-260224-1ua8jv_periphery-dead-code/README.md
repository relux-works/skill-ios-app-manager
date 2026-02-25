# STORY-260224-1ua8jv: periphery-dead-code

## Description
Periphery (dead code detection) integration via make periphery. Catches unused code, protocols, imports. Config auto-generated based on project structure and modules. Integrated into make validate pipeline. Need to research Periphery + Tuist interop (periphery scan --project vs --index-store-path). May need make periphery-setup to build index first.

IMPORTANT: NEVER run as Xcode pre-build phase / build tool plugin. Periphery is a standalone make target, invoked explicitly by dev or CI. It is slow by nature (full index scan) — shoving it into build pipeline is insanity.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
