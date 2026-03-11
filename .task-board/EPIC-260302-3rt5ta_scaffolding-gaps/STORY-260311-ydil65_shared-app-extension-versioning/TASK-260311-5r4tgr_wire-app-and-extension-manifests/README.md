# TASK-260311-5r4tgr: wire-app-and-extension-manifests

## Description
Update app and app extension manifests/templates so every generated target reads version/build values from the shared source instead of per-target literals.

## Scope
Patch the main app Project.swift template and every generated app extension Project.swift path to reference the shared version/build definition. Cover existing extension types and the default path for newly scaffolded extensions so future .appex targets inherit the shared values without manual edits.

## Acceptance Criteria
App Project.swift consumes the shared version/build source; extension Project.swift templates consume the same source; newly scaffolded extensions inherit shared values by default; scaffold code makes it structurally hard for new extensions to introduce their own independent version/build values.
