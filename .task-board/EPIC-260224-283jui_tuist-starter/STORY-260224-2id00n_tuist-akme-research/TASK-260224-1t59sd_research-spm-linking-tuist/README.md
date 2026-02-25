# TASK-260224-1t59sd: research-spm-linking-tuist

## Description
CRITICAL RESEARCH: SPM intra-package target linking is always static (no way to specify dynamic for target-to-target deps within one Package.swift). This means Impl target statically links Interface target — breaks our all-dynamic rule.

Two options to investigate:
1. Can Tuist override/control linking type for SPM targets within a package? Test with a real Tuist project: create one SPM package with Interface+Impl targets, see if Tuist can force dynamic linking between them.
2. If NOT — split every product module into TWO separate SPM packages:
   - ModuleName/ (Package.swift with .dynamic library, Interface only)
   - ModuleNameImpl/ (Package.swift with .dynamic library, depends on ModuleName package)
   - This gives full control: each package is a .dynamic product

Test BOTH approaches with actual code. Build, check Mach-O types with otool -L or file command.

Output: .research/260224_spm-linking-tuist.md with test results and recommendation.

Reference: bsim packages (see .research/ref_bsim-*-package.swift) currently use single-package approach — this is the problem we are solving.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
