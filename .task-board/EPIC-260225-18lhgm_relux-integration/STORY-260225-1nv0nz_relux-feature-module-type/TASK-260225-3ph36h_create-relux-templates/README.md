# TASK-260225-3ph36h: create-relux-templates

## Description
Create new Swift template files in internal/relux/templates/ for relux-feature modules. Templates: 1) relux_namespace.swift.tmpl - extended namespace with Business sub-namespace (enum ModuleName { enum Business {} }), 2) relux_interface.swift.tmpl - Module.Interface protocol conforming to Relux.Module, 3) relux_action.swift.tmpl - Business.Action enum conforming to Relux.Action, 4) relux_effect.swift.tmpl - Business.Effect enum conforming to Relux.Effect, 5) relux_impl.swift.tmpl - Module.Impl struct conforming to Interface with states/sagas arrays and async init with manual DI, 6) relux_state.swift.tmpl - Business.State as @Observable Relux.HybridState with reduce/cleanup stubs, 7) relux_flow.swift.tmpl - Business.Flow actor conforming to Relux.Flow with apply stub. All templates use TemplateVariables (ModuleName, InterfacePackageName, ImplPackageName).

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
