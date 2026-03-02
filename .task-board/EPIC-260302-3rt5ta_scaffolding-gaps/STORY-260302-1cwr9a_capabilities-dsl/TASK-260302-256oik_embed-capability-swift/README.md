# TASK-260302-256oik: embed-capability-swift

## Description
Port Capability DSL Swift files from tuist-akme into our scaffold pipeline as embedded Go files (embed.FS).

Files to port (from /Users/alexis/src/relux-works/tuist-akme/TuistPlugins/ProjectInfraPlugin/ProjectDescriptionHelpers/):
1. Capability.swift (526 lines) — full capability type system
2. EntitlementsFactory.swift — converts [Capability] to Tuist Entitlements
3. Capability+PortalCapability.swift — 118 Apple portal capabilities

ALSO add Capability struct to registry.go (same pattern as ExternalDep):

type Capability struct {
    Type string            // keychainSharing, appGroups, pushNotifications, etc.
    Args map[string]string // optional params (e.g. group ID for appGroups)
}

Add Capabilities []Capability field to Module struct.

Then populate each modules register.go with its needed capabilities:
- securestore: keychainSharing
- scaffold (init): appGroups (if config has app_groups)
- Others: empty for now

Place Swift files in internal/scaffold/capability_files/ with embed.FS.

Adaptations from tuist-akme:
- Remove Namespacing enum and environment suffix logic
- Remove ConfigurationHelper references
- Simplify Identifier: keep .default and .custom(id: String), remove .shared
- Keep all capability kinds and portal capabilities intact
- Keep EntitlementsFactory multi-platform validation
- Keep customEntitlements escape hatch

Tests: embed_test.go verifying files + registry_test.go for new Capability field.
Run make test to verify.

## Scope
(define task scope)

## Acceptance Criteria
(define acceptance criteria)
