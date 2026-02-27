## Status
done

## Assigned To
[implementer] developer (claude)

## Created
2026-02-25T11:29:59Z

## Last Update
2026-02-27T11:04:59Z

## Blocked By
- (none)

## Blocks
- TASK-260225-3ph36h

## Checklist
- [ ] Code written per task description and AC
- [ ] Tests written and passing
- [ ] Lint clean
- [ ] Build not broken
- [ ] Result noted on the board
- [ ] `module create Auth --type relux-feature` creates Auth/ and AuthImpl/ packages under Packages/
- [ ] Auth/Package.swift has swift-relux as external dependency with product "Relux"
- [ ] AuthImpl/Package.swift has swift-relux as external dependency with product "Relux"
- [ ] Root Package.swift gets swift-relux added automatically
- [ ] Interface templates generate: Namespace.swift, Module.swift, Module+Interface.swift, Business+Action.swift, Business+Effect.swift
- [ ] Impl templates generate: Module+Impl.swift, Business+State.swift, Business+Flow.swift
- [ ] Module+Interface.swift has protocol conforming to Relux.Module
- [ ] Business+State.swift has @Observable class conforming to Relux.HybridState
- [ ] Business+Flow.swift has actor conforming to Relux.Flow
- [ ] Module+Impl.swift has async init with manual DI wiring states and sagas
- [ ] All existing tests pass (`go test ./...`)
- [ ] New unit tests for relux-feature type registration, template rendering, and template target mapping
- [ ] CLI integration test for `module create --type relux-feature`
- [ ] Existing module types (feature, kit, shared, ui, utility) work unchanged

## Notes
Implemented relux-feature module type.
Files modified:
- internal/modules/types.go (added ModuleTypeReluxFeature constant + ExternalDep type)
- internal/modules/registry.go (added descriptor + template set + swift-relux dep)
- internal/relux/templates/ (7 new .tmpl files: relux_namespace, relux_interface, relux_action, relux_effect, relux_impl, relux_state, relux_flow)
- internal/relux/init_cmd.go (added template target definitions)
- internal/relux/template_engine.go (registered new templates in requiredTemplateNames)
- internal/tuistproj/package_gen.go (added ExternalProductDep, extended PackageGenerationInput)
- internal/tuistproj/templates/package.swift.tmpl (renders external deps)
- internal/tuistproj/manager.go (added relux-feature to isProductModuleType, external deps pipeline, root Package.swift update)
- internal/components/interfaces.go (added ExternalDep type to ModuleOpts)
- internal/cli/module.go (updated help text)
Tests added:
- TestCreatorCreateReluxFeatureModule (creator_test.go)
- TestInitCommandRunReluxFeatureTemplateSet (init_cmd_test.go)
- TestTuistProjectManagerCreateModuleReluxFeatureScaffoldsTwoPackagesWithExternalDeps (manager_test.go)
- 7 golden file tests for new templates (templates_test.go)
- Updated registry_test.go for new type
All 17 packages pass: go test ./..., go vet, gofmt clean.
agent completed: [implementer] developer (claude) (exit=0)
agent spawned: claude (pid=59878, exit=0)

## Precondition Resources
- [impl-guide](file://TASK-260225-2uxthe/impl-guide) — Implementation guide

## Outcome Resources
(none)
