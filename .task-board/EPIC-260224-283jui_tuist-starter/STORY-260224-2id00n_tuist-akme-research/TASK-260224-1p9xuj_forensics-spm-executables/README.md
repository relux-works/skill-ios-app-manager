# TASK-260224-1p9xuj: forensics-spm-executables

## Description
Research SPM (Swift Package Manager) capabilities focused on: package types, executable targets, plugins, build tool plugins, macros. How to structure a Swift package with executable products. How tuist-akme uses Package.swift. SPM + Tuist interop patterns.

SPM MODULE NAMING CONVENTION — reference: bsim SDK packages (see .research/ref_bsim-*-package.swift). Pattern:
- Package: BSimSDK
- Interface target: BSimSDK (path: Interface/)
- Implementation target: BSimSDKImpl (path: Impl/)
- Test target: BSimSDKTests (path: Tests/)
- Two library products: BSimSDK (interfaces) + BSimSDKImpl (implementation)
- Impl depends on Interface. Consumers depend only on Interface.
- Cross-package: BSimID depends on BSimSDK (interface) and BSimSDKImpl (for wiring)

This is THE naming/structure convention for our generated modules.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
