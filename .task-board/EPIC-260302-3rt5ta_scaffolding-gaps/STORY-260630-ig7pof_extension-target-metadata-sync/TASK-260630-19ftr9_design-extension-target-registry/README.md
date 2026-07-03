# TASK-260630-19ftr9: design-extension-target-registry

## Description
Define the target registry/discovery shape consumed by project-config plugins for host app, app-like targets, test targets, and extension targets.

## Scope
Define reusable extension target discovery for project-config sync plugins. Prefer filesystem discovery of scaffold-owned Extensions/*/Project.swift plus existing host/nested app manifest discovery; do not make concrete extension plugins call metadata sync code.

## Acceptance Criteria
There is one shared discovery/helper path for extension Project.swift manifests; versions/min-target/team-id/build-flags/application-configuration use the same target set where applicable; extension plugins only create/register scaffold shape and do not duplicate metadata propagation.
