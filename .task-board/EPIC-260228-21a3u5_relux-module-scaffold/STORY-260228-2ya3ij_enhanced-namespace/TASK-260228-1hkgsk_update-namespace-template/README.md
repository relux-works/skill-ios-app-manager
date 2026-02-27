# TASK-260228-1hkgsk: update-namespace-template

## Description
Update internal/relux/templates/relux_namespace.swift.tmpl to generate richer namespace hierarchy. Current template generates only: enum <Name> { enum Business {} }. New template should generate:

enum <Name> {
    enum Data {
        enum Api {
            enum DTO {}
        }
    }
    enum Business {
        enum Model {}
    }
    enum UI {
        enum Model {}
    }
}

This matches the canonical pattern from .research/260228_relux_module_internal_layout.md section 1. All sub-namespaces are public enums. Keep the template simple — no conditional logic needed, all relux-feature modules get the full hierarchy.

Files to modify:
- internal/relux/templates/relux_namespace.swift.tmpl

Verification:
- make test passes
- Golden files updated if needed (make test-update)
- Demo app regenerated and compiles

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
