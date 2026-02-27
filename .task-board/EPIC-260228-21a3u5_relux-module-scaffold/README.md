# EPIC-260228-21a3u5: relux-module-scaffold

## Description
Enhance relux-feature module scaffolding to generate production-ready Relux business logic content. Current templates create correct structure with empty switches/enums. This epic covers: richer namespace with Data/UI sub-namespaces, per-module IoC container pattern, Flow testability (optional dispatcher injection), test file generation with @Suite hierarchy and mock patterns, and Container/View scaffolding for relux-features with UI layer.

## Scope
All relux-feature template files in internal/relux/templates/ and module creation logic in internal/modules/. Affects generated Swift code in Packages/<Name>/ and Packages/<Name>Impl/. Does NOT change non-relux module types.

## Acceptance Criteria
1. Namespace template generates Data/UI sub-namespaces when appropriate
2. Module impl uses per-module IoC container with buildIoC() pattern
3. Flow template includes optional dispatcher injection for testability
4. Test files generated with @Suite hierarchy matching namespace structure
5. All existing tests pass (make test)
6. Demo app regenerated and compiles with tuist generate + xcodebuild
