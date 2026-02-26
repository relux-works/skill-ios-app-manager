# TASK-260225-11y0jr: wire-template-targets-and-deps

## Description
1) Update templateTargetsForSet() in internal/relux/init_cmd.go to map new template names to output paths: relux_namespace→InterfaceSourcesDir/Namespace.swift, relux_interface→InterfaceSourcesDir/Module+Interface.swift, relux_action→InterfaceSourcesDir/Business+Action.swift, relux_effect→InterfaceSourcesDir/Business+Effect.swift, relux_impl→ImplSourcesDir/Module+Impl.swift, relux_state→ImplSourcesDir/Business+State.swift, relux_flow→ImplSourcesDir/Business+Flow.swift. 2) Update Package.swift generation in internal/tuistproj/package_gen.go so relux-feature packages include swift-relux as external dependency. 3) Register new templates in template_engine.go. 4) Update CLI help text in module.go to list relux-feature as valid type.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
