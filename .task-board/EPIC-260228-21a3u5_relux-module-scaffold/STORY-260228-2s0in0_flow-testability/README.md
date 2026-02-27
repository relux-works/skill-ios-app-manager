# STORY-260228-2s0in0: flow-testability

## Description
Add optional dispatcher injection to Flow template for testability. Currently Flow actor takes only State as dependency. The canonical pattern injects optional Relux.Dispatcher (defaulting to Self.defaultDispatcher) which allows tests to provide a Relux.Testing.Logger-backed dispatcher for action/effect assertion. Reference: .research/260228_relux_module_internal_layout.md section Authoritative Architecture > Testing Pattern. Key changes: (1) Flow init takes optional dispatcher param, (2) Flow stores dispatcher property, (3) Update internalApply to use dispatcher for action dispatch.

## Scope
(define story scope)

## Acceptance Criteria
(define acceptance criteria)
