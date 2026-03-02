## Status
done

## Assigned To
[implementer] developer (claude)

## Created
2026-03-02T10:57:43Z

## Last Update
2026-03-02T11:26:55Z

## Blocked By
- TASK-260302-256oik

## Blocks
- (none)

## Checklist
- [ ] Code written per task description and AC
- [ ] Tests written and passing
- [ ] Lint clean
- [ ] Build not broken
- [ ] Result noted on the board
- [ ] Capability struct added to registry.go with Type and Args fields
- [ ] Module struct has Capabilities []Capability field
- [ ] securestore/register.go declares keychainSharing capability
- [ ] Embedded Swift files copied to Tuist/ProjectDescriptionHelpers/ during scaffold init
- [ ] AppCapabilities.swift generated with empty app capability array
- [ ] Project.swift.tmpl uses EntitlementsFactory.make() instead of .file(path:) for entitlements
- [ ] AddToAppCapabilities function works idempotently (exists → skip, missing → add)
- [ ] setup_command.go applies module capabilities after external deps
- [ ] Registry tests updated for Capability field
- [ ] make test passes
- [ ] make lint passes

## Notes
agent spawned: claude (pid=93862, exit=0)
Implemented capability integration:
1. registry.go: Added Capability struct (Type + Args) and Capabilities field to Module
2. securestore/register.go: Declares keychainSharing capability
3. scaffold/scaffold.go: Copies embedded capability Swift files to Tuist/ProjectDescriptionHelpers/ during init
4. scaffold/app_capabilities.go: GenerateAppCapabilities() + AddToAppCapabilities() (idempotent)
5. Project.swift.tmpl: entitlements now use EntitlementsFactory.make() instead of .file(path:)
6. setup_command.go: Applies module capabilities after external deps
Files changed:
- internal/registry/registry.go
- internal/securestore/register.go
- internal/scaffold/scaffold.go
- internal/scaffold/app_capabilities.go (new)
- internal/scaffold/app_capabilities_test.go (new)
- internal/template/tuist/Project.swift.tmpl
- internal/cli/setup_command.go
- internal/cli/ioc_test.go (writeProjectScaffold helper)
- internal/cli/init_test.go
- internal/scaffold/scaffold_test.go
- internal/registry/registry_test.go
- testdata/golden/project-swift.golden
make test: PASS
make lint: PASS

## Precondition Resources
(none)

## Outcome Resources
(none)
