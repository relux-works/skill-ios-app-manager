# TASK-260302-35m022: scaffold-capabilities-integration

## Description
Integrate capability Swift files into scaffold pipeline and update Project.swift template.

PREREQ — Capability struct in registry.go (skipped by embed task):

Add to internal/registry/registry.go, same pattern as ExternalDep:

// Capability describes a Tuist capability declared by a module.
type Capability struct {
    Type string            // keychainSharing, appGroups, pushNotifications, etc.
    Args map[string]string // optional params: e.g. {"group": "group.xxx"} for appGroups
}

Add field to Module struct: Capabilities []Capability

Populate in existing register.go files:
- securestore/register.go: keychainSharing
- scaffold init: appGroups (if config has app_groups)
- Others: empty for now

APPROACH: Capabilities follow ExternalDeps pattern — declared in each modules register.go, applied generically by setup_command.go.

1. Update setup_command.go to handle mod.Capabilities:
   - After external deps loop, add capability application loop
   - For each Capability, call AddToAppCapabilities() to patch AppCapabilities.swift
   - Idempotent: check if already present, skip if yes

2. Update scaffold pipeline (init):
   - Copy embedded Swift files (from internal/scaffold/capability_files/ embed.FS) to Tuist/ProjectDescriptionHelpers/
   - Generate initial AppCapabilities.swift with base capabilities from config (app_groups -> .appGroups())
   - Old entitlements generation already removed (done in TASK-260302-2ufnu7)

3. Update Project.swift.tmpl:
   - Replace: entitlements: .file(path: "AppName.entitlements")
   - With: entitlements: EntitlementsFactory.make(hostBundleId: bundleId, destinations: .iOS, capabilities: AppCapabilities.app)
   - Add import for AppCapabilities if needed (it is a ProjectDescriptionHelper, so auto-available)

4. Write AddToAppCapabilities function (in internal/capabilities/ or internal/scaffold/):
   - Read AppCapabilities.swift, find insertion point, add capability line
   - Map Capability.Type to Swift DSL: keychainSharing -> .keychainSharing(), appGroups -> .appGroups(group: .custom(id: Value("..."))), pushNotifications -> .pushNotifications(environment: .production)
   - Idempotent: string-check before inserting

5. Update registry_test.go for new Capability field.
6. Run make test to verify no regressions.
7. Run make build + demo app rebuild to verify Swift compiles.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
